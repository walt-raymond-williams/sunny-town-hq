package inventory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPurchaseStudentShopItemBuysCookie(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := db.Exec(ctx, "insert into student_wallet (app_user_id, star_balance) values (123, 125)"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}
	if _, _, err := CommitShopStockDelta(ctx, db, ShopStockEventRequest{
		EventID: "seed-stock",
		Source:  "test",
		ShopID:  CookieKeeperShopID,
		ItemKey: CookieKey,
		Delta:   2,
	}); err != nil {
		t.Fatalf("seed shop stock: %v", err)
	}

	response, err := PurchaseStudentShopItem(ctx, db, 123, ShopPurchaseRequest{
		ShopID:   CookieKeeperShopID,
		ItemKey:  CookieKey,
		Quantity: 1,
	})
	if err != nil {
		t.Fatalf("purchase error = %v", err)
	}
	if response.StarBalance != 75 {
		t.Fatalf("star balance = %d, want 75", response.StarBalance)
	}
	if len(response.Inventory.Items) != 1 || response.Inventory.Items[0].Key != CookieKey || response.Inventory.Items[0].Quantity != 1 {
		t.Fatalf("inventory = %#v, want 1 cookie", response.Inventory)
	}

	var walletBalance int
	var cookieQuantity int
	var shopStockQuantity int
	var ledgerDelta int
	if err := db.QueryRow(ctx, "select star_balance from student_wallet where app_user_id = 123").Scan(&walletBalance); err != nil {
		t.Fatalf("load wallet: %v", err)
	}
	if err := db.QueryRow(
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
	if err := db.QueryRow(ctx, "select delta from student_star_ledger where source = 'shop_purchase'").Scan(&ledgerDelta); err != nil {
		t.Fatalf("load shop ledger: %v", err)
	}
	if err := db.QueryRow(
		ctx,
		`
			select ssi.quantity
			from shop_stock_item ssi
			join inventory_item_type iit on iit.id = ssi.item_type_id
			where ssi.shop_id = $1 and iit.key = $2
		`,
		CookieKeeperShopID,
		CookieKey,
	).Scan(&shopStockQuantity); err != nil {
		t.Fatalf("load shop stock quantity: %v", err)
	}
	if walletBalance != 75 || cookieQuantity != 1 || shopStockQuantity != 1 || ledgerDelta != -50 {
		t.Fatalf("wallet=%d cookies=%d shopStock=%d ledgerDelta=%d, want 75, 1, 1, -50", walletBalance, cookieQuantity, shopStockQuantity, ledgerDelta)
	}
}

func TestPurchaseStudentShopItemRequiresStars(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := db.Exec(ctx, "insert into student_wallet (app_user_id, star_balance) values (123, 49)"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}

	_, err := PurchaseStudentShopItem(ctx, db, 123, ShopPurchaseRequest{
		ShopID:   CookieKeeperShopID,
		ItemKey:  CookieKey,
		Quantity: 1,
	})
	if err != ErrInsufficientStars {
		t.Fatalf("purchase error = %v, want ErrInsufficientStars", err)
	}

	var walletBalance int
	var inventoryRows int
	if err := db.QueryRow(ctx, "select star_balance from student_wallet where app_user_id = 123").Scan(&walletBalance); err != nil {
		t.Fatalf("load wallet: %v", err)
	}
	if err := db.QueryRow(ctx, "select count(*) from student_inventory_item").Scan(&inventoryRows); err != nil {
		t.Fatalf("count inventory rows: %v", err)
	}
	if walletBalance != 49 || inventoryRows != 0 {
		t.Fatalf("wallet=%d inventoryRows=%d, want 49 and 0", walletBalance, inventoryRows)
	}
}

func TestPurchaseStudentShopItemRequiresStock(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := db.Exec(ctx, "insert into student_wallet (app_user_id, star_balance) values (123, 125)"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}

	_, err := PurchaseStudentShopItem(ctx, db, 123, ShopPurchaseRequest{
		ShopID:   CookieKeeperShopID,
		ItemKey:  CookieKey,
		Quantity: 1,
	})
	if err != ErrInsufficientShopStock {
		t.Fatalf("purchase error = %v, want ErrInsufficientShopStock", err)
	}

	var walletBalance int
	var ledgerRows int
	if err := db.QueryRow(ctx, "select star_balance from student_wallet where app_user_id = 123").Scan(&walletBalance); err != nil {
		t.Fatalf("load wallet: %v", err)
	}
	if err := db.QueryRow(ctx, "select count(*) from student_star_ledger").Scan(&ledgerRows); err != nil {
		t.Fatalf("count star ledger rows: %v", err)
	}
	if walletBalance != 125 || ledgerRows != 0 {
		t.Fatalf("wallet=%d ledgerRows=%d, want 125 and 0", walletBalance, ledgerRows)
	}
}

func TestLoadShopStockReturnsCookieStock(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	stock, err := LoadShopStock(ctx, db, CookieKeeperShopID)
	if err != nil {
		t.Fatalf("load empty shop stock: %v", err)
	}
	if stock.ShopID != CookieKeeperShopID ||
		len(stock.Items) != 1 ||
		stock.Items[0].ItemKey != CookieKey ||
		stock.Items[0].Quantity != 0 ||
		stock.Items[0].Capacity != CookieKeeperCookieStockCapacity {
		t.Fatalf("empty shop stock = %#v, want cookie quantity 0 capacity %d", stock, CookieKeeperCookieStockCapacity)
	}

	if _, _, err := CommitShopStockDelta(ctx, db, ShopStockEventRequest{
		EventID: "stock-event",
		Source:  "test",
		ShopID:  CookieKeeperShopID,
		ItemKey: CookieKey,
		Delta:   3,
	}); err != nil {
		t.Fatalf("seed shop stock: %v", err)
	}

	stock, err = LoadShopStock(ctx, db, CookieKeeperShopID)
	if err != nil {
		t.Fatalf("load shop stock: %v", err)
	}
	if len(stock.Items) != 1 ||
		stock.Items[0].ItemKey != CookieKey ||
		stock.Items[0].Quantity != 3 ||
		stock.Items[0].Capacity != CookieKeeperCookieStockCapacity {
		t.Fatalf("shop stock = %#v, want cookie quantity 3 capacity %d", stock, CookieKeeperCookieStockCapacity)
	}
}

func TestPurchaseStudentShopItemRejectsInvalidPurchase(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	invalidRequests := []ShopPurchaseRequest{
		{ShopID: "other-shop", ItemKey: CookieKey, Quantity: 1},
		{ShopID: "cookie-keeper-shop", ItemKey: "star", Quantity: 1},
		{ShopID: CookieKeeperShopID, ItemKey: CookieKey, Quantity: 0},
	}
	for _, request := range invalidRequests {
		if _, err := PurchaseStudentShopItem(context.Background(), db, 123, request); err == nil {
			t.Fatalf("purchase %#v succeeded, want error", request)
		}
	}
}

func TestCommitShopStockDeltaIsIdempotent(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	inserted, quantity, err := CommitShopStockDelta(ctx, db, ShopStockEventRequest{
		EventID: "event-1",
		Source:  "npc_job_production",
		ShopID:  CookieKeeperShopID,
		ItemKey: CookieKey,
		Delta:   2,
	})
	if err != nil {
		t.Fatalf("commit stock delta: %v", err)
	}
	if !inserted || quantity != 2 {
		t.Fatalf("inserted=%v quantity=%d, want true and 2", inserted, quantity)
	}

	inserted, quantity, err = CommitShopStockDelta(ctx, db, ShopStockEventRequest{
		EventID: "event-1",
		Source:  "npc_job_production",
		ShopID:  CookieKeeperShopID,
		ItemKey: CookieKey,
		Delta:   2,
	})
	if err != nil {
		t.Fatalf("commit duplicate stock delta: %v", err)
	}
	if inserted || quantity != 2 {
		t.Fatalf("inserted=%v quantity=%d, want false and 2", inserted, quantity)
	}
}

func TestCommitShopStockDeltaClampsAtCapacity(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	inserted, quantity, err := CommitShopStockDelta(ctx, db, ShopStockEventRequest{
		EventID: "near-capacity",
		Source:  "npc_job_production",
		ShopID:  CookieKeeperShopID,
		ItemKey: CookieKey,
		Delta:   CookieKeeperCookieStockCapacity - 1,
	})
	if err != nil {
		t.Fatalf("seed near capacity: %v", err)
	}
	if !inserted || quantity != CookieKeeperCookieStockCapacity-1 {
		t.Fatalf("inserted=%v quantity=%d, want true and %d", inserted, quantity, CookieKeeperCookieStockCapacity-1)
	}

	inserted, quantity, err = CommitShopStockDelta(ctx, db, ShopStockEventRequest{
		EventID: "over-capacity",
		Source:  "npc_job_production",
		ShopID:  CookieKeeperShopID,
		ItemKey: CookieKey,
		Delta:   3,
	})
	if err != nil {
		t.Fatalf("commit over capacity: %v", err)
	}
	if !inserted || quantity != CookieKeeperCookieStockCapacity {
		t.Fatalf("inserted=%v quantity=%d, want true and %d", inserted, quantity, CookieKeeperCookieStockCapacity)
	}

	inserted, quantity, err = CommitShopStockDelta(ctx, db, ShopStockEventRequest{
		EventID: "over-capacity",
		Source:  "npc_job_production",
		ShopID:  CookieKeeperShopID,
		ItemKey: CookieKey,
		Delta:   3,
	})
	if err != nil {
		t.Fatalf("commit duplicate over capacity: %v", err)
	}
	if inserted || quantity != CookieKeeperCookieStockCapacity {
		t.Fatalf("inserted=%v quantity=%d, want false and %d", inserted, quantity, CookieKeeperCookieStockCapacity)
	}

	var ledgerRows int
	if err := db.QueryRow(ctx, "select count(*) from shop_stock_ledger").Scan(&ledgerRows); err != nil {
		t.Fatalf("count stock ledger rows: %v", err)
	}
	if ledgerRows != 2 {
		t.Fatalf("stock ledger rows = %d, want 2", ledgerRows)
	}

	if _, err := db.Exec(ctx, "insert into student_wallet (app_user_id, star_balance) values (123, 50)"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}
	if _, err := PurchaseStudentShopItem(ctx, db, 123, ShopPurchaseRequest{
		ShopID:   CookieKeeperShopID,
		ItemKey:  CookieKey,
		Quantity: 1,
	}); err != nil {
		t.Fatalf("purchase after full stock: %v", err)
	}

	stock, err := LoadShopStock(ctx, db, CookieKeeperShopID)
	if err != nil {
		t.Fatalf("load shop stock after purchase: %v", err)
	}
	if len(stock.Items) != 1 || stock.Items[0].Quantity != CookieKeeperCookieStockCapacity-1 {
		t.Fatalf("shop stock after purchase = %#v, want %d", stock, CookieKeeperCookieStockCapacity-1)
	}
}

func TestShopInputStoragePersistsAndLoadsItems(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	storage, err := LoadShopInputStorage(ctx, db, CookieKeeperShopID)
	if err != nil {
		t.Fatalf("load empty input storage: %v", err)
	}
	if storage.ShopID != CookieKeeperShopID || storage.Capacity != CookieKeeperInputStorageCapacity || len(storage.Items) != 0 {
		t.Fatalf("empty input storage = %#v, want shop, capacity, and no items", storage)
	}

	accepted, total, err := IncrementShopInputStorageItem(ctx, db, CookieKeeperShopID, "rock", 12)
	if err != nil {
		t.Fatalf("increment input storage: %v", err)
	}
	if !accepted || total != 12 {
		t.Fatalf("accepted=%v total=%d, want true and 12", accepted, total)
	}

	storage, err = LoadShopInputStorage(ctx, db, CookieKeeperShopID)
	if err != nil {
		t.Fatalf("load input storage: %v", err)
	}
	if len(storage.Items) != 1 || storage.Items[0].ItemKey != "rock" || storage.Items[0].Quantity != 12 {
		t.Fatalf("input storage = %#v, want 12 rocks", storage)
	}
}

func TestShopInputStorageCapacityRejectsOverflow(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	accepted, total, err := IncrementShopInputStorageItem(ctx, db, CookieKeeperShopID, "rock", CookieKeeperInputStorageCapacity)
	if err != nil {
		t.Fatalf("fill input storage: %v", err)
	}
	if !accepted || total != CookieKeeperInputStorageCapacity {
		t.Fatalf("accepted=%v total=%d, want true and capacity", accepted, total)
	}

	accepted, total, err = IncrementShopInputStorageItem(ctx, db, CookieKeeperShopID, "crystal", 1)
	if err != nil {
		t.Fatalf("overflow input storage: %v", err)
	}
	if accepted || total != CookieKeeperInputStorageCapacity {
		t.Fatalf("accepted=%v total=%d, want false and capacity", accepted, total)
	}

	storage, err := LoadShopInputStorage(ctx, db, CookieKeeperShopID)
	if err != nil {
		t.Fatalf("load input storage after overflow: %v", err)
	}
	if len(storage.Items) != 1 || storage.Items[0].ItemKey != "rock" || storage.Items[0].Quantity != CookieKeeperInputStorageCapacity {
		t.Fatalf("input storage after overflow = %#v, want only capped rocks", storage)
	}
}

func TestConsumeShopInputStorageItem(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if accepted, _, err := IncrementShopInputStorageItem(ctx, db, CookieKeeperShopID, "rock", 5); err != nil || !accepted {
		t.Fatalf("seed input storage accepted=%v err=%v, want accepted", accepted, err)
	}

	consumed, err := ConsumeShopInputStorageItem(ctx, db, CookieKeeperShopID, "rock", 4)
	if err != nil {
		t.Fatalf("consume input storage: %v", err)
	}
	if !consumed {
		t.Fatalf("consumed = false, want true")
	}

	consumed, err = ConsumeShopInputStorageItem(ctx, db, CookieKeeperShopID, "rock", 2)
	if err != nil {
		t.Fatalf("consume missing input storage: %v", err)
	}
	if consumed {
		t.Fatalf("consumed = true, want false for insufficient input storage")
	}

	storage, err := LoadShopInputStorage(ctx, db, CookieKeeperShopID)
	if err != nil {
		t.Fatalf("load input storage after consume: %v", err)
	}
	if len(storage.Items) != 1 || storage.Items[0].Quantity != 1 {
		t.Fatalf("input storage after consume = %#v, want 1 rock", storage)
	}
}

func TestStudentInventoryResponsesExposeItemMetadata(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := IncrementStudentItem(ctx, db, 123, "sunny_hoodie", 1); err != nil {
		t.Fatalf("seed hoodie inventory: %v", err)
	}
	if err := IncrementStudentItem(ctx, db, 123, "pickaxe", 1); err != nil {
		t.Fatalf("seed pickaxe inventory: %v", err)
	}
	if err := IncrementStudentItem(ctx, db, 123, "rock", 5); err != nil {
		t.Fatalf("seed rock inventory: %v", err)
	}
	if err := IncrementStudentItem(ctx, db, 123, "stone_block", 2); err != nil {
		t.Fatalf("seed stone block inventory: %v", err)
	}
	if _, err := EquipStudentItem(ctx, db, 123, EquipmentChangeRequest{
		Slot:    EquipmentSlotGear,
		ItemKey: "sunny_hoodie",
	}); err != nil {
		t.Fatalf("equip hoodie: %v", err)
	}

	inventory, err := LoadStudent(ctx, db, 123)
	if err != nil {
		t.Fatalf("load inventory: %v", err)
	}
	hoodie := inventoryItemByKey(t, inventory.Items, "sunny_hoodie")
	assertItemMetadata(t, hoodie.IconKey, hoodie.MaxStack, hoodie.Category, "sunny_hoodie", 1, "gear")
	if hoodie.VisualKey != "sunny_hoodie_visual" {
		t.Fatalf("hoodie visual key = %q, want separate render key", hoodie.VisualKey)
	}

	hotbar, err := LoadStudentHotbar(ctx, db, 123)
	if err != nil {
		t.Fatalf("load hotbar: %v", err)
	}
	if hotbar.Slots[0].Item == nil {
		t.Fatalf("hotbar slot 1 item = nil, want pickaxe")
	}
	assertItemMetadata(t, hotbar.Slots[0].Item.IconKey, hotbar.Slots[0].Item.MaxStack, hotbar.Slots[0].Item.Category, "pickaxe", 1, "tool")
	if hotbar.Slots[1].Item == nil {
		t.Fatalf("hotbar slot 2 item = nil, want stone block")
	}
	assertItemMetadata(t, hotbar.Slots[1].Item.IconKey, hotbar.Slots[1].Item.MaxStack, hotbar.Slots[1].Item.Category, "stone_block", 64, "building")

	equipment, err := LoadStudentEquipment(ctx, db, 123)
	if err != nil {
		t.Fatalf("load equipment: %v", err)
	}
	if equipment.Slots[0].Item == nil {
		t.Fatalf("equipment gear item = nil, want hoodie")
	}
	assertItemMetadata(t, equipment.Slots[0].Item.IconKey, equipment.Slots[0].Item.MaxStack, equipment.Slots[0].Item.Category, "sunny_hoodie", 1, "gear")
	if equipment.Slots[0].Item.VisualKey != "sunny_hoodie_visual" {
		t.Fatalf("equipment visual key = %q, want separate render key", equipment.Slots[0].Item.VisualKey)
	}

	recipes, err := LoadCraftingRecipes(ctx, db, 123)
	if err != nil {
		t.Fatalf("load crafting recipes: %v", err)
	}
	if len(recipes.Recipes) != 1 {
		t.Fatalf("recipes = %#v, want one student recipe", recipes)
	}
	recipe := recipes.Recipes[0]
	assertItemMetadata(t, recipe.OutputIconKey, recipe.OutputMaxStack, recipe.OutputCategory, "stone_block", 64, "building")
	if len(recipe.Ingredients) != 1 {
		t.Fatalf("ingredients = %#v, want one rock ingredient", recipe.Ingredients)
	}
	assertItemMetadata(t, recipe.Ingredients[0].IconKey, recipe.Ingredients[0].MaxStack, recipe.Ingredients[0].Category, "rock", 64, "resource")
}

func TestStudentInventorySlotsLoadEmptyAndOccupiedSlots(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := IncrementStudentItem(ctx, db, 123, "rock", 5); err != nil {
		t.Fatalf("seed rock inventory: %v", err)
	}
	if err := IncrementStudentItem(ctx, db, 123, "pickaxe", 1); err != nil {
		t.Fatalf("seed pickaxe inventory: %v", err)
	}

	response, err := LoadStudentSlots(ctx, db, 123)
	if err != nil {
		t.Fatalf("load slots: %v", err)
	}
	if response.SlotCount != StudentInventorySlotCount || len(response.Slots) != StudentInventorySlotCount {
		t.Fatalf("slots count = %d len=%d, want %d", response.SlotCount, len(response.Slots), StudentInventorySlotCount)
	}
	if response.Slots[0].SlotIndex != 0 || response.Slots[0].Item == nil || response.Slots[0].Item.Key != "rock" || response.Slots[0].Item.Quantity != 5 {
		t.Fatalf("slot 0 = %#v, want 5 rocks", response.Slots[0])
	}
	if response.Slots[1].SlotIndex != 1 || response.Slots[1].Item == nil || response.Slots[1].Item.Key != "pickaxe" || response.Slots[1].Item.Quantity != 1 {
		t.Fatalf("slot 1 = %#v, want pickaxe", response.Slots[1])
	}
	if response.Slots[2].Item != nil {
		t.Fatalf("slot 2 = %#v, want empty slot", response.Slots[2])
	}
	if len(response.Items) != 2 {
		t.Fatalf("aggregate items = %#v, want two entries", response.Items)
	}
}

func TestIncrementStudentItemSplitsStacksAndMaintainsAggregate(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := IncrementStudentItem(ctx, db, 123, "rock", 70); err != nil {
		t.Fatalf("increment rocks: %v", err)
	}

	slots, err := LoadStudentSlots(ctx, db, 123)
	if err != nil {
		t.Fatalf("load slots: %v", err)
	}
	if slots.Slots[0].Item == nil || slots.Slots[0].Item.Key != "rock" || slots.Slots[0].Item.Quantity != 64 {
		t.Fatalf("slot 0 = %#v, want 64 rocks", slots.Slots[0])
	}
	if slots.Slots[1].Item == nil || slots.Slots[1].Item.Key != "rock" || slots.Slots[1].Item.Quantity != 6 {
		t.Fatalf("slot 1 = %#v, want 6 rocks", slots.Slots[1])
	}

	var aggregate int
	if err := db.QueryRow(
		ctx,
		`
			select sii.quantity
			from student_inventory_item sii
			join inventory_item_type iit on iit.id = sii.item_type_id
			where sii.app_user_id = 123 and iit.key = 'rock'
		`,
	).Scan(&aggregate); err != nil {
		t.Fatalf("load aggregate: %v", err)
	}
	if aggregate != 70 {
		t.Fatalf("aggregate = %d, want 70", aggregate)
	}
}

func TestConsumeStudentItemConsumesAcrossStacksAndMaintainsAggregate(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := IncrementStudentItem(ctx, db, 123, "rock", 70); err != nil {
		t.Fatalf("increment rocks: %v", err)
	}

	consumed, err := ConsumeStudentItem(ctx, db, 123, "rock", 68)
	if err != nil {
		t.Fatalf("consume rocks: %v", err)
	}
	if !consumed {
		t.Fatalf("consumed = false, want true")
	}

	slots, err := LoadStudentSlots(ctx, db, 123)
	if err != nil {
		t.Fatalf("load slots: %v", err)
	}
	if slots.Slots[0].Item == nil || slots.Slots[0].Item.Key != "rock" || slots.Slots[0].Item.Quantity != 2 {
		t.Fatalf("slot 0 = %#v, want 2 rocks after consuming from high slots first", slots.Slots[0])
	}
	if slots.Slots[1].Item != nil {
		t.Fatalf("slot 1 = %#v, want empty after consuming stack", slots.Slots[1])
	}

	var aggregate int
	if err := db.QueryRow(
		ctx,
		`
			select sii.quantity
			from student_inventory_item sii
			join inventory_item_type iit on iit.id = sii.item_type_id
			where sii.app_user_id = 123 and iit.key = 'rock'
		`,
	).Scan(&aggregate); err != nil {
		t.Fatalf("load aggregate: %v", err)
	}
	if aggregate != 2 {
		t.Fatalf("aggregate = %d, want 2", aggregate)
	}
}

func TestMoveStudentInventoryStackMovesIntoEmptySlot(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := IncrementStudentItem(ctx, db, 123, "rock", 5); err != nil {
		t.Fatalf("seed rock inventory: %v", err)
	}

	response, err := MoveStudentInventoryStack(ctx, db, 123, InventoryMoveRequest{
		Source:      InventorySlotDescriptor{Kind: inventoryStorageKindPlayer, SlotIndex: 0},
		Destination: InventorySlotDescriptor{Kind: inventoryStorageKindPlayer, SlotIndex: 5},
		Mode:        "move",
	})
	if err != nil {
		t.Fatalf("move stack: %v", err)
	}
	if response.Slots[0].Item != nil {
		t.Fatalf("slot 0 = %#v, want empty after move", response.Slots[0])
	}
	if response.Slots[5].Item == nil || response.Slots[5].Item.Key != "rock" || response.Slots[5].Item.Quantity != 5 {
		t.Fatalf("slot 5 = %#v, want 5 rocks", response.Slots[5])
	}
}

func TestMoveStudentInventoryStackSwapsOccupiedSlots(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := IncrementStudentItem(ctx, db, 123, "rock", 5); err != nil {
		t.Fatalf("seed rock inventory: %v", err)
	}
	if err := IncrementStudentItem(ctx, db, 123, "stone_block", 2); err != nil {
		t.Fatalf("seed stone block inventory: %v", err)
	}

	response, err := MoveStudentInventoryStack(ctx, db, 123, InventoryMoveRequest{
		Source:      InventorySlotDescriptor{Kind: inventoryStorageKindPlayer, SlotIndex: 0},
		Destination: InventorySlotDescriptor{Kind: inventoryStorageKindPlayer, SlotIndex: 1},
		Mode:        "swap",
	})
	if err != nil {
		t.Fatalf("swap stacks: %v", err)
	}
	if response.Slots[0].Item == nil || response.Slots[0].Item.Key != "stone_block" || response.Slots[0].Item.Quantity != 2 {
		t.Fatalf("slot 0 = %#v, want stone blocks", response.Slots[0])
	}
	if response.Slots[1].Item == nil || response.Slots[1].Item.Key != "rock" || response.Slots[1].Item.Quantity != 5 {
		t.Fatalf("slot 1 = %#v, want rocks", response.Slots[1])
	}
}

func TestMoveStudentInventoryStackMergesCompatibleStacks(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := IncrementStudentItem(ctx, db, 123, "rock", 70); err != nil {
		t.Fatalf("seed rock inventory: %v", err)
	}

	_, err := MoveStudentInventoryStack(ctx, db, 123, InventoryMoveRequest{
		Source:      InventorySlotDescriptor{Kind: inventoryStorageKindPlayer, SlotIndex: 1},
		Destination: InventorySlotDescriptor{Kind: inventoryStorageKindPlayer, SlotIndex: 0},
		Mode:        "merge",
	})
	if !errors.Is(err, ErrInventoryStackFull) {
		t.Fatalf("merge full stack error = %v, want ErrInventoryStackFull", err)
	}

	if _, err := db.Exec(ctx, "update student_inventory_slot set quantity = 60 where app_user_id = 123 and slot_index = 0"); err != nil {
		t.Fatalf("make destination partially full: %v", err)
	}
	if _, err := db.Exec(
		ctx,
		`
			update student_inventory_item sii
			set quantity = 66
			from inventory_item_type iit
			where sii.item_type_id = iit.id
				and sii.app_user_id = 123
				and iit.key = 'rock'
		`,
	); err != nil {
		t.Fatalf("keep aggregate consistent: %v", err)
	}
	response, err := MoveStudentInventoryStack(ctx, db, 123, InventoryMoveRequest{
		Source:      InventorySlotDescriptor{Kind: inventoryStorageKindPlayer, SlotIndex: 1},
		Destination: InventorySlotDescriptor{Kind: inventoryStorageKindPlayer, SlotIndex: 0},
		Mode:        "merge",
	})
	if err != nil {
		t.Fatalf("merge partial stacks: %v", err)
	}
	if response.Slots[0].Item == nil || response.Slots[0].Item.Quantity != 64 {
		t.Fatalf("slot 0 = %#v, want full merged stack", response.Slots[0])
	}
	if response.Slots[1].Item == nil || response.Slots[1].Item.Quantity != 2 {
		t.Fatalf("slot 1 = %#v, want two rocks remaining", response.Slots[1])
	}
}

func TestMoveStudentInventoryStackRejectsInvalidOperations(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := IncrementStudentItem(ctx, db, 123, "rock", 5); err != nil {
		t.Fatalf("seed rock inventory: %v", err)
	}
	if err := IncrementStudentItem(ctx, db, 123, "stone_block", 2); err != nil {
		t.Fatalf("seed stone block inventory: %v", err)
	}

	tests := []struct {
		name    string
		request InventoryMoveRequest
		wantErr error
	}{
		{
			name: "invalid slot",
			request: InventoryMoveRequest{
				Source:      InventorySlotDescriptor{Kind: inventoryStorageKindPlayer, SlotIndex: -1},
				Destination: InventorySlotDescriptor{Kind: inventoryStorageKindPlayer, SlotIndex: 0},
				Mode:        "move",
			},
			wantErr: ErrInvalidInventorySlot,
		},
		{
			name: "empty source",
			request: InventoryMoveRequest{
				Source:      InventorySlotDescriptor{Kind: inventoryStorageKindPlayer, SlotIndex: 5},
				Destination: InventorySlotDescriptor{Kind: inventoryStorageKindPlayer, SlotIndex: 6},
				Mode:        "move",
			},
			wantErr: ErrInventorySourceEmpty,
		},
		{
			name: "move to occupied destination",
			request: InventoryMoveRequest{
				Source:      InventorySlotDescriptor{Kind: inventoryStorageKindPlayer, SlotIndex: 0},
				Destination: InventorySlotDescriptor{Kind: inventoryStorageKindPlayer, SlotIndex: 1},
				Mode:        "move",
			},
			wantErr: ErrInventoryDestinationOccupied,
		},
		{
			name: "incompatible merge",
			request: InventoryMoveRequest{
				Source:      InventorySlotDescriptor{Kind: inventoryStorageKindPlayer, SlotIndex: 0},
				Destination: InventorySlotDescriptor{Kind: inventoryStorageKindPlayer, SlotIndex: 1},
				Mode:        "merge",
			},
			wantErr: ErrInventoryIncompatibleMerge,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := MoveStudentInventoryStack(ctx, db, 123, test.request)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("move error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestCraftStudentRecipeCreatesStoneBlock(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := IncrementStudentItem(ctx, db, 123, "rock", 5); err != nil {
		t.Fatalf("seed rock inventory: %v", err)
	}

	recipes, err := LoadCraftingRecipes(ctx, db, 123)
	if err != nil {
		t.Fatalf("load recipes: %v", err)
	}
	if len(recipes.Recipes) != 1 || recipes.Recipes[0].Key != "stone_block" || !recipes.Recipes[0].CanCraft {
		t.Fatalf("recipes = %#v, want craftable stone_block", recipes)
	}

	response, err := CraftStudentRecipe(ctx, db, 123, CraftRecipeRequest{RecipeKey: "stone_block"})
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
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := IncrementStudentItem(ctx, db, 123, "rock", 3); err != nil {
		t.Fatalf("seed rock inventory: %v", err)
	}

	_, err := CraftStudentRecipe(ctx, db, 123, CraftRecipeRequest{RecipeKey: "stone_block"})
	if err != ErrInsufficientIngredient {
		t.Fatalf("craft recipe error = %v, want ErrInsufficientIngredient", err)
	}

	var rockQuantity int
	var stoneBlockRows int
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
	if err := db.QueryRow(
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
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	_, err := CraftStudentRecipe(context.Background(), db, 123, CraftRecipeRequest{RecipeKey: "missing"})
	if err != ErrUnknownRecipe {
		t.Fatalf("craft recipe error = %v, want ErrUnknownRecipe", err)
	}
}

func TestEquipStudentItemRequiresOwnedEquippableItem(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := IncrementStudentItem(ctx, db, 123, "sunny_hoodie", 1); err != nil {
		t.Fatalf("seed hoodie inventory: %v", err)
	}

	equipment, err := EquipStudentItem(ctx, db, 123, EquipmentChangeRequest{
		Slot:    EquipmentSlotGear,
		ItemKey: "sunny_hoodie",
	})
	if err != nil {
		t.Fatalf("equip hoodie error = %v", err)
	}
	if equipment.Slots[0].Item == nil || equipment.Slots[0].Item.Key != "sunny_hoodie" {
		t.Fatalf("equipment = %#v, want hoodie in gear slot", equipment)
	}

	inventory, err := LoadStudent(ctx, db, 123)
	if err != nil {
		t.Fatalf("load inventory: %v", err)
	}
	if len(inventory.Items) != 1 || !inventory.Items[0].Equipped {
		t.Fatalf("inventory = %#v, want equipped hoodie", inventory)
	}
}

func TestEquipStudentItemRejectsCookieAndMissingOwnership(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := IncrementStudentItem(ctx, db, 123, CookieKey, 1); err != nil {
		t.Fatalf("seed cookie inventory: %v", err)
	}

	_, err := EquipStudentItem(ctx, db, 123, EquipmentChangeRequest{
		Slot:    EquipmentSlotGear,
		ItemKey: CookieKey,
	})
	if err != ErrItemNotEquippable {
		t.Fatalf("equip cookie error = %v, want ErrItemNotEquippable", err)
	}

	_, err = EquipStudentItem(ctx, db, 123, EquipmentChangeRequest{
		Slot:    EquipmentSlotAccessory,
		ItemKey: "star_cap",
	})
	if err != ErrItemNotOwned {
		t.Fatalf("equip unowned cap error = %v, want ErrItemNotOwned", err)
	}
}

func TestUnequipStudentItemClearsSlot(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := IncrementStudentItem(ctx, db, 123, "star_cap", 1); err != nil {
		t.Fatalf("seed cap inventory: %v", err)
	}
	if _, err := EquipStudentItem(ctx, db, 123, EquipmentChangeRequest{
		Slot:    EquipmentSlotAccessory,
		ItemKey: "star_cap",
	}); err != nil {
		t.Fatalf("equip cap error = %v", err)
	}

	equipment, err := UnequipStudentItem(ctx, db, 123, EquipmentChangeRequest{Slot: EquipmentSlotAccessory})
	if err != nil {
		t.Fatalf("unequip cap error = %v", err)
	}
	if equipment.Slots[1].Item != nil {
		t.Fatalf("equipment = %#v, want empty accessory slot", equipment)
	}
}

func TestEquipStudentItemSupportsToolSlot(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := IncrementStudentItem(ctx, db, 123, "pickaxe", 1); err != nil {
		t.Fatalf("seed pickaxe inventory: %v", err)
	}

	equipment, err := EquipStudentItem(ctx, db, 123, EquipmentChangeRequest{
		Slot:    EquipmentSlotTool,
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
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := IncrementStudentItem(ctx, db, 123, "pickaxe", 1); err != nil {
		t.Fatalf("seed pickaxe inventory: %v", err)
	}
	if err := IncrementStudentItem(ctx, db, 123, "stone_block", 2); err != nil {
		t.Fatalf("seed stone block inventory: %v", err)
	}

	hotbar, err := LoadStudentHotbar(ctx, db, 123)
	if err != nil {
		t.Fatalf("load hotbar: %v", err)
	}
	if len(hotbar.Slots) != 5 || hotbar.Slots[0].Item == nil || hotbar.Slots[0].Item.Key != "pickaxe" {
		t.Fatalf("hotbar = %#v, want pickaxe in slot 1", hotbar)
	}
	if hotbar.Slots[1].Item == nil || hotbar.Slots[1].Item.Key != "stone_block" {
		t.Fatalf("hotbar = %#v, want stone_block in slot 2", hotbar)
	}

	hotbar, err = SetStudentHotbarSlot(ctx, db, 123, HotbarSlotRequest{Slot: 3, ItemKey: "stone_block"})
	if err != nil {
		t.Fatalf("set hotbar slot: %v", err)
	}
	if hotbar.Slots[2].Item == nil || hotbar.Slots[2].Item.Key != "stone_block" {
		t.Fatalf("hotbar = %#v, want stone_block in slot 3", hotbar)
	}

	hotbar, err = SetStudentHotbarSlot(ctx, db, 123, HotbarSlotRequest{Slot: 3})
	if err != nil {
		t.Fatalf("clear hotbar slot: %v", err)
	}
	if hotbar.Slots[2].Item != nil {
		t.Fatalf("hotbar = %#v, want empty slot 3", hotbar)
	}
}

func TestStudentHotbarRejectsUnownedItem(t *testing.T) {
	db, cleanup := testInventoryDB(t)
	defer cleanup()

	_, err := SetStudentHotbarSlot(context.Background(), db, 123, HotbarSlotRequest{Slot: 1, ItemKey: "stone_block"})
	if !errors.Is(err, ErrHotbarItemNotOwned) {
		t.Fatalf("set hotbar error = %v, want ErrHotbarItemNotOwned", err)
	}
}

func inventoryItemByKey(t *testing.T, items []ItemResponse, key string) ItemResponse {
	t.Helper()
	for _, item := range items {
		if item.Key == key {
			return item
		}
	}
	t.Fatalf("inventory items = %#v, want item %q", items, key)
	return ItemResponse{}
}

func assertItemMetadata(t *testing.T, iconKey string, maxStack int, category string, wantIconKey string, wantMaxStack int, wantCategory string) {
	t.Helper()
	if iconKey != wantIconKey || maxStack != wantMaxStack || category != wantCategory {
		t.Fatalf(
			"metadata iconKey=%q maxStack=%d category=%q, want iconKey=%q maxStack=%d category=%q",
			iconKey,
			maxStack,
			category,
			wantIconKey,
			wantMaxStack,
			wantCategory,
		)
	}
}

func testInventoryDB(t *testing.T) (*pgxpool.Pool, func()) {
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

	schema := fmt.Sprintf("inventory_test_%d", time.Now().UnixNano())
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
			values ('cookie', 'Cookie', 'A treat for your pet.')`,
		`insert into inventory_item_type (key, name, description, equip_slot, visual_key)
			values
				('sunny_hoodie', 'Sunny Hoodie', 'A cozy hoodie for Sunny Town.', 'gear', 'sunny_hoodie_visual'),
				('star_cap', 'Star Cap', 'A bright cap for sunny adventures.', 'accessory', 'star_cap_visual'),
				('pickaxe', 'Pickaxe', 'A sturdy starter tool.', 'tool', 'pickaxe_visual')`,
		`insert into inventory_item_type (key, name, description)
			values
				('rock', 'Rock', 'A sturdy rock from Forest Crossing.'),
				('crystal', 'Crystal', 'A bright crystal from Forest Crossing.'),
				('stone_block', 'Stone Block', 'A solid block crafted from stone.')`,
		`update inventory_item_type
			set icon_key = key,
				max_stack = case when key in ('sunny_hoodie', 'star_cap', 'pickaxe') then 1 else 64 end,
				category = case
					when key = 'cookie' then 'consumable'
					when key in ('sunny_hoodie', 'star_cap') then 'gear'
					when key = 'pickaxe' then 'tool'
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
