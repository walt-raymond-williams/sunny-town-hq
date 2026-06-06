package server

import (
	"sync"
	"time"

	"hq/internal/sunnytown/hqclient"
	stmaps "hq/internal/sunnytown/maps"
	stprotocol "hq/internal/sunnytown/protocol"

	"github.com/gorilla/websocket"
)

type gameMap = stmaps.GameMap
type point = stmaps.Point
type rect = stmaps.Rect
type portal = stmaps.Portal
type npc = stmaps.NPC
type activity = stmaps.Activity
type shop = stmaps.Shop
type shopItem = stmaps.ShopItem
type resourceNodeDefinition = stmaps.ResourceNodeDefinition

type clientMessage = stprotocol.ClientMessage
type equipmentSnapshot = stprotocol.EquipmentSnapshot
type inventorySnapshot = stprotocol.InventorySnapshot
type serverMessage = stprotocol.ServerMessage
type playerSnapshot = stprotocol.PlayerSnapshot

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
	server           *Server
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

type collectibleSnapshot = stprotocol.CollectibleSnapshot
type resourceNodeSnapshot = stprotocol.ResourceNodeSnapshot
type placedObjectSnapshot = stprotocol.PlacedObjectSnapshot
type worldObjectSnapshot = stprotocol.WorldObjectSnapshot

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

type rewardCommitRequest = hqclient.RewardCommitRequest
type rewardCommitResponse = hqclient.RewardCommitResponse
type resourceCommitRequest = hqclient.ResourceCommitRequest
type resourceCommitResponse = hqclient.ResourceCommitResponse
type mapObjectResponse = hqclient.MapObjectResponse
type placeMapObjectRequest = hqclient.PlaceMapObjectRequest
type removeMapObjectRequest = hqclient.RemoveMapObjectRequest
type studentPositionResponse = hqclient.StudentPositionResponse
type studentPositionRequest = hqclient.StudentPositionRequest
