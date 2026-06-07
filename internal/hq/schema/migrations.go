package schema

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Migration struct {
	Version string
	SQL     string
}

func RunMigrations(ctx context.Context, db *pgxpool.Pool, dir string) error {
	migrations, err := LoadMigrations(dir)
	if err != nil {
		return err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(
		ctx,
		`
			create table if not exists schema_migration (
				version text primary key,
				applied_at timestamptz not null default now()
			)
		`,
	); err != nil {
		return err
	}

	for _, migration := range migrations {
		var applied bool
		if err := tx.QueryRow(
			ctx,
			`select exists (select 1 from schema_migration where version = $1)`,
			migration.Version,
		).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}

		if _, err := tx.Exec(ctx, migration.SQL); err != nil {
			return fmt.Errorf("apply migration %s: %w", migration.Version, err)
		}
		if _, err := tx.Exec(
			ctx,
			`insert into schema_migration (version) values ($1)`,
			migration.Version,
		); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func LoadMigrations(dir string) ([]Migration, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, errors.New("migration directory is required")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	migrations := []Migration{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		sql, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		migrations = append(migrations, Migration{
			Version: strings.TrimSuffix(entry.Name(), ".sql"),
			SQL:     string(sql),
		})
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	if len(migrations) == 0 {
		return nil, fmt.Errorf("no SQL migrations found in %s", dir)
	}

	return migrations, nil
}
