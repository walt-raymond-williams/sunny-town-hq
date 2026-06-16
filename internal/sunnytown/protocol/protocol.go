package protocol

import stmaps "hq/internal/sunnytown/maps"

type ClientMessage struct {
	Type         string         `json:"type"`
	Seq          int64          `json:"seq,omitempty"`
	ClientTimeMS int64          `json:"client_time_ms,omitempty"`
	X            float64        `json:"x,omitempty"`
	Y            float64        `json:"y,omitempty"`
	Facing       string         `json:"facing,omitempty"`
	Moving       bool           `json:"moving,omitempty"`
	ToolKey      string         `json:"toolKey,omitempty"`
	ItemKey      string         `json:"itemKey,omitempty"`
	GridX        int            `json:"gridX,omitempty"`
	GridY        int            `json:"gridY,omitempty"`
	ObjectSource string         `json:"objectSource,omitempty"`
	ObjectID     string         `json:"objectId,omitempty"`
	Source       StorageSlotRef `json:"source,omitempty"`
	Destination  StorageSlotRef `json:"destination,omitempty"`
}

type EquipmentSnapshot map[string]string
type InventorySnapshot map[string]int

type StorageSlotRef struct {
	Kind        string `json:"kind,omitempty"`
	ContainerID string `json:"containerId,omitempty"`
	SlotIndex   int    `json:"slotIndex,omitempty"`
}

type InventoryItemSnapshot struct {
	Key         string `json:"key,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Quantity    int    `json:"quantity,omitempty"`
	EquipSlot   string `json:"equipSlot,omitempty"`
	VisualKey   string `json:"visualKey,omitempty"`
	IconKey     string `json:"iconKey,omitempty"`
	MaxStack    int    `json:"maxStack,omitempty"`
	Category    string `json:"category,omitempty"`
	Equipped    bool   `json:"equipped,omitempty"`
}

type InventorySlotSnapshot struct {
	SlotIndex int                    `json:"slotIndex"`
	Item      *InventoryItemSnapshot `json:"item,omitempty"`
}

type StudentInventorySlotsSnapshot struct {
	SlotCount int                     `json:"slotCount"`
	Slots     []InventorySlotSnapshot `json:"slots"`
	Items     []InventoryItemSnapshot `json:"items"`
}

type ContainerSlotsSnapshot struct {
	ContainerID string                  `json:"containerId"`
	SlotCount   int                     `json:"slotCount"`
	Revision    int64                   `json:"revision"`
	Slots       []InventorySlotSnapshot `json:"slots"`
}

type ServerMessage struct {
	Type           string                         `json:"type"`
	SelfID         string                         `json:"selfId,omitempty"`
	RoomID         string                         `json:"roomId,omitempty"`
	MapID          string                         `json:"mapId,omitempty"`
	Map            *stmaps.GameMap                `json:"map,omitempty"`
	Tick           int64                          `json:"tick,omitempty"`
	ServerTimeMS   int64                          `json:"serverTimeMs,omitempty"`
	Players        []PlayerSnapshot               `json:"players,omitempty"`
	NPCs           []NPCSnapshot                  `json:"npcs"`
	Collectibles   []CollectibleSnapshot          `json:"collectibles,omitempty"`
	ResourceNodes  []ResourceNodeSnapshot         `json:"resourceNodes,omitempty"`
	PlacedObjects  []PlacedObjectSnapshot         `json:"placedObjects,omitempty"`
	PlacedObject   *PlacedObjectSnapshot          `json:"placedObject,omitempty"`
	WorldObjects   []WorldObjectSnapshot          `json:"worldObjects,omitempty"`
	WorldObject    *WorldObjectSnapshot           `json:"worldObject,omitempty"`
	Code           string                         `json:"code,omitempty"`
	EventID        string                         `json:"eventId,omitempty"`
	Kind           string                         `json:"kind,omitempty"`
	Amount         int                            `json:"amount,omitempty"`
	NewStarBalance int                            `json:"newStarBalance,omitempty"`
	CollectibleID  string                         `json:"collectibleId,omitempty"`
	NodeID         string                         `json:"nodeId,omitempty"`
	ResourceKey    string                         `json:"resourceKey,omitempty"`
	Quantity       int                            `json:"quantity,omitempty"`
	Reason         string                         `json:"reason,omitempty"`
	Inventory      *StudentInventorySlotsSnapshot `json:"inventory,omitempty"`
	Container      *ContainerSlotsSnapshot        `json:"container,omitempty"`
}

type PlayerSnapshot struct {
	ID               string            `json:"id"`
	CharacterID      int64             `json:"characterId"`
	DisplayName      string            `json:"displayName"`
	X                float64           `json:"x"`
	Y                float64           `json:"y"`
	Facing           string            `json:"facing"`
	Moving           bool              `json:"moving"`
	AvatarID         string            `json:"avatarId"`
	Equipment        EquipmentSnapshot `json:"equipment"`
	LastProcessedSeq int64             `json:"lastProcessedSeq"`
}

type NPCSnapshot struct {
	ID            string           `json:"id"`
	CharacterID   int64            `json:"characterId,omitempty"`
	Name          string           `json:"name"`
	X             float64          `json:"x"`
	Y             float64          `json:"y"`
	Facing        string           `json:"facing"`
	Moving        bool             `json:"moving"`
	SpriteKey     string           `json:"spriteKey"`
	Dialogue      []string         `json:"dialogue"`
	RoutineStatus string           `json:"routineStatus,omitempty"`
	Shop          *stmaps.Shop     `json:"shop,omitempty"`
	Activity      *stmaps.Activity `json:"activity,omitempty"`
}

type CollectibleSnapshot struct {
	ID     string  `json:"id"`
	Kind   string  `json:"kind"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Active bool    `json:"active"`
}

type ResourceNodeSnapshot struct {
	ID     string  `json:"id"`
	Kind   string  `json:"kind"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radius"`
	Active bool    `json:"active"`
	Hits   int     `json:"hits"`
	Needed int     `json:"needed"`
}

type PlacedObjectSnapshot struct {
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

type WorldObjectSnapshot struct {
	ID                string   `json:"id"`
	Kind              string   `json:"kind"`
	Source            string   `json:"source"`
	ItemKey           string   `json:"itemKey,omitempty"`
	ResourceKind      string   `json:"resourceKind,omitempty"`
	Name              string   `json:"name,omitempty"`
	LocationID        string   `json:"locationId,omitempty"`
	ShopID            string   `json:"shopId,omitempty"`
	StorageRole       string   `json:"storageRole,omitempty"`
	X                 float64  `json:"x"`
	Y                 float64  `json:"y"`
	Width             float64  `json:"width,omitempty"`
	Height            float64  `json:"height,omitempty"`
	Radius            float64  `json:"radius,omitempty"`
	InteractionRadius float64  `json:"interactionRadius,omitempty"`
	Active            bool     `json:"active"`
	Collision         bool     `json:"collision"`
	Breakable         bool     `json:"breakable"`
	ReservesPlacement bool     `json:"reservesPlacement"`
	Hits              int      `json:"hits,omitempty"`
	Needed            int      `json:"needed,omitempty"`
	GridX             int      `json:"gridX,omitempty"`
	GridY             int      `json:"gridY,omitempty"`
	PlacedByAppUserID int64    `json:"placedByAppUserId,omitempty"`
	Tags              []string `json:"tags,omitempty"`
}
