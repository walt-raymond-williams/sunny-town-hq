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
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	stconfig "hq/internal/sunnytown/config"
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

type gameMap struct {
	ID            string                   `json:"id"`
	Name          string                   `json:"name"`
	TileSize      int                      `json:"tileSize"`
	Width         int                      `json:"width"`
	Height        int                      `json:"height"`
	Spawns        []point                  `json:"spawns"`
	BlockedRects  []rect                   `json:"blockedRects"`
	StarSpawns    []point                  `json:"starSpawns"`
	Portals       []portal                 `json:"portals"`
	NPCs          []npc                    `json:"npcs"`
	ResourceNodes []resourceNodeDefinition `json:"resourceNodes"`
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

type resourceNodeDefinition struct {
	ID                string  `json:"id"`
	Kind              string  `json:"kind"`
	X                 float64 `json:"x"`
	Y                 float64 `json:"y"`
	Radius            float64 `json:"radius"`
	InteractionRadius float64 `json:"interactionRadius"`
	RespawnSeconds    int     `json:"respawnSeconds"`
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
	ItemKey      string  `json:"itemKey,omitempty"`
	GridX        int     `json:"gridX,omitempty"`
	GridY        int     `json:"gridY,omitempty"`
}

type equipmentSnapshot map[string]string
type inventorySnapshot map[string]int

type serverMessage struct {
	Type           string                 `json:"type"`
	SelfID         string                 `json:"selfId,omitempty"`
	RoomID         string                 `json:"roomId,omitempty"`
	MapID          string                 `json:"mapId,omitempty"`
	Map            *gameMap               `json:"map,omitempty"`
	Tick           int64                  `json:"tick,omitempty"`
	ServerTimeMS   int64                  `json:"serverTimeMs,omitempty"`
	Players        []playerSnapshot       `json:"players,omitempty"`
	Collectibles   []collectibleSnapshot  `json:"collectibles,omitempty"`
	ResourceNodes  []resourceNodeSnapshot `json:"resourceNodes,omitempty"`
	PlacedObjects  []placedObjectSnapshot `json:"placedObjects,omitempty"`
	PlacedObject   *placedObjectSnapshot  `json:"placedObject,omitempty"`
	WorldObjects   []worldObjectSnapshot  `json:"worldObjects,omitempty"`
	WorldObject    *worldObjectSnapshot   `json:"worldObject,omitempty"`
	Code           string                 `json:"code,omitempty"`
	EventID        string                 `json:"eventId,omitempty"`
	Kind           string                 `json:"kind,omitempty"`
	Amount         int                    `json:"amount,omitempty"`
	NewStarBalance int                    `json:"newStarBalance,omitempty"`
	CollectibleID  string                 `json:"collectibleId,omitempty"`
	NodeID         string                 `json:"nodeId,omitempty"`
	ResourceKey    string                 `json:"resourceKey,omitempty"`
	Quantity       int                    `json:"quantity,omitempty"`
	Reason         string                 `json:"reason,omitempty"`
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
	appUserID     int64
	id            string
	displayName   string
	avatarID      string
	equipment     equipmentSnapshot
	inventory     inventorySnapshot
	x             float64
	y             float64
	facing        string
	moving        bool
	lastMoveAt    time.Time
	lastMoveSeq   int64
	lastToolUseAt time.Time
	portalLocked  bool
	client        *client
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

	mu             sync.Mutex
	players        map[string]*player
	collectibles   map[string]*collectible
	resourceNodes  map[string]*resourceNode
	placedObjects  map[string]*placedObject
	worldObjects   map[string]*worldObject
	tick           int64
	rewardRunID    string
	rewardEvents   chan rewardEvent
	resourceEvents chan resourceEvent
	world          *world
}

type server struct {
	config   stconfig.Config
	world    *world
	client   *http.Client
	upgrader websocket.Upgrader
}

type world struct {
	roomID         string
	rooms          map[string]*room
	defaultRoom    *room
	rewardEvents   chan rewardEvent
	resourceEvents chan resourceEvent
	transferMu     sync.Mutex
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

type worldObject struct {
	id                string
	kind              string
	source            string
	itemKey           string
	resourceKind      string
	mapID             string
	x                 float64
	y                 float64
	width             float64
	height            float64
	radius            float64
	interactionRadius float64
	collision         bool
	breakable         bool
	toolKey           string
	hitsRequired      int
	reservesPlacement bool
	respawnDelay      time.Duration
	active            bool
	hitCount          int
	respawnAt         time.Time
	harvestSeq        int64
	gridX             int
	gridY             int
	placedByAppUserID int64
}

type resourceNode = worldObject
type placedObject = worldObject

type resourceNodeSnapshot struct {
	ID     string  `json:"id"`
	Kind   string  `json:"kind"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radius"`
	Active bool    `json:"active"`
	Hits   int     `json:"hits"`
	Needed int     `json:"needed"`
}

type placedObjectSnapshot struct {
	ID                string  `json:"id"`
	ItemKey           string  `json:"itemKey"`
	GridX             int     `json:"gridX"`
	GridY             int     `json:"gridY"`
	X                 float64 `json:"x"`
	Y                 float64 `json:"y"`
	Width             float64 `json:"width"`
	Height            float64 `json:"height"`
	PlacedByAppUserID int64   `json:"placedByAppUserId,omitempty"`
}

type worldObjectSnapshot struct {
	ID                string  `json:"id"`
	Kind              string  `json:"kind"`
	Source            string  `json:"source"`
	ItemKey           string  `json:"itemKey,omitempty"`
	ResourceKind      string  `json:"resourceKind,omitempty"`
	X                 float64 `json:"x"`
	Y                 float64 `json:"y"`
	Width             float64 `json:"width,omitempty"`
	Height            float64 `json:"height,omitempty"`
	Radius            float64 `json:"radius,omitempty"`
	Active            bool    `json:"active"`
	Collision         bool    `json:"collision"`
	Breakable         bool    `json:"breakable"`
	ReservesPlacement bool    `json:"reservesPlacement"`
	Hits              int     `json:"hits,omitempty"`
	Needed            int     `json:"needed,omitempty"`
	GridX             int     `json:"gridX,omitempty"`
	GridY             int     `json:"gridY,omitempty"`
	PlacedByAppUserID int64   `json:"placedByAppUserId,omitempty"`
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

type resourceEvent struct {
	eventID     string
	appUserID   int64
	roomID      string
	mapID       string
	nodeID      string
	resourceKey string
	amount      int
	client      *client
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

type resourceCommitRequest struct {
	EventID     string `json:"event_id"`
	AppUserID   int64  `json:"app_user_id"`
	Source      string `json:"source"`
	RoomID      string `json:"room_id"`
	MapID       string `json:"map_id"`
	NodeID      string `json:"node_id"`
	ResourceKey string `json:"resource_key"`
	Amount      int    `json:"amount"`
}

type resourceCommitResponse struct {
	Accepted    bool   `json:"accepted"`
	Duplicate   bool   `json:"duplicate"`
	ResourceKey string `json:"resource_key"`
	Quantity    int    `json:"quantity"`
}

type mapObjectsResponse struct {
	Objects []mapObjectResponse `json:"objects"`
}

type mapObjectResponse struct {
	ID                  int64  `json:"id"`
	RoomID              string `json:"room_id"`
	MapID               string `json:"map_id"`
	GridX               int    `json:"grid_x"`
	GridY               int    `json:"grid_y"`
	ItemKey             string `json:"item_key"`
	PlacedByAppUserID   int64  `json:"placed_by_app_user_id"`
	RemainingItemAmount int    `json:"remaining_item_amount"`
}

type placeMapObjectRequest struct {
	AppUserID int64  `json:"app_user_id"`
	RoomID    string `json:"room_id"`
	MapID     string `json:"map_id"`
	GridX     int    `json:"grid_x"`
	GridY     int    `json:"grid_y"`
	ItemKey   string `json:"item_key"`
}

type removeMapObjectRequest struct {
	AppUserID int64  `json:"app_user_id"`
	RoomID    string `json:"room_id"`
	MapID     string `json:"map_id"`
	GridX     int    `json:"grid_x"`
	GridY     int    `json:"grid_y"`
}

type studentEquipmentResponse struct {
	Slots []equipmentSlotResponse `json:"slots"`
}

type inventoryQuantityResponse struct {
	ItemKey  string `json:"item_key"`
	Quantity int    `json:"quantity"`
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
	cfg := stconfig.Load()
	maps, err := loadMaps(cfg.MapsDir)
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

func newServer(cfg stconfig.Config, world *world) *server {
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
			return origin == "" || cfg.AllowedOrigins["*"] || cfg.AllowedOrigins[origin]
		},
	}
	return srv
}

func (srv *server) internalRequest(ctx context.Context, method string, path string, body []byte) (*http.Response, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		method,
		srv.config.HQInternalURL+path,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("X-HQ-Service-Secret", srv.config.ServiceSecret)
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

func (srv *server) loadStudentInventoryQuantity(ctx context.Context, appUserID int64, itemKey string) (int, error) {
	query := url.Values{}
	query.Set("app_user_id", strconv.FormatInt(appUserID, 10))
	query.Set("item_key", itemKey)
	response, err := srv.internalRequest(
		ctx,
		http.MethodGet,
		"/api/internal/sunny-town/inventory-quantity?"+query.Encode(),
		nil,
	)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return 0, fmt.Errorf("inventory quantity request failed status=%d", response.StatusCode)
	}

	var quantity inventoryQuantityResponse
	if err := json.NewDecoder(response.Body).Decode(&quantity); err != nil {
		return 0, err
	}
	return quantity.Quantity, nil
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

func (srv *server) loadInitialMapObjects(ctx context.Context) error {
	for _, room := range srv.world.rooms {
		loadCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		objects, err := srv.loadMapObjects(loadCtx, room.id, room.gameMap.ID)
		cancel()
		if err != nil {
			return err
		}
		room.mu.Lock()
		room.setPlacedObjectsLocked(objects)
		room.mu.Unlock()
	}
	return nil
}

func (srv *server) loadMapObjects(ctx context.Context, roomID string, mapID string) (map[string]*placedObject, error) {
	query := url.Values{}
	query.Set("room_id", roomID)
	query.Set("map_id", mapID)
	response, err := srv.internalRequest(ctx, http.MethodGet, "/api/internal/sunny-town/map-objects?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("map objects request failed status=%d", response.StatusCode)
	}

	var loaded mapObjectsResponse
	if err := json.NewDecoder(response.Body).Decode(&loaded); err != nil {
		return nil, err
	}

	room := srv.world.rooms[mapID]
	if room == nil {
		return nil, fmt.Errorf("unknown map %q", mapID)
	}
	objects := map[string]*placedObject{}
	for _, object := range loaded.Objects {
		placed := placedObjectFromResponse(room.gameMap, object)
		objects[placed.id] = placed
	}
	return objects, nil
}

func (srv *server) refreshRoomMapObjects(ctx context.Context, mapID string) error {
	room := srv.world.rooms[mapID]
	if room == nil {
		return fmt.Errorf("unknown map %q", mapID)
	}
	objects, err := srv.loadMapObjects(ctx, room.id, room.gameMap.ID)
	if err != nil {
		return err
	}
	room.mu.Lock()
	room.setPlacedObjectsLocked(objects)
	room.mu.Unlock()
	return nil
}

func (srv *server) placeMapObject(ctx context.Context, request placeMapObjectRequest, gameMap gameMap) (*placedObject, int, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, 0, err
	}
	response, err := srv.internalRequest(ctx, http.MethodPost, "/api/internal/sunny-town/map-objects/place", body)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, 0, fmt.Errorf("place map object failed status=%d", response.StatusCode)
	}

	var placed mapObjectResponse
	if err := json.NewDecoder(response.Body).Decode(&placed); err != nil {
		return nil, 0, err
	}
	return placedObjectFromResponse(gameMap, placed), placed.RemainingItemAmount, nil
}

func (srv *server) removeMapObject(ctx context.Context, request removeMapObjectRequest, gameMap gameMap) (*placedObject, int, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, 0, err
	}
	response, err := srv.internalRequest(ctx, http.MethodPost, "/api/internal/sunny-town/map-objects/remove", body)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, 0, fmt.Errorf("remove map object failed status=%d", response.StatusCode)
	}

	var removed mapObjectResponse
	if err := json.NewDecoder(response.Body).Decode(&removed); err != nil {
		return nil, 0, err
	}
	return placedObjectFromResponse(gameMap, removed), removed.RemainingItemAmount, nil
}

