package inventory

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const inventoryStorageKindPlayer = "player_inventory"
const inventoryStorageKindContainer = "container"

var (
	ErrUnsupportedInventoryMoveMode = errors.New("unsupported inventory move mode")
	ErrUnsupportedInventoryStorage  = errors.New("unsupported inventory storage")
	ErrInvalidInventorySlot         = errors.New("invalid inventory slot")
	ErrInventorySourceEmpty         = errors.New("inventory source slot is empty")
	ErrInventoryDestinationEmpty    = errors.New("inventory destination slot is empty")
	ErrInventoryDestinationOccupied = errors.New("inventory destination slot is occupied")
	ErrInventoryIncompatibleMerge   = errors.New("inventory stacks cannot be merged")
	ErrInventoryStackFull           = errors.New("inventory stack is full")
)

type InventorySlotDescriptor struct {
	Kind        string `json:"kind"`
	SlotIndex   int    `json:"slotIndex"`
	ContainerID string `json:"containerId,omitempty"`
}

type InventoryMoveRequest struct {
	Source      InventorySlotDescriptor `json:"source"`
	Destination InventorySlotDescriptor `json:"destination"`
	Mode        string                  `json:"mode"`
}

func MoveStudentInventoryStack(ctx context.Context, db *pgxpool.Pool, userID int64, request InventoryMoveRequest) (StudentInventorySlotsResponse, error) {
	request.Mode = strings.TrimSpace(request.Mode)
	if request.Mode == "" {
		request.Mode = "auto"
	}
	if err := validateInventorySlotDescriptor(request.Source); err != nil {
		return StudentInventorySlotsResponse{}, err
	}
	if err := validateInventorySlotDescriptor(request.Destination); err != nil {
		return StudentInventorySlotsResponse{}, err
	}
	if request.Source.SlotIndex == request.Destination.SlotIndex {
		return StudentInventorySlotsResponse{}, ErrInvalidInventorySlot
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return StudentInventorySlotsResponse{}, err
	}
	defer tx.Rollback(ctx)

	if err := lockStudentInventory(ctx, tx, userID); err != nil {
		return StudentInventorySlotsResponse{}, err
	}

	source, err := loadInventorySlot(ctx, tx, userID, request.Source.SlotIndex)
	if err != nil {
		return StudentInventorySlotsResponse{}, err
	}
	destination, err := loadInventorySlot(ctx, tx, userID, request.Destination.SlotIndex)
	if err != nil {
		return StudentInventorySlotsResponse{}, err
	}

	mode := request.Mode
	if mode == "auto" {
		mode = automaticInventoryMoveMode(source, destination)
	}

	switch mode {
	case "move":
		err = moveInventoryStack(ctx, tx, userID, request.Source.SlotIndex, source, request.Destination.SlotIndex, destination)
	case "swap":
		err = swapInventoryStacks(ctx, tx, userID, request.Source.SlotIndex, source, request.Destination.SlotIndex, destination)
	case "merge":
		err = mergeInventoryStacks(ctx, tx, userID, request.Source.SlotIndex, source, request.Destination.SlotIndex, destination)
	default:
		err = ErrUnsupportedInventoryMoveMode
	}
	if err != nil {
		return StudentInventorySlotsResponse{}, err
	}

	response, err := LoadStudentSlots(ctx, tx, userID)
	if err != nil {
		return StudentInventorySlotsResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StudentInventorySlotsResponse{}, err
	}
	return response, nil
}

func validateInventorySlotDescriptor(descriptor InventorySlotDescriptor) error {
	if strings.TrimSpace(descriptor.Kind) != inventoryStorageKindPlayer {
		return ErrUnsupportedInventoryStorage
	}
	if descriptor.SlotIndex < 0 || descriptor.SlotIndex >= StudentInventorySlotCount {
		return ErrInvalidInventorySlot
	}
	return nil
}

type inventorySlotState struct {
	ItemTypeID int64
	Quantity   int
	MaxStack   int
	Occupied   bool
}

