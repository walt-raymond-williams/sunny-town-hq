package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	stconfig "hq/internal/sunnytown/config"
	stmaps "hq/internal/sunnytown/maps"
	"hq/internal/sunnytownauth"

	"github.com/gorilla/websocket"
)

const (
	defaultRoomID           = "sunny-town-main"
	defaultMapID            = "sunny-town-v1"
	equipmentSlotGear       = "gear"
	equipmentSlotAccessory  = "accessory"
	equipmentSlotTool       = "tool"
	playerSize              = 28.0
	playerSpeed             = 150.0
	starPickupRadius        = 30.0
	resourceCommitQueueSize = 32
	resourceHitsRequired    = 3
	resourceToolCooldown    = 500 * time.Millisecond
	simulationInterval      = 50 * time.Millisecond
	snapshotInterval        = 100 * time.Millisecond
	movingStateTTL          = 250 * time.Millisecond
	starRespawnDelay        = 10 * time.Second
)

const (
	worldObjectSourceNatural = "natural"
	worldObjectSourcePlaced  = "placed"

	worldObjectKindRockNode   = "rock_node"
	worldObjectKindStoneBlock = "stone_block"
)

func main() {
	cfg := stconfig.Load()
	maps, err := stmaps.LoadMaps(cfg.MapsDir)
	if err != nil {
		log.Fatalf("load maps: %v", err)
	}
	if _, ok := maps[defaultMapID]; !ok {
		log.Fatalf("default map %q was not loaded", defaultMapID)
	}

	world := newWorld(defaultRoomID, maps)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	for _, room := range world.rooms {
		go room.run(ctx)
	}

	srv := newServer(cfg, world)
	if err := srv.loadInitialMapObjects(ctx); err != nil {
		log.Printf("load initial sunny town map objects: %v", err)
	}
	go srv.runRewardWorker(ctx)
	go srv.runResourceWorker(ctx)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/sunny-town/ws", srv.handleWebSocket)

	httpServer := &http.Server{
		Addr:              cfg.Host + ":" + cfg.Port,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	log.Printf("Sunny Town listening on http://%s:%s", cfg.Host, cfg.Port)
	if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped: %v", err)
	}
}

func (client *client) refreshEquipment() {
	room := client.currentRoom()
	if room == nil || client.server == nil {
		return
	}

	room.mu.Lock()
	player := room.players[client.id]
	if player == nil {
		room.mu.Unlock()
		return
	}
	userID := player.appUserID
	room.mu.Unlock()

	equipment, err := client.server.hq.LoadStudentEquipment(context.Background(), userID)
	if err != nil {
		log.Printf("refresh equipment player=%s: %v", client.id, err)
		client.trySend(serverMessage{Type: "error", Code: "equipment_refresh_failed"})
		return
	}

	room.mu.Lock()
	if player := room.players[client.id]; player != nil {
		player.equipment = equipment
	}
	room.mu.Unlock()
	room.broadcastSnapshot(time.Now())
}

func (client *client) ownsInventoryItem(ctx context.Context, appUserID int64, itemKey string) (bool, error) {
	itemKey = strings.TrimSpace(itemKey)
	if itemKey == "" || appUserID < 1 {
		return false, nil
	}
	if client.server != nil {
		quantity, err := client.server.hq.LoadStudentInventoryQuantity(ctx, appUserID, itemKey)
		if err != nil {
			return false, err
		}
		return quantity > 0, nil
	}

	room := client.currentRoom()
	if room == nil {
		return false, nil
	}
	room.mu.Lock()
	defer room.mu.Unlock()
	player := room.players[client.id]
	if player == nil {
		return false, nil
	}
	return player.inventory[itemKey] > 0, nil
}

