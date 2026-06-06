package main

import (
	"context"
	"errors"
	"strings"
)

const sunnyTownHotbarSlotCount = 5

var (
	errInvalidHotbarSlot  = errors.New("invalid hotbar slot")
	errHotbarItemNotOwned = errors.New("hotbar item is not owned")
)

type hotbarItemResponse struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	EquipSlot   string `json:"equipSlot,omitempty"`
	VisualKey   string `json:"visualKey,omitempty"`
}

type hotbarSlotResponse struct {
	Slot int                 `json:"slot"`
	Item *hotbarItemResponse `json:"item"`
}

type studentHotbarResponse struct {
	Slots []hotbarSlotResponse `json:"slots"`
}

type hotbarSlotRequest struct {
	Slot    int    `json:"slot"`
	ItemKey string `json:"itemKey"`
}

func (app *app) loadStudentHotbar(ctx context.Context, userID int64) (studentHotbarResponse, error) {
	if err := app.seedDefaultStudentHotbar(ctx, userID); err != nil {
		return studentHotbarResponse{}, err
	}

	rows, err := app.db.Query(
		ctx,
		`
			select slots.slot_index,
				iit.key,
				iit.name,
				iit.description,
				coalesce(sii.quantity, 0) as quantity,
				coalesce(iit.equip_slot, '') as equip_slot,
				coalesce(iit.visual_key, '') as visual_key
			from generate_series(1, $2) as slots(slot_index)
			left join student_hotbar_slot shs on shs.app_user_id = $1
				and shs.slot_index = slots.slot_index
			left join inventory_item_type iit on iit.id = shs.item_type_id
			left join student_inventory_item sii on sii.app_user_id = shs.app_user_id
				and sii.item_type_id = shs.item_type_id
				and sii.quantity > 0
			order by slots.slot_index
		`,
		userID,
		sunnyTownHotbarSlotCount,
	)
	if err != nil {
		return studentHotbarResponse{}, err
	}
	defer rows.Close()

	response := studentHotbarResponse{Slots: []hotbarSlotResponse{}}
	for rows.Next() {
		var slot int
		var key *string
		var name *string
		var description *string
		var quantity int
		var equipSlot string
		var visualKey string
		if err := rows.Scan(&slot, &key, &name, &description, &quantity, &equipSlot, &visualKey); err != nil {
			return studentHotbarResponse{}, err
		}

		slotResponse := hotbarSlotResponse{Slot: slot}
		if key != nil && quantity > 0 {
			slotResponse.Item = &hotbarItemResponse{
				Key:         *key,
				Name:        stringValue(name),
				Description: stringValue(description),
				Quantity:    quantity,
				EquipSlot:   equipSlot,
				VisualKey:   visualKey,
			}
		}
		response.Slots = append(response.Slots, slotResponse)
	}
	if err := rows.Err(); err != nil {
		return studentHotbarResponse{}, err
	}
	return response, nil
}

func (app *app) setStudentHotbarSlot(ctx context.Context, userID int64, request hotbarSlotRequest) (studentHotbarResponse, error) {
	if request.Slot < 1 || request.Slot > sunnyTownHotbarSlotCount {
		return studentHotbarResponse{}, errInvalidHotbarSlot
	}
	itemKey := strings.TrimSpace(request.ItemKey)
	if itemKey == "" {
		_, err := app.db.Exec(
			ctx,
			"delete from student_hotbar_slot where app_user_id = $1 and slot_index = $2",
			userID,
			request.Slot,
		)
		if err != nil {
			return studentHotbarResponse{}, err
		}
		return app.loadStudentHotbar(ctx, userID)
	}

	result, err := app.db.Exec(
		ctx,
		`
			insert into student_hotbar_slot (app_user_id, slot_index, item_type_id)
			select $1, $2, iit.id
			from inventory_item_type iit
			join student_inventory_item sii on sii.item_type_id = iit.id
				and sii.app_user_id = $1
				and sii.quantity > 0
			where iit.key = $3
			on conflict (app_user_id, slot_index) do update
			set item_type_id = excluded.item_type_id,
				updated_at = now()
		`,
		userID,
		request.Slot,
		itemKey,
	)
	if err != nil {
		return studentHotbarResponse{}, err
	}
	if result.RowsAffected() == 0 {
		return studentHotbarResponse{}, errHotbarItemNotOwned
	}
	return app.loadStudentHotbar(ctx, userID)
}

func (app *app) seedDefaultStudentHotbar(ctx context.Context, userID int64) error {
	_, err := app.db.Exec(
		ctx,
		`
			insert into student_hotbar_slot (app_user_id, slot_index, item_type_id)
			select $1, defaults.slot_index, iit.id
			from (values (1, 'pickaxe'), (2, 'stone_block')) as defaults(slot_index, item_key)
			join inventory_item_type iit on iit.key = defaults.item_key
			join student_inventory_item sii on sii.item_type_id = iit.id
				and sii.app_user_id = $1
				and sii.quantity > 0
			where not exists (
				select 1 from student_hotbar_slot existing
				where existing.app_user_id = $1
			)
			on conflict (app_user_id, slot_index) do nothing
		`,
		userID,
	)
	return err
}
