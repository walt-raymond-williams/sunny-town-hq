package main

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
	"testing"
	"time"

	"hq/internal/aiapi"
	hqinventory "hq/internal/hq/inventory"
	hqpet "hq/internal/hq/pet"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	testSunnyTownRoomID = "sunny-town-main"
	testSunnyTownMapID  = "sunny-town-v1"
)

func TestCommitSunnyTownRewardIdempotent(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	request := sunnyTownRewardEventRequest{
		EventID:       "sunny-town-main:star-0001:1:123",
		AppUserID:     123,
		RoomID:        "sunny-town-main",
		MapID:         "sunny-town-v1",
		CollectibleID: "star-0001",
		RewardKind:    "star",
		Amount:        1,
	}

	first, err := app.commitSunnyTownReward(ctx, request)
	if err != nil {
		t.Fatalf("first commit error = %v", err)
	}
	if !first.Accepted || first.Duplicate || first.NewStarBalance != 1 {
		t.Fatalf("first commit = %#v, want accepted non-duplicate balance 1", first)
	}

	second, err := app.commitSunnyTownReward(ctx, request)
	if err != nil {
		t.Fatalf("second commit error = %v", err)
	}
	if !second.Accepted || !second.Duplicate || second.NewStarBalance != 1 {
		t.Fatalf("second commit = %#v, want accepted duplicate balance 1", second)
	}

	var ledgerRows int
	var balance int
	if err := app.db.QueryRow(ctx, "select count(*) from student_star_ledger").Scan(&ledgerRows); err != nil {
		t.Fatalf("count ledger rows: %v", err)
	}
	if err := app.db.QueryRow(ctx, "select star_balance from student_wallet where app_user_id = 123").Scan(&balance); err != nil {
		t.Fatalf("load wallet balance: %v", err)
	}
	if ledgerRows != 1 || balance != 1 {
		t.Fatalf("ledgerRows=%d balance=%d, want 1 and 1", ledgerRows, balance)
	}
}

