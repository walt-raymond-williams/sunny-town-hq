package inventory

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const CookieKey = "cookie"
const StudentInventorySlotCount = 30

var ErrInventoryFull = errors.New("inventory is full")

type ItemResponse struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	EquipSlot   string `json:"equipSlot,omitempty"`
	VisualKey   string `json:"visualKey,omitempty"`
	IconKey     string `json:"iconKey,omitempty"`
	MaxStack    int    `json:"maxStack,omitempty"`
	Category    string `json:"category,omitempty"`
	Equipped    bool   `json:"equipped"`
}

type StudentResponse struct {
	Items []ItemResponse `json:"items"`
}

type InventorySlotItemResponse struct {
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

type InventorySlotResponse struct {
	SlotIndex int                        `json:"slotIndex"`
	Item      *InventorySlotItemResponse `json:"item"`
}

type StudentInventorySlotsResponse struct {
	SlotCount int                     `json:"slotCount"`
	Slots     []InventorySlotResponse `json:"slots"`
	Items     []ItemResponse          `json:"items"`
}

type Querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Loader interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type inventoryMutationQuerier interface {
	Querier
	rowQuerier
	Loader
}

type txBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

type inventorySlotQuantity struct {
	slotIndex int
	quantity  int
}

func LoadStudent(ctx context.Context, querier Loader, userID int64) (StudentResponse, error) {
	rows, err := querier.Query(
		ctx,
		`
			select iit.key,
				iit.name,
				iit.description,
				coalesce(sii.quantity, 0) as quantity,
				coalesce(iit.equip_slot, '') as equip_slot,
				coalesce(iit.visual_key, '') as visual_key,
				coalesce(iit.icon_key, '') as icon_key,
				coalesce(iit.max_stack, 0) as max_stack,
				coalesce(iit.category, '') as category,
				sei.app_user_id is not null as equipped
			from inventory_item_type iit
			join student_inventory_item sii on sii.item_type_id = iit.id
				and sii.app_user_id = $1
				and sii.quantity > 0
			left join student_equipped_item sei on sei.app_user_id = sii.app_user_id
				and sei.item_type_id = iit.id
			order by iit.id
		`,
		userID,
	)
	if err != nil {
		return StudentResponse{}, err
	}
	defer rows.Close()

	inventory := StudentResponse{Items: []ItemResponse{}}
	for rows.Next() {
		var item ItemResponse
		if err := rows.Scan(
			&item.Key,
			&item.Name,
			&item.Description,
			&item.Quantity,
			&item.EquipSlot,
			&item.VisualKey,
			&item.IconKey,
			&item.MaxStack,
			&item.Category,
			&item.Equipped,
		); err != nil {
			return StudentResponse{}, err
		}
		inventory.Items = append(inventory.Items, item)
	}
	if err := rows.Err(); err != nil {
		return StudentResponse{}, err
	}
	return inventory, nil
}

func LoadStudentSlots(ctx context.Context, querier Loader, userID int64) (StudentInventorySlotsResponse, error) {
	rows, err := querier.Query(
		ctx,
		`
			select slots.slot_index,
				iit.key,
				iit.name,
				iit.description,
				coalesce(sis.quantity, 0) as quantity,
				coalesce(iit.equip_slot, '') as equip_slot,
				coalesce(iit.visual_key, '') as visual_key,
				coalesce(iit.icon_key, '') as icon_key,
				coalesce(iit.max_stack, 0) as max_stack,
				coalesce(iit.category, '') as category
			from generate_series(0, $2 - 1) as slots(slot_index)
			left join student_inventory_slot sis on sis.app_user_id = $1
				and sis.slot_index = slots.slot_index
			left join inventory_item_type iit on iit.id = sis.item_type_id
			order by slots.slot_index
		`,
		userID,
		StudentInventorySlotCount,
	)
	if err != nil {
		return StudentInventorySlotsResponse{}, err
	}
	defer rows.Close()

	response := StudentInventorySlotsResponse{
		SlotCount: StudentInventorySlotCount,
		Slots:     []InventorySlotResponse{},
		Items:     []ItemResponse{},
	}
	for rows.Next() {
		var slotIndex int
		var key *string
		var name *string
		var description *string
		var quantity int
		var equipSlot string
		var visualKey string
		var iconKey string
		var maxStack int
		var category string
		if err := rows.Scan(&slotIndex, &key, &name, &description, &quantity, &equipSlot, &visualKey, &iconKey, &maxStack, &category); err != nil {
			return StudentInventorySlotsResponse{}, err
		}

		slot := InventorySlotResponse{SlotIndex: slotIndex}
		if key != nil && quantity > 0 {
			slot.Item = &InventorySlotItemResponse{
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
		response.Slots = append(response.Slots, slot)
	}
	if err := rows.Err(); err != nil {
		return StudentInventorySlotsResponse{}, err
	}

	aggregate, err := LoadStudent(ctx, querier, userID)
	if err != nil {
		return StudentInventorySlotsResponse{}, err
	}
	response.Items = aggregate.Items
	return response, nil
}

func IncrementStudentItem(ctx context.Context, querier Querier, userID int64, itemKey string, delta int) error {
	itemKey = strings.TrimSpace(itemKey)
	if userID < 1 || itemKey == "" || delta < 1 {
		return errors.New("inventory increment is missing required fields")
	}

	if beginner, ok := querier.(txBeginner); ok {
		tx, err := beginner.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		if err := incrementStudentItemInTx(ctx, tx, userID, itemKey, delta); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	mutator, ok := querier.(inventoryMutationQuerier)
	if !ok {
		return errors.New("inventory increment requires query support")
	}
	return incrementStudentItemInTx(ctx, mutator, userID, itemKey, delta)
}

func incrementStudentItemInTx(ctx context.Context, querier inventoryMutationQuerier, userID int64, itemKey string, delta int) error {
	if err := lockStudentInventory(ctx, querier, userID); err != nil {
		return err
	}

	itemTypeID, maxStack, err := loadItemTypeForInventoryMutation(ctx, querier, itemKey)
	if err != nil {
		return err
	}

	remaining := delta
	fillableSlots := []inventorySlotQuantity{}
	rows, err := querier.Query(
		ctx,
		`
			select slot_index, quantity
			from student_inventory_slot
			where app_user_id = $1
				and item_type_id = $2
				and quantity < $3
			order by slot_index
		`,
		userID,
		itemTypeID,
		maxStack,
	)
	if err != nil {
		return err
	}
	for rows.Next() {
		var slotIndex int
		var quantity int
		if err := rows.Scan(&slotIndex, &quantity); err != nil {
			rows.Close()
			return err
		}
		fillableSlots = append(fillableSlots, inventorySlotQuantity{slotIndex: slotIndex, quantity: quantity})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for _, slot := range fillableSlots {
		space := maxStack - slot.quantity
		added := minInt(remaining, space)
		if added < 1 {
			continue
		}
		if _, err := querier.Exec(
			ctx,
			`
				update student_inventory_slot
				set quantity = quantity + $3,
					updated_at = now()
				where app_user_id = $1 and slot_index = $2
			`,
			userID,
			slot.slotIndex,
			added,
		); err != nil {
			return err
		}
		remaining -= added
		if remaining == 0 {
			break
		}
	}

	if remaining > 0 {
		emptySlots := []int{}
		emptyRows, err := querier.Query(
			ctx,
			`
				select slots.slot_index
				from generate_series(0, $2 - 1) as slots(slot_index)
				left join student_inventory_slot sis on sis.app_user_id = $1
					and sis.slot_index = slots.slot_index
				where sis.slot_index is null
					or sis.item_type_id is null
				order by slots.slot_index
			`,
			userID,
			StudentInventorySlotCount,
		)
		if err != nil {
			return err
		}
		for emptyRows.Next() {
			var slotIndex int
			if err := emptyRows.Scan(&slotIndex); err != nil {
				emptyRows.Close()
				return err
			}
			emptySlots = append(emptySlots, slotIndex)
		}
		if err := emptyRows.Err(); err != nil {
			emptyRows.Close()
			return err
		}
		emptyRows.Close()

		for _, slotIndex := range emptySlots {
			added := minInt(remaining, maxStack)
			if _, err := querier.Exec(
				ctx,
				`
					insert into student_inventory_slot (app_user_id, slot_index, item_type_id, quantity)
					values ($1, $2, $3, $4)
					on conflict (app_user_id, slot_index) do update
					set item_type_id = excluded.item_type_id,
						quantity = excluded.quantity,
						updated_at = now()
				`,
				userID,
				slotIndex,
				itemTypeID,
				added,
			); err != nil {
				return err
			}
			remaining -= added
			if remaining == 0 {
				break
			}
		}
	}

	if remaining > 0 {
		return ErrInventoryFull
	}

	_, err = querier.Exec(
		ctx,
		`
			insert into student_inventory_item (app_user_id, item_type_id, quantity)
			select $1, iit.id, $3
			from inventory_item_type iit
			where iit.key = $2
			on conflict (app_user_id, item_type_id) do update
			set quantity = student_inventory_item.quantity + excluded.quantity,
				updated_at = now()
		`,
		userID,
		itemKey,
		delta,
	)
	return err
}

func ConsumeStudentItem(ctx context.Context, querier Querier, userID int64, itemKey string, quantity int) (bool, error) {
	itemKey = strings.TrimSpace(itemKey)
	if userID < 1 || itemKey == "" || quantity < 1 {
		return false, errors.New("inventory consume is missing required fields")
	}

	if beginner, ok := querier.(txBeginner); ok {
		tx, err := beginner.Begin(ctx)
		if err != nil {
			return false, err
		}
		defer tx.Rollback(ctx)
		consumed, err := consumeStudentItemInTx(ctx, tx, userID, itemKey, quantity)
		if err != nil || !consumed {
			return consumed, err
		}
		return consumed, tx.Commit(ctx)
	}
	mutator, ok := querier.(inventoryMutationQuerier)
	if !ok {
		return false, errors.New("inventory consume requires query support")
	}
	return consumeStudentItemInTx(ctx, mutator, userID, itemKey, quantity)
}

func consumeStudentItemInTx(ctx context.Context, querier inventoryMutationQuerier, userID int64, itemKey string, quantity int) (bool, error) {
	if err := lockStudentInventory(ctx, querier, userID); err != nil {
		return false, err
	}

	itemTypeID, _, err := loadItemTypeForInventoryMutation(ctx, querier, itemKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	var available int
	if err := querier.QueryRow(
		ctx,
		`
			select coalesce(sum(quantity), 0)::integer
			from student_inventory_slot
			where app_user_id = $1 and item_type_id = $2
		`,
		userID,
		itemTypeID,
	).Scan(&available); err != nil {
		return false, err
	}
	if available < quantity {
		return false, nil
	}

	remaining := quantity
	slots := []inventorySlotQuantity{}
	rows, err := querier.Query(
		ctx,
		`
			select slot_index, quantity
			from student_inventory_slot
			where app_user_id = $1 and item_type_id = $2
			order by slot_index desc
		`,
		userID,
		itemTypeID,
	)
	if err != nil {
		return false, err
	}
	for rows.Next() {
		var slotIndex int
		var slotQuantity int
		if err := rows.Scan(&slotIndex, &slotQuantity); err != nil {
			rows.Close()
			return false, err
		}
		slots = append(slots, inventorySlotQuantity{slotIndex: slotIndex, quantity: slotQuantity})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return false, err
	}
	rows.Close()

	for _, slot := range slots {
		removed := minInt(remaining, slot.quantity)
		newQuantity := slot.quantity - removed
		if newQuantity == 0 {
			if _, err := querier.Exec(ctx, "delete from student_inventory_slot where app_user_id = $1 and slot_index = $2", userID, slot.slotIndex); err != nil {
				return false, err
			}
		} else {
			if _, err := querier.Exec(
				ctx,
				`
					update student_inventory_slot
					set quantity = $3,
						updated_at = now()
					where app_user_id = $1 and slot_index = $2
				`,
				userID,
				slot.slotIndex,
				newQuantity,
			); err != nil {
				return false, err
			}
		}
		remaining -= removed
		if remaining == 0 {
			break
		}
	}

	result, err := querier.Exec(
		ctx,
		`
			update student_inventory_item sii
			set quantity = quantity - $3,
				updated_at = now()
			from inventory_item_type iit
			where sii.item_type_id = iit.id
				and sii.app_user_id = $1
				and iit.key = $2
				and sii.quantity >= $3
		`,
		userID,
		itemKey,
		quantity,
	)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

func lockStudentInventory(ctx context.Context, querier rowQuerier, userID int64) error {
	var locked int
	return querier.QueryRow(
		ctx,
		`select 1 from (select pg_advisory_xact_lock($1)) as locked`,
		userID,
	).Scan(&locked)
}

func loadItemTypeForInventoryMutation(ctx context.Context, querier rowQuerier, itemKey string) (int64, int, error) {
	var itemTypeID int64
	var maxStack int
	err := querier.QueryRow(
		ctx,
		`
			select id, greatest(coalesce(max_stack, 0), 1)
			from inventory_item_type
			where key = $1
		`,
		itemKey,
	).Scan(&itemTypeID, &maxStack)
	return itemTypeID, maxStack, err
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
