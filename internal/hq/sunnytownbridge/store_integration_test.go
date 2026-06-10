package sunnytownbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	hqinventory "hq/internal/hq/inventory"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	testSunnyTownRoomID = "sunny-town-main"
	testSunnyTownMapID  = "sunny-town-v1"
)

func TestCommitSunnyTownRewardIdempotent(t *testing.T) {
	db, cleanup := testBridgeDB(t)
	defer cleanup()

	ctx := context.Background()
	store := Store{DB: db}
	request := RewardEventRequest{
		EventID:       "sunny-town-main:star-0001:1:123",
		AppUserID:     123,
		RoomID:        "sunny-town-main",
		MapID:         "sunny-town-v1",
		CollectibleID: "star-0001",
		RewardKind:    "star",
		Amount:        1,
	}

	first, err := store.CommitReward(ctx, request)
	if err != nil {
		t.Fatalf("first commit error = %v", err)
	}
	if !first.Accepted || first.Duplicate || first.NewStarBalance != 1 {
		t.Fatalf("first commit = %#v, want accepted non-duplicate balance 1", first)
	}

	second, err := store.CommitReward(ctx, request)
	if err != nil {
		t.Fatalf("second commit error = %v", err)
	}
	if !second.Accepted || !second.Duplicate || second.NewStarBalance != 1 {
		t.Fatalf("second commit = %#v, want accepted duplicate balance 1", second)
	}

	var ledgerRows int
	var balance int
	if err := db.QueryRow(ctx, "select count(*) from student_star_ledger").Scan(&ledgerRows); err != nil {
		t.Fatalf("count ledger rows: %v", err)
	}
	if err := db.QueryRow(ctx, "select star_balance from student_wallet where app_user_id = 123").Scan(&balance); err != nil {
		t.Fatalf("load wallet balance: %v", err)
	}
	if ledgerRows != 1 || balance != 1 {
		t.Fatalf("ledgerRows=%d balance=%d, want 1 and 1", ledgerRows, balance)
	}
}

func TestCommitSunnyTownResourceIdempotent(t *testing.T) {
	db, cleanup := testBridgeDB(t)
	defer cleanup()

	ctx := context.Background()
	store := Store{DB: db}
	request := ResourceEventRequest{
		EventID:     "forest-crossing-v1:rock-node-001:1:123",
		AppUserID:   123,
		Source:      "sunny_town_mining",
		RoomID:      "sunny-town-main",
		MapID:       "forest-crossing-v1",
		NodeID:      "rock-node-001",
		ResourceKey: "rock",
		Amount:      2,
	}

	first, err := store.CommitResource(ctx, request)
	if err != nil {
		t.Fatalf("first resource commit error = %v", err)
	}
	if !first.Accepted || first.Duplicate || first.Quantity != 2 || first.ResourceKey != "rock" {
		t.Fatalf("first resource commit = %#v, want accepted non-duplicate quantity 2", first)
	}

	second, err := store.CommitResource(ctx, request)
	if err != nil {
		t.Fatalf("second resource commit error = %v", err)
	}
	if !second.Accepted || !second.Duplicate || second.Quantity != 2 {
		t.Fatalf("second resource commit = %#v, want accepted duplicate quantity 2", second)
	}

	var ledgerRows int
	var rockQuantity int
	if err := db.QueryRow(ctx, "select count(*) from student_inventory_ledger").Scan(&ledgerRows); err != nil {
		t.Fatalf("count inventory ledger rows: %v", err)
	}
	if err := db.QueryRow(
		ctx,
		`
			select sii.quantity
			from student_inventory_item sii
			join inventory_item_type iit on iit.id = sii.item_type_id
			where sii.app_user_id = 123 and iit.key = 'rock'
		`,
	).Scan(&rockQuantity); err != nil {
		t.Fatalf("load rock quantity: %v", err)
	}
	if ledgerRows != 1 || rockQuantity != 2 {
		t.Fatalf("ledgerRows=%d rockQuantity=%d, want 1 and 2", ledgerRows, rockQuantity)
	}
}