func (client *client) handleToolUse(message clientMessage) {
	toolKey := strings.TrimSpace(message.ToolKey)
	if toolKey == "" {
		client.trySend(serverMessage{Type: "error", Code: "missing_tool"})
		return
	}

	room := client.currentRoom()
	if room == nil {
		client.trySend(serverMessage{Type: "error", Code: "not_in_room"})
		return
	}

	room.mu.Lock()
	player := room.players[client.id]
	if player == nil {
		room.mu.Unlock()
		client.trySend(serverMessage{Type: "error", Code: "player_not_found"})
		return
	}
	appUserID := player.appUserID
	room.mu.Unlock()

	if toolKey != "pickaxe" {
		return
	}
	ownsTool, err := client.ownsInventoryItem(context.Background(), appUserID, toolKey)
	if err != nil {
		log.Printf("validate tool ownership player=%s tool=%s: %v", client.id, toolKey, err)
		client.trySend(serverMessage{Type: "error", Code: "tool_validation_failed"})
		return
	}
	if !ownsTool {
		client.trySend(serverMessage{Type: "error", Code: "tool_not_owned"})
		return
	}

	var event *resourceEvent
	var removeRequest *removeMapObjectRequest
	var removeTargetID string
	now := time.Now()
	room.mu.Lock()
	player = room.players[client.id]
	if player == nil {
		room.mu.Unlock()
		client.trySend(serverMessage{Type: "error", Code: "player_not_found"})
		return
	}
	if !player.lastToolUseAt.IsZero() && now.Sub(player.lastToolUseAt) < resourceToolCooldown {
		room.mu.Unlock()
		client.trySend(serverMessage{Type: "error", Code: "tool_cooldown"})
		return
	}
	target := room.nearestBreakableWorldObjectLocked(player, toolKey)
	if target == nil {
		room.mu.Unlock()
		client.trySend(serverMessage{Type: "error", Code: "resource_node_not_found"})
		return
	}
	if target.source == worldObjectSourcePlaced && client.server == nil {
		room.mu.Unlock()
		client.trySend(serverMessage{Type: "error", Code: "map_object_remove_failed"})
		return
	}
	player.lastToolUseAt = now
	target.hitCount++
	if target.hitCount >= target.hitsRequired {
		switch target.source {
		case worldObjectSourcePlaced:
			removeTargetID = target.id
			removeRequest = &removeMapObjectRequest{
				AppUserID: player.appUserID,
				RoomID:    room.id,
				MapID:     room.gameMap.ID,
				GridX:     target.gridX,
				GridY:     target.gridY,
			}
		case worldObjectSourceNatural:
			target.active = false
			target.hitCount = 0
			target.harvestSeq++
			target.respawnAt = now.Add(target.respawnDelay)
			resourceKey, amount := rollMiningDrop()
			eventID := fmt.Sprintf("%s:%s:%d:%d", room.gameMap.ID, target.id, target.harvestSeq, player.appUserID)
			event = &resourceEvent{
				eventID:     eventID,
				appUserID:   player.appUserID,
				roomID:      room.id,
				mapID:       room.gameMap.ID,
				nodeID:      target.id,
				resourceKey: resourceKey,
				amount:      amount,
				client:      player.client,
			}
		}
	}
	room.mu.Unlock()

	if removeRequest != nil {
		client.removePlacedWorldObject(room, *removeRequest, removeTargetID, now)
		return
	}
	room.broadcastSnapshot(now)
	if event == nil {
		return
	}
	select {
	case room.resourceEvents <- *event:
	default:
		log.Printf("resource queue full event=%s", event.eventID)
		client.trySend(serverMessage{
			Type:   "resource_failed",
			NodeID: event.nodeID,
			Reason: "temporary_error",
		})
	}
}

func (client *client) removePlacedWorldObject(room *room, request removeMapObjectRequest, targetID string, now time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	removed, quantity, err := client.server.removeMapObject(ctx, request, room.gameMap)
	cancel()
	if err != nil {
		log.Printf("remove placed object player=%s object=%s: %v", client.id, targetID, err)
		client.trySend(serverMessage{Type: "error", Code: "map_object_remove_failed"})
		return
	}

	var snapshot placedObjectSnapshot
	var worldSnapshot worldObjectSnapshot
	room.mu.Lock()
	room.removePlacedObjectLocked(removed.id)
	snapshot = removed.snapshot()
	worldSnapshot = removed.worldObjectSnapshot()
	room.mu.Unlock()

	room.broadcastSnapshot(now)
	room.broadcast(serverMessage{
		Type:         "map_object_removed",
		MapID:        room.gameMap.ID,
		PlacedObject: &snapshot,
		WorldObject:  &worldSnapshot,
	})
	client.trySend(serverMessage{
		Type:         "map_object_removed",
		MapID:        room.gameMap.ID,
		PlacedObject: &snapshot,
		WorldObject:  &worldSnapshot,
		ResourceKey:  removed.itemKey,
		Quantity:     quantity,
	})
}

