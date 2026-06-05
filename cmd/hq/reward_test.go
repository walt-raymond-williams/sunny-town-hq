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

	return &app{db: db}, func() {
		db.Close()
		_, _ = adminPool.Exec(ctx, "drop schema "+schema+" cascade")
		adminPool.Close()
	}
}
