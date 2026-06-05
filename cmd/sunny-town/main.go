package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"hq/internal/sunnytownauth"

	"github.com/gorilla/websocket"
)

const (
	defaultRoomID      = "sunny-town-main"
	defaultMapID       = "sunny-town-v1"
	playerSize         = 28.0
	playerSpeed        = 150.0
	starPickupRadius   = 30.0
	simulationInterval = 50 * time.Millisecond
	snapshotInterval   = 100 * time.Millisecond
	starRespawnDelay   = 10 * time.Second
)

type config struct {
	host           string
	port           string
	joinSecret     string
	serviceSecret  string
	hqInternalURL  string
	allowedOrigins map[string]bool
	mapPath        string
}

type gameMap struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	TileSize     int     `json:"tileSize"`
	Width        int     `json:"width"`
	Height       int     `json:"height"`
	Spawns       []point `json:"spawns"`
	BlockedRects []rect  `json:"blockedRects"`
	StarSpawns   []point `json:"starSpawns"`
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

type inputState struct {
	Up    bool `json:"up"`
	Down  bool `json:"down"`
	Left  bool `json:"left"`
	Right bool `json:"right"`
}

type clientMessage struct {
	Type         string `json:"type"`
	Seq          int64  `json:"seq,omitempty"`
	ClientTimeMS int64  `json:"client_time_ms,omitempty"`
	inputState
}

type serverMessage struct {
	Type           string                `json:"type"`
	SelfID         string                `json:"selfId,omitempty"`
	RoomID         string                `json:"roomId,omitempty"`
	MapID          string                `json:"mapId,omitempty"`
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
	ID               string  `json:"id"`
	DisplayName      string  `json:"displayName"`
	X                float64 `json:"x"`
	Y                float64 `json:"y"`
	Facing           string  `json:"facing"`
	Moving           bool    `json:"moving"`
	AvatarID         string  `json:"avatarId"`
	LastProcessedSeq int64   `json:"lastProcessedSeq"`
}

type player struct {
	appUserID         int64
	id                string
	displayName       string
	avatarID          string
	x                 float64
	y                 float64
	facing            string
	moving            bool
	input             inputState
	receivedInputSeq  int64
	simulatedInputSeq int64
	client            *client
}

type client struct {
	conn             *websocket.Conn
	send             chan serverMessage
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
	rewardEvents chan rewardEvent
}

type server struct {
	config   config
	gameMap  gameMap
	room     *room
	client   *http.Client
	upgrader websocket.Upgrader
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

func main() {
	cfg := loadConfig()
	gameMap, err := loadMap(cfg.mapPath)
	if err != nil {
		log.Fatalf("load map: %v", err)
	}
	if gameMap.ID != defaultMapID {
		log.Fatalf("map id %q does not match expected %q", gameMap.ID, defaultMapID)
	}

	room := newRoom(defaultRoomID, gameMap)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go room.run(ctx)

	srv := newServer(cfg, gameMap, room)
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
		mapPath:        envOrDefault("SUNNY_TOWN_MAP_PATH", filepath.Join("sunny-town", "maps", "sunny-town-v1.json")),
	}
}