func (client *client) handlePlaceObject(message clientMessage) {
	itemKey := strings.TrimSpace(message.ItemKey)
	if itemKey != "stone_block" {
		client.trySend(serverMessage{Type: "error", Code: "unsupported_map_object"})
		return
	}
	room := client.currentRoom()
	if room == nil {
		client.trySend(serverMessage{Type: "error", Code: "not_in_room"})
		return
	}
	if client.server == nil {
		client.trySend(serverMessage{Type: "error", Code: "map_object_place_failed"})
		return
	}

	var request placeMapObjectRequest
	room.mu.Lock()
	player := room.players[client.id]
	if player == nil {
		room.mu.Unlock()
		client.trySend(serverMessage{Type: "error", Code: "player_not_found"})
		return
	}
	if !room.canPlaceObjectLocked(message.GridX, message.GridY, itemKey) {
		room.mu.Unlock()
		client.trySend(serverMessage{Type: "error", Code: "invalid_map_object_location"})
		return
	}
	request = placeMapObjectRequest{
		AppUserID: player.appUserID,
		RoomID:    room.id,
		MapID:     room.gameMap.ID,
		GridX:     message.GridX,
		GridY:     message.GridY,
		ItemKey:   itemKey,
	}
	room.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	placed, quantity, err := client.server.placeMapObject(ctx, request, room.gameMap)
	cancel()
	if err != nil {
		log.Printf("place object player=%s map=%s grid=(%d,%d): %v", client.id, room.gameMap.ID, message.GridX, message.GridY, err)
		client.trySend(serverMessage{Type: "error", Code: "map_object_place_failed"})
		return
	}

	room.mu.Lock()
	if !room.canPlaceObjectLocked(placed.gridX, placed.gridY, placed.itemKey) {
		deleteRequest := removeMapObjectRequest{
			AppUserID: request.AppUserID,
			RoomID:    request.RoomID,
			MapID:     request.MapID,
			GridX:     request.GridX,
			GridY:     request.GridY,
		}
		room.mu.Unlock()
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, _, _ = client.server.removeMapObject(cleanupCtx, deleteRequest, room.gameMap)
		cleanupCancel()
		client.trySend(serverMessage{Type: "error", Code: "invalid_map_object_location"})
		return
	}
	room.addPlacedObjectLocked(placed)
	snapshot := placed.snapshot()
	worldSnapshot := placed.worldObjectSnapshot()
	room.mu.Unlock()

	room.broadcastSnapshot(time.Now())
	room.broadcast(serverMessage{
		Type:         "map_object_placed",
		MapID:        room.gameMap.ID,
		PlacedObject: &snapshot,
		WorldObject:  &worldSnapshot,
	})
	client.trySend(serverMessage{
		Type:         "map_object_placed",
		MapID:        room.gameMap.ID,
		PlacedObject: &snapshot,
		WorldObject:  &worldSnapshot,
		ResourceKey:  placed.itemKey,
		Quantity:     quantity,
	})
}

func (room *room) step(dt float64, now time.Time) {
	room.mu.Lock()

	room.tick++
	var rewards []rewardEvent
	for _, player := range room.players {
		if !player.lastMoveAt.IsZero() && now.Sub(player.lastMoveAt) > movingStateTTL {
			player.moving = false
		}

		rewards = append(rewards, room.collectStarsLocked(player, now)...)
	}

	room.respawnCollectiblesLocked(now)
	room.respawnResourceNodesLocked(now)
	room.mu.Unlock()

	for _, reward := range rewards {
		select {
		case room.rewardEvents <- reward:
		default:
			log.Printf("reward queue full event=%s", reward.eventID)
			reward.client.trySend(serverMessage{
				Type:          "reward_failed",
				CollectibleID: reward.collectibleID,
				Reason:        "temporary_error",
			})
		}
	}
}

func (room *room) nearestBreakableWorldObjectLocked(player *player, toolKey string) *worldObject {
	var nearest *worldObject
	nearestDistance := math.MaxFloat64
	for _, object := range room.worldObjects {
		if !object.active || !object.breakable || object.toolKey != toolKey {
			continue
		}
		centerX, centerY := object.center()
		distance := math.Hypot(player.x-centerX, player.y-centerY)
		if distance > object.interactionRadius || distance >= nearestDistance {
			continue
		}
		nearest = object
		nearestDistance = distance
	}
	return nearest
}

