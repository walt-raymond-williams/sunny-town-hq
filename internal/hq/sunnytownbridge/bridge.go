package sunnytownbridge

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	hqinventory "hq/internal/hq/inventory"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

type Store struct {
	DB *pgxpool.Pool
}

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (store Store) CommitReward(ctx context.Context, request RewardEventRequest) (RewardEventResponse, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	request.CollectibleID = strings.TrimSpace(request.CollectibleID)
	request.RewardKind = strings.TrimSpace(request.RewardKind)
	if request.EventID == "" || request.AppUserID < 1 || request.RoomID == "" || request.MapID == "" || request.CollectibleID == "" {
		return RewardEventResponse{}, errors.New("reward event is missing required fields")
	}
	if request.RewardKind != "star" || request.Amount != 1 {
		return RewardEventResponse{}, errors.New("unsupported sunny town reward")
	}

	inserted, balance, err := CommitStudentStarReward(ctx, store.DB, StarRewardRequest{
		EventID:       request.EventID,
		AppUserID:     request.AppUserID,
		Source:        "sunny_town_star_collect",
		Delta:         request.Amount,
		RoomID:        request.RoomID,
		MapID:         request.MapID,
		CollectibleID: request.CollectibleID,
	})
	if err != nil {
		return RewardEventResponse{}, err
	}

	return RewardEventResponse{Accepted: true, Duplicate: !inserted, NewStarBalance: balance}, nil
}

func (store Store) CommitResource(ctx context.Context, request ResourceEventRequest) (ResourceEventResponse, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.Source = strings.TrimSpace(request.Source)
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	request.NodeID = strings.TrimSpace(request.NodeID)
	request.ResourceKey = strings.TrimSpace(request.ResourceKey)
	if request.EventID == "" || request.AppUserID < 1 || request.Source == "" || request.RoomID == "" || request.MapID == "" || request.NodeID == "" {
		return ResourceEventResponse{}, errors.New("resource event is missing required fields")
	}
	if request.Source != "sunny_town_mining" {
		return ResourceEventResponse{}, errors.New("unsupported resource event source")
	}
	if request.ResourceKey != "rock" && request.ResourceKey != "crystal" {
		return ResourceEventResponse{}, errors.New("unsupported resource")
	}
	if request.Amount < 1 {
		return ResourceEventResponse{}, errors.New("resource amount must be positive")
	}

	inserted, quantity, err := CommitStudentInventoryLedgerDelta(ctx, store.DB, InventoryLedgerRequest{
		EventID:   request.EventID,
		AppUserID: request.AppUserID,
		Source:    request.Source,
		ItemKey:   request.ResourceKey,
		Delta:     request.Amount,
		RoomID:    request.RoomID,
		MapID:     request.MapID,
		NodeID:    request.NodeID,
	})
	if err != nil {
		return ResourceEventResponse{}, err
	}

	return ResourceEventResponse{Accepted: true, Duplicate: !inserted, ResourceKey: request.ResourceKey, Quantity: quantity}, nil
}

func (store Store) LoadPosition(ctx context.Context, appUserID int64) (PositionResponse, error) {
	if appUserID < 1 {
		return PositionResponse{}, errors.New("app_user_id is required")
	}

	var position PositionResponse
	err := store.DB.QueryRow(
		ctx,
		`
			select app_user_id, room_id, map_id, x, y, facing, updated_at
			from student_sunny_town_position
			where app_user_id = $1
		`,
		appUserID,
	).Scan(&position.AppUserID, &position.RoomID, &position.MapID, &position.X, &position.Y, &position.Facing, &position.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return PositionResponse{Found: false}, nil
	}
	if err != nil {
		return PositionResponse{}, err
	}
	position.Found = true
	return position, nil
}

