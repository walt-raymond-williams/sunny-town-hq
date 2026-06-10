package inventory

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	EquipmentSlotGear      = "gear"
	EquipmentSlotAccessory = "accessory"
	EquipmentSlotTool      = "tool"
)

var (
	ErrInvalidEquipmentSlot = errors.New("invalid equipment slot")
	ErrItemNotEquippable    = errors.New("item is not equippable")
	ErrItemNotOwned         = errors.New("item is not owned")
)

type EquipmentItemResponse struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	EquipSlot   string `json:"equipSlot"`
	VisualKey   string `json:"visualKey"`
	IconKey     string `json:"iconKey,omitempty"`
	MaxStack    int    `json:"maxStack,omitempty"`
	Category    string `json:"category,omitempty"`
}

type EquipmentSlotResponse struct {
	Slot string                 `json:"slot"`
	Item *EquipmentItemResponse `json:"item"`
}

type StudentEquipmentResponse struct {
	Slots []EquipmentSlotResponse `json:"slots"`
}

type EquipmentChangeRequest struct {
	Slot    string `json:"slot"`
	ItemKey string `json:"itemKey"`
}

func NormalizeEquipmentSlot(slot string) (string, error) {
	slot = strings.TrimSpace(slot)
	switch slot {
	case EquipmentSlotGear, EquipmentSlotAccessory, EquipmentSlotTool:
		return slot, nil
	default:
		return "", ErrInvalidEquipmentSlot
	}
}

func LoadStudentEquipment(ctx context.Context, db *pgxpool.Pool, userID int64) (StudentEquipmentResponse, error) {
	rows, err := db.Query(
		ctx,
		`
			select slots.slot,
				iit.key,
				iit.name,
				iit.description,
				coalesce(iit.equip_slot, '') as equip_slot,
				coalesce(iit.visual_key, '') as visual_key,
				coalesce(iit.icon_key, '') as icon_key,
				coalesce(iit.max_stack, 0) as max_stack,
				coalesce(iit.category, '') as category
			from (values ('gear'), ('accessory'), ('tool')) as slots(slot)
			left join student_equipped_item sei on sei.app_user_id = $1
				and sei.slot = slots.slot
			left join student_inventory_item sii on sii.app_user_id = sei.app_user_id
				and sii.item_type_id = sei.item_type_id
				and sii.quantity > 0
			left join inventory_item_type iit on iit.id = sei.item_type_id
				and sii.app_user_id is not null
			order by case slots.slot when 'gear' then 1 when 'accessory' then 2 else 3 end
		`,
		userID,
	)
	if err != nil {
		return StudentEquipmentResponse{}, err
	}
	defer rows.Close()

	response := StudentEquipmentResponse{Slots: []EquipmentSlotResponse{}}
	for rows.Next() {
		var slot string
		var key *string
		var name *string
		var description *string
		var equipSlot *string
		var visualKey *string
		var iconKey *string
		var maxStack *int
		var category *string
		if err := rows.Scan(&slot, &key, &name, &description, &equipSlot, &visualKey, &iconKey, &maxStack, &category); err != nil {
			return StudentEquipmentResponse{}, err
		}

		slotResponse := EquipmentSlotResponse{Slot: slot}
		if key != nil {
			slotResponse.Item = &EquipmentItemResponse{
				Key:         *key,
				Name:        StringValue(name),
				Description: StringValue(description),
				EquipSlot:   StringValue(equipSlot),
				VisualKey:   StringValue(visualKey),
				IconKey:     StringValue(iconKey),
				MaxStack:    IntValue(maxStack),
				Category:    StringValue(category),
			}
		}
		response.Slots = append(response.Slots, slotResponse)
	}
	if err := rows.Err(); err != nil {
		return StudentEquipmentResponse{}, err
	}
	return response, nil
}

func EquipStudentItem(ctx context.Context, db *pgxpool.Pool, userID int64, request EquipmentChangeRequest) (StudentEquipmentResponse, error) {
	slot, err := NormalizeEquipmentSlot(request.Slot)
	if err != nil {
		return StudentEquipmentResponse{}, err
	}
	itemKey := strings.TrimSpace(request.ItemKey)
	if itemKey == "" {
		return StudentEquipmentResponse{}, ErrItemNotEquippable
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return StudentEquipmentResponse{}, err
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
		return StudentEquipmentResponse{}, err
	}
	if equipSlot == "" || equipSlot != slot {
		return StudentEquipmentResponse{}, ErrItemNotEquippable
	}
	if ownedQuantity < 1 {
		return StudentEquipmentResponse{}, ErrItemNotOwned
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
		return StudentEquipmentResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return StudentEquipmentResponse{}, err
	}
	return LoadStudentEquipment(ctx, db, userID)
}

func UnequipStudentItem(ctx context.Context, db *pgxpool.Pool, userID int64, request EquipmentChangeRequest) (StudentEquipmentResponse, error) {
	slot, err := NormalizeEquipmentSlot(request.Slot)
	if err != nil {
		return StudentEquipmentResponse{}, err
	}
	_, err = db.Exec(
		ctx,
		"delete from student_equipped_item where app_user_id = $1 and slot = $2",
		userID,
		slot,
	)
	if err != nil {
		return StudentEquipmentResponse{}, err
	}
	return LoadStudentEquipment(ctx, db, userID)
}

func StringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func IntValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
