package sunnytownbridge

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"

	hqcharacters "hq/internal/hq/characters"
	hqinventory "hq/internal/hq/inventory"

	"github.com/jackc/pgx/v5"
)

var (
	supportedNPCJobKeys    = []string{"shopkeeper_stock", "teacher_lesson_prep"}
	supportedNPCOutputKeys = []string{"shop_stock_progress", "lesson_prep_progress"}
)

const (
	npcJobProductionBlockedMissingInputs = "missing_inputs"
	npcJobProductionBlockedOutputFull    = "output_full"
)

func (store Store) CommitReward(ctx context.Context, request RewardEventRequest) (RewardEventResponse, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	request.CollectibleID = strings.TrimSpace(request.CollectibleID)
	request.RewardKind = strings.TrimSpace(request.RewardKind)
	if request.EventID == "" || request.AppUserID < 1 || request.RoomID == "" || request.MapID == "" || request.CollectibleID == "" {
		return RewardEventResponse{}, errors.New("reward event is missing required fields")
	}
	if request.RewardKind != "star" || request.Amount != 1 {
		return RewardEventResponse{}, errors.New("unsupported sunny town reward")
	}

	inserted, balance, err := CommitStudentStarReward(ctx, store.DB, StarRewardRequest{
		EventID:       request.EventID,
		AppUserID:     request.AppUserID,
		Source:        "sunny_town_star_collect",
		Delta:         request.Amount,
		RoomID:        request.RoomID,
		MapID:         request.MapID,
		CollectibleID: request.CollectibleID,
	})
	if err != nil {
		return RewardEventResponse{}, err
	}

	return RewardEventResponse{Accepted: true, Duplicate: !inserted, NewStarBalance: balance}, nil
}

func (store Store) CommitResource(ctx context.Context, request ResourceEventRequest) (ResourceEventResponse, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.Source = strings.TrimSpace(request.Source)
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	request.NodeID = strings.TrimSpace(request.NodeID)
	request.ResourceKey = strings.TrimSpace(request.ResourceKey)
	if request.EventID == "" || request.AppUserID < 1 || request.Source == "" || request.RoomID == "" || request.MapID == "" || request.NodeID == "" {
		return ResourceEventResponse{}, errors.New("resource event is missing required fields")
	}
	if request.Source != "sunny_town_mining" {
		return ResourceEventResponse{}, errors.New("unsupported resource event source")
	}
	if request.ResourceKey != "rock" && request.ResourceKey != "crystal" {
		return ResourceEventResponse{}, errors.New("unsupported resource")
	}
	if request.Amount < 1 {
		return ResourceEventResponse{}, errors.New("resource amount must be positive")
	}

	inserted, quantity, err := CommitStudentInventoryLedgerDelta(ctx, store.DB, InventoryLedgerRequest{
		EventID:   request.EventID,
		AppUserID: request.AppUserID,
		Source:    request.Source,
		ItemKey:   request.ResourceKey,
		Delta:     request.Amount,
		RoomID:    request.RoomID,
		MapID:     request.MapID,
		NodeID:    request.NodeID,
	})
	if err != nil {
		return ResourceEventResponse{}, err
	}

	return ResourceEventResponse{Accepted: true, Duplicate: !inserted, ResourceKey: request.ResourceKey, Quantity: quantity}, nil
}

