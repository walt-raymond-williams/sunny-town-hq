package server

import (
	"sync"
	"time"

	"hq/internal/sunnytown/hqclient"
	stmaps "hq/internal/sunnytown/maps"
	stnavigation "hq/internal/sunnytown/navigation"
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
type fixtureDefinition = stmaps.FixtureDefinition
type npcRoute = stnavigation.Route

type clientMessage = stprotocol.ClientMessage
type storageSlotRef = stprotocol.StorageSlotRef
type equipmentSnapshot = stprotocol.EquipmentSnapshot
type inventorySnapshot = stprotocol.InventorySnapshot
type serverMessage = stprotocol.ServerMessage
type npcSnapshot = stprotocol.NPCSnapshot
type playerSnapshot = stprotocol.PlayerSnapshot

type player struct {
	appUserID     int64
	characterID   int64
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
	liveNPCs       map[string]*liveNPC
	collectibles   map[string]*collectible
	resourceNodes  map[string]*resourceNode
	placedObjects  map[string]*placedObject
	worldObjects   map[string]*worldObject
	tick           int64
	npcPausedAt    time.Time
	rewardRunID    string
	rewardEvents   chan rewardEvent
	resourceEvents chan resourceEvent
	npcJobEvents   chan npcJobProductionEvent
	world          *world
}

type liveNPC struct {
	characterID int64
	npcKey      string
	displayName string
	spriteKey   string
	mapID       string
	x           float64
	y           float64
	facing      string
	moving      bool
	dialogue    []string
	shop        *shop
	activity    *activity

	drives      npcDrives
	activeDrive npcDrive
	anchors     npcRoutineAnchors
	goal        *npcGoal
	route       *npcRoute
	routeStep   int
	pathIndex   int

	goalStartedAt   time.Time
	focusUntil      time.Time
	reevaluateAt    time.Time
	goalArrivedAt   time.Time
	goalArriveDrive float64
	failureCount    int
	failedTargets   map[string]time.Time
	jobProduction   npcJobProductionState
}

type world struct {
	roomID         string
	rooms          map[string]*room
	defaultRoom    *room
	navigation     *stnavigation.Graph
	npcMu          sync.RWMutex
	npcCharacters  map[string]npcCharacter
	rewardEvents   chan rewardEvent
	resourceEvents chan resourceEvent
	npcJobEvents   chan npcJobProductionEvent
	transferMu     sync.Mutex
	npcDayLength   time.Duration
}

type npcDrive string

const (
	npcDriveHunger npcDrive = "hunger"
	npcDriveEnergy npcDrive = "energy"
	npcDriveSocial npcDrive = "social"
	npcDriveWork   npcDrive = "work"
	npcDriveIdle   npcDrive = "idle"
)

type npcDrives struct {
	Hunger float64
	Energy float64
	Social float64
	Work   float64
}

type npcGoal struct {
	drive      npcDrive
	mapID      string
	location   stmaps.Location
	anchorKind string
}

type npcRoutineAnchors struct {
	Home   *npcLocationAnchor
	Work   *npcLocationAnchor
	Food   *npcLocationAnchor
	Social *npcLocationAnchor
}

type npcLocationAnchor struct {
	Kind         string
	MapID        string
	LocationID   string
	LocationName string
	Source       string
	Tags         []string
}

type npcCharacter struct {
	characterID int64
	displayName string
	avatarID    string
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
	name              string
	locationID        string
	shopID            string
	storageRole       string
	tags              []string
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
	characterID int64
	roomID      string
	mapID       string
	nodeID      string
	resourceKey string
	amount      int
	client      *client
}

type npcJobProductionState struct {
	Progress  float64
	Sequence  int64
	LastAt    time.Time
	LastEvent string
}

type npcJobProductionEvent struct {
	eventID     string
	characterID int64
	roomID      string
	mapID       string
	npcKey      string
	jobKey      string
	locationID  string
	outputKey   string
	amount      int
}

type rewardCommitRequest = hqclient.RewardCommitRequest
type rewardCommitResponse = hqclient.RewardCommitResponse
type resourceCommitRequest = hqclient.ResourceCommitRequest
type resourceCommitResponse = hqclient.ResourceCommitResponse
type npcJobProductionRequest = hqclient.NPCJobProductionRequest
type npcJobProductionResponse = hqclient.NPCJobProductionResponse
type mapObjectResponse = hqclient.MapObjectResponse
type placeMapObjectRequest = hqclient.PlaceMapObjectRequest
type removeMapObjectRequest = hqclient.RemoveMapObjectRequest
type studentPositionResponse = hqclient.StudentPositionResponse
type studentPositionRequest = hqclient.StudentPositionRequest
type containerTransferRequest = hqclient.ContainerTransferRequest
