package schema

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestLoadMigrationsSortsSQLFiles(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"0002_second.sql": "select 2;",
		"notes.txt":       "ignored",
		"0001_first.sql":  "select 1;",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	migrations, err := LoadMigrations(dir)
	if err != nil {
		t.Fatalf("LoadMigrations() error = %v", err)
	}
	if len(migrations) != 2 {
		t.Fatalf("migration count = %d, want 2", len(migrations))
	}
	if migrations[0].Version != "0001_first" || migrations[1].Version != "0002_second" {
		t.Fatalf("versions = %#v, want sorted SQL migrations", migrations)
	}
}

func TestRunMigrationsIntegration(t *testing.T) {
	databaseURL := os.Getenv("HQ_SCHEMA_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("HQ_SCHEMA_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	defer db.Close()

	if err := RunMigrations(ctx, db, "../../../deploy/postgres/migrations"); err != nil {
		t.Fatalf("RunMigrations() first run error = %v", err)
	}
	if err := RunMigrations(ctx, db, "../../../deploy/postgres/migrations"); err != nil {
		t.Fatalf("RunMigrations() second run error = %v", err)
	}

	var migrationCount int
	if err := db.QueryRow(ctx, "select count(*) from schema_migration").Scan(&migrationCount); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if migrationCount != 5 {
		t.Fatalf("migration count = %d, want 5", migrationCount)
	}

	var tableCount int
	if err := db.QueryRow(
		ctx,
		`
			select count(*)
			from information_schema.tables
			where table_schema = 'public'
				and table_name in (
					'assignment',
					'assignment_attempt',
					'assignment_ai_grade',
					'app_user',
					'app_user_role',
					'pet_state',
					'student_wallet',
					'student_star_ledger',
					'inventory_item_type',
					'student_inventory_item',
					'student_inventory_ledger',
					'student_equipped_item',
					'student_hotbar_slot',
					'student_sunny_town_position',
					'sunny_town_map_object'
				)
		`,
	).Scan(&tableCount); err != nil {
		t.Fatalf("count tables: %v", err)
	}
	if tableCount != 15 {
		t.Fatalf("table count = %d, want 15", tableCount)
	}
}