func (store Store) SavePosition(ctx context.Context, request PositionRequest) (PositionResponse, error) {
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	request.Facing = strings.TrimSpace(request.Facing)
	if request.AppUserID < 1 || request.RoomID == "" || request.MapID == "" {
		return PositionResponse{}, errors.New("position is missing required fields")
	}
	if !IsFacing(request.Facing) {
		return PositionResponse{}, errors.New("invalid sunny town facing")
	}
	if math.IsNaN(request.X) || math.IsInf(request.X, 0) || math.IsNaN(request.Y) || math.IsInf(request.Y, 0) {
		return PositionResponse{}, errors.New("invalid sunny town coordinates")
	}

	var position PositionResponse
	err := store.DB.QueryRow(
		ctx,
		`
			insert into student_sunny_town_position (
				app_user_id,
				room_id,
				map_id,
				x,
				y,
				facing
			)
			values ($1, $2, $3, $4, $5, $6)
			on conflict (app_user_id) do update
			set room_id = excluded.room_id,
				map_id = excluded.map_id,
				x = excluded.x,
				y = excluded.y,
				facing = excluded.facing,
				updated_at = now()
			returning app_user_id, room_id, map_id, x, y, facing, updated_at
		`,
		request.AppUserID,
		request.RoomID,
		request.MapID,
		request.X,
		request.Y,
		request.Facing,
	).Scan(&position.AppUserID, &position.RoomID, &position.MapID, &position.X, &position.Y, &position.Facing, &position.UpdatedAt)
	if err != nil {
		return PositionResponse{}, err
	}
	position.Found = true
	return position, nil
}

func IsFacing(value string) bool {
	return value == "up" || value == "down" || value == "left" || value == "right"
}

func (store Store) LoadMapObjects(ctx context.Context, roomID string, mapID string) (MapObjectsResponse, error) {
	roomID = strings.TrimSpace(roomID)
	mapID = strings.TrimSpace(mapID)
	if roomID == "" || mapID == "" {
		return MapObjectsResponse{}, ErrInvalidMapObject
	}

	rows, err := store.DB.Query(
		ctx,
		`
			select id, room_id, map_id, grid_x, grid_y, item_key, placed_by_app_user_id
			from sunny_town_map_object
			where room_id = $1 and map_id = $2
			order by id
		`,
		roomID,
		mapID,
	)
	if err != nil {
		return MapObjectsResponse{}, err
	}
	defer rows.Close()

	response := MapObjectsResponse{Objects: []MapObjectResponse{}}
	for rows.Next() {
		var object MapObjectResponse
		if err := rows.Scan(&object.ID, &object.RoomID, &object.MapID, &object.GridX, &object.GridY, &object.ItemKey, &object.PlacedByAppUserID); err != nil {
			return MapObjectsResponse{}, err
		}
		response.Objects = append(response.Objects, object)
	}
	return response, rows.Err()
}

func (store Store) PlaceMapObject(ctx context.Context, request PlaceMapObjectRequest) (MapObjectResponse, error) {
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	request.ItemKey = strings.TrimSpace(request.ItemKey)
	if request.AppUserID < 1 || request.RoomID == "" || request.MapID == "" || request.GridX < 0 || request.GridY < 0 {
		return MapObjectResponse{}, ErrInvalidMapObject
	}
	if request.ItemKey != StoneBlockItemKey {
		return MapObjectResponse{}, ErrUnsupportedMapObject
	}

	tx, err := store.DB.Begin(ctx)
	if err != nil {
		return MapObjectResponse{}, err
	}
	defer tx.Rollback(ctx)

	consumed, err := hqinventory.ConsumeStudentItem(ctx, tx, request.AppUserID, request.ItemKey, 1)
	if err != nil {
		return MapObjectResponse{}, err
	}
	if !consumed {
		return MapObjectResponse{}, ErrMapObjectNotOwned
	}

	var object MapObjectResponse
	err = tx.QueryRow(
		ctx,
		`
			insert into sunny_town_map_object (
				room_id,
				map_id,
				grid_x,
				grid_y,
				item_key,
				placed_by_app_user_id
			)
			values ($1, $2, $3, $4, $5, $6)
			returning id, room_id, map_id, grid_x, grid_y, item_key, placed_by_app_user_id
		`,
		request.RoomID,
		request.MapID,
		request.GridX,
		request.GridY,
		request.ItemKey,
		request.AppUserID,
	).Scan(&object.ID, &object.RoomID, &object.MapID, &object.GridX, &object.GridY, &object.ItemKey, &object.PlacedByAppUserID)
	if IsUniqueViolation(err) {
		return MapObjectResponse{}, ErrMapObjectOccupied
	}
	if err != nil {
		return MapObjectResponse{}, err
	}

	quantity, err := LoadStudentInventoryQuantity(ctx, tx, request.AppUserID, request.ItemKey)
	if err != nil {
		return MapObjectResponse{}, err
	}
	object.RemainingItemAmount = quantity

	if err := tx.Commit(ctx); err != nil {
		return MapObjectResponse{}, err
	}
	return object, nil
}

