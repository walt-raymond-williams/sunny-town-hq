package main

import (
	"context"
	"errors"
	"strings"
)

const (
	equipmentSlotGear      = "gear"
	equipmentSlotAccessory = "accessory"
)

var (
	errInvalidEquipmentSlot = errors.New("invalid equipment slot")
	errItemNotEquippable    = errors.New("item is not equippable")
	errItemNotOwned         = errors.New("item is not owned")
)

type equipmentItemResponse struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	EquipSlot   string `json:"equipSlot"`
	VisualKey   string `json:"visualKey"`
}

type equipmentSlotResponse struct {
	Slot string                 `json:"slot"`
	Item *equipmentItemResponse `json:"item"`
}

type studentEquipmentResponse struct {
	Slots []equipmentSlotResponse `json:"slots"`
}

type equipmentChangeRequest struct {
	Slot    string `json:"slot"`
	ItemKey string `json:"itemKey"`
}

func normalizeEquipmentSlot(slot string) (string, error) {
	slot = strings.TrimSpace(slot)
	switch slot {
	case equipmentSlotGear, equipmentSlotAccessory:
		return slot, nil
	default:
		return "", errInvalidEquipmentSlot
	}
}

func (app *app) loadStudentEquipment(ctx context.Context, userID int64) (studentEquipmentResponse, error) {
	rows, err := app.db.Query(
		ctx,
		`
			select slots.slot,
				iit.key,
				iit.name,
				iit.description,
				coalesce(iit.equip_slot, '') as equip_slot,
				coalesce(iit.visual_key, '') as visual_key
			from (values ('gear'), ('accessory')) as slots(slot)
			left join student_equipped_item sei on sei.app_user_id = $1
				and sei.slot = slots.slot
			left join student_inventory_item sii on sii.app_user_id = sei.app_user_id
				and sii.item_type_id = sei.item_type_id
				and sii.quantity > 0
			left join inventory_item_type iit on iit.id = sei.item_type_id
				and sii.app_user_id is not null
			order by case slots.slot when 'gear' then 1 else 2 end
		`,
		userID,
	)
	if err != nil {
		return studentEquipmentResponse{}, err
	}
	defer rows.Close()

	response := studentEquipmentResponse{Slots: []equipmentSlotResponse{}}
	for rows.Next() {
		var slot string
		var key *string
		var name *string
		var description *string
		var equipSlot *string
		var visualKey *string
		if err := rows.Scan(&slot, &key, &name, &description, &equipSlot, &visualKey); err != nil {
			return studentEquipmentResponse{}, err
		}

		slotResponse := equipmentSlotResponse{Slot: slot}
		if key != nil {
			slotResponse.Item = &equipmentItemResponse{
				Key:         *key,
				Name:        stringValue(name),
				Description: stringValue(description),
				EquipSlot:   stringValue(equipSlot),
				VisualKey:   stringValue(visualKey),
			}
		}
		response.Slots = append(response.Slots, slotResponse)
	}
	if err := rows.Err(); err != nil {
		return studentEquipmentResponse{}, err
	}
	return response, nil
}

func (app *app) equipStudentItem(ctx context.Context, userID int64, request equipmentChangeRequest) (studentEquipmentResponse, error) {
	slot, err := normalizeEquipmentSlot(request.Slot)
	if err != nil {
		return studentEquipmentResponse{}, err
	}
	itemKey := strings.TrimSpace(request.ItemKey)
	if itemKey == "" {
		return studentEquipmentResponse{}, errItemNotEquippable
	}

	tx, err := app.db.Begin(ctx)
	if err != nil {
		return studentEquipmentResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var itemTypeID int64
	var equipSlot string
	var ownedQuantity int
	err = tx.QueryRow(
		ctx,
		`
			select iit.id,
				coalesce(iit.equip_slot, ''),
				coalesce(sii.quantity, 0)
			from inventory_item_type iit
			left join student_inventory_item sii on sii.item_type_id = iit.id
				and sii.app_user_id = $1
			where iit.key = $2
		`,
		userID,
		itemKey,
	).Scan(&itemTypeID, &equipSlot, &ownedQuantity)
	if err != nil {
		return studentEquipmentResponse{}, err
	}
	if equipSlot == "" || equipSlot != slot {
		return studentEquipmentResponse{}, errItemNotEquippable
	}
	if ownedQuantity < 1 {
		return studentEquipmentResponse{}, errItemNotOwned
	}

	_, err = tx.Exec(
		ctx,
		`
			insert into student_equipped_item (app_user_id, slot, item_type_id)
			values ($1, $2, $3)
			on conflict (app_user_id, slot) do update
			set item_type_id = excluded.item_type_id,
				updated_at = now()
		`,
		userID,
		slot,
		itemTypeID,
	)
	if err != nil {
		return studentEquipmentResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return studentEquipmentResponse{}, err
	}
	return app.loadStudentEquipment(ctx, userID)
}

func (app *app) unequipStudentItem(ctx context.Context, userID int64, request equipmentChangeRequest) (studentEquipmentResponse, error) {
	slot, err := normalizeEquipmentSlot(request.Slot)
	if err != nil {
		return studentEquipmentResponse{}, err
	}
	_, err = app.db.Exec(
		ctx,
		"delete from student_equipped_item where app_user_id = $1 and slot = $2",
		userID,
		slot,
	)
	if err != nil {
		return studentEquipmentResponse{}, err
	}
	return app.loadStudentEquipment(ctx, userID)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
