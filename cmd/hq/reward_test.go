package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now()
		)`,
		`insert into inventory_item_type (key, name, description)
			values ('cookie', 'Cookie', 'A treat for your pet.')`,
		`create table student_inventory_item (
			app_user_id bigint not null references app_user(id) on delete cascade,
			item_type_id bigint not null references inventory_item_type(id) on delete restrict,
			quantity integer not null default 0,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			primary key (app_user_id, item_type_id),
			constraint student_inventory_item_quantity_nonnegative check (quantity >= 0)
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