func (store Store) CommitNPCJobProduction(ctx context.Context, request NPCJobProductionRequest) (NPCJobProductionResponse, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	request.NPCKey = strings.TrimSpace(request.NPCKey)
	request.JobKey = strings.TrimSpace(request.JobKey)
	request.LocationID = strings.TrimSpace(request.LocationID)
	request.OutputKey = strings.TrimSpace(request.OutputKey)
	if request.EventID == "" || request.CharacterID < 1 || request.RoomID == "" || request.MapID == "" || request.NPCKey == "" || request.JobKey == "" || request.LocationID == "" || request.OutputKey == "" {
		return NPCJobProductionResponse{}, errors.New("npc job production event is missing required fields")
	}
	if !slices.Contains(supportedNPCJobKeys, request.JobKey) {
		return NPCJobProductionResponse{}, errors.New("unsupported npc job")
	}
	if !slices.Contains(supportedNPCOutputKeys, request.OutputKey) {
		return NPCJobProductionResponse{}, errors.New("unsupported npc job output")
	}
	if request.Amount < 1 {
		return NPCJobProductionResponse{}, errors.New("npc job production amount must be positive")
	}

	tx, err := store.DB.Begin(ctx)
	if err != nil {
		return NPCJobProductionResponse{}, err
	}
	defer tx.Rollback(ctx)

	if err := lockNPCJobProductionEvent(ctx, tx, request.EventID); err != nil {
		return NPCJobProductionResponse{}, err
	}
	found, blocked, blockedReason, err := loadNPCJobProductionEventOutcome(ctx, tx, request.EventID)
	if err != nil {
		return NPCJobProductionResponse{}, err
	}
	if found {
		if err := tx.Commit(ctx); err != nil {
			return NPCJobProductionResponse{}, err
		}
		return NPCJobProductionResponse{Accepted: true, Duplicate: true, Blocked: blocked, BlockedReason: blockedReason}, nil
	}

	ledgerRequest := NPCJobProductionLedgerRequest{
		EventID:     request.EventID,
		CharacterID: request.CharacterID,
		RoomID:      request.RoomID,
		MapID:       request.MapID,
		NPCKey:      request.NPCKey,
		JobKey:      request.JobKey,
		LocationID:  request.LocationID,
		OutputKey:   request.OutputKey,
		Amount:      request.Amount,
	}

	if request.JobKey == "shopkeeper_stock" && request.OutputKey == "shop_stock_progress" {
		err := hqinventory.CommitShopRecipeProduction(ctx, tx, hqinventory.ShopRecipeProductionRequest{
			EventID:   "npc-job-production:" + request.EventID,
			Source:    "npc_job_production",
			ShopID:    hqinventory.CookieKeeperShopID,
			RecipeKey: hqinventory.CookieRecipeKey,
			Amount:    request.Amount,
		})
		if errors.Is(err, hqinventory.ErrInsufficientIngredient) || errors.Is(err, hqinventory.ErrRecipeOutputFull) {
			reason := npcJobProductionBlockedMissingInputs
			if errors.Is(err, hqinventory.ErrRecipeOutputFull) {
				reason = npcJobProductionBlockedOutputFull
			}
			if _, err := CommitNPCJobProductionBlockedLedger(ctx, tx, NPCJobProductionBlockedLedgerRequest{
				EventID:     request.EventID,
				CharacterID: request.CharacterID,
				RoomID:      request.RoomID,
				MapID:       request.MapID,
				NPCKey:      request.NPCKey,
				JobKey:      request.JobKey,
				LocationID:  request.LocationID,
				OutputKey:   request.OutputKey,
				Amount:      request.Amount,
				Reason:      reason,
			}); err != nil {
				return NPCJobProductionResponse{}, err
			}
			if err := tx.Commit(ctx); err != nil {
				return NPCJobProductionResponse{}, err
			}
			return NPCJobProductionResponse{Accepted: true, Blocked: true, BlockedReason: reason}, nil
		}
		if err != nil {
			return NPCJobProductionResponse{}, err
		}
	}

	inserted, err := CommitNPCJobProductionLedger(ctx, tx, ledgerRequest)
	if err != nil {
		return NPCJobProductionResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return NPCJobProductionResponse{}, err
	}

	return NPCJobProductionResponse{Accepted: true, Duplicate: !inserted}, nil
}

func lockNPCJobProductionEvent(ctx context.Context, querier rowQuerier, eventID string) error {
	var locked int
	return querier.QueryRow(
		ctx,
		`select 1 from (select pg_advisory_xact_lock(hashtext('npc-job-production:' || $1))) as locked`,
		eventID,
	).Scan(&locked)
}

func loadNPCJobProductionEventOutcome(ctx context.Context, querier rowQuerier, eventID string) (bool, bool, string, error) {
	var found bool
	var blocked bool
	var reason string
	err := querier.QueryRow(
		ctx,
		`
			select exists(
					select 1
					from sunny_town_npc_job_production_ledger
					where event_id = $1
				) or exists(
					select 1
					from sunny_town_npc_job_production_blocked_ledger
					where event_id = $1
				) as found,
				exists(
					select 1
					from sunny_town_npc_job_production_blocked_ledger
					where event_id = $1
				) as blocked,
				coalesce(
					(
						select reason
						from sunny_town_npc_job_production_blocked_ledger
						where event_id = $1
					),
					''
				) as reason
		`,
		eventID,
	).Scan(&found, &blocked, &reason)
	return found, blocked, reason, err
}

func (store Store) LoadNPCJobProductionProgress(ctx context.Context, request NPCJobProductionProgressRequest) (NPCJobProductionProgressResponse, error) {
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.JobKey = strings.TrimSpace(request.JobKey)
	request.NPCKey = strings.TrimSpace(request.NPCKey)
	request.LocationID = strings.TrimSpace(request.LocationID)
	if request.RoomID == "" {
		return NPCJobProductionProgressResponse{}, errors.New("room_id is required")
	}
	if request.JobKey != "" && !slices.Contains(supportedNPCJobKeys, request.JobKey) {
		return NPCJobProductionProgressResponse{}, errors.New("unsupported npc job")
	}

	rows, err := store.DB.Query(
		ctx,
		`
			select
				character_id,
				room_id,
				map_id,
				npc_key,
				job_key,
				location_id,
				output_key,
				sum(amount)::integer as total_amount,
				count(*)::integer as event_count,
				min(created_at) as first_at,
				max(created_at) as last_at
			from sunny_town_npc_job_production_ledger
			where room_id = $1
				and ($2 = '' or job_key = $2)
				and ($3 = '' or npc_key = $3)
				and ($4 = '' or location_id = $4)
			group by character_id, room_id, map_id, npc_key, job_key, location_id, output_key
			order by job_key, npc_key, location_id, output_key, character_id
		`,
		request.RoomID,
		request.JobKey,
		request.NPCKey,
		request.LocationID,
	)
	if err != nil {
		return NPCJobProductionProgressResponse{}, err
	}
	defer rows.Close()

	response := NPCJobProductionProgressResponse{Progress: []NPCJobProductionProgressEntry{}}
	for rows.Next() {
		var entry NPCJobProductionProgressEntry
		if err := rows.Scan(
			&entry.CharacterID,
			&entry.RoomID,
			&entry.MapID,
			&entry.NPCKey,
			&entry.JobKey,
			&entry.LocationID,
			&entry.OutputKey,
			&entry.TotalAmount,
			&entry.EventCount,
			&entry.FirstAt,
			&entry.LastAt,
		); err != nil {
			return NPCJobProductionProgressResponse{}, err
		}
		response.Progress = append(response.Progress, entry)
	}
	if err := rows.Err(); err != nil {
		return NPCJobProductionProgressResponse{}, err
	}
	return response, nil
}