func TestCommitNPCJobProductionIdempotent(t *testing.T) {
	db, cleanup := testBridgeDB(t)
	defer cleanup()

	ctx := context.Background()
	store := Store{DB: db}
	ensured, err := store.EnsureNPCCharacters(ctx, EnsureNPCCharactersRequest{
		RoomID: "sunny-town-main",
		NPCs: []EnsureNPCCharacterInput{{
			NPCKey:      "cookie-keeper",
			DisplayName: "Cookie Keeper",
			AvatarID:    "keeper",
		}},
	})
	if err != nil {
		t.Fatalf("ensure npc error = %v", err)
	}
	characterID := ensured.NPCs[0].CharacterID
	if accepted, _, err := hqinventory.IncrementShopInputStorageItem(ctx, db, hqinventory.CookieKeeperShopID, hqinventory.FlourKey, 1); err != nil || !accepted {
		t.Fatalf("seed flour input accepted=%v err=%v, want accepted", accepted, err)
	}
	if accepted, _, err := hqinventory.IncrementShopInputStorageItem(ctx, db, hqinventory.CookieKeeperShopID, hqinventory.SugarKey, 1); err != nil || !accepted {
		t.Fatalf("seed sugar input accepted=%v err=%v, want accepted", accepted, err)
	}
	request := NPCJobProductionRequest{
		EventID:     "sunny-town-main:cookie-keeper:shopkeeper_stock:cookie-keeper-counter:1",
		CharacterID: characterID,
		RoomID:      "sunny-town-main",
		MapID:       "sunny-town-house-1",
		NPCKey:      "cookie-keeper",
		JobKey:      "shopkeeper_stock",
		LocationID:  "cookie-keeper-counter",
		OutputKey:   "shop_stock_progress",
		Amount:      1,
	}

	first, err := store.CommitNPCJobProduction(ctx, request)
	if err != nil {
		t.Fatalf("first production commit error = %v", err)
	}
	if !first.Accepted || first.Duplicate {
		t.Fatalf("first production commit = %#v, want accepted non-duplicate", first)
	}

	second, err := store.CommitNPCJobProduction(ctx, request)
	if err != nil {
		t.Fatalf("second production commit error = %v", err)
	}
	if !second.Accepted || !second.Duplicate {
		t.Fatalf("second production commit = %#v, want accepted duplicate", second)
	}

	var rows int
	var blockedRows int
	var stockQuantity int
	var stockLedgerRows int
	if err := db.QueryRow(ctx, "select count(*) from sunny_town_npc_job_production_ledger where character_id = $1", characterID).Scan(&rows); err != nil {
		t.Fatalf("count production ledger rows: %v", err)
	}
	if err := db.QueryRow(ctx, "select count(*) from sunny_town_npc_job_production_blocked_ledger where character_id = $1", characterID).Scan(&blockedRows); err != nil {
		t.Fatalf("count blocked production ledger rows: %v", err)
	}
	if err := db.QueryRow(
		ctx,
		`
			select ssi.quantity
			from shop_stock_item ssi
			join inventory_item_type iit on iit.id = ssi.item_type_id
			where ssi.shop_id = $1 and iit.key = $2
		`,
		hqinventory.CookieKeeperShopID,
		hqinventory.CookieKey,
	).Scan(&stockQuantity); err != nil {
		t.Fatalf("load shop stock: %v", err)
	}
	if err := db.QueryRow(ctx, "select count(*) from shop_stock_ledger").Scan(&stockLedgerRows); err != nil {
		t.Fatalf("count shop stock ledger rows: %v", err)
	}
	if rows != 1 || blockedRows != 0 || stockQuantity != 1 || stockLedgerRows != 1 {
		t.Fatalf("productionRows=%d blockedRows=%d stockQuantity=%d stockLedgerRows=%d, want 1, 0, 1, 1", rows, blockedRows, stockQuantity, stockLedgerRows)
	}
}

func TestCommitNPCJobProductionBlocksWhenCookieInputsAreMissing(t *testing.T) {
	db, cleanup := testBridgeDB(t)
	defer cleanup()

	ctx := context.Background()
	store := Store{DB: db}
	ensured, err := store.EnsureNPCCharacters(ctx, EnsureNPCCharactersRequest{
		RoomID: "sunny-town-main",
		NPCs: []EnsureNPCCharacterInput{{
			NPCKey:      "cookie-keeper",
			DisplayName: "Cookie Keeper",
			AvatarID:    "keeper",
		}},
	})
	if err != nil {
		t.Fatalf("ensure npc error = %v", err)
	}
	if accepted, _, err := hqinventory.IncrementShopInputStorageItem(ctx, db, hqinventory.CookieKeeperShopID, hqinventory.FlourKey, 1); err != nil || !accepted {
		t.Fatalf("seed flour input accepted=%v err=%v, want accepted", accepted, err)
	}
	request := NPCJobProductionRequest{
		EventID:     "sunny-town-main:cookie-keeper:shopkeeper_stock:cookie-keeper-counter:missing-inputs",
		CharacterID: ensured.NPCs[0].CharacterID,
		RoomID:      "sunny-town-main",
		MapID:       "sunny-town-house-1",
		NPCKey:      "cookie-keeper",
		JobKey:      "shopkeeper_stock",
		LocationID:  "cookie-keeper-counter",
		OutputKey:   "shop_stock_progress",
		Amount:      1,
	}

	first, err := store.CommitNPCJobProduction(ctx, request)
	if err != nil {
		t.Fatalf("first production commit error = %v", err)
	}
	if !first.Accepted || first.Duplicate || !first.Blocked || first.BlockedReason != npcJobProductionBlockedMissingInputs {
		t.Fatalf("first production commit = %#v, want accepted blocked missing_inputs", first)
	}

	second, err := store.CommitNPCJobProduction(ctx, request)
	if err != nil {
		t.Fatalf("second production commit error = %v", err)
	}
	if !second.Accepted || !second.Duplicate || !second.Blocked || second.BlockedReason != npcJobProductionBlockedMissingInputs {
		t.Fatalf("second production commit = %#v, want duplicate blocked missing_inputs", second)
	}

	var productionRows int
	var blockedRows int
	var stockRows int
	if err := db.QueryRow(ctx, "select count(*) from sunny_town_npc_job_production_ledger").Scan(&productionRows); err != nil {
		t.Fatalf("count production rows: %v", err)
	}
	if err := db.QueryRow(ctx, "select count(*) from sunny_town_npc_job_production_blocked_ledger").Scan(&blockedRows); err != nil {
		t.Fatalf("count blocked rows: %v", err)
	}
	if err := db.QueryRow(ctx, "select count(*) from shop_stock_item").Scan(&stockRows); err != nil {
		t.Fatalf("count stock rows: %v", err)
	}
	if productionRows != 0 || blockedRows != 1 || stockRows != 0 {
		t.Fatalf("productionRows=%d blockedRows=%d stockRows=%d, want 0, 1, 0", productionRows, blockedRows, stockRows)
	}

	inputStorage, err := hqinventory.LoadShopInputStorage(ctx, db, hqinventory.CookieKeeperShopID)
	if err != nil {
		t.Fatalf("load input storage: %v", err)
	}
	if len(inputStorage.Items) != 1 || inputStorage.Items[0].ItemKey != hqinventory.FlourKey || inputStorage.Items[0].Quantity != 1 {
		t.Fatalf("input storage = %#v, want flour unchanged after blocked production", inputStorage)
	}
}