func TestCommitSunnyTownResourceIdempotent(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	request := sunnyTownResourceEventRequest{
		EventID:     "forest-crossing-v1:rock-node-001:1:123",
		AppUserID:   123,
		Source:      "sunny_town_mining",
		RoomID:      "sunny-town-main",
		MapID:       "forest-crossing-v1",
		NodeID:      "rock-node-001",
		ResourceKey: "rock",
		Amount:      2,
	}

	first, err := app.commitSunnyTownResource(ctx, request)
	if err != nil {
		t.Fatalf("first resource commit error = %v", err)
	}
	if !first.Accepted || first.Duplicate || first.Quantity != 2 || first.ResourceKey != "rock" {
		t.Fatalf("first resource commit = %#v, want accepted non-duplicate quantity 2", first)
	}

	second, err := app.commitSunnyTownResource(ctx, request)
	if err != nil {
		t.Fatalf("second resource commit error = %v", err)
	}
	if !second.Accepted || !second.Duplicate || second.Quantity != 2 {
		t.Fatalf("second resource commit = %#v, want accepted duplicate quantity 2", second)
	}

	var ledgerRows int
	var rockQuantity int
	if err := app.db.QueryRow(ctx, "select count(*) from student_inventory_ledger").Scan(&ledgerRows); err != nil {
		t.Fatalf("count inventory ledger rows: %v", err)
	}
	if err := app.db.QueryRow(
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

func TestCommitSunnyTownResourceRejectsInvalidRequest(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	requests := []sunnyTownResourceEventRequest{
		{},
		{EventID: "event", AppUserID: 123, Source: "sunny_town_mining", RoomID: "sunny-town-main", MapID: "forest-crossing-v1", NodeID: "rock-node-001", ResourceKey: "star", Amount: 1},
		{EventID: "event", AppUserID: 123, Source: "sunny_town_mining", RoomID: "sunny-town-main", MapID: "forest-crossing-v1", NodeID: "rock-node-001", ResourceKey: "rock", Amount: 0},
		{EventID: "event", AppUserID: 123, Source: "other", RoomID: "sunny-town-main", MapID: "forest-crossing-v1", NodeID: "rock-node-001", ResourceKey: "rock", Amount: 1},
	}
	for _, request := range requests {
		if _, err := app.commitSunnyTownResource(context.Background(), request); err == nil {
			t.Fatalf("resource request %#v succeeded, want error", request)
		}
	}
}

func TestSunnyTownResourceEndpointRequiresServiceSecret(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()
	app.sunnyTownServiceSecret = "test-secret"

	body, err := json.Marshal(sunnyTownResourceEventRequest{
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

	app.handleSunnyTownResourceEvent(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestSaveSunnyTownPositionUpsertsLastLocation(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	first, err := app.saveSunnyTownPosition(ctx, sunnyTownPositionRequest{
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

	second, err := app.saveSunnyTownPosition(ctx, sunnyTownPositionRequest{
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
	loaded, err := app.loadSunnyTownPosition(ctx, 123)
	if err != nil {
		t.Fatalf("load position error = %v", err)
	}
	if !loaded.Found || loaded.AppUserID != second.AppUserID || loaded.RoomID != second.RoomID || loaded.MapID != second.MapID || loaded.X != second.X || loaded.Y != second.Y || loaded.Facing != second.Facing {
		t.Fatalf("loaded position = %#v, want persisted fields from %#v", loaded, second)
	}
}

func TestApplyGameResultCreditsPetStarsOnce(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	first, err := app.petStore().ApplyGameResult(ctx, 123, 7, 9, "round-1")
	if err != nil {
		t.Fatalf("first game result error = %v", err)
	}
	if first.StarBalance != 9 || first.PetState.Happiness != 57 || first.PetState.Energy != 45 {
		t.Fatalf("first profile = %#v, want balance 9 happiness 57 energy 45", first)
	}

	second, err := app.petStore().ApplyGameResult(ctx, 123, 7, 9, "round-1")
	if err != nil {
		t.Fatalf("second game result error = %v", err)
	}
	if second.StarBalance != 9 || second.PetState.Happiness != 57 || second.PetState.Energy != 45 {
		t.Fatalf("second profile = %#v, want duplicate to keep balance and pet stats unchanged", second)
	}

	var ledgerRows int
	var source string
	if err := app.db.QueryRow(ctx, "select count(*) from student_star_ledger").Scan(&ledgerRows); err != nil {
		t.Fatalf("count ledger rows: %v", err)
	}
	if err := app.db.QueryRow(ctx, "select source from student_star_ledger where event_id = 'pet-falling-stars:123:round-1'").Scan(&source); err != nil {
		t.Fatalf("load pet star ledger source: %v", err)
	}
	if ledgerRows != 1 || source != "pet_falling_stars" {
		t.Fatalf("ledgerRows=%d source=%q, want 1 pet_falling_stars", ledgerRows, source)
	}
}

func TestFeedStudentPetConsumesCookieInventory(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if err := hqinventory.IncrementStudentItem(ctx, app.db, 123, hqinventory.CookieKey, 2); err != nil {
		t.Fatalf("seed cookie inventory: %v", err)
	}

	profile, err := app.petStore().Feed(ctx, 123)
	if err != nil {
		t.Fatalf("feed pet error = %v", err)
	}
	if profile.Cookies != 1 || profile.PetState.Hunger != 60 {
		t.Fatalf("profile = %#v, want 1 cookie and hunger 60", profile)
	}
}

func TestFeedStudentPetRequiresCookieInventory(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	_, err := app.petStore().Feed(context.Background(), 123)
	if err != errNoCookies {
		t.Fatalf("feed pet error = %v, want errNoCookies", err)
	}
}

func TestPetProfileJSONUsesStudentAPIFieldNames(t *testing.T) {
	profile := hqpet.Profile{
		ID:          123,
		DisplayName: "Student",
		Cookies:     2,
		StarBalance: 7,
		PetState: hqpet.State{
			Hunger:    50,
			Happiness: 60,
			Energy:    70,
			Mood:      "idle",
		},
	}

	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("marshal profile: %v", err)
	}
	body := string(data)
	for _, field := range []string{`"display_name"`, `"star_balance"`, `"pet_state"`, `"last_decay_at"`} {
		if !strings.Contains(body, field) {
			t.Fatalf("profile json = %s, want field %s", body, field)
		}
	}
}

func TestPurchaseStudentShopItemBuysCookie(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := app.db.Exec(ctx, "insert into student_wallet (app_user_id, star_balance) values (123, 125)"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}

	response, err := hqinventory.PurchaseStudentShopItem(ctx, app.db, 123, hqinventory.ShopPurchaseRequest{
		ShopID:   "cookie-keeper-shop",
		ItemKey:  hqinventory.CookieKey,
		Quantity: 1,
	})
	if err != nil {
		t.Fatalf("purchase error = %v", err)
	}
	if response.StarBalance != 75 {
		t.Fatalf("star balance = %d, want 75", response.StarBalance)
	}
	if len(response.Inventory.Items) != 1 || response.Inventory.Items[0].Key != hqinventory.CookieKey || response.Inventory.Items[0].Quantity != 1 {
		t.Fatalf("inventory = %#v, want 1 cookie", response.Inventory)
	}

	var walletBalance int
	var cookieQuantity int
	var ledgerDelta int
	if err := app.db.QueryRow(ctx, "select star_balance from student_wallet where app_user_id = 123").Scan(&walletBalance); err != nil {
		t.Fatalf("load wallet: %v", err)
	}
	if err := app.db.QueryRow(
		ctx,
		`
			select sii.quantity
			from student_inventory_item sii
			join inventory_item_type iit on iit.id = sii.item_type_id
			where sii.app_user_id = 123 and iit.key = 'cookie'
		`,
	).Scan(&cookieQuantity); err != nil {
		t.Fatalf("load cookie quantity: %v", err)
	}
	if err := app.db.QueryRow(ctx, "select delta from student_star_ledger where source = 'shop_purchase'").Scan(&ledgerDelta); err != nil {
		t.Fatalf("load shop ledger: %v", err)
	}
	if walletBalance != 75 || cookieQuantity != 1 || ledgerDelta != -50 {
		t.Fatalf("wallet=%d cookies=%d ledgerDelta=%d, want 75, 1, -50", walletBalance, cookieQuantity, ledgerDelta)
	}
}

func TestPurchaseStudentShopItemRequiresStars(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := app.db.Exec(ctx, "insert into student_wallet (app_user_id, star_balance) values (123, 49)"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}

	_, err := hqinventory.PurchaseStudentShopItem(ctx, app.db, 123, hqinventory.ShopPurchaseRequest{
		ShopID:   "cookie-keeper-shop",
		ItemKey:  hqinventory.CookieKey,
		Quantity: 1,
	})
	if err != hqinventory.ErrInsufficientStars {
		t.Fatalf("purchase error = %v, want ErrInsufficientStars", err)
	}

	var walletBalance int
	var inventoryRows int
	if err := app.db.QueryRow(ctx, "select star_balance from student_wallet where app_user_id = 123").Scan(&walletBalance); err != nil {
		t.Fatalf("load wallet: %v", err)
	}
	if err := app.db.QueryRow(ctx, "select count(*) from student_inventory_item").Scan(&inventoryRows); err != nil {
		t.Fatalf("count inventory rows: %v", err)
	}
	if walletBalance != 49 || inventoryRows != 0 {
		t.Fatalf("wallet=%d inventoryRows=%d, want 49 and 0", walletBalance, inventoryRows)
	}
}

func TestPurchaseStudentShopItemRejectsInvalidPurchase(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	invalidRequests := []hqinventory.ShopPurchaseRequest{
		{ShopID: "other-shop", ItemKey: hqinventory.CookieKey, Quantity: 1},
		{ShopID: "cookie-keeper-shop", ItemKey: "star", Quantity: 1},
		{ShopID: "cookie-keeper-shop", ItemKey: hqinventory.CookieKey, Quantity: 0},
	}
	for _, request := range invalidRequests {
		if _, err := hqinventory.PurchaseStudentShopItem(context.Background(), app.db, 123, request); err == nil {
			t.Fatalf("purchase %#v succeeded, want error", request)
		}
	}
}

func TestCraftStudentRecipeCreatesStoneBlock(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if err := hqinventory.IncrementStudentItem(ctx, app.db, 123, "rock", 5); err != nil {
		t.Fatalf("seed rock inventory: %v", err)
	}

	recipes, err := hqinventory.LoadCraftingRecipes(ctx, app.db, 123)
	if err != nil {
		t.Fatalf("load recipes: %v", err)
	}
	if len(recipes.Recipes) != 1 || recipes.Recipes[0].Key != "stone_block" || !recipes.Recipes[0].CanCraft {
		t.Fatalf("recipes = %#v, want craftable stone_block", recipes)
	}

	response, err := hqinventory.CraftStudentRecipe(ctx, app.db, 123, hqinventory.CraftRecipeRequest{RecipeKey: "stone_block"})
	if err != nil {
		t.Fatalf("craft recipe error = %v", err)
	}

	quantities := map[string]int{}
	for _, item := range response.Inventory.Items {
		quantities[item.Key] = item.Quantity
	}
	if quantities["rock"] != 1 || quantities["stone_block"] != 1 {
		t.Fatalf("inventory quantities = %#v, want rock=1 stone_block=1", quantities)
	}
	if len(response.Recipes) != 1 || response.Recipes[0].CanCraft {
		t.Fatalf("recipes after craft = %#v, want stone_block not craftable", response.Recipes)
	}
}

func TestCraftStudentRecipeRequiresIngredients(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if err := hqinventory.IncrementStudentItem(ctx, app.db, 123, "rock", 3); err != nil {
		t.Fatalf("seed rock inventory: %v", err)
	}

	_, err := hqinventory.CraftStudentRecipe(ctx, app.db, 123, hqinventory.CraftRecipeRequest{RecipeKey: "stone_block"})
	if err != hqinventory.ErrInsufficientIngredient {
		t.Fatalf("craft recipe error = %v, want ErrInsufficientIngredient", err)
	}

	var rockQuantity int
	var stoneBlockRows int
	if err := app.db.QueryRow(
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
	if err := app.db.QueryRow(
		ctx,
		`
			select count(*)
			from student_inventory_item sii
			join inventory_item_type iit on iit.id = sii.item_type_id
			where sii.app_user_id = 123 and iit.key = 'stone_block'
		`,
	).Scan(&stoneBlockRows); err != nil {
		t.Fatalf("count stone block rows: %v", err)
	}
	if rockQuantity != 3 || stoneBlockRows != 0 {
		t.Fatalf("rockQuantity=%d stoneBlockRows=%d, want 3 and 0", rockQuantity, stoneBlockRows)
	}
}

func TestCraftStudentRecipeRejectsUnknownRecipe(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	_, err := hqinventory.CraftStudentRecipe(context.Background(), app.db, 123, hqinventory.CraftRecipeRequest{RecipeKey: "missing"})
	if err != hqinventory.ErrUnknownRecipe {
		t.Fatalf("craft recipe error = %v, want ErrUnknownRecipe", err)
	}
}

func TestPlaceSunnyTownMapObjectConsumesStoneBlock(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if err := hqinventory.IncrementStudentItem(ctx, app.db, 123, stoneBlockItemKey, 2); err != nil {
		t.Fatalf("seed stone block inventory: %v", err)
	}

	placed, err := app.placeSunnyTownMapObject(ctx, sunnyTownPlaceMapObjectRequest{
		AppUserID: 123,
		RoomID:    testSunnyTownRoomID,
		MapID:     testSunnyTownMapID,
		GridX:     4,
		GridY:     5,
		ItemKey:   stoneBlockItemKey,
	})
	if err != nil {
		t.Fatalf("place map object error = %v", err)
	}
	if placed.ID == 0 || placed.ItemKey != stoneBlockItemKey || placed.GridX != 4 || placed.GridY != 5 || placed.RemainingItemAmount != 1 {
		t.Fatalf("placed object = %#v, want stone block at 4,5 with 1 remaining", placed)
	}

	loaded, err := app.loadSunnyTownMapObjects(ctx, testSunnyTownRoomID, testSunnyTownMapID)
	if err != nil {
		t.Fatalf("load map objects error = %v", err)
	}
	if len(loaded.Objects) != 1 || loaded.Objects[0].ID != placed.ID {
		t.Fatalf("loaded objects = %#v, want placed object", loaded.Objects)
	}
}

func TestPlaceSunnyTownMapObjectRollsBackInventoryWhenOccupied(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if err := hqinventory.IncrementStudentItem(ctx, app.db, 123, stoneBlockItemKey, 2); err != nil {
		t.Fatalf("seed stone block inventory: %v", err)
	}
	request := sunnyTownPlaceMapObjectRequest{
		AppUserID: 123,
		RoomID:    testSunnyTownRoomID,
		MapID:     testSunnyTownMapID,
		GridX:     4,
		GridY:     5,
		ItemKey:   stoneBlockItemKey,
	}
	if _, err := app.placeSunnyTownMapObject(ctx, request); err != nil {
		t.Fatalf("first place map object error = %v", err)
	}
	_, err := app.placeSunnyTownMapObject(ctx, request)
	if err != errMapObjectOccupied {
		t.Fatalf("second place map object error = %v, want errMapObjectOccupied", err)
	}
	quantity, err := loadStudentInventoryQuantity(ctx, app.db, 123, stoneBlockItemKey)
	if err != nil {
		t.Fatalf("load stone block quantity: %v", err)
	}
	if quantity != 1 {
		t.Fatalf("stone block quantity = %d, want 1", quantity)
	}
}

func TestRemoveSunnyTownMapObjectRefundsStoneBlock(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if err := hqinventory.IncrementStudentItem(ctx, app.db, 123, stoneBlockItemKey, 1); err != nil {
		t.Fatalf("seed stone block inventory: %v", err)
	}
	if _, err := app.placeSunnyTownMapObject(ctx, sunnyTownPlaceMapObjectRequest{
		AppUserID: 123,
		RoomID:    testSunnyTownRoomID,
		MapID:     testSunnyTownMapID,
		GridX:     4,
		GridY:     5,
		ItemKey:   stoneBlockItemKey,
	}); err != nil {
		t.Fatalf("place map object error = %v", err)
	}

	removed, err := app.removeSunnyTownMapObject(ctx, sunnyTownRemoveMapObjectRequest{
		AppUserID: 123,
		RoomID:    testSunnyTownRoomID,
		MapID:     testSunnyTownMapID,
		GridX:     4,
		GridY:     5,
	})
	if err != nil {
		t.Fatalf("remove map object error = %v", err)
	}
	if removed.ItemKey != stoneBlockItemKey || removed.RemainingItemAmount != 1 {
		t.Fatalf("removed object = %#v, want refunded stone block", removed)
	}

	loaded, err := app.loadSunnyTownMapObjects(ctx, testSunnyTownRoomID, testSunnyTownMapID)
	if err != nil {
		t.Fatalf("load map objects error = %v", err)
	}
	if len(loaded.Objects) != 0 {
		t.Fatalf("loaded objects = %#v, want empty", loaded.Objects)
	}
}

func TestEquipStudentItemRequiresOwnedEquippableItem(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if err := hqinventory.IncrementStudentItem(ctx, app.db, 123, "sunny_hoodie", 1); err != nil {
		t.Fatalf("seed hoodie inventory: %v", err)
	}

	equipment, err := hqinventory.EquipStudentItem(ctx, app.db, 123, hqinventory.EquipmentChangeRequest{
		Slot:    hqinventory.EquipmentSlotGear,
		ItemKey: "sunny_hoodie",
	})
	if err != nil {
		t.Fatalf("equip hoodie error = %v", err)
	}
	if equipment.Slots[0].Item == nil || equipment.Slots[0].Item.Key != "sunny_hoodie" {
		t.Fatalf("equipment = %#v, want hoodie in gear slot", equipment)
	}

	inventory, err := hqinventory.LoadStudent(ctx, app.db, 123)
	if err != nil {
		t.Fatalf("load inventory: %v", err)
	}
	if len(inventory.Items) != 1 || !inventory.Items[0].Equipped {
		t.Fatalf("inventory = %#v, want equipped hoodie", inventory)
	}
}

func TestEquipStudentItemRejectsCookieAndMissingOwnership(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if err := hqinventory.IncrementStudentItem(ctx, app.db, 123, hqinventory.CookieKey, 1); err != nil {
		t.Fatalf("seed cookie inventory: %v", err)
	}

	_, err := hqinventory.EquipStudentItem(ctx, app.db, 123, hqinventory.EquipmentChangeRequest{
		Slot:    hqinventory.EquipmentSlotGear,
		ItemKey: hqinventory.CookieKey,
	})
	if err != hqinventory.ErrItemNotEquippable {
		t.Fatalf("equip cookie error = %v, want ErrItemNotEquippable", err)
	}

	_, err = hqinventory.EquipStudentItem(ctx, app.db, 123, hqinventory.EquipmentChangeRequest{
		Slot:    hqinventory.EquipmentSlotAccessory,
		ItemKey: "star_cap",
	})
	if err != hqinventory.ErrItemNotOwned {
		t.Fatalf("equip unowned cap error = %v, want ErrItemNotOwned", err)
	}
}

func TestUnequipStudentItemClearsSlot(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if err := hqinventory.IncrementStudentItem(ctx, app.db, 123, "star_cap", 1); err != nil {
		t.Fatalf("seed cap inventory: %v", err)
	}
	if _, err := hqinventory.EquipStudentItem(ctx, app.db, 123, hqinventory.EquipmentChangeRequest{
		Slot:    hqinventory.EquipmentSlotAccessory,
		ItemKey: "star_cap",
	}); err != nil {
		t.Fatalf("equip cap error = %v", err)
	}

	equipment, err := hqinventory.UnequipStudentItem(ctx, app.db, 123, hqinventory.EquipmentChangeRequest{Slot: hqinventory.EquipmentSlotAccessory})
	if err != nil {
		t.Fatalf("unequip cap error = %v", err)
	}
	if equipment.Slots[1].Item != nil {
		t.Fatalf("equipment = %#v, want empty accessory slot", equipment)
	}
}

func TestEquipStudentItemSupportsToolSlot(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if err := hqinventory.IncrementStudentItem(ctx, app.db, 123, "pickaxe", 1); err != nil {
		t.Fatalf("seed pickaxe inventory: %v", err)
	}

	equipment, err := hqinventory.EquipStudentItem(ctx, app.db, 123, hqinventory.EquipmentChangeRequest{
		Slot:    hqinventory.EquipmentSlotTool,
		ItemKey: "pickaxe",
	})
	if err != nil {
		t.Fatalf("equip pickaxe error = %v", err)
	}
	if equipment.Slots[2].Item == nil || equipment.Slots[2].Item.Key != "pickaxe" {
		t.Fatalf("equipment = %#v, want pickaxe in tool slot", equipment)
	}
}

func TestStudentHotbarDefaultsAndUpdates(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if err := hqinventory.IncrementStudentItem(ctx, app.db, 123, "pickaxe", 1); err != nil {
		t.Fatalf("seed pickaxe inventory: %v", err)
	}
	if err := hqinventory.IncrementStudentItem(ctx, app.db, 123, "stone_block", 2); err != nil {
		t.Fatalf("seed stone block inventory: %v", err)
	}

	hotbar, err := hqinventory.LoadStudentHotbar(ctx, app.db, 123)
	if err != nil {
		t.Fatalf("load hotbar: %v", err)
	}
	if len(hotbar.Slots) != 5 || hotbar.Slots[0].Item == nil || hotbar.Slots[0].Item.Key != "pickaxe" {
		t.Fatalf("hotbar = %#v, want pickaxe in slot 1", hotbar)
	}
	if hotbar.Slots[1].Item == nil || hotbar.Slots[1].Item.Key != "stone_block" {
		t.Fatalf("hotbar = %#v, want stone_block in slot 2", hotbar)
	}

	hotbar, err = hqinventory.SetStudentHotbarSlot(ctx, app.db, 123, hqinventory.HotbarSlotRequest{Slot: 3, ItemKey: "stone_block"})
	if err != nil {
		t.Fatalf("set hotbar slot: %v", err)
	}
	if hotbar.Slots[2].Item == nil || hotbar.Slots[2].Item.Key != "stone_block" {
		t.Fatalf("hotbar = %#v, want stone_block in slot 3", hotbar)
	}

	hotbar, err = hqinventory.SetStudentHotbarSlot(ctx, app.db, 123, hqinventory.HotbarSlotRequest{Slot: 3})
	if err != nil {
		t.Fatalf("clear hotbar slot: %v", err)
	}
	if hotbar.Slots[2].Item != nil {
		t.Fatalf("hotbar = %#v, want empty slot 3", hotbar)
	}
}

func TestStudentHotbarRejectsUnownedItem(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	_, err := hqinventory.SetStudentHotbarSlot(context.Background(), app.db, 123, hqinventory.HotbarSlotRequest{Slot: 1, ItemKey: "stone_block"})
	if !errors.Is(err, hqinventory.ErrHotbarItemNotOwned) {
		t.Fatalf("set hotbar error = %v, want ErrHotbarItemNotOwned", err)
	}
}

func TestRecordAIGradeResultStoresRecommendationWithoutAutoApply(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()
	app.aiAutoApplyGrades = false

	ctx := context.Background()
	assignmentID, attemptID := seedAssignmentAttempt(t, app)
	passed := true
	confidence := 0.91

	response, err := app.recordAIGradeResult(ctx, attemptID, aiapi.AIGradeResultRequest{
		RequestID:           "assignment-attempt-1:assignment-grader-v1",
		Status:              aiGradeStatusCompleted,
		RecommendedPassed:   &passed,
		RecommendedFeedback: "Correct.",
		Confidence:          &confidence,
		RubricScores: []aiapi.RubricScore{{
			Name:   "correctness",
			Score:  1,
			Reason: "Matches expected answer.",
		}},
		Model:         "fake-grader",
		PromptVersion: "assignment-grader-v1",
	})
	if err != nil {
		t.Fatalf("record ai grade result: %v", err)
	}
	if response.Applied {
		t.Fatal("expected AI result to be stored without auto-applying grade")
	}
	if response.ApplySkippedReason != aiApplySkippedAutoApplyDisabled {
		t.Fatalf("apply skipped reason = %q, want %q", response.ApplySkippedReason, aiApplySkippedAutoApplyDisabled)
	}

	var count int
	var currentPassed *bool
	if err := app.db.QueryRow(ctx, "select count(*) from assignment_ai_grade where assignment_attempt_id = $1", attemptID).Scan(&count); err != nil {
		t.Fatalf("count ai grades: %v", err)
	}
	if err := app.db.QueryRow(ctx, "select passed from assignment_attempt where id = $1 and assignment_id = $2", attemptID, assignmentID).Scan(&currentPassed); err != nil {
		t.Fatalf("load attempt passed: %v", err)
	}
	if count != 1 || currentPassed != nil {
		t.Fatalf("ai grade count=%d passed=%v, want stored recommendation and ungraded attempt", count, currentPassed)
	}
}

func TestRecordAIGradeResultAutoAppliesThroughSharedGradeCommand(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()
	app.aiAutoApplyGrades = true

	ctx := context.Background()
	_, attemptID := seedAssignmentAttempt(t, app)
	passed := true
	confidence := 0.97

	response, err := app.recordAIGradeResult(ctx, attemptID, aiapi.AIGradeResultRequest{
		RequestID:           "assignment-attempt-1:assignment-grader-v1",
		Status:              aiGradeStatusCompleted,
		RecommendedPassed:   &passed,
		RecommendedFeedback: "Correct.",
		Confidence:          &confidence,
		Model:               "fake-grader",
		PromptVersion:       "assignment-grader-v1",
	})
	if err != nil {
		t.Fatalf("record ai grade result: %v", err)
	}
	if !response.Applied {
		t.Fatal("expected AI grade to auto-apply")
	}

	var gradedByType string
	var gradeSource string
	var reviewStatus string
	var cookieAwarded bool
	var cookieQuantity int
	if err := app.db.QueryRow(
		ctx,
		`
			select graded_by_type, grade_source, ai_review_status, cookie_awarded
			from assignment_attempt
			where id = $1
		`,
		attemptID,
	).Scan(&gradedByType, &gradeSource, &reviewStatus, &cookieAwarded); err != nil {
		t.Fatalf("load graded attempt metadata: %v", err)
	}
	if err := app.db.QueryRow(
		ctx,
		`
			select sii.quantity
			from student_inventory_item sii
			join inventory_item_type iit on iit.id = sii.item_type_id
			where sii.app_user_id = 123 and iit.key = 'cookie'
		`,
	).Scan(&cookieQuantity); err != nil {
		t.Fatalf("load cookie quantity: %v", err)
	}
	if gradedByType != graderTypeAI || gradeSource != gradeSourceAIAuto || reviewStatus != aiReviewStatusPending || !cookieAwarded || cookieQuantity != 1 {
		t.Fatalf("metadata type=%q source=%q review=%q cookieAwarded=%v cookies=%d, want AI auto grade with one cookie", gradedByType, gradeSource, reviewStatus, cookieAwarded, cookieQuantity)
	}
}

func TestTeacherOverrideMarksAIAttemptOverridden(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()
	app.aiAutoApplyGrades = true

	ctx := context.Background()
	assignmentID, attemptID := seedAssignmentAttempt(t, app)
	passed := true
	if _, err := app.recordAIGradeResult(ctx, attemptID, aiapi.AIGradeResultRequest{
		RequestID:           "assignment-attempt-1:assignment-grader-v1",
		Status:              aiGradeStatusCompleted,
		RecommendedPassed:   &passed,
		RecommendedFeedback: "Correct.",
		Model:               "fake-grader",
		PromptVersion:       "assignment-grader-v1",
	}); err != nil {
		t.Fatalf("record ai grade result: %v", err)
	}

	teacherID := int64(124)
	if _, err := app.db.Exec(ctx, "insert into app_user (id, display_name) values ($1, 'Teacher')", teacherID); err != nil {
		t.Fatalf("seed teacher: %v", err)
	}
	if err := app.gradeAssignmentAttempt(ctx, gradeAttemptCommand{
		AssignmentID:     assignmentID,
		AttemptID:        attemptID,
		Passed:           false,
		Feedback:         "Try again.",
		GradedByType:     graderTypeTeacher,
		GradedByUserID:   &teacherID,
		GradeSource:      gradeSourceManual,
		PreserveAIReview: false,
	}); err != nil {
		t.Fatalf("teacher override grade: %v", err)
	}

	var gradedByType string
	var gradeSource string
	var reviewStatus string
	var currentPassed bool
	if err := app.db.QueryRow(
		ctx,
		`
			select graded_by_type, grade_source, ai_review_status, passed
			from assignment_attempt
			where id = $1
		`,
		attemptID,
	).Scan(&gradedByType, &gradeSource, &reviewStatus, &currentPassed); err != nil {
		t.Fatalf("load override metadata: %v", err)
	}
	if gradedByType != graderTypeTeacher || gradeSource != gradeSourceTeacherOverride || reviewStatus != aiReviewStatusOverridden || currentPassed {
		t.Fatalf("override metadata type=%q source=%q review=%q passed=%v, want teacher override failure", gradedByType, gradeSource, reviewStatus, currentPassed)
	}
}

func TestAIGradeRetryDoesNotOverwriteTeacherOverride(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()
	app.aiAutoApplyGrades = true

	ctx := context.Background()
	assignmentID, attemptID := seedAssignmentAttempt(t, app)
	passed := true
	request := aiapi.AIGradeResultRequest{
		RequestID:           "assignment-attempt-1:assignment-grader-v1",
		Status:              aiGradeStatusCompleted,
		RecommendedPassed:   &passed,
		RecommendedFeedback: "Correct.",
		Model:               "fake-grader",
		PromptVersion:       "assignment-grader-v1",
	}
	if _, err := app.recordAIGradeResult(ctx, attemptID, request); err != nil {
		t.Fatalf("record initial ai grade result: %v", err)
	}

	teacherID := int64(124)
	if _, err := app.db.Exec(ctx, "insert into app_user (id, display_name) values ($1, 'Teacher')", teacherID); err != nil {
		t.Fatalf("seed teacher: %v", err)
	}
	if err := app.gradeAssignmentAttempt(ctx, gradeAttemptCommand{
		AssignmentID:     assignmentID,
		AttemptID:        attemptID,
		Passed:           false,
		Feedback:         "Try again.",
		GradedByType:     graderTypeTeacher,
		GradedByUserID:   &teacherID,
		GradeSource:      gradeSourceManual,
		PreserveAIReview: false,
	}); err != nil {
		t.Fatalf("teacher override grade: %v", err)
	}

	response, err := app.recordAIGradeResult(ctx, attemptID, request)
	if err != nil {
		t.Fatalf("retry ai grade result: %v", err)
	}
	if response.Applied || response.ApplySkippedReason != aiApplySkippedTeacherOverride {
		t.Fatalf("retry applied=%v reason=%q, want skipped teacher override", response.Applied, response.ApplySkippedReason)
	}

	var gradedByType string
	var gradeSource string
	var reviewStatus string
	var currentPassed bool
	if err := app.db.QueryRow(
		ctx,
		`
			select graded_by_type, grade_source, ai_review_status, passed
			from assignment_attempt
			where id = $1
		`,
		attemptID,
	).Scan(&gradedByType, &gradeSource, &reviewStatus, &currentPassed); err != nil {
		t.Fatalf("load retry metadata: %v", err)
	}
	if gradedByType != graderTypeTeacher || gradeSource != gradeSourceTeacherOverride || reviewStatus != aiReviewStatusOverridden || currentPassed {
		t.Fatalf("retry metadata type=%q source=%q review=%q passed=%v, want teacher override preserved", gradedByType, gradeSource, reviewStatus, currentPassed)
	}
}

func TestAIGradeRetryDoesNotOverwriteReviewedAttempt(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()
	app.aiAutoApplyGrades = true

	ctx := context.Background()
	_, attemptID := seedAssignmentAttempt(t, app)
	passed := true
	request := aiapi.AIGradeResultRequest{
		RequestID:           "assignment-attempt-1:assignment-grader-v1",
		Status:              aiGradeStatusCompleted,
		RecommendedPassed:   &passed,
		RecommendedFeedback: "Correct.",
		Model:               "fake-grader",
		PromptVersion:       "assignment-grader-v1",
	}
	if _, err := app.recordAIGradeResult(ctx, attemptID, request); err != nil {
		t.Fatalf("record initial ai grade result: %v", err)
	}
	if _, err := app.db.Exec(ctx, "update assignment_attempt set ai_review_status = $2 where id = $1", attemptID, aiReviewStatusReviewed); err != nil {
		t.Fatalf("mark ai grade reviewed: %v", err)
	}

	passed = false
	request.RecommendedPassed = &passed
	request.RecommendedFeedback = "Incorrect."
	response, err := app.recordAIGradeResult(ctx, attemptID, request)
	if err != nil {
		t.Fatalf("retry ai grade result: %v", err)
	}
	if response.Applied || response.ApplySkippedReason != aiApplySkippedTeacherReviewed {
		t.Fatalf("retry applied=%v reason=%q, want skipped teacher reviewed", response.Applied, response.ApplySkippedReason)
	}

	var reviewStatus string
	var currentPassed bool
	if err := app.db.QueryRow(ctx, "select ai_review_status, passed from assignment_attempt where id = $1", attemptID).Scan(&reviewStatus, &currentPassed); err != nil {
		t.Fatalf("load reviewed attempt: %v", err)
	}
	if reviewStatus != aiReviewStatusReviewed || !currentPassed {
		t.Fatalf("review=%q passed=%v, want reviewed AI pass preserved", reviewStatus, currentPassed)
	}
}

func TestAIGradeDuplicateRequestIDSameAttemptIsIdempotent(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()
	app.aiAutoApplyGrades = true

	ctx := context.Background()
	_, attemptID := seedAssignmentAttempt(t, app)
	passed := true
	request := aiapi.AIGradeResultRequest{
		RequestID:           "assignment-attempt-1:assignment-grader-v1",
		Status:              aiGradeStatusCompleted,
		RecommendedPassed:   &passed,
		RecommendedFeedback: "Correct.",
		Model:               "fake-grader",
		PromptVersion:       "assignment-grader-v1",
	}
	first, err := app.recordAIGradeResult(ctx, attemptID, request)
	if err != nil {
		t.Fatalf("record first ai grade result: %v", err)
	}
	request.RecommendedFeedback = "Still correct."
	second, err := app.recordAIGradeResult(ctx, attemptID, request)
	if err != nil {
		t.Fatalf("record second ai grade result: %v", err)
	}
	if !first.Applied || !second.Applied || first.ID != second.ID {
		t.Fatalf("first=%#v second=%#v, want same applied AI grade", first, second)
	}

	var aiGradeCount int
	var cookieQuantity int
	if err := app.db.QueryRow(ctx, "select count(*) from assignment_ai_grade where assignment_attempt_id = $1", attemptID).Scan(&aiGradeCount); err != nil {
		t.Fatalf("count ai grades: %v", err)
	}
	if err := app.db.QueryRow(
		ctx,
		`
			select sii.quantity
			from student_inventory_item sii
			join inventory_item_type iit on iit.id = sii.item_type_id
			where sii.app_user_id = 123 and iit.key = 'cookie'
		`,
	).Scan(&cookieQuantity); err != nil {
		t.Fatalf("load cookie quantity: %v", err)
	}
	if aiGradeCount != 1 || cookieQuantity != 1 {
		t.Fatalf("aiGradeCount=%d cookieQuantity=%d, want one idempotent AI grade and one cookie", aiGradeCount, cookieQuantity)
	}
}

func TestAIGradeDuplicateRequestIDDifferentAttemptIsRejected(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()
	app.aiAutoApplyGrades = true

	ctx := context.Background()
	_, firstAttemptID := seedAssignmentAttempt(t, app)
	_, secondAttemptID := seedAssignmentAttempt(t, app)
	passed := true
	request := aiapi.AIGradeResultRequest{
		RequestID:           "assignment-attempt-1:assignment-grader-v1",
		Status:              aiGradeStatusCompleted,
		RecommendedPassed:   &passed,
		RecommendedFeedback: "Correct.",
		Model:               "fake-grader",
		PromptVersion:       "assignment-grader-v1",
	}
	if _, err := app.recordAIGradeResult(ctx, firstAttemptID, request); err != nil {
		t.Fatalf("record first ai grade result: %v", err)
	}

	_, err := app.recordAIGradeResult(ctx, secondAttemptID, request)
	if !errors.Is(err, errAIGradeRequestAttemptMismatch) {
		t.Fatalf("second ai grade error = %v, want request attempt mismatch", err)
	}

	var secondAIGradeID *int64
	var secondPassed *bool
	if err := app.db.QueryRow(ctx, "select ai_grade_id, passed from assignment_attempt where id = $1", secondAttemptID).Scan(&secondAIGradeID, &secondPassed); err != nil {
		t.Fatalf("load second attempt: %v", err)
	}
	if secondAIGradeID != nil || secondPassed != nil {
		t.Fatalf("second ai_grade_id=%v passed=%v, want untouched second attempt", secondAIGradeID, secondPassed)
	}
}

func seedAssignmentAttempt(t *testing.T, app *app) (int64, int64) {
	t.Helper()
	ctx := context.Background()

	var assignmentID int64
	if err := app.db.QueryRow(
		ctx,
		"insert into assignment (category, prompt, expected_answer) values ('MATH', 'What is 2 + 2?', '4') returning id",
	).Scan(&assignmentID); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}

	var attemptID int64
	if err := app.db.QueryRow(
		ctx,
		`
			insert into assignment_attempt (
				assignment_id,
				student_user_id,
				attempt_number,
				submitted_answer
			)
			values ($1, 123, 1, '4')
			returning id
		`,
		assignmentID,
	).Scan(&attemptID); err != nil {
		t.Fatalf("seed assignment attempt: %v", err)
	}

	return assignmentID, attemptID
}

func testRewardApp(t *testing.T) (*app, func()) {
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

	schema := fmt.Sprintf("reward_test_%d", time.Now().UnixNano())
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
		`create table pet_state (
			user_id bigint primary key references app_user(id) on delete cascade,
			hunger integer not null default 50,
			happiness integer not null default 50,
			energy integer not null default 50,
			sleeping boolean not null default false,
			updated_at timestamptz not null default now(),
			last_decay_at timestamptz not null default now(),
			sleep_started_at timestamptz null,
			sleep_started_energy integer null
		)`,
		`create table assignment (
			id bigserial primary key,
			category text not null,
			prompt text not null,
			expected_answer text not null,
			created_at timestamptz not null default now()
		)`,
		`create table assignment_attempt (
			id bigserial primary key,
			assignment_id bigint not null references assignment(id) on delete cascade,
			student_user_id bigint not null references app_user(id) on delete cascade,
			attempt_number integer not null,
			submitted_answer text not null,
			date_submitted timestamptz not null default now(),
			passed boolean null,
			feedback text null,
			date_graded timestamptz null,
			cookie_awarded boolean not null default false,
			reset_at timestamptz null,
			graded_by_type text null,
			graded_by_user_id bigint null references app_user(id) on delete set null,
			graded_by_service text null,
			grade_source text null,
			ai_review_status text null,
			ai_grade_id bigint null,
			unique (assignment_id, student_user_id, attempt_number)
		)`,
		`create table assignment_ai_grade (
			id bigserial primary key,
			assignment_attempt_id bigint not null references assignment_attempt(id) on delete cascade,
			request_id text not null unique,
			status text not null,
			recommended_passed boolean null,
			recommended_feedback text null,
			confidence numeric null,
			rubric_scores jsonb not null default '[]'::jsonb,
			model text null,
			prompt_version text not null,
			raw_response jsonb null,
			error_message text null,
			created_at timestamptz not null default now(),
			completed_at timestamptz null
		)`,
		`alter table assignment_attempt
			add constraint assignment_attempt_ai_grade_id_fkey
			foreign key (ai_grade_id) references assignment_ai_grade(id) on delete set null`,
		`create table inventory_item_type (
			id bigserial primary key,
			key text not null unique,
			name text not null,
			description text not null default '',
			equip_slot text null,
			visual_key text null,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now()
		)`,
		`insert into inventory_item_type (key, name, description)
			values ('cookie', 'Cookie', 'A treat for your pet.')`,
		`insert into inventory_item_type (key, name, description, equip_slot, visual_key)
			values
				('sunny_hoodie', 'Sunny Hoodie', 'A cozy hoodie for Sunny Town.', 'gear', 'sunny_hoodie'),
				('star_cap', 'Star Cap', 'A bright cap for sunny adventures.', 'accessory', 'star_cap'),
				('pickaxe', 'Pickaxe', 'A sturdy starter tool.', 'tool', 'pickaxe')`,
		`insert into inventory_item_type (key, name, description)
			values
				('rock', 'Rock', 'A sturdy rock from Forest Crossing.'),
				('crystal', 'Crystal', 'A bright crystal from Forest Crossing.'),
				('stone_block', 'Stone Block', 'A solid block crafted from stone.')`,
		`create table student_inventory_item (
			app_user_id bigint not null references app_user(id) on delete cascade,
			item_type_id bigint not null references inventory_item_type(id) on delete restrict,
			quantity integer not null default 0,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			primary key (app_user_id, item_type_id),
			constraint student_inventory_item_quantity_nonnegative check (quantity >= 0)
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
		`create table student_equipped_item (
			app_user_id bigint not null references app_user(id) on delete cascade,
			slot text not null,
			item_type_id bigint not null references inventory_item_type(id) on delete restrict,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			primary key (app_user_id, slot),
			constraint student_equipped_item_slot_check check (slot in ('gear', 'accessory', 'tool'))
		)`,
		`create table student_hotbar_slot (
			app_user_id bigint not null references app_user(id) on delete cascade,
			slot_index integer not null,
			item_type_id bigint not null references inventory_item_type(id) on delete restrict,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			primary key (app_user_id, slot_index),
			constraint student_hotbar_slot_index_check check (slot_index between 1 and 5)
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
		`insert into pet_state (user_id, hunger, happiness, energy) values (123, 50, 50, 50)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(ctx, statement); err != nil {
			db.Close()
			_, _ = adminPool.Exec(ctx, "drop schema "+schema+" cascade")
			adminPool.Close()
			t.Fatalf("setup statement failed: %v", err)
		}
	}

	return &app{db: db}, func() {
		db.Close()
		_, _ = adminPool.Exec(ctx, "drop schema "+schema+" cascade")
		adminPool.Close()
	}
}
