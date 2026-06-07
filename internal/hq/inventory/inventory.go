package inventory

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const CookieKey = "cookie"

type ItemResponse struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	EquipSlot   string `json:"equipSlot,omitempty"`
	VisualKey   string `json:"visualKey,omitempty"`
	Equipped    bool   `json:"equipped"`
}

type StudentResponse struct {
	Items []ItemResponse `json:"items"`
}

type Querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Loader interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
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

func IncrementStudentItem(ctx context.Context, querier Querier, userID int64, itemKey string, delta int) error {
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

func ConsumeStudentItem(ctx context.Context, querier Querier, userID int64, itemKey string, quantity int) (bool, error) {
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
