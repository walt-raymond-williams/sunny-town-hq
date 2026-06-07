package main

import (
	"context"

	hqinventory "hq/internal/hq/inventory"
	hqpet "hq/internal/hq/pet"

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
