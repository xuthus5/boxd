// Package sqlite is the only package allowed to import the SQLite driver.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const (
	busyTimeoutMS = 5000
	maxOpenConns  = 4
	maxIdleConns  = 4
)

// DB is the infrastructure connection wrapper. Business packages receive
// repository ports instead of this type.
type DB struct {
	sql  *sql.DB
	path string
}

// Open creates the data directory, opens SQLite with required pragmas, and
// verifies the connection.
func Open(ctx context.Context, path string) (*DB, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	parameters := url.Values{}
	parameters.Add("_pragma", "journal_mode(WAL)")
	parameters.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", busyTimeoutMS))
	parameters.Add("_pragma", "foreign_keys(ON)")
	parameters.Add("_pragma", "synchronous(NORMAL)")
	dsn := "file:" + filepath.ToSlash(absolute) + "?" + parameters.Encode()

	connection, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	connection.SetMaxOpenConns(maxOpenConns)
	connection.SetMaxIdleConns(maxIdleConns)
	connection.SetConnMaxLifetime(0)

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := connection.PingContext(pingCtx); err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := os.Chmod(absolute, 0o600); err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("secure database file: %w", err)
	}
	return &DB{sql: connection, path: absolute}, nil
}

// Close releases the connection pool.
func (db *DB) Close() error {
	if db == nil || db.sql == nil {
		return nil
	}
	return db.sql.Close()
}

// Path returns the absolute database path.
func (db *DB) Path() string {
	return db.path
}

// Ping checks that the database is reachable.
func (db *DB) Ping(ctx context.Context) error {
	return db.sql.PingContext(ctx)
}

// Checkpoint flushes the WAL before shutdown.
func (db *DB) Checkpoint(ctx context.Context) error {
	_, err := db.sql.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")
	return err
}

// Configs returns the configuration repository.
func (db *DB) Configs() *ConfigRepository {
	return &ConfigRepository{db: db.sql}
}

// Settings returns the settings repository.
func (db *DB) Settings() *SettingsRepository {
	return &SettingsRepository{db: db.sql}
}

// Profiles returns the managed desired-state repository.
func (db *DB) Profiles() *ProfileRepository {
	return &ProfileRepository{db: db.sql}
}

// RuleSets returns the rule-set status repository.
func (db *DB) RuleSets() *RuleSetRepository {
	return &RuleSetRepository{db: db.sql}
}

// TestResults returns the node test-result repository.
func (db *DB) TestResults() *TestResultRepository {
	return &TestResultRepository{db: db.sql}
}

// Security returns the local administrator credential repository.
func (db *DB) Security() *SecurityRepository {
	return &SecurityRepository{db: db.sql}
}

// Audit returns the audit event repository.
func (db *DB) Audit() *AuditRepository {
	return &AuditRepository{db: db.sql}
}
