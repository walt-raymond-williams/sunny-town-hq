package main

import (
	"context"

	hqinventory "hq/internal/hq/inventory"
)

const (
	equipmentSlotGear      = hqinventory.EquipmentSlotGear
	equipmentSlotAccessory = hqinventory.EquipmentSlotAccessory
	equipmentSlotTool      = hqinventory.EquipmentSlotTool
)

var (
	errInvalidEquipmentSlot = hqinventory.ErrInvalidEquipmentSlot
	errItemNotEquippable    = hqinventory.ErrItemNotEquippable
	errItemNotOwned         = hqinventory.ErrItemNotOwned
)

type equipmentItemResponse = hqinventory.EquipmentItemResponse
type equipmentSlotResponse = hqinventory.EquipmentSlotResponse
type studentEquipmentResponse = hqinventory.StudentEquipmentResponse
type equipmentChangeRequest = hqinventory.EquipmentChangeRequest

func normalizeEquipmentSlot(slot string) (string, error) {
	return hqinventory.NormalizeEquipmentSlot(slot)
}

func (app *app) loadStudentEquipment(ctx context.Context, userID int64) (studentEquipmentResponse, error) {
	return hqinventory.LoadStudentEquipment(ctx, app.db, userID)
}

func (app *app) equipStudentItem(ctx context.Context, userID int64, request equipmentChangeRequest) (studentEquipmentResponse, error) {
	return hqinventory.EquipStudentItem(ctx, app.db, userID, request)
}

func (app *app) unequipStudentItem(ctx context.Context, userID int64, request equipmentChangeRequest) (studentEquipmentResponse, error) {
	return hqinventory.UnequipStudentItem(ctx, app.db, userID, request)
}

func stringValue(value *string) string {
	return hqinventory.StringValue(value)
}
