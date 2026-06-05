package main

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

const cookieInventoryKey = "cookie"

type inventoryItemResponse struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
}

type studentInventoryResponse struct {
	Items []inventoryItemResponse `json:"items"`
}

type inventoryQuerier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func (app *app) loadStudentInventory(ctx context.Context, userID int64) (studentInventoryResponse, error) {
	rows, err := app.db.Query(
		ctx,
		`
			select iit.key,
				iit.name,
				iit.description,
				coalesce(sii.quantity, 0) as quantity
			from inventory_item_type iit
			join student_inventory_item sii on sii.item_type_id = iit.id
				and sii.app_user_id = $1
				and sii.quantity > 0
			order by iit.id
		`,
		userID,
	)
	if err != nil {
		return studentInventoryResponse{}, err
	}
	defer rows.Close()

	inventory := studentInventoryResponse{Items: []inventoryItemResponse{}}
	for rows.Next() {
		var item inventoryItemResponse
		if err := rows.Scan(&item.Key, &item.Name, &item.Description, &item.Quantity); err != nil {
			return studentInventoryResponse{}, err
		}
		inventory.Items = append(inventory.Items, item)
	}
	if err := rows.Err(); err != nil {
		return studentInventoryResponse{}, err
	}
	return inventory, nil
}

func incrementStudentInventoryItem(ctx context.Context, querier inventoryQuerier, userID int64, itemKey string, delta int) error {
	itemKey = strings.TrimSpace(itemKey)
	if userID < 1 || itemKey == "" || delta < 1 {
		return errors.New("inventory increment is missing required fields")
	}

	_, err := querier.Exec(
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

func consumeStudentInventoryItem(ctx context.Context, querier inventoryQuerier, userID int64, itemKey string, quantity int) (bool, error) {
	itemKey = strings.TrimSpace(itemKey)
	if userID < 1 || itemKey == "" || quantity < 1 {
		return false, errors.New("inventory consume is missing required fields")
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
