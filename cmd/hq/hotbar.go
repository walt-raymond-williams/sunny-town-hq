package main

import (
	"context"

	hqinventory "hq/internal/hq/inventory"
)

const sunnyTownHotbarSlotCount = hqinventory.SunnyTownHotbarSlotCount

var (
	errInvalidHotbarSlot  = hqinventory.ErrInvalidHotbarSlot
	errHotbarItemNotOwned = hqinventory.ErrHotbarItemNotOwned
)

type hotbarItemResponse = hqinventory.HotbarItemResponse
type hotbarSlotResponse = hqinventory.HotbarSlotResponse
type studentHotbarResponse = hqinventory.StudentHotbarResponse
type hotbarSlotRequest = hqinventory.HotbarSlotRequest

func (app *app) loadStudentHotbar(ctx context.Context, userID int64) (studentHotbarResponse, error) {
	return hqinventory.LoadStudentHotbar(ctx, app.db, userID)
}

func (app *app) setStudentHotbarSlot(ctx context.Context, userID int64, request hotbarSlotRequest) (studentHotbarResponse, error) {
	return hqinventory.SetStudentHotbarSlot(ctx, app.db, userID, request)
}

func (app *app) seedDefaultStudentHotbar(ctx context.Context, userID int64) error {
	return hqinventory.SeedDefaultStudentHotbar(ctx, app.db, userID)
}
