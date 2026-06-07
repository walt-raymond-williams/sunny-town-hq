package pet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	hqinventory "hq/internal/hq/inventory"
	hqsunnytownbridge "hq/internal/hq/sunnytownbridge"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errTestNoCookies = errors.New("no cookies available")

func TestApplyGameResultCreditsPetStarsOnce(t *testing.T) {
	db, cleanup := testPetDB(t)
	defer cleanup()

	ctx := context.Background()
	first, err := testPetStore(db).ApplyGameResult(ctx, 123, 7, 9, "round-1")
	if err != nil {
		t.Fatalf("first game result error = %v", err)
	}
	if first.StarBalance != 9 || first.PetState.Happiness != 57 || first.PetState.Energy != 45 {
		t.Fatalf("first profile = %#v, want balance 9 happiness 57 energy 45", first)
	}

	second, err := testPetStore(db).ApplyGameResult(ctx, 123, 7, 9, "round-1")
	if err != nil {
		t.Fatalf("second game result error = %v", err)
	}
	if second.StarBalance != 9 || second.PetState.Happiness != 57 || second.PetState.Energy != 45 {
		t.Fatalf("second profile = %#v, want duplicate to keep balance and pet stats unchanged", second)
	}

	var ledgerRows int
	var source string
	if err := db.QueryRow(ctx, "select count(*) from student_star_ledger").Scan(&ledgerRows); err != nil {
		t.Fatalf("count ledger rows: %v", err)
	}
	if err := db.QueryRow(ctx, "select source from student_star_ledger where event_id = 'pet-falling-stars:123:round-1'").Scan(&source); err != nil {
		t.Fatalf("load pet star ledger source: %v", err)
	}
	if ledgerRows != 1 || source != "pet_falling_stars" {
		t.Fatalf("ledgerRows=%d source=%q, want 1 pet_falling_stars", ledgerRows, source)
	}
}

func TestFeedStudentPetConsumesCookieInventory(t *testing.T) {
	db, cleanup := testPetDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := hqinventory.IncrementStudentItem(ctx, db, 123, hqinventory.CookieKey, 2); err != nil {
		t.Fatalf("seed cookie inventory: %v", err)
	}

	profile, err := testPetStore(db).Feed(ctx, 123)
	if err != nil {
		t.Fatalf("feed pet error = %v", err)
	}
	if profile.Cookies != 1 || profile.PetState.Hunger != 60 {
		t.Fatalf("profile = %#v, want 1 cookie and hunger 60", profile)
	}
}

func TestFeedStudentPetRequiresCookieInventory(t *testing.T) {
	db, cleanup := testPetDB(t)
	defer cleanup()

	_, err := testPetStore(db).Feed(context.Background(), 123)
	if err != errTestNoCookies {
		t.Fatalf("feed pet error = %v, want errTestNoCookies", err)
	}
}

func TestPetProfileJSONUsesStudentAPIFieldNames(t *testing.T) {
	profile := Profile{
		ID:          123,
		DisplayName: "Student",
		Cookies:     2,
		StarBalance: 7,
		PetState: State{
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

func testPetStore(db *pgxpool.Pool) *Store {
	return &Store{
		DB:                    db,
		CookieInventoryKey:    hqinventory.CookieKey,
		NoCookiesError:        errTestNoCookies,
		ConsumeInventoryItem:  testConsumePetInventoryItem,
		CommitStudentStarOnce: testCommitPetStarReward,
	}
}

func testConsumePetInventoryItem(ctx context.Context, tx pgx.Tx, userID int64, itemKey string, quantity int) (bool, error) {
	return hqinventory.ConsumeStudentItem(ctx, tx, userID, itemKey, quantity)
}

func testCommitPetStarReward(ctx context.Context, tx pgx.Tx, request StarRewardRequest) (bool, int, error) {
	return hqsunnytownbridge.CommitStudentStarReward(ctx, tx, hqsunnytownbridge.StarRewardRequest{
		EventID:       request.EventID,
		AppUserID:     request.AppUserID,
		Source:        request.Source,
		Delta:         request.Delta,
		RoomID:        request.RoomID,
		MapID:         request.MapID,
		CollectibleID: request.CollectibleID,
	})
}

func testPetDB(t *testing.T) (*pgxpool.Pool, func()) {
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

	schema := fmt.Sprintf("pet_test_%d", time.Now().UnixNano())
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

	return db, func() {
		db.Close()
		_, _ = adminPool.Exec(ctx, "drop schema "+schema+" cascade")
		adminPool.Close()
	}
}