func (store Store) RemoveMapObject(ctx context.Context, request RemoveMapObjectRequest) (MapObjectResponse, error) {
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	if request.AppUserID < 1 || request.RoomID == "" || request.MapID == "" || request.GridX < 0 || request.GridY < 0 {
		return MapObjectResponse{}, ErrInvalidMapObject
	}

	tx, err := store.DB.Begin(ctx)
	if err != nil {
		return MapObjectResponse{}, err
	}
	defer tx.Rollback(ctx)

	var object MapObjectResponse
	err = tx.QueryRow(
		ctx,
		`
			delete from sunny_town_map_object
			where room_id = $1 and map_id = $2 and grid_x = $3 and grid_y = $4
			returning id, room_id, map_id, grid_x, grid_y, item_key, placed_by_app_user_id
		`,
		request.RoomID,
		request.MapID,
		request.GridX,
		request.GridY,
	).Scan(&object.ID, &object.RoomID, &object.MapID, &object.GridX, &object.GridY, &object.ItemKey, &object.PlacedByAppUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return MapObjectResponse{}, ErrMapObjectNotFound
	}
	if err != nil {
		return MapObjectResponse{}, err
	}
	if object.ItemKey != StoneBlockItemKey {
		return MapObjectResponse{}, ErrUnsupportedMapObject
	}

	if err := hqinventory.IncrementStudentItem(ctx, tx, request.AppUserID, object.ItemKey, 1); err != nil {
		return MapObjectResponse{}, err
	}
	quantity, err := LoadStudentInventoryQuantity(ctx, tx, request.AppUserID, object.ItemKey)
	if err != nil {
		return MapObjectResponse{}, err
	}
	object.RemainingItemAmount = quantity

	if err := tx.Commit(ctx); err != nil {
		return MapObjectResponse{}, err
	}
	return object, nil
}

func LoadStudentInventoryQuantity(ctx context.Context, querier rowQuerier, userID int64, itemKey string) (int, error) {
	var quantity int
	err := querier.QueryRow(
		ctx,
		`
			select coalesce(sii.quantity, 0)
			from inventory_item_type iit
			left join student_inventory_item sii on sii.item_type_id = iit.id
				and sii.app_user_id = $1
			where iit.key = $2
		`,
		userID,
		itemKey,
	).Scan(&quantity)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return quantity, err
}