func newServer(cfg config, gameMap gameMap, room *room) *server {
	srv := &server{
		config:  cfg,
		gameMap: gameMap,
		room:    room,
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

func (srv *server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	claims, err := sunnytownauth.Verify(r.URL.Query().Get("token"), srv.config.joinSecret, time.Now())
	if err != nil {
		log.Printf("sunny town auth failed: %v", err)
		http.Error(w, "invalid sunny town token", http.StatusUnauthorized)
		return
	}
	if err := validateJoinTarget(claims, srv.room.id, srv.gameMap.ID); err != nil {
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
		conn: conn,
		send: make(chan serverMessage, 16),
		room: srv.room,
		id:   playerID,
	}

	srv.room.join(client, claims)
	log.Printf("player joined room=%s player=%s", srv.room.id, playerID)

	go client.writePump()
	client.readPump()
}

func newRoom(id string, gameMap gameMap) *room {
	return &room{
		id:           id,
		gameMap:      gameMap,
		players:      map[string]*player{},
		collectibles: initialCollectibles(gameMap),
		rewardEvents: make(chan rewardEvent, 32),
	}
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

func (room *room) join(client *client, claims sunnytownauth.Claims) {
	room.mu.Lock()
	defer room.mu.Unlock()

	spawn := room.spawnPointLocked()
	player := &player{
		appUserID:   claims.AppUserID,
		id:          client.id,
		displayName: claims.DisplayName,
		avatarID:    claims.AvatarID,
		x:           spawn.X,
		y:           spawn.Y,
		facing:      "down",
		client:      client,
	}
	if player.displayName == "" {
		player.displayName = "Student"
	}
	if player.avatarID == "" {
		player.avatarID = "pet-default"
	}
	room.players[player.id] = player

	client.send <- serverMessage{
		Type:   "hello",
		SelfID: player.id,
		RoomID: room.id,
		MapID:  room.gameMap.ID,
	}
}

func (room *room) leave(client *client) {
	room.mu.Lock()
	if existing := room.players[client.id]; existing != nil && existing.client == client {
		delete(room.players, client.id)
		log.Printf("player left room=%s player=%s", room.id, client.id)
	}
	room.mu.Unlock()
}

func (room *room) updateInput(playerID string, seq int64, input inputState) {
	room.mu.Lock()
	if player := room.players[playerID]; player != nil {
		if seq <= player.receivedInputSeq {
			room.mu.Unlock()
			return
		}
		player.input = input
		player.receivedInputSeq = seq
	}
	room.mu.Unlock()
}

func (room *room) step(dt float64, now time.Time) {
	room.mu.Lock()

	room.tick++
	var rewards []rewardEvent
	for _, player := range room.players {
		dx := boolFloat(player.input.Right) - boolFloat(player.input.Left)
		dy := boolFloat(player.input.Down) - boolFloat(player.input.Up)
		if dx == 0 && dy == 0 {
			player.moving = false
		} else {
			length := math.Hypot(dx, dy)
			dx = dx / length
			dy = dy / length

			if math.Abs(dx) > math.Abs(dy) {
				if dx > 0 {
					player.facing = "right"
				} else {
					player.facing = "left"
				}
			} else if dy > 0 {
				player.facing = "down"
			} else {
				player.facing = "up"
			}

			nextX := player.x + dx*playerSpeed*dt
			nextY := player.y + dy*playerSpeed*dt
			if !room.collidesLocked(nextX, player.y) {
				player.x = room.clampXLocked(nextX)
			}
			if !room.collidesLocked(player.x, nextY) {
				player.y = room.clampYLocked(nextY)
			}
			player.moving = true
		}
		player.simulatedInputSeq = player.receivedInputSeq

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
			client.room.leave(client)
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
			LastProcessedSeq: player.simulatedInputSeq,
		})
	}
	return snapshots
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
		eventID := fmt.Sprintf("%s:%s:%d:%d", room.id, collectible.id, collectible.spawnSeq, player.appUserID)
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
	maxX := float64(room.gameMap.Width*room.gameMap.TileSize) - playerSize/2
	return math.Max(playerSize/2, math.Min(maxX, x))
}

func (room *room) clampYLocked(y float64) float64 {
	maxY := float64(room.gameMap.Height*room.gameMap.TileSize) - playerSize/2
	return math.Max(playerSize/2, math.Min(maxY, y))
}

func (client *client) readPump() {
	defer func() {
		client.room.leave(client)
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
		case "input":
			if !client.allowInput(time.Now()) {
				continue
			}
			client.room.updateInput(client.id, message.Seq, message.inputState)
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

func (client *client) allowInput(now time.Time) bool {
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
		case event := <-srv.room.rewardEvents:
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
	return loaded, nil
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

func boolFloat(value bool) float64 {
	if value {
		return 1
	}
	return 0
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

func validateJoinTarget(claims sunnytownauth.Claims, roomID string, mapID string) error {
	if claims.RoomID != roomID || claims.MapID != mapID {
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
