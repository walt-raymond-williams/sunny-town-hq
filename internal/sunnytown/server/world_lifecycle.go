package server

import (
	"context"
	"log"
	"math"
	"time"

	stmaps "hq/internal/sunnytown/maps"
	stnavigation "hq/internal/sunnytown/navigation"
	"hq/internal/sunnytownauth"
)

func newWorld(roomID string, maps map[string]gameMap) *world {
	return newWorldWithNPCDayLength(roomID, maps, defaultNPCScheduleDayLength)
}

func newWorldWithNPCDayLength(roomID string, maps map[string]gameMap, npcDayLength time.Duration) *world {
	if npcDayLength <= 0 {
		npcDayLength = defaultNPCScheduleDayLength
	}
	rewardEvents := make(chan rewardEvent, 32)
	resourceEvents := make(chan resourceEvent, resourceCommitQueueSize)
	npcJobEvents := make(chan npcJobProductionEvent, npcJobProductionQueueSize)
	navigationGraph, err := stnavigation.NewGraph(maps)
	if err != nil {
		log.Printf("build sunny town navigation graph: %v", err)
	}
	created := &world{
		roomID:         roomID,
		rooms:          map[string]*room{},
		navigation:     navigationGraph,
		npcCharacters:  map[string]npcCharacter{},
		rewardEvents:   rewardEvents,
		resourceEvents: resourceEvents,
		npcJobEvents:   npcJobEvents,
		npcDayLength:   npcDayLength,
	}
	for _, gameMap := range maps {
		created.rooms[gameMap.ID] = newRoom(roomID, gameMap, rewardEvents, resourceEvents, npcJobEvents, created)
	}
	created.defaultRoom = created.rooms[defaultMapID]
	created.configureNPCRoutineAnchors()
	return created
}

func (world *world) scheduleDayLength() time.Duration {
	if world == nil || world.npcDayLength <= 0 {
		return defaultNPCScheduleDayLength
	}
	return world.npcDayLength
}

func (room *room) scheduleDayLength() time.Duration {
	if room == nil {
		return defaultNPCScheduleDayLength
	}
	return room.world.scheduleDayLength()
}

func newRoom(id string, gameMap gameMap, rewardEvents chan rewardEvent, resourceEvents chan resourceEvent, npcJobEvents chan npcJobProductionEvent, world *world) *room {
	resourceNodes := initialResourceNodes(gameMap)
	fixtureObjects := initialFixtureObjects(gameMap)
	worldObjects := map[string]*worldObject{}
	for _, node := range resourceNodes {
		worldObjects[worldObjectKey(node.source, node.id)] = node
	}
	for _, object := range fixtureObjects {
		worldObjects[worldObjectKey(object.source, object.id)] = object
	}
	room := &room{
		id:             id,
		gameMap:        gameMap,
		players:        map[string]*player{},
		liveNPCs:       initialLiveNPCs(gameMap),
		collectibles:   initialCollectibles(gameMap),
		resourceNodes:  resourceNodes,
		placedObjects:  map[string]*placedObject{},
		worldObjects:   worldObjects,
		rewardRunID:    newRewardRunID(),
		rewardEvents:   rewardEvents,
		resourceEvents: resourceEvents,
		npcJobEvents:   npcJobEvents,
		world:          world,
	}
	room.configureNPCBehaviorLocked()
	return room
}

func (world *world) join(client *client, claims sunnytownauth.Claims, equipment equipmentSnapshot, position studentPositionResponse) {
	target := world.rooms[claims.MapID]
	if target == nil {
		target = world.defaultRoom
	}
	target.join(client, claims, equipment, position)
}

func (room *room) run(ctx context.Context) {
	ticker := time.NewTicker(simulationInterval)
	defer ticker.Stop()

	lastSnapshot := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			room.step(simulationInterval.Seconds(), now)
			if now.Sub(lastSnapshot) >= snapshotInterval {
				room.broadcastSnapshot(now)
				lastSnapshot = now
			}
		}
	}
}