func CommitStudentStarReward(ctx context.Context, querier rowQuerier, request StarRewardRequest) (bool, int, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.Source = strings.TrimSpace(request.Source)
	if request.EventID == "" || request.AppUserID < 1 || request.Source == "" || request.Delta == 0 {
		return false, 0, errors.New("star reward is missing required fields")
	}

	var inserted bool
	var balance int
	err := querier.QueryRow(
		ctx,
		`
			with inserted as (
				insert into student_star_ledger (
					app_user_id,
					event_id,
					source,
					delta,
					room_id,
					map_id,
					collectible_id
				)
				values ($1, $2, $3, $4, nullif($5, ''), nullif($6, ''), nullif($7, ''))
				on conflict (event_id) do nothing
				returning app_user_id, delta
			),
			updated_wallet as (
				insert into student_wallet (app_user_id, star_balance)
				select app_user_id, delta from inserted
				on conflict (app_user_id) do update
				set star_balance = student_wallet.star_balance + excluded.star_balance,
					updated_at = now()
				returning star_balance
			)
			select exists(select 1 from inserted) as inserted,
				coalesce(
					(select star_balance from updated_wallet),
					(select star_balance from student_wallet where app_user_id = $1),
					0
				) as star_balance
		`,
		request.AppUserID,
		request.EventID,
		request.Source,
		request.Delta,
		request.RoomID,
		request.MapID,
		request.CollectibleID,
	).Scan(&inserted, &balance)
	return inserted, balance, err
}

func CommitStudentInventoryLedgerDelta(ctx context.Context, querier rowQuerier, request InventoryLedgerRequest) (bool, int, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.Source = strings.TrimSpace(request.Source)
	request.ItemKey = strings.TrimSpace(request.ItemKey)
	if request.EventID == "" || request.AppUserID < 1 || request.Source == "" || request.ItemKey == "" || request.Delta == 0 {
		return false, 0, errors.New("inventory ledger event is missing required fields")
	}

	var inserted bool
	var quantity int
	err := querier.QueryRow(
		ctx,
		`
			with item_type as (
				select id
				from inventory_item_type
				where key = $4
			),
			inserted as (
				insert into student_inventory_ledger (
					app_user_id,
					event_id,
					source,
					item_type_id,
					delta,
					room_id,
					map_id,
					node_id
				)
				select $1, $2, $3, id, $5, nullif($6, ''), nullif($7, ''), nullif($8, '')
				from item_type
				on conflict (event_id) do nothing
				returning app_user_id, item_type_id, delta
			),
			updated_inventory as (
				insert into student_inventory_item (app_user_id, item_type_id, quantity)
				select app_user_id, item_type_id, delta from inserted
				on conflict (app_user_id, item_type_id) do update
				set quantity = student_inventory_item.quantity + excluded.quantity,
					updated_at = now()
				returning quantity
			)
			select exists(select 1 from inserted) as inserted,
				coalesce(
					(select quantity from updated_inventory),
					(
						select sii.quantity
						from student_inventory_item sii
						join item_type on item_type.id = sii.item_type_id
						where sii.app_user_id = $1
					),
					0
				) as quantity
		`,
		request.AppUserID,
		request.EventID,
		request.Source,
		request.ItemKey,
		request.Delta,
		request.RoomID,
		request.MapID,
		request.NodeID,
	).Scan(&inserted, &quantity)
	return inserted, quantity, err
}

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type HTTPHandler struct {
	store         Store
	serviceSecret string
}

func NewHTTPHandler(store Store, serviceSecret string) HTTPHandler {
	return HTTPHandler{store: store, serviceSecret: serviceSecret}
}

