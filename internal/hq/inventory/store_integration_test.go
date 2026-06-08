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
