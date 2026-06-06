package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"hq/internal/sunnytownauth"

	"github.com/gorilla/websocket"
)

const (
	defaultRoomID          = "sunny-town-main"
	defaultMapID           = "sunny-town-v1"
	equipmentSlotGear      = "gear"
	equipmentSlotAccessory = "accessory"
	equipmentSlotTool      = "tool"
	playerSize             = 28.0
	playerSpeed            = 150.0
	starPickupRadius       = 30.0
	simulationInterval     = 50 * time.Millisecond
	snapshotInterval       = 100 * time.Millisecond
	movingStateTTL         = 250 * time.Millisecond
	starRespawnDelay       = 10 * time.Second
)

type config struct {
	host           string
	port           string
	joinSecret     string
	serviceSecret  string
	hqInternalURL  string
	allowedOrigins map[string]bool
	mapsDir        string
}

type gameMap struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	TileSize     int      `json:"tileSize"`
	Width        int      `json:"width"`
	Height       int      `json:"height"`
	Spawns       []point  `json:"spawns"`
	BlockedRects []rect   `json:"blockedRects"`
	StarSpawns   []point  `json:"starSpawns"`
	Portals      []portal `json:"portals"`
	NPCs         []npc    `json:"npcs"`
}

type point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type portal struct {
	ID           string  `json:"id"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Width        float64 `json:"width"`
	Height       float64 `json:"height"`
	TargetMapID  string  `json:"targetMapId"`
	TargetX      float64 `json:"targetX"`
	TargetY      float64 `json:"targetY"`
	TargetFacing string  `json:"targetFacing"`
}

type npc struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	X         float64   `json:"x"`
	Y         float64   `json:"y"`
	Facing    string    `json:"facing"`
	SpriteKey string    `json:"spriteKey"`
	Dialogue  []string  `json:"dialogue"`
	Shop      *shop     `json:"shop,omitempty"`
	Activity  *activity `json:"activity,omitempty"`
}

type activity struct {
	Type string `json:"type"`
}

type shop struct {
	ID    string     `json:"id"`
	Items []shopItem `json:"items"`
}

type shopItem struct {
	ItemKey     string `json:"itemKey"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceStars  int    `json:"priceStars"`
}

type clientMessage struct {
	Type         string  `json:"type"`
	Seq          int64   `json:"seq,omitempty"`
	ClientTimeMS int64   `json:"client_time_ms,omitempty"`
	X            float64 `json:"x,omitempty"`
	Y            float64 `json:"y,omitempty"`
	Facing       string  `json:"facing,omitempty"`
	Moving       bool    `json:"moving,omitempty"`
	ToolKey      string  `json:"toolKey,omitempty"`
}

type equipmentSnapshot map[string]string

type serverMessage struct {
	Type           string                `json:"type"`
	SelfID         string                `json:"selfId,omitempty"`
	RoomID         string                `json:"roomId,omitempty"`
	MapID          string                `json:"mapId,omitempty"`
	Map            *gameMap              `json:"map,omitempty"`
	Tick           int64                 `json:"tick,omitempty"`
	ServerTimeMS   int64                 `json:"serverTimeMs,omitempty"`
	Players        []playerSnapshot      `json:"players,omitempty"`
	Collectibles   []collectibleSnapshot `json:"collectibles,omitempty"`
	Code           string                `json:"code,omitempty"`
	EventID        string                `json:"eventId,omitempty"`
	Kind           string                `json:"kind,omitempty"`
	Amount         int                   `json:"amount,omitempty"`
	NewStarBalance int                   `json:"newStarBalance,omitempty"`
	CollectibleID  string                `json:"collectibleId,omitempty"`
	Reason         string                `json:"reason,omitempty"`
}

type playerSnapshot struct {
	ID               string            `json:"id"`
	DisplayName      string            `json:"displayName"`
	X                float64           `json:"x"`
	Y                float64           `json:"y"`
	Facing           string            `json:"facing"`
	Moving           bool              `json:"moving"`
	AvatarID         string            `json:"avatarId"`
	Equipment        equipmentSnapshot `json:"equipment"`
	LastProcessedSeq int64             `json:"lastProcessedSeq"`
}

type player struct {
	appUserID   int64
	id          string
	displayName string
	avatarID    string
	equipment   equipmentSnapshot
	x           float64
	y           float64
	facing      string
	moving      bool
	lastMoveAt  time.Time
	lastMoveSeq int64
	client      *client
}

