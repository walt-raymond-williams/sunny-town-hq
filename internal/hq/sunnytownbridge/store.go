package sunnytownbridge

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"
)

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
