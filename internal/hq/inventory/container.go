package inventory

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrContainerNotFound = errors.New("container not found")
	ErrInvalidContainer  = errors.New("invalid container")
)

type ContainerSlotsResponse struct {
	ContainerID string                  `json:"containerId"`
	SlotCount   int                     `json:"slotCount"`
	Revision    int64                   `json:"revision"`
	Slots       []InventorySlotResponse `json:"slots"`
}

type ContainerTransferRequest struct {
	AppUserID   int64                   `json:"appUserId"`
	Source      InventorySlotDescriptor `json:"source"`
	Destination InventorySlotDescriptor `json:"destination"`
	Mode        string                  `json:"mode"`
}

type ContainerTransferResponse struct {
	Inventory StudentInventorySlotsResponse `json:"inventory"`
	Container ContainerSlotsResponse        `json:"container"`
}

type storageContainerState struct {
	ID        string
	SlotCount int
	Revision  int64
}

func LoadContainerSlots(ctx context.Context, querier inventoryMutationQuerier, containerID string) (ContainerSlotsResponse, error) {
	containerID = strings.TrimSpace(containerID)
	if containerID == "" {
		return ContainerSlotsResponse{}, ErrInvalidContainer
	}

	container, err := loadStorageContainer(ctx, querier, containerID, false)
	if err != nil {
		return ContainerSlotsResponse{}, err
	}

	rows, err := querier.Query(
		ctx,
		`
			select slots.slot_index,
				iit.key,
				iit.name,
				iit.description,
				coalesce(scs.quantity, 0) as quantity,
				coalesce(iit.equip_slot, '') as equip_slot,
				coalesce(iit.visual_key, '') as visual_key,
				coalesce(iit.icon_key, '') as icon_key,
				coalesce(iit.max_stack, 0) as max_stack,
				coalesce(iit.category, '') as category
			from generate_series(0, $2 - 1) as slots(slot_index)
			left join storage_container_slot scs on scs.container_id = $1
				and scs.slot_index = slots.slot_index
			left join inventory_item_type iit on iit.id = scs.item_type_id
			order by slots.slot_index
		`,
		container.ID,
		container.SlotCount,
	)
	if err != nil {
		return ContainerSlotsResponse{}, err
	}
	defer rows.Close()

	response := ContainerSlotsResponse{
		ContainerID: container.ID,
		SlotCount:   container.SlotCount,
		Revision:    container.Revision,
		Slots:       []InventorySlotResponse{},
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
			return ContainerSlotsResponse{}, err
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
		return ContainerSlotsResponse{}, err
	}
	return response, nil
}