func (room *room) collectStarsLocked(player *player, now time.Time) []rewardEvent {
	rewards := []rewardEvent{}
	for _, collectible := range room.collectibles {
		if !collectible.active || collectible.kind != "star" {
			continue
		}
		if math.Hypot(player.x-collectible.x, player.y-collectible.y) > starPickupRadius {
			continue
		}

		collectible.active = false
		collectible.spawnSeq++
		collectible.respawnAt = now.Add(starRespawnDelay)
		eventID := fmt.Sprintf("%s:%s:%s:%d:%d", room.id, room.rewardRunID, collectible.id, collectible.spawnSeq, player.appUserID)
		rewards = append(rewards, rewardEvent{
			eventID:       eventID,
			appUserID:     player.appUserID,
			roomID:        room.id,
			mapID:         room.gameMap.ID,
			collectibleID: collectible.id,
			kind:          collectible.kind,
			amount:        1,
			client:        player.client,
		})
	}
	return rewards
}

func (room *room) respawnCollectiblesLocked(now time.Time) {
	for _, collectible := range room.collectibles {
		if collectible.active || collectible.respawnAt.IsZero() || now.Before(collectible.respawnAt) {
			continue
		}
		collectible.active = true
		collectible.respawnAt = time.Time{}
	}
}

func (room *room) respawnResourceNodesLocked(now time.Time) {
	for _, node := range room.resourceNodes {
		if node.active || node.respawnAt.IsZero() || now.Before(node.respawnAt) {
			continue
		}
		node.active = true
		node.hitCount = 0
		node.respawnAt = time.Time{}
	}
}

func (room *room) canPlaceObjectLocked(gridX int, gridY int, itemKey string) bool {
	if itemKey != "stone_block" || gridX < 0 || gridY < 0 {
		return false
	}
	if gridX >= room.gameMap.Width || gridY >= room.gameMap.Height {
		return false
	}
	objectRect := gridRect(room.gameMap, gridX, gridY)
	for _, blocked := range room.gameMap.BlockedRects {
		if rectsOverlap(objectRect, blocked) {
			return false
		}
	}
	for _, portal := range room.gameMap.Portals {
		if rectsOverlap(objectRect, rect{X: portal.X, Y: portal.Y, Width: portal.Width, Height: portal.Height}) {
			return false
		}
	}
	for _, npc := range room.gameMap.NPCs {
		if rectsOverlap(objectRect, rect{X: npc.X - playerSize/2, Y: npc.Y - playerSize/2, Width: playerSize, Height: playerSize}) {
			return false
		}
	}
	for _, object := range room.worldObjects {
		if object.reservesPlacement && rectsOverlap(objectRect, object.rect()) {
			return false
		}
	}
	for _, player := range room.players {
		if rectsOverlap(objectRect, rect{X: player.x - playerSize/2, Y: player.y - playerSize/2, Width: playerSize, Height: playerSize}) {
			return false
		}
	}
	return true
}

func (client *client) readPump() {
	defer func() {
		if room := client.currentRoom(); room != nil {
			room.leave(client)
		}
		_ = client.conn.Close()
	}()

	client.conn.SetReadLimit(1024)
	_ = client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		return client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	for {
		var message clientMessage
		if err := client.conn.ReadJSON(&message); err != nil {
			if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("message decode error player=%s: %v", client.id, err)
			}
			return
		}

		switch message.Type {
		case "move":
			if !client.allowMove(time.Now(), message.Moving) {
				continue
			}
			if room := client.currentRoom(); room != nil {
				room.updateMove(client.id, message.Seq, message.X, message.Y, message.Facing, message.Moving, time.Now())
			}
		case "equipment_changed":
			client.refreshEquipment()
		case "tool_use":
			client.handleToolUse(message)
		case "place_object":
			client.handlePlaceObject(message)
		case "ping":
			_ = client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		default:
			client.trySend(serverMessage{Type: "error", Code: "invalid_message"})
		}
	}
}