func TestCommitNPCJobProductionBlocksWhenCookieOutputIsFull(t *testing.T) {
	db, cleanup := testBridgeDB(t)
	defer cleanup()

	ctx := context.Background()
	store := Store{DB: db}
	ensured, err := store.EnsureNPCCharacters(ctx, EnsureNPCCharactersRequest{
		RoomID: "sunny-town-main",
		NPCs: []EnsureNPCCharacterInput{{
			NPCKey:      "cookie-keeper",
			DisplayName: "Cookie Keeper",
			AvatarID:    "keeper",
		}},
	})
	if err != nil {
		t.Fatalf("ensure npc error = %v", err)
	}
	if _, _, err := hqinventory.CommitShopStockDelta(ctx, db, hqinventory.ShopStockEventRequest{
		EventID: "seed-full-output",
		Source:  "test",
		ShopID:  hqinventory.CookieKeeperShopID,
		ItemKey: hqinventory.CookieKey,
		Delta:   hqinventory.CookieKeeperCookieStockCapacity,
	}); err != nil {
		t.Fatalf("seed full output stock: %v", err)
	}
	if accepted, _, err := hqinventory.IncrementShopInputStorageItem(ctx, db, hqinventory.CookieKeeperShopID, hqinventory.FlourKey, 1); err != nil || !accepted {
		t.Fatalf("seed flour input accepted=%v err=%v, want accepted", accepted, err)
	}
	if accepted, _, err := hqinventory.IncrementShopInputStorageItem(ctx, db, hqinventory.CookieKeeperShopID, hqinventory.SugarKey, 1); err != nil || !accepted {
		t.Fatalf("seed sugar input accepted=%v err=%v, want accepted", accepted, err)
	}

	response, err := store.CommitNPCJobProduction(ctx, NPCJobProductionRequest{
		EventID:     "sunny-town-main:cookie-keeper:shopkeeper_stock:cookie-keeper-counter:output-full",
		CharacterID: ensured.NPCs[0].CharacterID,
		RoomID:      "sunny-town-main",
		MapID:       "sunny-town-house-1",
		NPCKey:      "cookie-keeper",
		JobKey:      "shopkeeper_stock",
		LocationID:  "cookie-keeper-counter",
		OutputKey:   "shop_stock_progress",
		Amount:      1,
	})
	if err != nil {
		t.Fatalf("production commit error = %v", err)
	}
	if !response.Accepted || response.Duplicate || !response.Blocked || response.BlockedReason != npcJobProductionBlockedOutputFull {
		t.Fatalf("production commit = %#v, want accepted blocked output_full", response)
	}

	inputStorage, err := hqinventory.LoadShopInputStorage(ctx, db, hqinventory.CookieKeeperShopID)
	if err != nil {
		t.Fatalf("load input storage: %v", err)
	}
	quantities := map[string]int{}
	for _, item := range inputStorage.Items {
		quantities[item.ItemKey] = item.Quantity
	}
	if quantities[hqinventory.FlourKey] != 1 || quantities[hqinventory.SugarKey] != 1 {
		t.Fatalf("input storage = %#v, want flour and sugar unchanged", inputStorage)
	}
}

