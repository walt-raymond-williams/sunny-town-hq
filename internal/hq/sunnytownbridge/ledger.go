package sunnytownbridge

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

func LoadStudentInventoryQuantity(ctx context.Context, querier rowQuerier, userID int64, itemKey string) (int, error) {
	var quantity int
	err := querier.QueryRow(
		ctx,
		`
			select coalesce(sii.quantity, 0)
			from inventory_item_type iit
			left join student_inventory_item sii on sii.item_type_id = iit.id
				and sii.app_user_id = $1
			where iit.key = $2
		`,
		userID,
		itemKey,
	).Scan(&quantity)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return quantity, err
}

func CommitStudentStarReward(ctx context.Context, querier rowQuerier, request StarRewardRequest) (bool, int, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.Source = strings.TrimSpace(request.Source)
	if request.EventID == "" || request.AppUserID < 1 || request.Source == "" || request.Delta == 0 {
		return false, 0, errors.New("star reward is missing required fields")
	}

	var inserted bool
	var balance int
	err := querier.QueryRow(
		ctx,
		`
			with inserted as (
				insert into student_star_ledger (
					app_user_id,
					event_id,
					source,
					delta,
					room_id,
					map_id,
					collectible_id
				)
				values ($1, $2, $3, $4, nullif($5, ''), nullif($6, ''), nullif($7, ''))
				on conflict (event_id) do nothing
				returning app_user_id, delta
			),
			updated_wallet as (
				insert into student_wallet (app_user_id, star_balance)
				select app_user_id, delta from inserted
				on conflict (app_user_id) do update
				set star_balance = student_wallet.star_balance + excluded.star_balance,
					updated_at = now()
				returning star_balance
			)
			select exists(select 1 from inserted) as inserted,
				coalesce(
					(select star_balance from updated_wallet),
					(select star_balance from student_wallet where app_user_id = $1),
					0
				) as star_balance
		`,
		request.AppUserID,
		request.EventID,
		request.Source,
		request.Delta,
		request.RoomID,
		request.MapID,
		request.CollectibleID,
	).Scan(&inserted, &balance)
	return inserted, balance, err
}

func CommitStudentInventoryLedgerDelta(ctx context.Context, querier rowQuerier, request InventoryLedgerRequest) (bool, int, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.Source = strings.TrimSpace(request.Source)
	request.ItemKey = strings.TrimSpace(request.ItemKey)
	if request.EventID == "" || request.AppUserID < 1 || request.Source == "" || request.ItemKey == "" || request.Delta == 0 {
		return false, 0, errors.New("inventory ledger event is missing required fields")
	}

	var inserted bool
	var quantity int
	err := querier.QueryRow(
		ctx,
		`
			with item_type as (
				select id
				from inventory_item_type
				where key = $4
			),
			inserted as (
				insert into student_inventory_ledger (
					app_user_id,
					event_id,
					source,
					item_type_id,
					delta,
					room_id,
					map_id,
					node_id
				)
				select $1, $2, $3, id, $5, nullif($6, ''), nullif($7, ''), nullif($8, '')
				from item_type
				on conflict (event_id) do nothing
				returning app_user_id, item_type_id, delta
			),
			updated_inventory as (
				insert into student_inventory_item (app_user_id, item_type_id, quantity)
				select app_user_id, item_type_id, delta from inserted
				on conflict (app_user_id, item_type_id) do update
				set quantity = student_inventory_item.quantity + excluded.quantity,
					updated_at = now()
				returning quantity
			)
			select exists(select 1 from inserted) as inserted,
				coalesce(
					(select quantity from updated_inventory),
					(
						select sii.quantity
						from student_inventory_item sii
						join item_type on item_type.id = sii.item_type_id
						where sii.app_user_id = $1
					),
					0
				) as quantity
		`,
		request.AppUserID,
		request.EventID,
		request.Source,
		request.ItemKey,
		request.Delta,
		request.RoomID,
		request.MapID,
		request.NodeID,
	).Scan(&inserted, &quantity)
	return inserted, quantity, err
}

func CommitNPCJobProductionLedger(ctx context.Context, querier rowQuerier, request NPCJobProductionLedgerRequest) (bool, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	request.NPCKey = strings.TrimSpace(request.NPCKey)
	request.JobKey = strings.TrimSpace(request.JobKey)
	request.LocationID = strings.TrimSpace(request.LocationID)
	request.OutputKey = strings.TrimSpace(request.OutputKey)
	if request.EventID == "" || request.CharacterID < 1 || request.RoomID == "" || request.MapID == "" || request.NPCKey == "" || request.JobKey == "" || request.LocationID == "" || request.OutputKey == "" || request.Amount < 1 {
		return false, errors.New("npc job production event is missing required fields")
	}

	var inserted bool
	err := querier.QueryRow(
		ctx,
		`
			insert into sunny_town_npc_job_production_ledger (
				event_id,
				character_id,
				room_id,
				map_id,
				npc_key,
				job_key,
				location_id,
				output_key,
				amount
			)
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			on conflict (event_id) do nothing
			returning true
		`,
		request.EventID,
		request.CharacterID,
		request.RoomID,
		request.MapID,
		request.NPCKey,
		request.JobKey,
		request.LocationID,
		request.OutputKey,
		request.Amount,
	).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return inserted, err
}

func CommitNPCJobProductionBlockedLedger(ctx context.Context, querier rowQuerier, request NPCJobProductionBlockedLedgerRequest) (bool, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	request.NPCKey = strings.TrimSpace(request.NPCKey)
	request.JobKey = strings.TrimSpace(request.JobKey)
	request.LocationID = strings.TrimSpace(request.LocationID)
	request.OutputKey = strings.TrimSpace(request.OutputKey)
	request.Reason = strings.TrimSpace(request.Reason)
	if request.EventID == "" || request.CharacterID < 1 || request.RoomID == "" || request.MapID == "" || request.NPCKey == "" || request.JobKey == "" || request.LocationID == "" || request.OutputKey == "" || request.Amount < 1 || request.Reason == "" {
		return false, errors.New("npc job production blocked event is missing required fields")
	}

	var inserted bool
	err := querier.QueryRow(
		ctx,
		`
			insert into sunny_town_npc_job_production_blocked_ledger (
				event_id,
				character_id,
				room_id,
				map_id,
				npc_key,
				job_key,
				location_id,
				output_key,
				amount,
				reason
			)
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			on conflict (event_id) do nothing
			returning true
		`,
		request.EventID,
		request.CharacterID,
		request.RoomID,
		request.MapID,
		request.NPCKey,
		request.JobKey,
		request.LocationID,
		request.OutputKey,
		request.Amount,
		request.Reason,
	).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return inserted, err
}
