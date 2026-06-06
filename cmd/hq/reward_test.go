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
	"testing"
	"time"

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
	first, err := app.applyGameResult(ctx, 123, 7, 9, "round-1")
	if err != nil {
		t.Fatalf("first game result error = %v", err)
	}
	if first.StarBalance != 9 || first.PetState.Happiness != 57 || first.PetState.Energy != 45 {
		t.Fatalf("first profile = %#v, want balance 9 happiness 57 energy 45", first)
	}

	second, err := app.applyGameResult(ctx, 123, 7, 9, "round-1")
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
	if err := incrementStudentInventoryItem(ctx, app.db, 123, cookieInventoryKey, 2); err != nil {
		t.Fatalf("seed cookie inventory: %v", err)
	}

	profile, err := app.feedStudentPet(ctx, 123)
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

	_, err := app.feedStudentPet(context.Background(), 123)
	if err != errNoCookies {
		t.Fatalf("feed pet error = %v, want errNoCookies", err)
	}
}

func TestPurchaseStudentShopItemBuysCookie(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := app.db.Exec(ctx, "insert into student_wallet (app_user_id, star_balance) values (123, 125)"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}

	response, err := app.purchaseStudentShopItem(ctx, 123, shopPurchaseRequest{
		ShopID:   "cookie-keeper-shop",
		ItemKey:  cookieInventoryKey,
		Quantity: 1,
	})
	if err != nil {
		t.Fatalf("purchase error = %v", err)
	}
	if response.StarBalance != 75 {
		t.Fatalf("star balance = %d, want 75", response.StarBalance)
	}
	if len(response.Inventory.Items) != 1 || response.Inventory.Items[0].Key != cookieInventoryKey || response.Inventory.Items[0].Quantity != 1 {
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

	_, err := app.purchaseStudentShopItem(ctx, 123, shopPurchaseRequest{
		ShopID:   "cookie-keeper-shop",
		ItemKey:  cookieInventoryKey,
		Quantity: 1,
	})
	if err != errInsufficientStars {
		t.Fatalf("purchase error = %v, want errInsufficientStars", err)
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

	invalidRequests := []shopPurchaseRequest{
		{ShopID: "other-shop", ItemKey: cookieInventoryKey, Quantity: 1},
		{ShopID: "cookie-keeper-shop", ItemKey: "star", Quantity: 1},
		{ShopID: "cookie-keeper-shop", ItemKey: cookieInventoryKey, Quantity: 0},
	}
	for _, request := range invalidRequests {
		if _, err := app.purchaseStudentShopItem(context.Background(), 123, request); err == nil {
			t.Fatalf("purchase %#v succeeded, want error", request)
		}
	}
}

func TestCraftStudentRecipeCreatesStoneBlock(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if err := incrementStudentInventoryItem(ctx, app.db, 123, "rock", 5); err != nil {
		t.Fatalf("seed rock inventory: %v", err)
	}

	recipes, err := app.loadCraftingRecipes(ctx, 123)
	if err != nil {
		t.Fatalf("load recipes: %v", err)
	}
	if len(recipes.Recipes) != 1 || recipes.Recipes[0].Key != "stone_block" || !recipes.Recipes[0].CanCraft {
		t.Fatalf("recipes = %#v, want craftable stone_block", recipes)
	}

	response, err := app.craftStudentRecipe(ctx, 123, craftRecipeRequest{RecipeKey: "stone_block"})
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
	if err := incrementStudentInventoryItem(ctx, app.db, 123, "rock", 3); err != nil {
		t.Fatalf("seed rock inventory: %v", err)
	}

	_, err := app.craftStudentRecipe(ctx, 123, craftRecipeRequest{RecipeKey: "stone_block"})
	if err != errInsufficientIngredient {
		t.Fatalf("craft recipe error = %v, want errInsufficientIngredient", err)
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

	_, err := app.craftStudentRecipe(context.Background(), 123, craftRecipeRequest{RecipeKey: "missing"})
	if err != errUnknownRecipe {
		t.Fatalf("craft recipe error = %v, want errUnknownRecipe", err)
	}
}

func TestPlaceSunnyTownMapObjectConsumesStoneBlock(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if err := incrementStudentInventoryItem(ctx, app.db, 123, stoneBlockItemKey, 2); err != nil {
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
	if err := incrementStudentInventoryItem(ctx, app.db, 123, stoneBlockItemKey, 2); err != nil {
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
	if err := incrementStudentInventoryItem(ctx, app.db, 123, stoneBlockItemKey, 1); err != nil {
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
	if err := incrementStudentInventoryItem(ctx, app.db, 123, "sunny_hoodie", 1); err != nil {
		t.Fatalf("seed hoodie inventory: %v", err)
	}

	equipment, err := app.equipStudentItem(ctx, 123, equipmentChangeRequest{
		Slot:    equipmentSlotGear,
		ItemKey: "sunny_hoodie",
	})
	if err != nil {
		t.Fatalf("equip hoodie error = %v", err)
	}
	if equipment.Slots[0].Item == nil || equipment.Slots[0].Item.Key != "sunny_hoodie" {
		t.Fatalf("equipment = %#v, want hoodie in gear slot", equipment)
	}

	inventory, err := app.loadStudentInventory(ctx, 123)
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
	if err := incrementStudentInventoryItem(ctx, app.db, 123, cookieInventoryKey, 1); err != nil {
		t.Fatalf("seed cookie inventory: %v", err)
	}

	_, err := app.equipStudentItem(ctx, 123, equipmentChangeRequest{
		Slot:    equipmentSlotGear,
		ItemKey: cookieInventoryKey,
	})
	if err != errItemNotEquippable {
		t.Fatalf("equip cookie error = %v, want errItemNotEquippable", err)
	}

	_, err = app.equipStudentItem(ctx, 123, equipmentChangeRequest{
		Slot:    equipmentSlotAccessory,
		ItemKey: "star_cap",
	})
	if err != errItemNotOwned {
		t.Fatalf("equip unowned cap error = %v, want errItemNotOwned", err)
	}
}

func TestUnequipStudentItemClearsSlot(t *testing.T) {
	app, cleanup := testRewardApp(t)
	defer cleanup()

	ctx := context.Background()
	if err := incrementStudentInventoryItem(ctx, app.db, 123, "star_cap", 1); err != nil {
		t.Fatalf("seed cap inventory: %v", err)
	}
	if _, err := app.equipStudentItem(ctx, 123, equipmentChangeRequest{
		Slot:    equipmentSlotAccessory,
		ItemKey: "star_cap",
	}); err != nil {
		t.Fatalf("equip cap error = %v", err)
	}

	equipment, err := app.unequipStudentItem(ctx, 123, equipmentChangeRequest{Slot: equipmentSlotAccessory})
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
	if err := incrementStudentInventoryItem(ctx, app.db, 123, "pickaxe", 1); err != nil {
		t.Fatalf("seed pickaxe inventory: %v", err)
	}

	equipment, err := app.equipStudentItem(ctx, 123, equipmentChangeRequest{
		Slot:    equipmentSlotTool,
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
	if err := incrementStudentInventoryItem(ctx, app.db, 123, "pickaxe", 1); err != nil {
		t.Fatalf("seed pickaxe inventory: %v", err)
	}
	if err := incrementStudentInventoryItem(ctx, app.db, 123, "stone_block", 2); err != nil {
		t.Fatalf("seed stone block inventory: %v", err)
	}

	hotbar, err := app.loadStudentHotbar(ctx, 123)
	if err != nil {
		t.Fatalf("load hotbar: %v", err)
	}
	if len(hotbar.Slots) != 5 || hotbar.Slots[0].Item == nil || hotbar.Slots[0].Item.Key != "pickaxe" {
		t.Fatalf("hotbar = %#v, want pickaxe in slot 1", hotbar)
	}
	if hotbar.Slots[1].Item == nil || hotbar.Slots[1].Item.Key != "stone_block" {
		t.Fatalf("hotbar = %#v, want stone_block in slot 2", hotbar)
	}

	hotbar, err = app.setStudentHotbarSlot(ctx, 123, hotbarSlotRequest{Slot: 3, ItemKey: "stone_block"})
	if err != nil {
		t.Fatalf("set hotbar slot: %v", err)
	}
	if hotbar.Slots[2].Item == nil || hotbar.Slots[2].Item.Key != "stone_block" {
		t.Fatalf("hotbar = %#v, want stone_block in slot 3", hotbar)
	}

	hotbar, err = app.setStudentHotbarSlot(ctx, 123, hotbarSlotRequest{Slot: 3})
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

	_, err := app.setStudentHotbarSlot(context.Background(), 123, hotbarSlotRequest{Slot: 1, ItemKey: "stone_block"})
	if !errors.Is(err, errHotbarItemNotOwned) {
		t.Fatalf("set hotbar error = %v, want errHotbarItemNotOwned", err)
	}
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
