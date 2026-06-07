package main

import (
	"context"

	hqinventory "hq/internal/hq/inventory"
)

const cookieInventoryKey = hqinventory.CookieKey

type inventoryItemResponse = hqinventory.ItemResponse
type studentInventoryResponse = hqinventory.StudentResponse
type inventoryQuerier = hqinventory.Querier
type inventoryLoader = hqinventory.Loader

func (app *app) loadStudentInventory(ctx context.Context, userID int64) (studentInventoryResponse, error) {
	return loadStudentInventory(ctx, app.db, userID)
}

func loadStudentInventory(ctx context.Context, querier inventoryLoader, userID int64) (studentInventoryResponse, error) {
	return hqinventory.LoadStudent(ctx, querier, userID)
}

func incrementStudentInventoryItem(ctx context.Context, querier inventoryQuerier, userID int64, itemKey string, delta int) error {
	return hqinventory.IncrementStudentItem(ctx, querier, userID, itemKey, delta)
}

func consumeStudentInventoryItem(ctx context.Context, querier inventoryQuerier, userID int64, itemKey string, quantity int) (bool, error) {
	return hqinventory.ConsumeStudentItem(ctx, querier, userID, itemKey, quantity)
}
