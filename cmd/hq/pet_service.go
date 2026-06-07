package main

import (
	"context"

	hqinventory "hq/internal/hq/inventory"
	hqpet "hq/internal/hq/pet"
	hqsunnytownbridge "hq/internal/hq/sunnytownbridge"

	"github.com/jackc/pgx/v5"
)

func (app *app) petStore() *hqpet.Store {
	return &hqpet.Store{
		DB:                    app.db,
		CookieInventoryKey:    hqinventory.CookieKey,
		NoCookiesError:        errNoCookies,
		ConsumeInventoryItem:  consumePetInventoryItem,
		CommitStudentStarOnce: commitPetStarReward,
	}
}

func consumePetInventoryItem(ctx context.Context, tx pgx.Tx, userID int64, itemKey string, quantity int) (bool, error) {
	return hqinventory.ConsumeStudentItem(ctx, tx, userID, itemKey, quantity)
}

func commitPetStarReward(ctx context.Context, tx pgx.Tx, request hqpet.StarRewardRequest) (bool, int, error) {
	return hqsunnytownbridge.CommitStudentStarReward(ctx, tx, hqsunnytownbridge.StarRewardRequest{
		EventID:       request.EventID,
		AppUserID:     request.AppUserID,
		Source:        request.Source,
		Delta:         request.Delta,
		RoomID:        request.RoomID,
		MapID:         request.MapID,
		CollectibleID: request.CollectibleID,
	})
}