func TestLoadNPCJobProductionProgressAggregatesLedger(t *testing.T) {
	db, cleanup := testBridgeDB(t)
	defer cleanup()

	ctx := context.Background()
	store := Store{DB: db}
	ensured, err := store.EnsureNPCCharacters(ctx, EnsureNPCCharactersRequest{
		RoomID: "sunny-town-main",
		NPCs: []EnsureNPCCharacterInput{
			{NPCKey: "cookie-keeper", DisplayName: "Cookie Keeper", AvatarID: "keeper"},
			{NPCKey: "teacher", DisplayName: "Teacher", AvatarID: "teacher"},
		},
	})
	if err != nil {
		t.Fatalf("ensure npc error = %v", err)
	}
	characterIDs := map[string]int64{}
	for _, npc := range ensured.NPCs {
		characterIDs[npc.NPCKey] = npc.CharacterID
	}
	if accepted, _, err := hqinventory.IncrementShopInputStorageItem(ctx, db, hqinventory.CookieKeeperShopID, hqinventory.FlourKey, 3); err != nil || !accepted {
		t.Fatalf("seed flour input accepted=%v err=%v, want accepted", accepted, err)
	}
	if accepted, _, err := hqinventory.IncrementShopInputStorageItem(ctx, db, hqinventory.CookieKeeperShopID, hqinventory.SugarKey, 3); err != nil || !accepted {
		t.Fatalf("seed sugar input accepted=%v err=%v, want accepted", accepted, err)
	}

	requests := []NPCJobProductionRequest{
		{
			EventID:     "sunny-town-main:cookie-keeper:shopkeeper_stock:cookie-keeper-counter:1",
			CharacterID: characterIDs["cookie-keeper"],
			RoomID:      "sunny-town-main",
			MapID:       "sunny-town-house-1",
			NPCKey:      "cookie-keeper",
			JobKey:      "shopkeeper_stock",
			LocationID:  "cookie-keeper-counter",
			OutputKey:   "shop_stock_progress",
			Amount:      1,
		},
		{
			EventID:     "sunny-town-main:cookie-keeper:shopkeeper_stock:cookie-keeper-counter:2",
			CharacterID: characterIDs["cookie-keeper"],
			RoomID:      "sunny-town-main",
			MapID:       "sunny-town-house-1",
			NPCKey:      "cookie-keeper",
			JobKey:      "shopkeeper_stock",
			LocationID:  "cookie-keeper-counter",
			OutputKey:   "shop_stock_progress",
			Amount:      2,
		},
		{
			EventID:     "sunny-town-main:teacher:teacher_lesson_prep:teacher-desk-work:1",
			CharacterID: characterIDs["teacher"],
			RoomID:      "sunny-town-main",
			MapID:       "sunny-town-classroom",
			NPCKey:      "teacher",
			JobKey:      "teacher_lesson_prep",
			LocationID:  "teacher-desk-work",
			OutputKey:   "lesson_prep_progress",
			Amount:      1,
		},
	}
	for _, request := range requests {
		if _, err := store.CommitNPCJobProduction(ctx, request); err != nil {
			t.Fatalf("commit production %#v: %v", request.EventID, err)
		}
	}

	progress, err := store.LoadNPCJobProductionProgress(ctx, NPCJobProductionProgressRequest{RoomID: "sunny-town-main"})
	if err != nil {
		t.Fatalf("load production progress: %v", err)
	}
	if len(progress.Progress) != 2 {
		t.Fatalf("progress entries = %#v, want two grouped entries", progress.Progress)
	}
	shop := progress.Progress[0]
	if shop.NPCKey != "cookie-keeper" || shop.JobKey != "shopkeeper_stock" || shop.TotalAmount != 3 || shop.EventCount != 2 {
		t.Fatalf("shop progress = %#v, want cookie keeper total 3 from two events", shop)
	}
	lesson := progress.Progress[1]
	if lesson.NPCKey != "teacher" || lesson.JobKey != "teacher_lesson_prep" || lesson.TotalAmount != 1 || lesson.EventCount != 1 {
		t.Fatalf("lesson progress = %#v, want teacher total 1 from one event", lesson)
	}

	filtered, err := store.LoadNPCJobProductionProgress(ctx, NPCJobProductionProgressRequest{
		RoomID: "sunny-town-main",
		JobKey: "shopkeeper_stock",
	})
	if err != nil {
		t.Fatalf("load filtered production progress: %v", err)
	}
	if len(filtered.Progress) != 1 || filtered.Progress[0].NPCKey != "cookie-keeper" {
		t.Fatalf("filtered progress = %#v, want only cookie keeper shop progress", filtered.Progress)
	}
}

func TestCommitSunnyTownResourceRejectsInvalidRequest(t *testing.T) {
	db, cleanup := testBridgeDB(t)
	defer cleanup()

	store := Store{DB: db}
	requests := []ResourceEventRequest{
		{},
		{EventID: "event", AppUserID: 123, Source: "sunny_town_mining", RoomID: "sunny-town-main", MapID: "forest-crossing-v1", NodeID: "rock-node-001", ResourceKey: "star", Amount: 1},
		{EventID: "event", AppUserID: 123, Source: "sunny_town_mining", RoomID: "forest-crossing-v1", MapID: "forest-crossing-v1", NodeID: "rock-node-001", ResourceKey: "rock", Amount: 0},
		{EventID: "event", AppUserID: 123, Source: "other", RoomID: "sunny-town-main", MapID: "forest-crossing-v1", NodeID: "rock-node-001", ResourceKey: "rock", Amount: 1},
	}
	for _, request := range requests {
		if _, err := store.CommitResource(context.Background(), request); err == nil {
			t.Fatalf("resource request %#v succeeded, want error", request)
		}
	}
}

func TestCommitNPCJobProductionRejectsInvalidRequest(t *testing.T) {
	db, cleanup := testBridgeDB(t)
	defer cleanup()

	store := Store{DB: db}
	requests := []NPCJobProductionRequest{
		{},
		{EventID: "event", CharacterID: 1, RoomID: "sunny-town-main", MapID: "sunny-town-house-1", NPCKey: "cookie-keeper", JobKey: "other", LocationID: "cookie-keeper-counter", OutputKey: "shop_stock_progress", Amount: 1},
		{EventID: "event", CharacterID: 1, RoomID: "sunny-town-main", MapID: "sunny-town-house-1", NPCKey: "cookie-keeper", JobKey: "shopkeeper_stock", LocationID: "cookie-keeper-counter", OutputKey: "other", Amount: 1},
		{EventID: "event", CharacterID: 1, RoomID: "sunny-town-main", MapID: "sunny-town-house-1", NPCKey: "cookie-keeper", JobKey: "shopkeeper_stock", LocationID: "cookie-keeper-counter", OutputKey: "shop_stock_progress", Amount: 0},
	}
	for _, request := range requests {
		if _, err := store.CommitNPCJobProduction(context.Background(), request); err == nil {
			t.Fatalf("production request %#v succeeded, want error", request)
		}
	}
}

