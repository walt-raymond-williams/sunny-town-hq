package sunnytownbridge

import (
	"context"
	"errors"
	"net/http"
	"strings"

	hqinventory "hq/internal/hq/inventory"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

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

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
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