func (srv *server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	claims, err := sunnytownauth.Verify(r.URL.Query().Get("token"), srv.config.JoinSecret, time.Now())
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
	if err := srv.refreshRoomMapObjects(r.Context(), claims.MapID); err != nil {
		log.Printf("refresh sunny town map objects map=%s: %v", claims.MapID, err)
	}

	srv.world.join(client, claims, equipment, position)
	log.Printf("player joined room=%s map=%s player=%s", srv.world.roomID, claims.MapID, playerID)

	go client.writePump()
	client.readPump()
}

func newWorld(roomID string, maps map[string]gameMap) *world {
	rewardEvents := make(chan rewardEvent, 32)
	resourceEvents := make(chan resourceEvent, resourceCommitQueueSize)
	created := &world{
		roomID:         roomID,
		rooms:          map[string]*room{},
		rewardEvents:   rewardEvents,
		resourceEvents: resourceEvents,
	}
	for _, gameMap := range maps {
		created.rooms[gameMap.ID] = newRoom(roomID, gameMap, rewardEvents, resourceEvents, created)
	}
	created.defaultRoom = created.rooms[defaultMapID]
	return created
}

func newRoom(id string, gameMap gameMap, rewardEvents chan rewardEvent, resourceEvents chan resourceEvent, world *world) *room {
	resourceNodes := initialResourceNodes(gameMap)
	worldObjects := map[string]*worldObject{}
	for _, node := range resourceNodes {
		worldObjects[worldObjectKey(node.source, node.id)] = node
	}
	return &room{
		id:             id,
		gameMap:        gameMap,
		players:        map[string]*player{},
		collectibles:   initialCollectibles(gameMap),
		resourceNodes:  resourceNodes,
		placedObjects:  map[string]*placedObject{},
		worldObjects:   worldObjects,
		rewardRunID:    newRewardRunID(),
		rewardEvents:   rewardEvents,
		resourceEvents: resourceEvents,
		world:          world,
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
	client.setRoom(room)

	client.send <- serverMessage{
		Type:          "hello",
		SelfID:        player.id,
		RoomID:        room.id,
		MapID:         room.gameMap.ID,
		Map:           &room.gameMap,
		Players:       room.snapshotsLocked(),
		Collectibles:  room.collectibleSnapshotsLocked(),
		ResourceNodes: room.resourceNodeSnapshotsLocked(),
		PlacedObjects: room.placedObjectSnapshotsLocked(),
		WorldObjects:  room.worldObjectSnapshotsLocked(),
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
	player.portalLocked = true

	target.mu.Lock()
	target.players[playerID] = player
	player.client.setRoom(target)
	message := serverMessage{
		Type:          "map_changed",
		SelfID:        player.id,
		RoomID:        target.id,
		MapID:         target.gameMap.ID,
		Map:           &target.gameMap,
		Players:       target.snapshotsLocked(),
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

func (client *client) ownsInventoryItem(ctx context.Context, appUserID int64, itemKey string) (bool, error) {
	itemKey = strings.TrimSpace(itemKey)
	if itemKey == "" || appUserID < 1 {
		return false, nil
	}
	if client.server != nil {
		quantity, err := client.server.loadStudentInventoryQuantity(ctx, appUserID, itemKey)
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
			currentPortal := room.portalForPlayerLocked(player)
			if currentPortal == nil {
				player.portalLocked = false
			} else if !player.portalLocked {
				triggered = currentPortal
			}
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

func (room *room) broadcastSnapshot(now time.Time) {
	room.mu.Lock()
	message := serverMessage{
		Type:          "snapshot",
		MapID:         room.gameMap.ID,
		Tick:          room.tick,
		ServerTimeMS:  now.UnixMilli(),
		Players:       room.snapshotsLocked(),
		Collectibles:  room.collectibleSnapshotsLocked(),
		ResourceNodes: room.resourceNodeSnapshotsLocked(),
		PlacedObjects: room.placedObjectSnapshotsLocked(),
		WorldObjects:  room.worldObjectSnapshotsLocked(),
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

func (room *room) broadcast(message serverMessage) {
	room.mu.Lock()
	clients := make([]*client, 0, len(room.players))
	for _, player := range room.players {
		clients = append(clients, player.client)
	}
	room.mu.Unlock()

	for _, client := range clients {
		client.trySend(message)
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

func inventoryFromEquipment(equipment equipmentSnapshot) inventorySnapshot {
	inventory := inventorySnapshot{}
	if equipment[equipmentSlotTool] == "pickaxe" {
		inventory["pickaxe"] = 1
	}
	return inventory
}

func (room *room) acceptedMoveLocked(player *player, proposedX float64, proposedY float64) (float64, float64, bool) {
	if math.IsNaN(proposedX) || math.IsNaN(proposedY) || math.IsInf(proposedX, 0) || math.IsInf(proposedY, 0) {
		return player.x, player.y, false
	}

	x := room.clampXLocked(proposedX)
	y := room.clampYLocked(proposedY)
	if room.collidesLocked(x, y) {
		x = player.x
		y = player.y
	}
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

func (room *room) resourceNodeSnapshotsLocked() []resourceNodeSnapshot {
	snapshots := make([]resourceNodeSnapshot, 0, len(room.resourceNodes))
	for _, node := range room.resourceNodes {
		snapshots = append(snapshots, resourceNodeSnapshot{
			ID:     node.id,
			Kind:   node.resourceKind,
			X:      node.x,
			Y:      node.y,
			Radius: node.radius,
			Active: node.active,
			Hits:   node.hitCount,
			Needed: node.hitsRequired,
		})
	}
	return snapshots
}

func (room *room) placedObjectSnapshotsLocked() []placedObjectSnapshot {
	snapshots := make([]placedObjectSnapshot, 0, len(room.placedObjects))
	for _, object := range room.placedObjects {
		snapshots = append(snapshots, object.snapshot())
	}
	return snapshots
}

func (room *room) worldObjectSnapshotsLocked() []worldObjectSnapshot {
	snapshots := make([]worldObjectSnapshot, 0, len(room.worldObjects))
	for _, object := range room.worldObjects {
		snapshots = append(snapshots, object.worldObjectSnapshot())
	}
	return snapshots
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
	for _, object := range room.worldObjects {
		if object.active && object.collision && rectsOverlap(playerRect, object.rect()) {
			return true
		}
	}
	return false
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
		srv.config.HQInternalURL+"/api/internal/sunny-town/reward-events",
		bytes.NewReader(body),
	)
	if err != nil {
		return rewardCommitResponse{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-HQ-Service-Secret", srv.config.ServiceSecret)

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
	resourceNodeIDs := map[string]bool{}
	for _, node := range loaded.ResourceNodes {
		if node.ID == "" || node.Kind == "" || node.X < 0 || node.Y < 0 || node.Radius <= 0 || node.InteractionRadius <= 0 || node.RespawnSeconds < 1 {
			return gameMap{}, fmt.Errorf("map %q has an invalid resource node", loaded.ID)
		}
		if resourceNodeIDs[node.ID] {
			return gameMap{}, fmt.Errorf("map %q has duplicate resource node id %q", loaded.ID, node.ID)
		}
		resourceNodeIDs[node.ID] = true
		if node.Kind != "rock" {
			return gameMap{}, fmt.Errorf("map %q resource node %q has unsupported kind %q", loaded.ID, node.ID, node.Kind)
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

func (object *placedObject) snapshot() placedObjectSnapshot {
	return placedObjectSnapshot{
		ID:                object.id,
		ItemKey:           object.itemKey,
		GridX:             object.gridX,
		GridY:             object.gridY,
		X:                 object.x,
		Y:                 object.y,
		Width:             object.width,
		Height:            object.height,
		PlacedByAppUserID: object.placedByAppUserID,
	}
}

func (object *worldObject) worldObjectSnapshot() worldObjectSnapshot {
	return worldObjectSnapshot{
		ID:                object.id,
		Kind:              object.kind,
		Source:            object.source,
		ItemKey:           object.itemKey,
		ResourceKind:      object.resourceKind,
		X:                 object.x,
		Y:                 object.y,
		Width:             object.width,
		Height:            object.height,
		Radius:            object.radius,
		Active:            object.active,
		Collision:         object.collision,
		Breakable:         object.breakable,
		ReservesPlacement: object.reservesPlacement,
		Hits:              object.hitCount,
		Needed:            object.hitsRequired,
		GridX:             object.gridX,
		GridY:             object.gridY,
		PlacedByAppUserID: object.placedByAppUserID,
	}
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
	body, err := json.Marshal(resourceCommitRequest{
		EventID:     event.eventID,
		AppUserID:   event.appUserID,
		Source:      "sunny_town_mining",
		RoomID:      event.roomID,
		MapID:       event.mapID,
		NodeID:      event.nodeID,
		ResourceKey: event.resourceKey,
		Amount:      event.amount,
	})
	if err != nil {
		return resourceCommitResponse{}, err
	}

	response, err := srv.internalRequest(ctx, http.MethodPost, "/api/internal/sunny-town/resource-events", body)
	if err != nil {
		return resourceCommitResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return resourceCommitResponse{}, fmt.Errorf("hq resource status %d", response.StatusCode)
	}

	var committed resourceCommitResponse
	if err := json.NewDecoder(response.Body).Decode(&committed); err != nil {
		return resourceCommitResponse{}, err
	}
	if !committed.Accepted {
		return resourceCommitResponse{}, errors.New("hq rejected resource")
	}
	return committed, nil
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

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