func TestSunnyTownResourceEndpointRequiresServiceSecret(t *testing.T) {
	db, cleanup := testBridgeDB(t)
	defer cleanup()

	body, err := json.Marshal(ResourceEventRequest{
		EventID:     "forest-crossing-v1:rock-node-001:1:123",
		AppUserID:   123,
		Source:      "sunny_town_mining",
		RoomID:      "sunny-town-main",
		MapID:       "forest-crossing-v1",
		NodeID:      "rock-node-001",
		ResourceKey: "rock",
		Amount:      1,
	})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/internal/sunny-town/resource-events", bytes.NewReader(body))
	response := httptest.NewRecorder()

	NewHTTPHandler(Store{DB: db}, "test-secret").HandleResourceEvent(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestSunnyTownNPCJobProductionProgressEndpointRequiresServiceSecret(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/internal/sunny-town/npc-job-production/progress?room_id=sunny-town-main", nil)
	response := httptest.NewRecorder()

	NewHTTPHandler(Store{}, "test-secret").HandleNPCJobProductionProgress(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestContainerTransferEndpointUsesServiceAuthenticatedRequest(t *testing.T) {
	db, cleanup := testBridgeDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := hqinventory.IncrementStudentItem(ctx, db, 123, "rock", 4); err != nil {
		t.Fatalf("seed rock inventory: %v", err)
	}
	if _, err := db.Exec(
		ctx,
		`
			insert into storage_container (
				id,
				kind,
				room_id,
				map_id,
				fixture_id,
				slot_count,
				access_policy
			)
			values ('fixture:sunny-town-main:sunny-town-house-1:test-chest', 'fixture', 'sunny-town-main', 'sunny-town-house-1', 'test-chest', 10, 'room_shared')
		`,
	); err != nil {
		t.Fatalf("seed storage container: %v", err)
	}

	body := []byte(`{
		"appUserId": 123,
		"source": {"kind": "player_inventory", "slotIndex": 0},
		"destination": {"kind": "container", "containerId": "fixture:sunny-town-main:sunny-town-house-1:test-chest", "slotIndex": 0},
		"mode": "move"
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/internal/sunny-town/container-transfer", bytes.NewReader(body))
	request.Header.Set("X-HQ-Service-Secret", "test-secret")
	response := httptest.NewRecorder()

	NewHTTPHandler(Store{DB: db}, "test-secret").HandleContainerTransfer(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"containerId":"fixture:sunny-town-main:sunny-town-house-1:test-chest"`) {
		t.Fatalf("body = %s, want container response", response.Body.String())
	}
}

func TestSaveSunnyTownPositionUpsertsLastLocation(t *testing.T) {
	db, cleanup := testBridgeDB(t)
	defer cleanup()

	ctx := context.Background()
	store := Store{DB: db}
	first, err := store.SavePosition(ctx, PositionRequest{
		AppUserID: 123,
		RoomID:    "sunny-town-main",
		MapID:     "sunny-town-v1",
		X:         120.5,
		Y:         140.25,
		Facing:    "right",
	})
	if err != nil {
		t.Fatalf("first save error = %v", err)
	}
	if !first.Found || first.MapID != "sunny-town-v1" || first.X != 120.5 || first.Facing != "right" {
		t.Fatalf("first position = %#v", first)
	}

	second, err := store.SavePosition(ctx, PositionRequest{
		AppUserID: 123,
		RoomID:    "sunny-town-main",
		MapID:     "sunny-town-classroom",
		X:         220,
		Y:         260,
		Facing:    "up",
	})
	if err != nil {
		t.Fatalf("second save error = %v", err)
	}
	loaded, err := store.LoadPosition(ctx, 123)
	if err != nil {
		t.Fatalf("load position error: %v", err)
	}
	if !loaded.Found || loaded.AppUserID != second.AppUserID || loaded.RoomID != second.RoomID || loaded.MapID != second.MapID || loaded.X != second.X || loaded.Y != second.Y || loaded.Facing != second.Facing {
		t.Fatalf("loaded position = %#v, want persisted fields from %#v", loaded, second)
	}
}

func TestEnsureNPCCharactersUpsertsDurableRows(t *testing.T) {
	db, cleanup := testBridgeDB(t)
	defer cleanup()

	ctx := context.Background()
	store := Store{DB: db}
	first, err := store.EnsureNPCCharacters(ctx, EnsureNPCCharactersRequest{
		RoomID: "sunny-town-main",
		NPCs: []EnsureNPCCharacterInput{{
			NPCKey:      "mayor-sunny",
			DisplayName: "Mayor Sunny",
			AvatarID:    "mayor",
		}},
	})
	if err != nil {
		t.Fatalf("first ensure error = %v", err)
	}
	if len(first.NPCs) != 1 || first.NPCs[0].CharacterID == 0 || first.NPCs[0].NPCKey != "mayor-sunny" {
		t.Fatalf("first ensure = %#v, want mayor-sunny with character id", first)
	}

	second, err := store.EnsureNPCCharacters(ctx, EnsureNPCCharactersRequest{
		RoomID: "sunny-town-main",
		NPCs: []EnsureNPCCharacterInput{{
			NPCKey:      "mayor-sunny",
			DisplayName: "Mayor Sunny",
			AvatarID:    "mayor",
		}},
	})
	if err != nil {
		t.Fatalf("second ensure error = %v", err)
	}
	if len(second.NPCs) != 1 || second.NPCs[0].CharacterID != first.NPCs[0].CharacterID {
		t.Fatalf("second ensure = %#v, want same character id %d", second, first.NPCs[0].CharacterID)
	}

	var characterRows int
	var mappingRows int
	if err := db.QueryRow(ctx, "select count(*) from sunny_town_character where character_type = 'npc'").Scan(&characterRows); err != nil {
		t.Fatalf("count character rows: %v", err)
	}
	if err := db.QueryRow(ctx, "select count(*) from sunny_town_npc_character where room_id = 'sunny-town-main' and npc_key = 'mayor-sunny'").Scan(&mappingRows); err != nil {
		t.Fatalf("count mapping rows: %v", err)
	}
	if characterRows != 1 || mappingRows != 1 {
		t.Fatalf("characterRows=%d mappingRows=%d, want 1 and 1", characterRows, mappingRows)
	}
}

func TestEnsureNPCCharactersConcurrentCallsShareDurableRow(t *testing.T) {
	db, cleanup := testBridgeDB(t)
	defer cleanup()

	store := Store{DB: db}
	const workers = 12
	start := make(chan struct{})
	errs := make(chan error, workers)
	ids := make(chan int64, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			response, err := store.EnsureNPCCharacters(context.Background(), EnsureNPCCharactersRequest{
				RoomID: "sunny-town-main",
				NPCs: []EnsureNPCCharacterInput{{
					NPCKey:      "mayor-sunny",
					DisplayName: "Mayor Sunny",
					AvatarID:    "mayor",
				}},
			})
			if err != nil {
				errs <- err
				return
			}
			if len(response.NPCs) != 1 {
				errs <- fmt.Errorf("response npcs = %#v, want one npc", response.NPCs)
				return
			}
			ids <- response.NPCs[0].CharacterID
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	close(ids)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent ensure error: %v", err)
		}
	}
	var firstID int64
	for id := range ids {
		if id == 0 {
			t.Fatal("concurrent ensure returned zero character id")
		}
		if firstID == 0 {
			firstID = id
			continue
		}
		if id != firstID {
			t.Fatalf("concurrent ensure returned character id %d, want %d", id, firstID)
		}
	}

	var characterRows int
	var mappingRows int
	if err := db.QueryRow(context.Background(), "select count(*) from sunny_town_character where character_type = 'npc'").Scan(&characterRows); err != nil {
		t.Fatalf("count character rows: %v", err)
	}
	if err := db.QueryRow(context.Background(), "select count(*) from sunny_town_npc_character where room_id = 'sunny-town-main' and npc_key = 'mayor-sunny'").Scan(&mappingRows); err != nil {
		t.Fatalf("count mapping rows: %v", err)
	}
	if characterRows != 1 || mappingRows != 1 {
		t.Fatalf("characterRows=%d mappingRows=%d, want 1 and 1", characterRows, mappingRows)
	}
}

func TestPlaceSunnyTownMapObjectConsumesStoneBlock(t *testing.T) {
	db, cleanup := testBridgeDB(t)
	defer cleanup()

	ctx := context.Background()
	store := Store{DB: db}
	if err := hqinventory.IncrementStudentItem(ctx, db, 123, StoneBlockItemKey, 2); err != nil {
		t.Fatalf("seed stone block inventory: %v", err)
	}

	placed, err := store.PlaceMapObject(ctx, PlaceMapObjectRequest{
		AppUserID: 123,
		RoomID:    testSunnyTownRoomID,
		MapID:     testSunnyTownMapID,
		GridX:     4,
		GridY:     5,
		ItemKey:   StoneBlockItemKey,
	})
	if err != nil {
		t.Fatalf("place map object: %v", err)
	}
	if placed.ID == 0 || placed.ItemKey != StoneBlockItemKey || placed.GridX != 4 || placed.GridY != 5 || placed.RemainingItemAmount != 1 {
		t.Fatalf("placed = %#v, want stone block at 4,5 with one remaining", placed)
	}
	quantity, err := LoadStudentInventoryQuantity(ctx, db, 123, StoneBlockItemKey)
	if err != nil {
		t.Fatalf("load quantity: %v", err)
	}
	if quantity != 1 {
		t.Fatalf("quantity = %d, want 1", quantity)
	}
}

func TestPlaceSunnyTownMapObjectRollsBackInventoryWhenOccupied(t *testing.T) {
	db, cleanup := testBridgeDB(t)
	defer cleanup()

	ctx := context.Background()
	store := Store{DB: db}
	if err := hqinventory.IncrementStudentItem(ctx, db, 123, StoneBlockItemKey, 2); err != nil {
		t.Fatalf("seed stone block inventory: %v", err)
	}
	request := PlaceMapObjectRequest{
		AppUserID: 123,
		RoomID:    testSunnyTownRoomID,
		MapID:     testSunnyTownMapID,
		GridX:     4,
		GridY:     5,
		ItemKey:   StoneBlockItemKey,
	}
	if _, err := store.PlaceMapObject(ctx, request); err != nil {
		t.Fatalf("place first object: %v", err)
	}
	if _, err := store.PlaceMapObject(ctx, request); !errors.Is(err, ErrMapObjectOccupied) {
		t.Fatalf("second place error = %v, want ErrMapObjectOccupied", err)
	}
	quantity, err := LoadStudentInventoryQuantity(ctx, db, 123, StoneBlockItemKey)
	if err != nil {
		t.Fatalf("load quantity: %v", err)
	}
	if quantity != 1 {
		t.Fatalf("quantity = %d, want rollback to keep 1", quantity)
	}
}

func TestRemoveSunnyTownMapObjectRefundsStoneBlock(t *testing.T) {
	db, cleanup := testBridgeDB(t)
	defer cleanup()

	ctx := context.Background()
	store := Store{DB: db}
	if err := hqinventory.IncrementStudentItem(ctx, db, 123, StoneBlockItemKey, 1); err != nil {
		t.Fatalf("seed stone block inventory: %v", err)
	}
	if _, err := store.PlaceMapObject(ctx, PlaceMapObjectRequest{
		AppUserID: 123,
		RoomID:    testSunnyTownRoomID,
		MapID:     testSunnyTownMapID,
		GridX:     7,
		GridY:     8,
		ItemKey:   StoneBlockItemKey,
	}); err != nil {
		t.Fatalf("place object: %v", err)
	}
	removed, err := store.RemoveMapObject(ctx, RemoveMapObjectRequest{
		AppUserID: 123,
		RoomID:    testSunnyTownRoomID,
		MapID:     testSunnyTownMapID,
		GridX:     7,
		GridY:     8,
	})
	if err != nil {
		t.Fatalf("remove object: %v", err)
	}
	if removed.ItemKey != StoneBlockItemKey || removed.RemainingItemAmount != 1 {
		t.Fatalf("removed = %#v, want refunded stone block with one remaining", removed)
	}
}

func testBridgeDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	databaseURL := os.Getenv("HQ_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("HQ_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect admin pool: %v", err)
	}

	schema := fmt.Sprintf("sunnytownbridge_test_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, "create schema "+schema); err != nil {
		adminPool.Close()
		t.Fatalf("create schema: %v", err)
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		adminPool.Close()
		t.Fatalf("parse pool config: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		adminPool.Close()
		t.Fatalf("connect test pool: %v", err)
	}

	statements := []string{
		`create table app_user (
			id bigint primary key,
			display_name text not null
		)`,
		`create table sunny_town_character (
			id bigserial primary key,
			character_type text not null,
			app_user_id bigint null unique references app_user(id) on delete cascade,
			room_id text not null default 'sunny-town-main',
			display_name text not null,
			avatar_id text not null default 'pet-default',
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			constraint sunny_town_character_type_check check (character_type in ('player', 'npc')),
			constraint sunny_town_character_owner_check check (
				(character_type = 'player' and app_user_id is not null) or
				(character_type = 'npc' and app_user_id is null)
			)
		)`,
		`create table sunny_town_npc_character (
			character_id bigint primary key references sunny_town_character(id) on delete cascade,
			room_id text not null,
			npc_key text not null,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			constraint sunny_town_npc_character_room_key unique (room_id, npc_key)
		)`,
		`create table sunny_town_npc_job_production_ledger (
			id bigserial primary key,
			event_id text not null unique,
			character_id bigint not null references sunny_town_character(id) on delete cascade,
			room_id text not null,
			map_id text not null,
			npc_key text not null,
			job_key text not null,
			location_id text not null,
			output_key text not null,
			amount integer not null,
			metadata jsonb not null default '{}'::jsonb,
			created_at timestamptz not null default now()
		)`,
		`create table sunny_town_npc_job_production_blocked_ledger (
			id bigserial primary key,
			event_id text not null unique,
			character_id bigint not null references sunny_town_character(id) on delete cascade,
			room_id text not null,
			map_id text not null,
			npc_key text not null,
			job_key text not null,
			location_id text not null,
			output_key text not null,
			amount integer not null,
			reason text not null,
			metadata jsonb not null default '{}'::jsonb,
			created_at timestamptz not null default now()
		)`,
		`create table student_wallet (
			app_user_id bigint primary key references app_user(id) on delete cascade,
			star_balance integer not null default 0,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			constraint student_wallet_star_balance_nonnegative check (star_balance >= 0)
		)`,
		`create table student_star_ledger (
			id bigserial primary key,
			app_user_id bigint not null references app_user(id) on delete cascade,
			event_id text not null unique,
			source text not null,
			delta integer not null,
			room_id text null,
			map_id text null,
			collectible_id text null,
			metadata jsonb not null default '{}'::jsonb,
			created_at timestamptz not null default now(),
			constraint student_star_ledger_delta_nonzero check (delta <> 0)
		)`,
		`create table inventory_item_type (
			id bigserial primary key,
			key text not null unique,
			name text not null,
			description text not null default '',
			equip_slot text null,
			visual_key text null,
			icon_key text null,
			max_stack integer null,
			category text null,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now()
		)`,
		`insert into inventory_item_type (key, name, description)
			values
				('cookie', 'Cookie', 'A treat for your pet.'),
				('flour', 'Flour', 'A basic baking ingredient.'),
				('sugar', 'Sugar', 'A sweet baking ingredient.'),
				('rock', 'Rock', 'A sturdy rock from Forest Crossing.'),
				('crystal', 'Crystal', 'A bright crystal from Forest Crossing.'),
				('stone_block', 'Stone Block', 'A solid block crafted from stone.')`,
		`update inventory_item_type
			set icon_key = key,
				max_stack = case when key = 'pickaxe' then 1 else 64 end,
				category = case
					when key = 'cookie' then 'consumable'
					when key in ('rock', 'crystal') then 'resource'
					when key = 'stone_block' then 'building'
					else category
				end`,
		`create table student_inventory_item (
			app_user_id bigint not null references app_user(id) on delete cascade,
			item_type_id bigint not null references inventory_item_type(id) on delete restrict,
			quantity integer not null default 0,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			primary key (app_user_id, item_type_id),
			constraint student_inventory_item_quantity_nonnegative check (quantity >= 0)
		)`,
		`create table student_inventory_slot (
			app_user_id bigint not null references app_user(id) on delete cascade,
			slot_index integer not null,
			item_type_id bigint null references inventory_item_type(id) on delete restrict,
			quantity integer null,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			primary key (app_user_id, slot_index),
			constraint student_inventory_slot_index_check check (slot_index >= 0 and slot_index < 30),
			constraint student_inventory_slot_quantity_check check (quantity is null or quantity > 0),
			constraint student_inventory_slot_empty_or_occupied_check check (
				(item_type_id is null and quantity is null)
				or
				(item_type_id is not null and quantity is not null)
			)
		)`,
		`create table student_inventory_ledger (
			id bigserial primary key,
			app_user_id bigint not null references app_user(id) on delete cascade,
			event_id text not null unique,
			source text not null,
			item_type_id bigint not null references inventory_item_type(id) on delete restrict,
			delta integer not null,
			room_id text null,
			map_id text null,
			node_id text null,
			metadata jsonb not null default '{}'::jsonb,
			created_at timestamptz not null default now(),
			constraint student_inventory_ledger_delta_nonzero check (delta <> 0)
		)`,
		`create table shop_stock_item (
			shop_id text not null,
			item_type_id bigint not null references inventory_item_type(id) on delete restrict,
			quantity integer not null default 0,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			primary key (shop_id, item_type_id),
			constraint shop_stock_item_quantity_nonnegative check (quantity >= 0)
		)`,
		`create table shop_stock_ledger (
			id bigserial primary key,
			event_id text not null unique,
			source text not null,
			shop_id text not null,
			item_type_id bigint not null references inventory_item_type(id) on delete restrict,
			delta integer not null,
			metadata jsonb not null default '{}'::jsonb,
			created_at timestamptz not null default now(),
			constraint shop_stock_ledger_delta_nonzero check (delta <> 0)
		)`,
		`create table shop_input_storage_item (
			shop_id text not null,
			item_type_id bigint not null references inventory_item_type(id) on delete restrict,
			quantity integer not null default 0,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			primary key (shop_id, item_type_id),
			constraint shop_input_storage_item_quantity_nonnegative check (quantity >= 0)
		)`,
		`create table storage_container (
			id text primary key,
			kind text not null,
			room_id text not null,
			map_id text not null,
			fixture_id text null,
			placed_object_id text null,
			shop_id text null,
			storage_role text null,
			location_id text null,
			owner_app_user_id bigint null references app_user(id) on delete cascade,
			slot_count integer not null,
			access_policy text not null,
			revision bigint not null default 0,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now()
		)`,
		`create table storage_container_slot (
			container_id text not null references storage_container(id) on delete cascade,
			slot_index integer not null,
			item_type_id bigint null references inventory_item_type(id) on delete restrict,
			quantity integer null,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			primary key (container_id, slot_index)
		)`,
		`create table student_sunny_town_position (
			app_user_id bigint primary key references app_user(id) on delete cascade,
			room_id text not null,
			map_id text not null,
			x double precision not null,
			y double precision not null,
			facing text not null,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			constraint student_sunny_town_position_facing_check check (facing in ('up', 'down', 'left', 'right'))
		)`,
		`create table sunny_town_map_object (
			id bigserial primary key,
			room_id text not null,
			map_id text not null,
			grid_x integer not null,
			grid_y integer not null,
			item_key text not null references inventory_item_type(key) on delete restrict,
			placed_by_app_user_id bigint not null references app_user(id) on delete cascade,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			constraint sunny_town_map_object_location_key unique (room_id, map_id, grid_x, grid_y)
		)`,
		`insert into app_user (id, display_name) values (123, 'Student')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(ctx, statement); err != nil {
			db.Close()
			_, _ = adminPool.Exec(ctx, "drop schema "+schema+" cascade")
			adminPool.Close()
			t.Fatalf("setup statement failed: %v", err)
		}
	}

	return db, func() {
		db.Close()
		_, _ = adminPool.Exec(ctx, "drop schema "+schema+" cascade")
		adminPool.Close()
	}
}