func (store Store) LoadPosition(ctx context.Context, appUserID int64) (PositionResponse, error) {
	if appUserID < 1 {
		return PositionResponse{}, errors.New("app_user_id is required")
	}

	var position PositionResponse
	err := store.DB.QueryRow(
		ctx,
		`
			select app_user_id, room_id, map_id, x, y, facing, updated_at
			from student_sunny_town_position
			where app_user_id = $1
		`,
		appUserID,
	).Scan(&position.AppUserID, &position.RoomID, &position.MapID, &position.X, &position.Y, &position.Facing, &position.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return PositionResponse{Found: false}, nil
	}
	if err != nil {
		return PositionResponse{}, err
	}
	position.Found = true
	return position, nil
}

func (store Store) SavePosition(ctx context.Context, request PositionRequest) (PositionResponse, error) {
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	request.Facing = strings.TrimSpace(request.Facing)
	if request.AppUserID < 1 || request.RoomID == "" || request.MapID == "" {
		return PositionResponse{}, errors.New("position is missing required fields")
	}
	if !IsFacing(request.Facing) {
		return PositionResponse{}, errors.New("invalid sunny town facing")
	}
	if math.IsNaN(request.X) || math.IsInf(request.X, 0) || math.IsNaN(request.Y) || math.IsInf(request.Y, 0) {
		return PositionResponse{}, errors.New("invalid sunny town coordinates")
	}

	var position PositionResponse
	err := store.DB.QueryRow(
		ctx,
		`
			insert into student_sunny_town_position (
				app_user_id,
				room_id,
				map_id,
				x,
				y,
				facing
			)
			values ($1, $2, $3, $4, $5, $6)
			on conflict (app_user_id) do update
			set room_id = excluded.room_id,
				map_id = excluded.map_id,
				x = excluded.x,
				y = excluded.y,
				facing = excluded.facing,
				updated_at = now()
			returning app_user_id, room_id, map_id, x, y, facing, updated_at
		`,
		request.AppUserID,
		request.RoomID,
		request.MapID,
		request.X,
		request.Y,
		request.Facing,
	).Scan(&position.AppUserID, &position.RoomID, &position.MapID, &position.X, &position.Y, &position.Facing, &position.UpdatedAt)
	if err != nil {
		return PositionResponse{}, err
	}
	position.Found = true
	return position, nil
}

func (store Store) EnsureNPCCharacters(ctx context.Context, request EnsureNPCCharactersRequest) (NPCCharactersResponse, error) {
	request.RoomID = strings.TrimSpace(request.RoomID)
	if request.RoomID == "" {
		return NPCCharactersResponse{}, errors.New("room_id is required")
	}
	specs := make([]hqcharacters.NPCSpec, 0, len(request.NPCs))
	seen := map[string]bool{}
	for _, npc := range request.NPCs {
		npc.NPCKey = strings.TrimSpace(npc.NPCKey)
		npc.DisplayName = strings.TrimSpace(npc.DisplayName)
		npc.AvatarID = strings.TrimSpace(npc.AvatarID)
		if npc.NPCKey == "" || npc.DisplayName == "" {
			return NPCCharactersResponse{}, errors.New("npc character is missing required fields")
		}
		if seen[npc.NPCKey] {
			continue
		}
		seen[npc.NPCKey] = true
		specs = append(specs, hqcharacters.NPCSpec{
			RoomID:      request.RoomID,
			NPCKey:      npc.NPCKey,
			DisplayName: npc.DisplayName,
			AvatarID:    npc.AvatarID,
		})
	}

	characters, err := hqcharacters.EnsureNPCs(ctx, store.DB, specs)
	if err != nil {
		return NPCCharactersResponse{}, err
	}

	response := NPCCharactersResponse{NPCs: make([]NPCCharacterResponse, 0, len(characters))}
	for _, character := range characters {
		response.NPCs = append(response.NPCs, NPCCharacterResponse{
			CharacterID: character.ID,
			RoomID:      character.RoomID,
			NPCKey:      character.NPCKey,
			DisplayName: character.DisplayName,
			AvatarID:    character.AvatarID,
		})
	}
	return response, nil
}

func IsFacing(value string) bool {
	return value == "up" || value == "down" || value == "left" || value == "right"
}