type client struct {
	conn             *websocket.Conn
	send             chan serverMessage
	server           *server
	mu               sync.Mutex
	room             *room
	id               string
	inputWindowStart time.Time
	inputWindowCount int
}

type room struct {
	id      string
	gameMap gameMap

	mu           sync.Mutex
	players      map[string]*player
	collectibles map[string]*collectible
	tick         int64
	rewardRunID  string
	rewardEvents chan rewardEvent
	world        *world
}

type server struct {
	config   config
	world    *world
	client   *http.Client
	upgrader websocket.Upgrader
}

type world struct {
	roomID       string
	rooms        map[string]*room
	defaultRoom  *room
	rewardEvents chan rewardEvent
	transferMu   sync.Mutex
}

type collectible struct {
	id        string
	kind      string
	x         float64
	y         float64
	active    bool
	spawnSeq  int64
	respawnAt time.Time
}

type collectibleSnapshot struct {
	ID     string  `json:"id"`
	Kind   string  `json:"kind"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Active bool    `json:"active"`
}

type rewardEvent struct {
	eventID       string
	appUserID     int64
	roomID        string
	mapID         string
	collectibleID string
	kind          string
	amount        int
	client        *client
}

type rewardCommitRequest struct {
	EventID       string `json:"event_id"`
	AppUserID     int64  `json:"app_user_id"`
	RoomID        string `json:"room_id"`
	MapID         string `json:"map_id"`
	CollectibleID string `json:"collectible_id"`
	RewardKind    string `json:"reward_kind"`
	Amount        int    `json:"amount"`
}

type rewardCommitResponse struct {
	Accepted       bool `json:"accepted"`
	Duplicate      bool `json:"duplicate"`
	NewStarBalance int  `json:"new_star_balance"`
}

type studentEquipmentResponse struct {
	Slots []equipmentSlotResponse `json:"slots"`
}

type studentPositionResponse struct {
	Found     bool    `json:"found"`
	AppUserID int64   `json:"app_user_id"`
	RoomID    string  `json:"room_id"`
	MapID     string  `json:"map_id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Facing    string  `json:"facing"`
}

