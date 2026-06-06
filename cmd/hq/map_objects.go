package main

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const stoneBlockItemKey = "stone_block"

var (
	errInvalidMapObject     = errors.New("invalid map object")
	errMapObjectOccupied    = errors.New("map object location is occupied")
	errMapObjectNotFound    = errors.New("map object not found")
	errMapObjectNotOwned    = errors.New("map object item is not owned")
	errUnsupportedMapObject = errors.New("unsupported map object")
)

type sunnyTownMapObjectResponse struct {
	ID                  int64  `json:"id"`
	RoomID              string `json:"room_id"`
	MapID               string `json:"map_id"`
	GridX               int    `json:"grid_x"`
	GridY               int    `json:"grid_y"`
	ItemKey             string `json:"item_key"`
	PlacedByAppUserID   int64  `json:"placed_by_app_user_id"`
	RemainingItemAmount int    `json:"remaining_item_amount,omitempty"`
}

type sunnyTownMapObjectsResponse struct {
	Objects []sunnyTownMapObjectResponse `json:"objects"`
}

type sunnyTownPlaceMapObjectRequest struct {
	AppUserID int64  `json:"app_user_id"`
	RoomID    string `json:"room_id"`
	MapID     string `json:"map_id"`
	GridX     int    `json:"grid_x"`
	GridY     int    `json:"grid_y"`
	ItemKey   string `json:"item_key"`
}

type sunnyTownRemoveMapObjectRequest struct {
	AppUserID int64  `json:"app_user_id"`
	RoomID    string `json:"room_id"`
	MapID     string `json:"map_id"`
	GridX     int    `json:"grid_x"`
	GridY     int    `json:"grid_y"`
}

func (app *app) loadSunnyTownMapObjects(ctx context.Context, roomID string, mapID string) (sunnyTownMapObjectsResponse, error) {
	roomID = strings.TrimSpace(roomID)
	mapID = strings.TrimSpace(mapID)
	if roomID == "" || mapID == "" {
		return sunnyTownMapObjectsResponse{}, errInvalidMapObject
	}

	rows, err := app.db.Query(
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
		return sunnyTownMapObjectsResponse{}, err
	}
	defer rows.Close()

	response := sunnyTownMapObjectsResponse{Objects: []sunnyTownMapObjectResponse{}}
	for rows.Next() {
		var object sunnyTownMapObjectResponse
		if err := rows.Scan(
			&object.ID,
			&object.RoomID,
			&object.MapID,
			&object.GridX,
			&object.GridY,
			&object.ItemKey,
			&object.PlacedByAppUserID,
		); err != nil {
			return sunnyTownMapObjectsResponse{}, err
		}
		response.Objects = append(response.Objects, object)
	}
	return response, rows.Err()
}

func (app *app) placeSunnyTownMapObject(ctx context.Context, request sunnyTownPlaceMapObjectRequest) (sunnyTownMapObjectResponse, error) {
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	request.ItemKey = strings.TrimSpace(request.ItemKey)
	if request.AppUserID < 1 || request.RoomID == "" || request.MapID == "" || request.GridX < 0 || request.GridY < 0 {
		return sunnyTownMapObjectResponse{}, errInvalidMapObject
	}
	if request.ItemKey != stoneBlockItemKey {
		return sunnyTownMapObjectResponse{}, errUnsupportedMapObject
	}

	tx, err := app.db.Begin(ctx)
	if err != nil {
		return sunnyTownMapObjectResponse{}, err
	}
	defer tx.Rollback(ctx)

	consumed, err := consumeStudentInventoryItem(ctx, tx, request.AppUserID, request.ItemKey, 1)
	if err != nil {
		return sunnyTownMapObjectResponse{}, err
	}
	if !consumed {
		return sunnyTownMapObjectResponse{}, errMapObjectNotOwned
	}

	var object sunnyTownMapObjectResponse
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
	).Scan(
		&object.ID,
		&object.RoomID,
		&object.MapID,
		&object.GridX,
		&object.GridY,
		&object.ItemKey,
		&object.PlacedByAppUserID,
	)
	if isUniqueViolation(err) {
		return sunnyTownMapObjectResponse{}, errMapObjectOccupied
	}
	if err != nil {
		return sunnyTownMapObjectResponse{}, err
	}

	quantity, err := loadStudentInventoryQuantity(ctx, tx, request.AppUserID, request.ItemKey)
	if err != nil {
		return sunnyTownMapObjectResponse{}, err
	}
	object.RemainingItemAmount = quantity

	if err := tx.Commit(ctx); err != nil {
		return sunnyTownMapObjectResponse{}, err
	}
	return object, nil
}

func (app *app) removeSunnyTownMapObject(ctx context.Context, request sunnyTownRemoveMapObjectRequest) (sunnyTownMapObjectResponse, error) {
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	if request.AppUserID < 1 || request.RoomID == "" || request.MapID == "" || request.GridX < 0 || request.GridY < 0 {
		return sunnyTownMapObjectResponse{}, errInvalidMapObject
	}

	tx, err := app.db.Begin(ctx)
	if err != nil {
		return sunnyTownMapObjectResponse{}, err
	}
	defer tx.Rollback(ctx)

	var object sunnyTownMapObjectResponse
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
	).Scan(
		&object.ID,
		&object.RoomID,
		&object.MapID,
		&object.GridX,
		&object.GridY,
		&object.ItemKey,
		&object.PlacedByAppUserID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sunnyTownMapObjectResponse{}, errMapObjectNotFound
	}
	if err != nil {
		return sunnyTownMapObjectResponse{}, err
	}
	if object.ItemKey != stoneBlockItemKey {
		return sunnyTownMapObjectResponse{}, errUnsupportedMapObject
	}

	if err := incrementStudentInventoryItem(ctx, tx, request.AppUserID, object.ItemKey, 1); err != nil {
		return sunnyTownMapObjectResponse{}, err
	}
	quantity, err := loadStudentInventoryQuantity(ctx, tx, request.AppUserID, object.ItemKey)
	if err != nil {
		return sunnyTownMapObjectResponse{}, err
	}
	object.RemainingItemAmount = quantity

	if err := tx.Commit(ctx); err != nil {
		return sunnyTownMapObjectResponse{}, err
	}
	return object, nil
}

type inventoryQuantityQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadStudentInventoryQuantity(ctx context.Context, querier inventoryQuantityQuerier, userID int64, itemKey string) (int, error) {
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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