func TransferPlayerContainerStack(ctx context.Context, db *pgxpool.Pool, request ContainerTransferRequest) (ContainerTransferResponse, error) {
	request.Mode = strings.TrimSpace(request.Mode)
	if request.Mode == "" {
		request.Mode = "auto"
	}
	request.Source.Kind = strings.TrimSpace(request.Source.Kind)
	request.Source.ContainerID = strings.TrimSpace(request.Source.ContainerID)
	request.Destination.Kind = strings.TrimSpace(request.Destination.Kind)
	request.Destination.ContainerID = strings.TrimSpace(request.Destination.ContainerID)
	if request.AppUserID < 1 {
		return ContainerTransferResponse{}, errors.New("container transfer is missing app_user_id")
	}
	if err := validatePlayerContainerTransfer(request.Source, request.Destination); err != nil {
		return ContainerTransferResponse{}, err
	}

	containerID := request.Source.ContainerID
	if strings.TrimSpace(containerID) == "" {
		containerID = request.Destination.ContainerID
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return ContainerTransferResponse{}, err
	}
	defer tx.Rollback(ctx)

	if err := lockStudentInventory(ctx, tx, request.AppUserID); err != nil {
		return ContainerTransferResponse{}, err
	}
	container, err := loadStorageContainer(ctx, tx, containerID, true)
	if err != nil {
		return ContainerTransferResponse{}, err
	}
	if err := lockStorageContainer(ctx, tx, container.ID); err != nil {
		return ContainerTransferResponse{}, err
	}
	if err := validateTransferSlotAgainstContainer(request.Source, container); err != nil {
		return ContainerTransferResponse{}, err
	}
	if err := validateTransferSlotAgainstContainer(request.Destination, container); err != nil {
		return ContainerTransferResponse{}, err
	}

	source, err := loadTransferSlot(ctx, tx, request.AppUserID, request.Source)
	if err != nil {
		return ContainerTransferResponse{}, err
	}
	destination, err := loadTransferSlot(ctx, tx, request.AppUserID, request.Destination)
	if err != nil {
		return ContainerTransferResponse{}, err
	}

	nextSource, nextDestination, err := nextTransferSlotStates(source, destination, request.Mode)
	if err != nil {
		return ContainerTransferResponse{}, err
	}
	if err := saveTransferSlot(ctx, tx, request.AppUserID, request.Source, nextSource); err != nil {
		return ContainerTransferResponse{}, err
	}
	if err := saveTransferSlot(ctx, tx, request.AppUserID, request.Destination, nextDestination); err != nil {
		return ContainerTransferResponse{}, err
	}
	if err := applyPlayerSlotAggregateDelta(ctx, tx, request.AppUserID, request.Source, source, nextSource); err != nil {
		return ContainerTransferResponse{}, err
	}
	if err := applyPlayerSlotAggregateDelta(ctx, tx, request.AppUserID, request.Destination, destination, nextDestination); err != nil {
		return ContainerTransferResponse{}, err
	}
	if err := incrementStorageContainerRevision(ctx, tx, container.ID); err != nil {
		return ContainerTransferResponse{}, err
	}

	inventory, err := LoadStudentSlots(ctx, tx, request.AppUserID)
	if err != nil {
		return ContainerTransferResponse{}, err
	}
	containerSlots, err := LoadContainerSlots(ctx, tx, container.ID)
	if err != nil {
		return ContainerTransferResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ContainerTransferResponse{}, err
	}
	return ContainerTransferResponse{Inventory: inventory, Container: containerSlots}, nil
}

func validatePlayerContainerTransfer(source InventorySlotDescriptor, destination InventorySlotDescriptor) error {
	source.Kind = strings.TrimSpace(source.Kind)
	destination.Kind = strings.TrimSpace(destination.Kind)
	if source.Kind == destination.Kind {
		return ErrUnsupportedInventoryStorage
	}
	if !isTransferStorageKind(source.Kind) || !isTransferStorageKind(destination.Kind) {
		return ErrUnsupportedInventoryStorage
	}
	if source.Kind == inventoryStorageKindContainer && strings.TrimSpace(source.ContainerID) == "" {
		return ErrInvalidContainer
	}
	if destination.Kind == inventoryStorageKindContainer && strings.TrimSpace(destination.ContainerID) == "" {
		return ErrInvalidContainer
	}
	if source.Kind == inventoryStorageKindContainer && destination.Kind == inventoryStorageKindContainer && strings.TrimSpace(source.ContainerID) != strings.TrimSpace(destination.ContainerID) {
		return ErrUnsupportedInventoryStorage
	}
	if source.Kind == inventoryStorageKindPlayer && (source.SlotIndex < 0 || source.SlotIndex >= StudentInventorySlotCount) {
		return ErrInvalidInventorySlot
	}
	if destination.Kind == inventoryStorageKindPlayer && (destination.SlotIndex < 0 || destination.SlotIndex >= StudentInventorySlotCount) {
		return ErrInvalidInventorySlot
	}
	if source.Kind == inventoryStorageKindPlayer && destination.Kind == inventoryStorageKindPlayer && source.SlotIndex == destination.SlotIndex {
		return ErrInvalidInventorySlot
	}
	return nil
}

func isTransferStorageKind(kind string) bool {
	return kind == inventoryStorageKindPlayer || kind == inventoryStorageKindContainer
}