func (client *client) writePump() {
	pingTicker := time.NewTicker(30 * time.Second)
	defer func() {
		pingTicker.Stop()
		_ = client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			_ = client.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if !ok {
				_ = client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := client.conn.WriteJSON(message); err != nil {
				return
			}
		case <-pingTicker.C:
			_ = client.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (client *client) trySend(message serverMessage) {
	select {
	case client.send <- message:
	default:
	}
}

func (client *client) allowMove(now time.Time, moving bool) bool {
	if !moving {
		return true
	}
	if client.inputWindowStart.IsZero() || now.Sub(client.inputWindowStart) >= time.Second {
		client.inputWindowStart = now
		client.inputWindowCount = 0
	}
	client.inputWindowCount++
	return client.inputWindowCount <= 30
}

func (srv *server) runRewardWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-srv.world.rewardEvents:
			response, err := srv.commitRewardWithRetry(ctx, event)
			if err != nil {
				log.Printf("reward commit failed event=%s player=%d: %v", event.eventID, event.appUserID, err)
				event.client.trySend(serverMessage{
					Type:          "reward_failed",
					CollectibleID: event.collectibleID,
					Reason:        "temporary_error",
				})
				continue
			}
			log.Printf("reward commit success event=%s player=%d duplicate=%v", event.eventID, event.appUserID, response.Duplicate)
			event.client.trySend(serverMessage{
				Type:           "reward_committed",
				EventID:        event.eventID,
				Kind:           event.kind,
				Amount:         event.amount,
				NewStarBalance: response.NewStarBalance,
			})
		}
	}
}

func (srv *server) commitRewardWithRetry(ctx context.Context, event rewardEvent) (rewardCommitResponse, error) {
	backoffs := []time.Duration{100 * time.Millisecond, 250 * time.Millisecond, 500 * time.Millisecond}
	var lastErr error
	for attempt := 0; attempt <= len(backoffs); attempt++ {
		response, err := srv.commitReward(ctx, event)
		if err == nil {
			return response, nil
		}
		lastErr = err
		if attempt == len(backoffs) {
			break
		}
		select {
		case <-ctx.Done():
			return rewardCommitResponse{}, ctx.Err()
		case <-time.After(backoffs[attempt]):
		}
	}
	return rewardCommitResponse{}, lastErr
}

func (srv *server) commitReward(ctx context.Context, event rewardEvent) (rewardCommitResponse, error) {
	return srv.hq.CommitReward(ctx, rewardCommitRequest{
		EventID:       event.eventID,
		AppUserID:     event.appUserID,
		RoomID:        event.roomID,
		MapID:         event.mapID,
		CollectibleID: event.collectibleID,
		RewardKind:    event.kind,
		Amount:        event.amount,
	})
}

func initialCollectibles(gameMap gameMap) map[string]*collectible {
	collectibles := map[string]*collectible{}
	for index, spawn := range gameMap.StarSpawns {
		id := fmt.Sprintf("star-%04d", index+1)
		collectibles[id] = &collectible{
			id:       id,
			kind:     "star",
			x:        spawn.X,
			y:        spawn.Y,
			active:   true,
			spawnSeq: 0,
		}
	}
	return collectibles
}

func initialResourceNodes(gameMap gameMap) map[string]*resourceNode {
	nodes := map[string]*resourceNode{}
	for _, definition := range gameMap.ResourceNodes {
		nodes[definition.ID] = &resourceNode{
			id:                definition.ID,
			kind:              worldObjectKindRockNode,
			source:            worldObjectSourceNatural,
			resourceKind:      definition.Kind,
			mapID:             gameMap.ID,
			x:                 definition.X,
			y:                 definition.Y,
			radius:            definition.Radius,
			interactionRadius: definition.InteractionRadius,
			collision:         true,
			breakable:         true,
			toolKey:           "pickaxe",
			hitsRequired:      resourceHitsRequired,
			reservesPlacement: true,
			respawnDelay:      time.Duration(definition.RespawnSeconds) * time.Second,
			active:            true,
		}
	}
	return nodes
}

func placedObjectFromResponse(gameMap gameMap, response mapObjectResponse) *placedObject {
	objectRect := gridRect(gameMap, response.GridX, response.GridY)
	return &placedObject{
		id:                strconv.FormatInt(response.ID, 10),
		kind:              worldObjectKindStoneBlock,
		source:            worldObjectSourcePlaced,
		itemKey:           response.ItemKey,
		mapID:             response.MapID,
		gridX:             response.GridX,
		gridY:             response.GridY,
		x:                 objectRect.X,
		y:                 objectRect.Y,
		width:             objectRect.Width,
		height:            objectRect.Height,
		interactionRadius: 58,
		collision:         true,
		breakable:         true,
		toolKey:           "pickaxe",
		hitsRequired:      1,
		reservesPlacement: true,
		active:            true,
		placedByAppUserID: response.PlacedByAppUserID,
	}
}