func loadInventorySlot(ctx context.Context, querier rowQuerier, userID int64, slotIndex int) (inventorySlotState, error) {
	var itemTypeID *int64
	var quantity *int
	var maxStack *int
	err := querier.QueryRow(
		ctx,
		`
			select sis.item_type_id,
				sis.quantity,
				iit.max_stack
			from generate_series($2, $2) as slots(slot_index)
			left join student_inventory_slot sis on sis.app_user_id = $1
				and sis.slot_index = slots.slot_index
			left join inventory_item_type iit on iit.id = sis.item_type_id
		`,
		userID,
		slotIndex,
	).Scan(&itemTypeID, &quantity, &maxStack)
	if err != nil {
		return inventorySlotState{}, err
	}
	if itemTypeID == nil || quantity == nil || *quantity < 1 {
		return inventorySlotState{}, nil
	}
	stackMax := IntValue(maxStack)
	if stackMax < 1 {
		stackMax = *quantity
	}
	return inventorySlotState{
		ItemTypeID: *itemTypeID,
		Quantity:   *quantity,
		MaxStack:   stackMax,
		Occupied:   true,
	}, nil
}

func automaticInventoryMoveMode(source inventorySlotState, destination inventorySlotState) string {
	if !source.Occupied {
		return "move"
	}
	if !destination.Occupied {
		return "move"
	}
	if source.ItemTypeID == destination.ItemTypeID {
		return "merge"
	}
	return "swap"
}

func moveInventoryStack(ctx context.Context, querier Querier, userID int64, sourceIndex int, source inventorySlotState, destinationIndex int, destination inventorySlotState) error {
	if !source.Occupied {
		return ErrInventorySourceEmpty
	}
	if destination.Occupied {
		return ErrInventoryDestinationOccupied
	}
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
		destinationIndex,
		source.ItemTypeID,
		source.Quantity,
	); err != nil {
		return err
	}
	_, err := querier.Exec(ctx, "delete from student_inventory_slot where app_user_id = $1 and slot_index = $2", userID, sourceIndex)
	return err
}

func swapInventoryStacks(ctx context.Context, querier Querier, userID int64, sourceIndex int, source inventorySlotState, destinationIndex int, destination inventorySlotState) error {
	if !source.Occupied {
		return ErrInventorySourceEmpty
	}
	if !destination.Occupied {
		return ErrInventoryDestinationEmpty
	}
	if _, err := querier.Exec(
		ctx,
		`
			update student_inventory_slot
			set item_type_id = case slot_index
					when $2 then $5::bigint
					when $4 then $3::bigint
				end,
				quantity = case slot_index
					when $2 then $6::integer
					when $4 then $7::integer
				end,
				updated_at = now()
			where app_user_id = $1
				and slot_index in ($2, $4)
		`,
		userID,
		sourceIndex,
		source.ItemTypeID,
		destinationIndex,
		destination.ItemTypeID,
		destination.Quantity,
		source.Quantity,
	); err != nil {
		return err
	}
	return nil
}

func mergeInventoryStacks(ctx context.Context, querier Querier, userID int64, sourceIndex int, source inventorySlotState, destinationIndex int, destination inventorySlotState) error {
	if !source.Occupied {
		return ErrInventorySourceEmpty
	}
	if !destination.Occupied {
		return ErrInventoryDestinationEmpty
	}
	if source.ItemTypeID != destination.ItemTypeID {
		return ErrInventoryIncompatibleMerge
	}
	capacity := destination.MaxStack - destination.Quantity
	if capacity < 1 {
		return ErrInventoryStackFull
	}
	moved := minInt(source.Quantity, capacity)
	sourceRemaining := source.Quantity - moved
	if _, err := querier.Exec(
		ctx,
		`
			update student_inventory_slot
			set quantity = quantity + $3,
				updated_at = now()
			where app_user_id = $1 and slot_index = $2
		`,
		userID,
		destinationIndex,
		moved,
	); err != nil {
		return err
	}
	if sourceRemaining == 0 {
		_, err := querier.Exec(ctx, "delete from student_inventory_slot where app_user_id = $1 and slot_index = $2", userID, sourceIndex)
		return err
	}
	_, err := querier.Exec(
		ctx,
		`
			update student_inventory_slot
			set quantity = $3,
				updated_at = now()
			where app_user_id = $1 and slot_index = $2
		`,
		userID,
		sourceIndex,
		sourceRemaining,
	)
	return err
}