func (room *room) join(client *client, claims sunnytownauth.Claims, equipment equipmentSnapshot, position studentPositionResponse) {
	room.mu.Lock()
	defer room.mu.Unlock()

	if len(room.players) == 0 {
		room.catchUpNPCsAfterNoPlayersLocked(time.Now())
	}

	spawn := room.spawnPointLocked()
	x := spawn.X
	y := spawn.Y
	facing := "down"
	if position.Found && position.RoomID == room.id && position.MapID == room.gameMap.ID {
		x = room.clampX(position.X)
		y = room.clampY(position.Y)
		if stmaps.IsFacing(position.Facing) {
			facing = position.Facing
		}
	}
	player := &player{
		appUserID:   claims.AppUserID,
		characterID: claims.CharacterID,
		id:          client.id,
		displayName: claims.DisplayName,
		avatarID:    claims.AvatarID,
		equipment:   equipment,
		inventory:   inventoryFromEquipment(equipment),
		x:           x,
		y:           y,
		facing:      facing,
		lastMoveAt:  time.Now(),
		client:      client,
	}
	if player.displayName == "" {
		player.displayName = "Student"
	}
	if player.avatarID == "" {
		player.avatarID = "pet-default"
	}
	room.players[player.id] = player
	room.npcPausedAt = time.Time{}
	client.setRoom(room)

	mapSnapshot := room.mapSnapshotLocked()
	client.send <- serverMessage{
		Type:          "hello",
		SelfID:        player.id,
		RoomID:        room.id,
		MapID:         room.gameMap.ID,
		Map:           &mapSnapshot,
		Players:       room.snapshotsLocked(),
		NPCs:          room.npcSnapshotsLocked(),
		Collectibles:  room.collectibleSnapshotsLocked(),
		ResourceNodes: room.resourceNodeSnapshotsLocked(),
		PlacedObjects: room.placedObjectSnapshotsLocked(),
		WorldObjects:  room.worldObjectSnapshotsLocked(),
	}
}

func (room *room) leave(client *client) {
	var saved *studentPositionRequest
	now := time.Now()
	room.mu.Lock()
	if existing := room.players[client.id]; existing != nil && existing.client == client {
		delete(room.players, client.id)
		if len(room.players) == 0 {
			room.npcPausedAt = now
		}
		saved = &studentPositionRequest{
			AppUserID: existing.appUserID,
			RoomID:    room.id,
			MapID:     room.gameMap.ID,
			X:         math.Round(existing.x*10) / 10,
			Y:         math.Round(existing.y*10) / 10,
			Facing:    existing.facing,
		}
		log.Printf("player left room=%s player=%s", room.id, client.id)
	}
	room.mu.Unlock()
	if saved != nil && client.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := client.server.hq.SaveStudentPosition(ctx, *saved); err != nil {
			log.Printf("save sunny town position player=%s map=%s: %v", client.id, saved.MapID, err)
		}
	}
}

func (world *world) leave(client *client) {
	if room := client.currentRoom(); room != nil {
		room.leave(client)
	}
}

func (world *world) transferPlayer(sourceMapID string, playerID string, usedPortal portal, seq int64, now time.Time) {
	world.transferMu.Lock()
	defer world.transferMu.Unlock()

	source := world.rooms[sourceMapID]
	target := world.rooms[usedPortal.TargetMapID]
	if source == nil || target == nil {
		return
	}

	source.mu.Lock()
	player := source.players[playerID]
	if player == nil {
		source.mu.Unlock()
		return
	}
	if player.client.currentRoom() != source {
		source.mu.Unlock()
		return
	}
	delete(source.players, playerID)
	if len(source.players) == 0 {
		source.npcPausedAt = now
	}
	source.mu.Unlock()

	player.x = target.clampX(usedPortal.TargetX)
	player.y = target.clampY(usedPortal.TargetY)
	if stmaps.IsFacing(usedPortal.TargetFacing) {
		player.facing = usedPortal.TargetFacing
	}
	player.moving = false
	player.lastMoveAt = now
	player.lastMoveSeq = seq
	player.portalLocked = true

	target.mu.Lock()
	if len(target.players) == 0 {
		target.catchUpNPCsAfterNoPlayersLocked(now)
	}
	target.players[playerID] = player
	target.npcPausedAt = time.Time{}
	player.client.setRoom(target)
	mapSnapshot := target.mapSnapshotLocked()
	message := serverMessage{
		Type:          "map_changed",
		SelfID:        player.id,
		RoomID:        target.id,
		MapID:         target.gameMap.ID,
		Map:           &mapSnapshot,
		Players:       target.snapshotsLocked(),
		NPCs:          target.npcSnapshotsLocked(),
		Collectibles:  target.collectibleSnapshotsLocked(),
		ResourceNodes: target.resourceNodeSnapshotsLocked(),
		PlacedObjects: target.placedObjectSnapshotsLocked(),
		WorldObjects:  target.worldObjectSnapshotsLocked(),
		ServerTimeMS:  now.UnixMilli(),
		Tick:          target.tick,
	}
	target.mu.Unlock()

	player.client.trySend(message)
	source.broadcastSnapshot(now)
	target.broadcastSnapshot(now)
}

func (client *client) currentRoom() *room {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.room
}

func (client *client) setRoom(room *room) {
	client.mu.Lock()
	client.room = room
	client.mu.Unlock()
}