func validateTransferSlotAgainstContainer(descriptor InventorySlotDescriptor, container storageContainerState) error {
	if descriptor.Kind != inventoryStorageKindContainer {
		return nil
	}
	if strings.TrimSpace(descriptor.ContainerID) != container.ID {
		return ErrInvalidContainer
	}
	if descriptor.SlotIndex < 0 || descriptor.SlotIndex >= container.SlotCount {
		return ErrInvalidInventorySlot
	}
	return nil
}

func loadStorageContainer(ctx context.Context, querier rowQuerier, containerID string, forUpdate bool) (storageContainerState, error) {
	query := `
		select id, slot_count, revision
		from storage_container
		where id = $1
	`
	if forUpdate {
		query += " for update"
	}
	var container storageContainerState
	err := querier.QueryRow(ctx, query, strings.TrimSpace(containerID)).Scan(&container.ID, &container.SlotCount, &container.Revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return storageContainerState{}, ErrContainerNotFound
	}
	return container, err
}

func lockStorageContainer(ctx context.Context, querier rowQuerier, containerID string) error {
	var locked int
	return querier.QueryRow(
		ctx,
		`select 1 from (select pg_advisory_xact_lock(hashtext('storage-container:' || $1))) as locked`,
		containerID,
	).Scan(&locked)
}

func loadTransferSlot(ctx context.Context, querier rowQuerier, userID int64, descriptor InventorySlotDescriptor) (inventorySlotState, error) {
	if descriptor.Kind == inventoryStorageKindPlayer {
		return loadInventorySlot(ctx, querier, userID, descriptor.SlotIndex)
	}
	return loadContainerSlot(ctx, querier, descriptor.ContainerID, descriptor.SlotIndex)
}