func gridRect(gameMap gameMap, gridX int, gridY int) rect {
	tileSize := float64(gameMap.TileSize)
	return rect{
		X:      float64(gridX) * tileSize,
		Y:      float64(gridY) * tileSize,
		Width:  tileSize,
		Height: tileSize,
	}
}

func (object *placedObject) rect() rect {
	if object.radius > 0 {
		return rect{
			X:      object.x - object.radius,
			Y:      object.y - object.radius,
			Width:  object.radius * 2,
			Height: object.radius * 2,
		}
	}
	return rect{
		X:      object.x,
		Y:      object.y,
		Width:  object.width,
		Height: object.height,
	}
}

func (object *worldObject) center() (float64, float64) {
	if object.radius > 0 {
		return object.x, object.y
	}
	return object.x + object.width/2, object.y + object.height/2
}

func worldObjectKey(source string, id string) string {
	return source + ":" + id
}

func (room *room) setPlacedObjectsLocked(objects map[string]*placedObject) {
	for _, object := range room.placedObjects {
		delete(room.worldObjects, worldObjectKey(object.source, object.id))
	}
	room.placedObjects = objects
	for _, object := range objects {
		room.addPlacedObjectLocked(object)
	}
}

func (room *room) addPlacedObjectLocked(object *placedObject) {
	room.placedObjects[object.id] = object
	room.worldObjects[worldObjectKey(object.source, object.id)] = object
}

func (room *room) removePlacedObjectLocked(id string) {
	if object := room.placedObjects[id]; object != nil {
		delete(room.worldObjects, worldObjectKey(object.source, object.id))
	}
	delete(room.placedObjects, id)
}

func rollMiningDrop() (string, int) {
	roll := randomInt(100)
	if roll < 85 {
		return "rock", 1
	}
	if roll < 95 {
		return "rock", 2
	}
	return "crystal", 1
}

func randomInt(max int) int {
	if max <= 1 {
		return 0
	}
	var bytes [1]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return int(bytes[0]) % max
	}
	return int(time.Now().UnixNano() % int64(max))
}

func (srv *server) runResourceWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-srv.world.resourceEvents:
			response, err := srv.commitResourceWithRetry(ctx, event)
			if err != nil {
				log.Printf("resource commit failed event=%s player=%d: %v", event.eventID, event.appUserID, err)
				event.client.trySend(serverMessage{
					Type:   "resource_failed",
					NodeID: event.nodeID,
					Reason: "temporary_error",
				})
				continue
			}
			log.Printf("resource commit success event=%s player=%d duplicate=%v", event.eventID, event.appUserID, response.Duplicate)
			event.client.trySend(serverMessage{
				Type:        "resource_committed",
				EventID:     event.eventID,
				NodeID:      event.nodeID,
				ResourceKey: response.ResourceKey,
				Amount:      event.amount,
				Quantity:    response.Quantity,
			})
		}
	}
}

func (srv *server) commitResourceWithRetry(ctx context.Context, event resourceEvent) (resourceCommitResponse, error) {
	backoffs := []time.Duration{100 * time.Millisecond, 250 * time.Millisecond, 500 * time.Millisecond}
	var lastErr error
	for attempt := 0; attempt <= len(backoffs); attempt++ {
		response, err := srv.commitResource(ctx, event)
		if err == nil {
			return response, nil
		}
		lastErr = err
		if attempt == len(backoffs) {
			break
		}
		select {
		case <-ctx.Done():
			return resourceCommitResponse{}, ctx.Err()
		case <-time.After(backoffs[attempt]):
		}
	}
	return resourceCommitResponse{}, lastErr
}

func (srv *server) commitResource(ctx context.Context, event resourceEvent) (resourceCommitResponse, error) {
	return srv.hq.CommitResource(ctx, resourceCommitRequest{
		EventID:     event.eventID,
		AppUserID:   event.appUserID,
		Source:      "sunny_town_mining",
		RoomID:      event.roomID,
		MapID:       event.mapID,
		NodeID:      event.nodeID,
		ResourceKey: event.resourceKey,
		Amount:      event.amount,
	})
}

func newRewardRunID() string {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return hex.EncodeToString(bytes[:])
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func validateJoinTarget(claims sunnytownauth.Claims, roomID string, rooms map[string]*room) error {
	if claims.RoomID != roomID {
		return errors.New("unknown room or map")
	}
	if rooms[claims.MapID] == nil {
		return errors.New("unknown room or map")
	}
	return nil
}
