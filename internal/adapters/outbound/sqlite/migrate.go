package sqlite

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migrate applies every unapplied migration in a transaction.
func (db *DB) Migrate(ctx context.Context) error {
	if _, err := db.sql.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at INTEGER NOT NULL
		)`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return err
	}
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version := strings.TrimSuffix(entry.Name(), ".sql")
		if _, exists := applied[version]; exists {
			continue
		}
		body, readErr := migrationFiles.ReadFile("migrations/" + entry.Name())
		if readErr != nil {
			return fmt.Errorf("read migration %s: %w", version, readErr)
		}
		if err := applyMigration(ctx, db, version, string(body)); err != nil {
			return err
		}
	}
	return nil
}

func appliedVersions(ctx context.Context, db *DB) (map[string]struct{}, error) {
	rows, err := db.sql.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("query migration versions: %w", err)
	}
	defer rows.Close()
	result := make(map[string]struct{})
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan migration version: %w", err)
		}
		result[version] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migration versions: %w", err)
	}
	return result, nil
}

func applyMigration(ctx context.Context, db *DB, version, body string) error {
	transaction, err := db.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", version, err)
	}
	if _, err := transaction.ExecContext(ctx, body); err != nil {
		_ = transaction.Rollback()
		return fmt.Errorf("execute migration %s: %w", version, err)
	}
	if _, err := transaction.ExecContext(ctx,
		"INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)",
		version, time.Now().UTC().UnixMilli(),
	); err != nil {
		_ = transaction.Rollback()
		return fmt.Errorf("record migration %s: %w", version, err)
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", version, err)
	}
	return nil
}