func loadContainerSlot(ctx context.Context, querier rowQuerier, containerID string, slotIndex int) (inventorySlotState, error) {
	var itemTypeID *int64
	var quantity *int
	var maxStack *int
	err := querier.QueryRow(
		ctx,
		`
			select scs.item_type_id,
				scs.quantity,
				iit.max_stack
			from generate_series($2, $2) as slots(slot_index)
			left join storage_container_slot scs on scs.container_id = $1
				and scs.slot_index = slots.slot_index
			left join inventory_item_type iit on iit.id = scs.item_type_id
		`,
		strings.TrimSpace(containerID),
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
	return inventorySlotState{ItemTypeID: *itemTypeID, Quantity: *quantity, MaxStack: stackMax, Occupied: true}, nil
}

func nextTransferSlotStates(source inventorySlotState, destination inventorySlotState, mode string) (inventorySlotState, inventorySlotState, error) {
	if mode == "auto" {
		mode = automaticInventoryMoveMode(source, destination)
	}
	switch mode {
	case "move":
		if !source.Occupied {
			return source, destination, ErrInventorySourceEmpty
		}
		if destination.Occupied {
			return source, destination, ErrInventoryDestinationOccupied
		}
		return inventorySlotState{}, source, nil
	case "swap":
		if !source.Occupied {
			return source, destination, ErrInventorySourceEmpty
		}
		if !destination.Occupied {
			return source, destination, ErrInventoryDestinationEmpty
		}
		return destination, source, nil
	case "merge":
		if !source.Occupied {
			return source, destination, ErrInventorySourceEmpty
		}
		if !destination.Occupied {
			return source, destination, ErrInventoryDestinationEmpty
		}
		if source.ItemTypeID != destination.ItemTypeID {
			return source, destination, ErrInventoryIncompatibleMerge
		}
		capacity := destination.MaxStack - destination.Quantity
		if capacity < 1 {
			return source, destination, ErrInventoryStackFull
		}
		moved := minInt(source.Quantity, capacity)
		nextSource := source
		nextSource.Quantity -= moved
		if nextSource.Quantity == 0 {
			nextSource = inventorySlotState{}
		}
		nextDestination := destination
		nextDestination.Quantity += moved
		return nextSource, nextDestination, nil
	default:
		return source, destination, ErrUnsupportedInventoryMoveMode
	}
}

func saveTransferSlot(ctx context.Context, querier Querier, userID int64, descriptor InventorySlotDescriptor, state inventorySlotState) error {
	if descriptor.Kind == inventoryStorageKindPlayer {
		return savePlayerInventorySlot(ctx, querier, userID, descriptor.SlotIndex, state)
	}
	return saveContainerInventorySlot(ctx, querier, descriptor.ContainerID, descriptor.SlotIndex, state)
}

func savePlayerInventorySlot(ctx context.Context, querier Querier, userID int64, slotIndex int, state inventorySlotState) error {
	if !state.Occupied {
		_, err := querier.Exec(ctx, "delete from student_inventory_slot where app_user_id = $1 and slot_index = $2", userID, slotIndex)
		return err
	}
	_, err := querier.Exec(
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
		state.ItemTypeID,
		state.Quantity,
	)
	return err
}

func saveContainerInventorySlot(ctx context.Context, querier Querier, containerID string, slotIndex int, state inventorySlotState) error {
	if !state.Occupied {
		_, err := querier.Exec(ctx, "delete from storage_container_slot where container_id = $1 and slot_index = $2", strings.TrimSpace(containerID), slotIndex)
		return err
	}
	_, err := querier.Exec(
		ctx,
		`
			insert into storage_container_slot (container_id, slot_index, item_type_id, quantity)
			values ($1, $2, $3, $4)
			on conflict (container_id, slot_index) do update
			set item_type_id = excluded.item_type_id,
				quantity = excluded.quantity,
				updated_at = now()
		`,
		strings.TrimSpace(containerID),
		slotIndex,
		state.ItemTypeID,
		state.Quantity,
	)
	return err
}

func applyPlayerSlotAggregateDelta(ctx context.Context, querier Querier, userID int64, descriptor InventorySlotDescriptor, before inventorySlotState, after inventorySlotState) error {
	if descriptor.Kind != inventoryStorageKindPlayer {
		return nil
	}
	deltas := map[int64]int{}
	if before.Occupied {
		deltas[before.ItemTypeID] -= before.Quantity
	}
	if after.Occupied {
		deltas[after.ItemTypeID] += after.Quantity
	}
	for itemTypeID, delta := range deltas {
		if delta == 0 {
			continue
		}
		if err := applyStudentInventoryAggregateDelta(ctx, querier, userID, itemTypeID, delta); err != nil {
			return err
		}
	}
	return nil
}

func applyStudentInventoryAggregateDelta(ctx context.Context, querier Querier, userID int64, itemTypeID int64, delta int) error {
	if delta > 0 {
		_, err := querier.Exec(
			ctx,
			`
				insert into student_inventory_item (app_user_id, item_type_id, quantity)
				values ($1, $2, $3)
				on conflict (app_user_id, item_type_id) do update
				set quantity = student_inventory_item.quantity + excluded.quantity,
					updated_at = now()
			`,
			userID,
			itemTypeID,
			delta,
		)
		return err
	}
	result, err := querier.Exec(
		ctx,
		`
			update student_inventory_item
			set quantity = quantity + $3,
				updated_at = now()
			where app_user_id = $1
				and item_type_id = $2
				and quantity >= $4
		`,
		userID,
		itemTypeID,
		delta,
		-delta,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrInventorySourceEmpty
	}
	_, err = querier.Exec(ctx, "delete from student_inventory_item where app_user_id = $1 and item_type_id = $2 and quantity = 0", userID, itemTypeID)
	return err
}

func incrementStorageContainerRevision(ctx context.Context, querier Querier, containerID string) error {
	_, err := querier.Exec(
		ctx,
		`
			update storage_container
			set revision = revision + 1,
				updated_at = now()
			where id = $1
		`,
		containerID,
	)
	return err
}
