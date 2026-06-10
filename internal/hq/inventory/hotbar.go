package inventory

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const SunnyTownHotbarSlotCount = 5

var (
	ErrInvalidHotbarSlot  = errors.New("invalid hotbar slot")
	ErrHotbarItemNotOwned = errors.New("hotbar item is not owned")
)

type HotbarItemResponse struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	EquipSlot   string `json:"equipSlot,omitempty"`
	VisualKey   string `json:"visualKey,omitempty"`
	IconKey     string `json:"iconKey,omitempty"`
	MaxStack    int    `json:"maxStack,omitempty"`
	Category    string `json:"category,omitempty"`
}

type HotbarSlotResponse struct {
	Slot int                 `json:"slot"`
	Item *HotbarItemResponse `json:"item"`
}

type StudentHotbarResponse struct {
	Slots []HotbarSlotResponse `json:"slots"`
}

type HotbarSlotRequest struct {
	Slot    int    `json:"slot"`
	ItemKey string `json:"itemKey"`
}

func LoadStudentHotbar(ctx context.Context, db *pgxpool.Pool, userID int64) (StudentHotbarResponse, error) {
	if err := SeedDefaultStudentHotbar(ctx, db, userID); err != nil {
		return StudentHotbarResponse{}, err
	}

	rows, err := db.Query(
		ctx,
		`
			select slots.slot_index,
				iit.key,
				iit.name,
				iit.description,
				coalesce(sii.quantity, 0) as quantity,
				coalesce(iit.equip_slot, '') as equip_slot,
				coalesce(iit.visual_key, '') as visual_key,
				coalesce(iit.icon_key, '') as icon_key,
				coalesce(iit.max_stack, 0) as max_stack,
				coalesce(iit.category, '') as category
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
		SunnyTownHotbarSlotCount,
	)
	if err != nil {
		return StudentHotbarResponse{}, err
	}
	defer rows.Close()

	response := StudentHotbarResponse{Slots: []HotbarSlotResponse{}}
	for rows.Next() {
		var slot int
		var key *string
		var name *string
		var description *string
		var quantity int
		var equipSlot string
		var visualKey string
		var iconKey string
		var maxStack int
		var category string
		if err := rows.Scan(&slot, &key, &name, &description, &quantity, &equipSlot, &visualKey, &iconKey, &maxStack, &category); err != nil {
			return StudentHotbarResponse{}, err
		}

		slotResponse := HotbarSlotResponse{Slot: slot}
		if key != nil && quantity > 0 {
			slotResponse.Item = &HotbarItemResponse{
				Key:         *key,
				Name:        StringValue(name),
				Description: StringValue(description),
				Quantity:    quantity,
				EquipSlot:   equipSlot,
				VisualKey:   visualKey,
				IconKey:     iconKey,
				MaxStack:    maxStack,
				Category:    category,
			}
		}
		response.Slots = append(response.Slots, slotResponse)
	}
	if err := rows.Err(); err != nil {
		return StudentHotbarResponse{}, err
	}
	return response, nil
}

func SetStudentHotbarSlot(ctx context.Context, db *pgxpool.Pool, userID int64, request HotbarSlotRequest) (StudentHotbarResponse, error) {
	if request.Slot < 1 || request.Slot > SunnyTownHotbarSlotCount {
		return StudentHotbarResponse{}, ErrInvalidHotbarSlot
	}
	itemKey := strings.TrimSpace(request.ItemKey)
	if itemKey == "" {
		_, err := db.Exec(
			ctx,
			"delete from student_hotbar_slot where app_user_id = $1 and slot_index = $2",
			userID,
			request.Slot,
		)
		if err != nil {
			return StudentHotbarResponse{}, err
		}
		return LoadStudentHotbar(ctx, db, userID)
	}

	result, err := db.Exec(
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
		return StudentHotbarResponse{}, err
	}
	if result.RowsAffected() == 0 {
		return StudentHotbarResponse{}, ErrHotbarItemNotOwned
	}
	return LoadStudentHotbar(ctx, db, userID)
}

func SeedDefaultStudentHotbar(ctx context.Context, db *pgxpool.Pool, userID int64) error {
	_, err := db.Exec(
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