type studentPositionRequest struct {
	AppUserID int64   `json:"app_user_id"`
	RoomID    string  `json:"room_id"`
	MapID     string  `json:"map_id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Facing    string  `json:"facing"`
}

type equipmentSlotResponse struct {
	Slot string                 `json:"slot"`
	Item *equipmentItemResponse `json:"item"`
}

type equipmentItemResponse struct {
	VisualKey string `json:"visualKey"`
}

func main() {
	cfg := loadConfig()
	maps, err := loadMaps(cfg.mapsDir)
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
	go srv.runRewardWorker(ctx)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/sunny-town/ws", srv.handleWebSocket)

	httpServer := &http.Server{
		Addr:              cfg.host + ":" + cfg.port,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	log.Printf("Sunny Town listening on http://%s:%s", cfg.host, cfg.port)
	if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped: %v", err)
	}
}

func loadConfig() config {
	return config{
		host:           envOrDefault("SUNNY_TOWN_HOST", "0.0.0.0"),
		port:           envOrDefault("SUNNY_TOWN_PORT", "18082"),
		joinSecret:     envOrDefault("SUNNY_TOWN_JOIN_SECRET", "local-dev-secret"),
		serviceSecret:  envOrDefault("SUNNY_TOWN_SERVICE_SECRET", "local-dev-service-secret"),
		hqInternalURL:  strings.TrimRight(envOrDefault("HQ_INTERNAL_BASE_URL", "http://127.0.0.1:8080"), "/"),
		allowedOrigins: allowedOrigins(envOrDefault("SUNNY_TOWN_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:18080,http://127.0.0.1:5173,http://127.0.0.1:18080")),
		mapsDir:        envOrDefault("SUNNY_TOWN_MAPS_DIR", filepath.Join("sunny-town", "maps")),
	}
}

func newServer(cfg config, world *world) *server {
	srv := &server{
		config: cfg,
		world:  world,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
	srv.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			return origin == "" || cfg.allowedOrigins["*"] || cfg.allowedOrigins[origin]
		},
	}
	return srv
}

func (srv *server) internalRequest(ctx context.Context, method string, path string, body []byte) (*http.Response, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		method,
		srv.config.hqInternalURL+path,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("X-HQ-Service-Secret", srv.config.serviceSecret)
	return srv.client.Do(request)
}

func (srv *server) loadStudentEquipment(ctx context.Context, appUserID int64) (equipmentSnapshot, error) {
	response, err := srv.internalRequest(
		ctx,
		http.MethodGet,
		"/api/internal/sunny-town/student-equipment?app_user_id="+strconv.FormatInt(appUserID, 10),
		nil,
	)
	if err != nil {
		return equipmentSnapshot{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return equipmentSnapshot{}, fmt.Errorf("equipment request failed status=%d", response.StatusCode)
	}

	var equipment studentEquipmentResponse
	if err := json.NewDecoder(response.Body).Decode(&equipment); err != nil {
		return equipmentSnapshot{}, err
	}

	snapshot := equipmentSnapshot{}
	for _, slot := range equipment.Slots {
		if slot.Slot != equipmentSlotGear && slot.Slot != equipmentSlotAccessory && slot.Slot != equipmentSlotTool {
			continue
		}
		if slot.Item == nil || strings.TrimSpace(slot.Item.VisualKey) == "" {
			continue
		}
		snapshot[slot.Slot] = strings.TrimSpace(slot.Item.VisualKey)
	}
	return snapshot, nil
}

func (srv *server) loadStudentPosition(ctx context.Context, appUserID int64) (studentPositionResponse, error) {
	response, err := srv.internalRequest(
		ctx,
		http.MethodGet,
		"/api/internal/sunny-town/player-position?app_user_id="+strconv.FormatInt(appUserID, 10),
		nil,
	)
	if err != nil {
		return studentPositionResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return studentPositionResponse{}, fmt.Errorf("position request failed status=%d", response.StatusCode)
	}

	var position studentPositionResponse
	if err := json.NewDecoder(response.Body).Decode(&position); err != nil {
		return studentPositionResponse{}, err
	}
	return position, nil
}

func (srv *server) saveStudentPosition(ctx context.Context, position studentPositionRequest) error {
	body, err := json.Marshal(position)
	if err != nil {
		return err
	}
	response, err := srv.internalRequest(ctx, http.MethodPost, "/api/internal/sunny-town/player-position", body)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("save position failed status=%d", response.StatusCode)
	}
	return nil
}

func (srv *server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	claims, err := sunnytownauth.Verify(r.URL.Query().Get("token"), srv.config.joinSecret, time.Now())
	if err != nil {
		log.Printf("sunny town auth failed: %v", err)
		http.Error(w, "invalid sunny town token", http.StatusUnauthorized)
		return
	}
	if err := validateJoinTarget(claims, srv.world.roomID, srv.world.rooms); err != nil {
		log.Printf("sunny town auth failed: unknown room=%q map=%q", claims.RoomID, claims.MapID)
		http.Error(w, "unknown room or map", http.StatusBadRequest)
		return
	}

	conn, err := srv.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade websocket: %v", err)
		return
	}

	playerID := sunnytownauth.PlayerID(claims.AppUserID)
	client := &client{
		conn:   conn,
		send:   make(chan serverMessage, 16),
		server: srv,
		id:     playerID,
	}

	equipment, err := srv.loadStudentEquipment(r.Context(), claims.AppUserID)
	if err != nil {
		log.Printf("load sunny town equipment: %v", err)
		equipment = equipmentSnapshot{}
	}

	position, err := srv.loadStudentPosition(r.Context(), claims.AppUserID)
	if err != nil {
		log.Printf("load sunny town position: %v", err)
		position = studentPositionResponse{}
	}

	srv.world.join(client, claims, equipment, position)
	log.Printf("player joined room=%s map=%s player=%s", srv.world.roomID, claims.MapID, playerID)

	go client.writePump()
	client.readPump()
}

func newWorld(roomID string, maps map[string]gameMap) *world {
	rewardEvents := make(chan rewardEvent, 32)
	created := &world{
		roomID:       roomID,
		rooms:        map[string]*room{},
		rewardEvents: rewardEvents,
	}
	for _, gameMap := range maps {
		created.rooms[gameMap.ID] = newRoom(roomID, gameMap, rewardEvents, created)
	}
	created.defaultRoom = created.rooms[defaultMapID]
	return created
}

func newRoom(id string, gameMap gameMap, rewardEvents chan rewardEvent, world *world) *room {
	return &room{
		id:           id,
		gameMap:      gameMap,
		players:      map[string]*player{},
		collectibles: initialCollectibles(gameMap),
		rewardRunID:  newRewardRunID(),
		rewardEvents: rewardEvents,
		world:        world,
	}
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

	spawn := room.spawnPointLocked()
	x := spawn.X
	y := spawn.Y
	facing := "down"
	if position.Found && position.RoomID == room.id && position.MapID == room.gameMap.ID {
		x = room.clampX(position.X)
		y = room.clampY(position.Y)
		if isFacing(position.Facing) {
			facing = position.Facing
		}
	}
	player := &player{
		appUserID:   claims.AppUserID,
		id:          client.id,
		displayName: claims.DisplayName,
		avatarID:    claims.AvatarID,
		equipment:   equipment,
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
	client.setRoom(room)

	client.send <- serverMessage{
		Type:         "hello",
		SelfID:       player.id,
		RoomID:       room.id,
		MapID:        room.gameMap.ID,
		Map:          &room.gameMap,
		Players:      room.snapshotsLocked(),
		Collectibles: room.collectibleSnapshotsLocked(),
	}
}

func (room *room) leave(client *client) {
	var saved *studentPositionRequest
	room.mu.Lock()
	if existing := room.players[client.id]; existing != nil && existing.client == client {
		delete(room.players, client.id)
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
		if err := client.server.saveStudentPosition(ctx, *saved); err != nil {
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
	source.mu.Unlock()

	player.x = target.clampX(usedPortal.TargetX)
	player.y = target.clampY(usedPortal.TargetY)
	if isFacing(usedPortal.TargetFacing) {
		player.facing = usedPortal.TargetFacing
	}
	player.moving = false
	player.lastMoveAt = now
	player.lastMoveSeq = seq

	target.mu.Lock()
	target.players[playerID] = player
	player.client.setRoom(target)
	message := serverMessage{
		Type:         "map_changed",
		SelfID:       player.id,
		RoomID:       target.id,
		MapID:        target.gameMap.ID,
		Map:          &target.gameMap,
		Players:      target.snapshotsLocked(),
		Collectibles: target.collectibleSnapshotsLocked(),
		ServerTimeMS: now.UnixMilli(),
		Tick:         target.tick,
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

	equipment, err := client.server.loadStudentEquipment(context.Background(), userID)
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
	equippedTool := strings.TrimSpace(player.equipment[equipmentSlotTool])
	room.mu.Unlock()

	if equippedTool == "" || equippedTool != toolKey {
		client.trySend(serverMessage{Type: "error", Code: "tool_not_equipped"})
		return
	}
}

func (room *room) updateMove(playerID string, seq int64, x float64, y float64, facing string, moving bool, now time.Time) {
	var triggered *portal
	room.mu.Lock()
	if player := room.players[playerID]; player != nil {
		if seq <= player.lastMoveSeq {
			room.mu.Unlock()
			return
		}
		acceptedX, acceptedY, ok := room.acceptedMoveLocked(player, x, y)
		if ok {
			player.x = acceptedX
			player.y = acceptedY
			player.lastMoveAt = now
		}
		if isFacing(facing) {
			player.facing = facing
		}
		player.moving = moving && ok
		player.lastMoveSeq = seq
		if ok {
			triggered = room.portalForPlayerLocked(player)
		}
	}
	room.mu.Unlock()

	if triggered != nil {
		room.world.transferPlayer(room.gameMap.ID, playerID, *triggered, seq, now)
	}
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

func (room *room) broadcastSnapshot(now time.Time) {
	room.mu.Lock()
	message := serverMessage{
		Type:         "snapshot",
		MapID:        room.gameMap.ID,
		Tick:         room.tick,
		ServerTimeMS: now.UnixMilli(),
		Players:      room.snapshotsLocked(),
		Collectibles: room.collectibleSnapshotsLocked(),
	}
	clients := make([]*client, 0, len(room.players))
	for _, player := range room.players {
		clients = append(clients, player.client)
	}
	room.mu.Unlock()

	for _, client := range clients {
		select {
		case client.send <- message:
		default:
			room.leave(client)
		}
	}
}

func (room *room) snapshotsLocked() []playerSnapshot {
	snapshots := make([]playerSnapshot, 0, len(room.players))
	for _, player := range room.players {
		snapshots = append(snapshots, playerSnapshot{
			ID:               player.id,
			DisplayName:      player.displayName,
			X:                math.Round(player.x*10) / 10,
			Y:                math.Round(player.y*10) / 10,
			Facing:           player.facing,
			Moving:           player.moving,
			AvatarID:         player.avatarID,
			Equipment:        cloneEquipment(player.equipment),
			LastProcessedSeq: player.lastMoveSeq,
		})
	}
	return snapshots
}

func cloneEquipment(equipment equipmentSnapshot) equipmentSnapshot {
	if len(equipment) == 0 {
		return equipmentSnapshot{}
	}
	cloned := make(equipmentSnapshot, len(equipment))
	for slot, visualKey := range equipment {
		cloned[slot] = visualKey
	}
	return cloned
}

func (room *room) acceptedMoveLocked(player *player, proposedX float64, proposedY float64) (float64, float64, bool) {
	if math.IsNaN(proposedX) || math.IsNaN(proposedY) || math.IsInf(proposedX, 0) || math.IsInf(proposedY, 0) {
		return player.x, player.y, false
	}

	x := room.clampXLocked(proposedX)
	y := room.clampYLocked(proposedY)
	return x, y, true
}

func (room *room) collectibleSnapshotsLocked() []collectibleSnapshot {
	snapshots := make([]collectibleSnapshot, 0, len(room.collectibles))
	for _, collectible := range room.collectibles {
		snapshots = append(snapshots, collectibleSnapshot{
			ID:     collectible.id,
			Kind:   collectible.kind,
			X:      collectible.x,
			Y:      collectible.y,
			Active: collectible.active,
		})
	}
	return snapshots
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

func (room *room) spawnPointLocked() point {
	if len(room.gameMap.Spawns) == 0 {
		return point{X: 64, Y: 64}
	}
	index := len(room.players) % len(room.gameMap.Spawns)
	return room.gameMap.Spawns[index]
}

func (room *room) collidesLocked(x float64, y float64) bool {
	playerRect := rect{
		X:      x - playerSize/2,
		Y:      y - playerSize/2,
		Width:  playerSize,
		Height: playerSize,
	}
	for _, blocked := range room.gameMap.BlockedRects {
		if rectsOverlap(playerRect, blocked) {
			return true
		}
	}
	return false
}

func (room *room) clampXLocked(x float64) float64 {
	return room.clampX(x)
}

func (room *room) clampYLocked(y float64) float64 {
	return room.clampY(y)
}

func (room *room) clampX(x float64) float64 {
	maxX := float64(room.gameMap.Width*room.gameMap.TileSize) - playerSize/2
	return math.Max(playerSize/2, math.Min(maxX, x))
}

func (room *room) clampY(y float64) float64 {
	maxY := float64(room.gameMap.Height*room.gameMap.TileSize) - playerSize/2
	return math.Max(playerSize/2, math.Min(maxY, y))
}

func (room *room) portalForPlayerLocked(player *player) *portal {
	playerRect := rect{
		X:      player.x - playerSize/2,
		Y:      player.y - playerSize/2,
		Width:  playerSize,
		Height: playerSize,
	}
	for index := range room.gameMap.Portals {
		portal := room.gameMap.Portals[index]
		if rectsOverlap(playerRect, rect{
			X:      portal.X,
			Y:      portal.Y,
			Width:  portal.Width,
			Height: portal.Height,
		}) {
			return &room.gameMap.Portals[index]
		}
	}
	return nil
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
	body, err := json.Marshal(rewardCommitRequest{
		EventID:       event.eventID,
		AppUserID:     event.appUserID,
		RoomID:        event.roomID,
		MapID:         event.mapID,
		CollectibleID: event.collectibleID,
		RewardKind:    event.kind,
		Amount:        event.amount,
	})
	if err != nil {
		return rewardCommitResponse{}, err
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		srv.config.hqInternalURL+"/api/internal/sunny-town/reward-events",
		bytes.NewReader(body),
	)
	if err != nil {
		return rewardCommitResponse{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-HQ-Service-Secret", srv.config.serviceSecret)

	response, err := srv.client.Do(request)
	if err != nil {
		return rewardCommitResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return rewardCommitResponse{}, fmt.Errorf("hq reward status %d", response.StatusCode)
	}

	var committed rewardCommitResponse
	if err := json.NewDecoder(response.Body).Decode(&committed); err != nil {
		return rewardCommitResponse{}, err
	}
	if !committed.Accepted {
		return rewardCommitResponse{}, errors.New("hq rejected reward")
	}
	return committed, nil
}

func loadMap(path string) (gameMap, error) {
	file, err := os.Open(path)
	if err != nil {
		return gameMap{}, err
	}
	defer file.Close()

	var loaded gameMap
	if err := json.NewDecoder(file).Decode(&loaded); err != nil {
		return gameMap{}, err
	}
	if loaded.ID == "" || loaded.TileSize < 1 || loaded.Width < 1 || loaded.Height < 1 {
		return gameMap{}, errors.New("map is missing required dimensions")
	}
	for _, portal := range loaded.Portals {
		if portal.ID == "" || portal.Width <= 0 || portal.Height <= 0 || portal.TargetMapID == "" {
			return gameMap{}, fmt.Errorf("map %q has an invalid portal", loaded.ID)
		}
		if !isFacing(portal.TargetFacing) {
			return gameMap{}, fmt.Errorf("map %q portal %q has invalid target facing", loaded.ID, portal.ID)
		}
	}
	npcIDs := map[string]bool{}
	for _, loadedNPC := range loaded.NPCs {
		if loadedNPC.ID == "" || loadedNPC.Name == "" || len(loadedNPC.Dialogue) == 0 {
			return gameMap{}, fmt.Errorf("map %q has an invalid npc", loaded.ID)
		}
		if npcIDs[loadedNPC.ID] {
			return gameMap{}, fmt.Errorf("map %q has duplicate npc id %q", loaded.ID, loadedNPC.ID)
		}
		npcIDs[loadedNPC.ID] = true
		if loadedNPC.Facing != "" && !isFacing(loadedNPC.Facing) {
			return gameMap{}, fmt.Errorf("map %q npc %q has invalid facing", loaded.ID, loadedNPC.ID)
		}
		for _, line := range loadedNPC.Dialogue {
			if strings.TrimSpace(line) == "" {
				return gameMap{}, fmt.Errorf("map %q npc %q has blank dialogue", loaded.ID, loadedNPC.ID)
			}
		}
		if loadedNPC.Shop != nil {
			if loadedNPC.Shop.ID == "" || len(loadedNPC.Shop.Items) == 0 {
				return gameMap{}, fmt.Errorf("map %q npc %q has an invalid shop", loaded.ID, loadedNPC.ID)
			}
			shopItemKeys := map[string]bool{}
			for _, item := range loadedNPC.Shop.Items {
				if item.ItemKey == "" || item.Name == "" || item.PriceStars < 1 {
					return gameMap{}, fmt.Errorf("map %q npc %q has an invalid shop item", loaded.ID, loadedNPC.ID)
				}
				if shopItemKeys[item.ItemKey] {
					return gameMap{}, fmt.Errorf("map %q npc %q has duplicate shop item %q", loaded.ID, loadedNPC.ID, item.ItemKey)
				}
				shopItemKeys[item.ItemKey] = true
			}
		}
		if loadedNPC.Activity != nil && loadedNPC.Activity.Type != "schoolwork" {
			return gameMap{}, fmt.Errorf("map %q npc %q has invalid activity type %q", loaded.ID, loadedNPC.ID, loadedNPC.Activity.Type)
		}
	}
	return loaded, nil
}

func loadMaps(dir string) (map[string]gameMap, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	maps := map[string]gameMap{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		loaded, err := loadMap(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if _, exists := maps[loaded.ID]; exists {
			return nil, fmt.Errorf("duplicate map id %q", loaded.ID)
		}
		maps[loaded.ID] = loaded
	}
	if len(maps) == 0 {
		return nil, fmt.Errorf("no maps found in %s", dir)
	}
	for _, loaded := range maps {
		for _, portal := range loaded.Portals {
			if _, ok := maps[portal.TargetMapID]; !ok {
				return nil, fmt.Errorf("map %q portal %q targets unknown map %q", loaded.ID, portal.ID, portal.TargetMapID)
			}
		}
	}
	return maps, nil
}

func allowedOrigins(value string) map[string]bool {
	origins := map[string]bool{}
	for _, origin := range strings.Split(value, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins[origin] = true
		}
	}
	return origins
}

func rectsOverlap(a rect, b rect) bool {
	return a.X < b.X+b.Width &&
		a.X+a.Width > b.X &&
		a.Y < b.Y+b.Height &&
		a.Y+a.Height > b.Y
}

func isFacing(value string) bool {
	return value == "up" || value == "down" || value == "left" || value == "right"
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

func envOrDefault(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