func (handler HTTPHandler) HandleRewardEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !handler.authorized(w, r) {
		return
	}

	var request RewardEventRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
		return
	}

	response, err := handler.store.CommitReward(r.Context(), request)
	if err != nil {
		log.Printf("commit sunny town reward: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "reward event could not be accepted"})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (handler HTTPHandler) HandleResourceEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !handler.authorized(w, r) {
		return
	}

	var request ResourceEventRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
		return
	}

	response, err := handler.store.CommitResource(r.Context(), request)
	if err != nil {
		log.Printf("commit sunny town resource: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "resource event could not be accepted"})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (handler HTTPHandler) HandleStudentEquipment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !handler.authorized(w, r) {
		return
	}

	userID, err := parsePositiveIntQuery(r, "app_user_id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "app_user_id is required"})
		return
	}

	equipment, err := hqinventory.LoadStudentEquipment(r.Context(), handler.store.DB, userID)
	if err != nil {
		log.Printf("load internal sunny town student equipment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "equipment could not be loaded"})
		return
	}

	writeJSON(w, http.StatusOK, equipment)
}

func (handler HTTPHandler) HandleInventoryQuantity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !handler.authorized(w, r) {
		return
	}

	userID, err := parsePositiveIntQuery(r, "app_user_id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "app_user_id is required"})
		return
	}
	itemKey := strings.TrimSpace(r.URL.Query().Get("item_key"))
	if itemKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "item_key is required"})
		return
	}

	quantity, err := LoadStudentInventoryQuantity(r.Context(), handler.store.DB, userID, itemKey)
	if err != nil {
		log.Printf("load internal sunny town inventory quantity: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "inventory quantity could not be loaded"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"item_key": itemKey, "quantity": quantity})
}

func (handler HTTPHandler) HandlePlayerPosition(w http.ResponseWriter, r *http.Request) {
	if !handler.authorized(w, r) {
		return
	}

	switch r.Method {
	case http.MethodGet:
		userID, err := parsePositiveIntQuery(r, "app_user_id")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "app_user_id is required"})
			return
		}
		position, err := handler.store.LoadPosition(r.Context(), userID)
		if err != nil {
			log.Printf("load internal sunny town position: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "position could not be loaded"})
			return
		}
		writeJSON(w, http.StatusOK, position)
	case http.MethodPost:
		var request PositionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
			return
		}
		position, err := handler.store.SavePosition(r.Context(), request)
		if err != nil {
			log.Printf("save internal sunny town position: %v", err)
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "position could not be saved"})
			return
		}
		writeJSON(w, http.StatusOK, position)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (handler HTTPHandler) HandleMapObjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !handler.authorized(w, r) {
		return
	}

	response, err := handler.store.LoadMapObjects(r.Context(), r.URL.Query().Get("room_id"), r.URL.Query().Get("map_id"))
	if err != nil {
		log.Printf("load internal sunny town map objects: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "map objects could not be loaded"})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (handler HTTPHandler) HandlePlaceMapObject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !handler.authorized(w, r) {
		return
	}

	var request PlaceMapObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
		return
	}

	response, err := handler.store.PlaceMapObject(r.Context(), request)
	if err != nil {
		log.Printf("place internal sunny town map object: %v", err)
		writeJSON(w, StatusForMapObjectError(err), map[string]string{"error": MapObjectErrorMessage(err)})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (handler HTTPHandler) HandleRemoveMapObject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !handler.authorized(w, r) {
		return
	}

	var request RemoveMapObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
		return
	}

	response, err := handler.store.RemoveMapObject(r.Context(), request)
	if err != nil {
		log.Printf("remove internal sunny town map object: %v", err)
		writeJSON(w, StatusForMapObjectError(err), map[string]string{"error": MapObjectErrorMessage(err)})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (handler HTTPHandler) authorized(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-HQ-Service-Secret")) == handler.serviceSecret {
		return true
	}
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "service authentication required"})
	return false
}

func parsePositiveIntQuery(r *http.Request, key string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get(key)), 10, 64)
	if err != nil || value < 1 {
		return 0, errors.New("positive integer is required")
	}
	return value, nil
}

func StatusForMapObjectError(err error) int {
	switch {
	case errors.Is(err, ErrMapObjectNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrMapObjectOccupied), errors.Is(err, ErrMapObjectNotOwned):
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}

func MapObjectErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrMapObjectOccupied):
		return "location is occupied"
	case errors.Is(err, ErrMapObjectNotFound):
		return "map object not found"
	case errors.Is(err, ErrMapObjectNotOwned):
		return "item is not in inventory"
	case errors.Is(err, ErrUnsupportedMapObject):
		return "unsupported map object"
	default:
		return "map object could not be updated"
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
