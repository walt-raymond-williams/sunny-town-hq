package protocol

import stmaps "hq/internal/sunnytown/maps"

type ClientMessage struct {
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

type EquipmentSnapshot map[string]string
type InventorySnapshot map[string]int

type ServerMessage struct {
	Type           string                 `json:"type"`
	SelfID         string                 `json:"selfId,omitempty"`
	RoomID         string                 `json:"roomId,omitempty"`
	MapID          string                 `json:"mapId,omitempty"`
	Map            *stmaps.GameMap        `json:"map,omitempty"`
	Tick           int64                  `json:"tick,omitempty"`
	ServerTimeMS   int64                  `json:"serverTimeMs,omitempty"`
	Players        []PlayerSnapshot       `json:"players,omitempty"`
	Collectibles   []CollectibleSnapshot  `json:"collectibles,omitempty"`
	ResourceNodes  []ResourceNodeSnapshot `json:"resourceNodes,omitempty"`
	PlacedObjects  []PlacedObjectSnapshot `json:"placedObjects,omitempty"`
	PlacedObject   *PlacedObjectSnapshot  `json:"placedObject,omitempty"`
	WorldObjects   []WorldObjectSnapshot  `json:"worldObjects,omitempty"`
	WorldObject    *WorldObjectSnapshot   `json:"worldObject,omitempty"`
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
