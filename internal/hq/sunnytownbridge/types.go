package sunnytownbridge

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const StoneBlockItemKey = "stone_block"

var (
	ErrInvalidMapObject     = errors.New("invalid map object")
	ErrMapObjectOccupied    = errors.New("map object location is occupied")
	ErrMapObjectNotFound    = errors.New("map object not found")
	ErrMapObjectNotOwned    = errors.New("map object item is not owned")
	ErrUnsupportedMapObject = errors.New("unsupported map object")
)

type RewardEventRequest struct {
	EventID       string `json:"event_id"`
	AppUserID     int64  `json:"app_user_id"`
	RoomID        string `json:"room_id"`
	MapID         string `json:"map_id"`
	CollectibleID string `json:"collectible_id"`
	RewardKind    string `json:"reward_kind"`
	Amount        int    `json:"amount"`
}

type RewardEventResponse struct {
	Accepted       bool `json:"accepted"`
	Duplicate      bool `json:"duplicate"`
	NewStarBalance int  `json:"new_star_balance"`
}

type ResourceEventRequest struct {
	EventID     string `json:"event_id"`
	AppUserID   int64  `json:"app_user_id"`
	Source      string `json:"source"`
	RoomID      string `json:"room_id"`
	MapID       string `json:"map_id"`
	NodeID      string `json:"node_id"`
	ResourceKey string `json:"resource_key"`
	Amount      int    `json:"amount"`
}

type ResourceEventResponse struct {
	Accepted    bool   `json:"accepted"`
	Duplicate   bool   `json:"duplicate"`
	ResourceKey string `json:"resource_key"`
	Quantity    int    `json:"quantity"`
}

type NPCJobProductionRequest struct {
	EventID     string `json:"event_id"`
	CharacterID int64  `json:"character_id"`
	RoomID      string `json:"room_id"`
	MapID       string `json:"map_id"`
	NPCKey      string `json:"npc_key"`
	JobKey      string `json:"job_key"`
	LocationID  string `json:"location_id"`
	OutputKey   string `json:"output_key"`
	Amount      int    `json:"amount"`
}

type NPCJobProductionResponse struct {
	Accepted  bool `json:"accepted"`
	Duplicate bool `json:"duplicate"`
}

type PositionRequest struct {
	AppUserID int64   `json:"app_user_id"`
	RoomID    string  `json:"room_id"`
	MapID     string  `json:"map_id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Facing    string  `json:"facing"`
}

type PositionResponse struct {
	Found     bool      `json:"found"`
	AppUserID int64     `json:"app_user_id,omitempty"`
	RoomID    string    `json:"room_id,omitempty"`
	MapID     string    `json:"map_id,omitempty"`
	X         float64   `json:"x,omitempty"`
	Y         float64   `json:"y,omitempty"`
	Facing    string    `json:"facing,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type MapObjectResponse struct {
	ID                  int64  `json:"id"`
	RoomID              string `json:"room_id"`
	MapID               string `json:"map_id"`
	GridX               int    `json:"grid_x"`
	GridY               int    `json:"grid_y"`
	ItemKey             string `json:"item_key"`
	PlacedByAppUserID   int64  `json:"placed_by_app_user_id"`
	RemainingItemAmount int    `json:"remaining_item_amount,omitempty"`
}

type MapObjectsResponse struct {
	Objects []MapObjectResponse `json:"objects"`
}

type EnsureNPCCharactersRequest struct {
	RoomID string                    `json:"room_id"`
	NPCs   []EnsureNPCCharacterInput `json:"npcs"`
}

type EnsureNPCCharacterInput struct {
	NPCKey      string `json:"npc_key"`
	DisplayName string `json:"display_name"`
	AvatarID    string `json:"avatar_id"`
}

type NPCCharacterResponse struct {
	CharacterID int64  `json:"character_id"`
	RoomID      string `json:"room_id"`
	NPCKey      string `json:"npc_key"`
	DisplayName string `json:"display_name"`
	AvatarID    string `json:"avatar_id"`
}

type NPCCharactersResponse struct {
	NPCs []NPCCharacterResponse `json:"npcs"`
}

type PlaceMapObjectRequest struct {
	AppUserID int64  `json:"app_user_id"`
	RoomID    string `json:"room_id"`
	MapID     string `json:"map_id"`
	GridX     int    `json:"grid_x"`
	GridY     int    `json:"grid_y"`
	ItemKey   string `json:"item_key"`
}

type RemoveMapObjectRequest struct {
	AppUserID int64  `json:"app_user_id"`
	RoomID    string `json:"room_id"`
	MapID     string `json:"map_id"`
	GridX     int    `json:"grid_x"`
	GridY     int    `json:"grid_y"`
}

type StarRewardRequest struct {
	EventID       string
	AppUserID     int64
	Source        string
	Delta         int
	RoomID        string
	MapID         string
	CollectibleID string
}

type InventoryLedgerRequest struct {
	EventID   string
	AppUserID int64
	Source    string
	ItemKey   string
	Delta     int
	RoomID    string
	MapID     string
	NodeID    string
}

type NPCJobProductionLedgerRequest struct {
	EventID     string
	CharacterID int64
	RoomID      string
	MapID       string
	NPCKey      string
	JobKey      string
	LocationID  string
	OutputKey   string
	Amount      int
}

type Store struct {
	DB *pgxpool.Pool
}

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}
