package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xuthus5/boxd/control"
)

// BackupStore creates and verifies SQLite snapshots for the control plane.
type BackupStore struct {
	db   *sql.DB
	path string
}

var _ control.BackupStore = (*BackupStore)(nil)

func (s *BackupStore) Create(ctx context.Context, destinationDir string, createdAt time.Time) (control.BackupInfo, error) {
	if strings.TrimSpace(destinationDir) == "" {
		return control.BackupInfo{}, fmt.Errorf("backup directory is required")
	}
	directory, err := filepath.Abs(destinationDir)
	if err != nil {
		return control.BackupInfo{}, fmt.Errorf("resolve backup directory: %w", err)
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return control.BackupInfo{}, fmt.Errorf("create backup directory: %w", err)
	}
	if err := s.db.PingContext(ctx); err != nil {
		return control.BackupInfo{}, fmt.Errorf("check database before backup: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return control.BackupInfo{}, fmt.Errorf("checkpoint database before backup: %w", err)
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	name := "boxd-" + createdAt.UTC().Format("20060102-150405.000") + ".sqlite"
	target := filepath.Join(directory, name)
	if _, err := os.Stat(target); err == nil {
		return control.BackupInfo{}, fmt.Errorf("backup %s already exists", target)
	} else if !errors.Is(err, os.ErrNotExist) {
		return control.BackupInfo{}, fmt.Errorf("inspect backup destination: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, "VACUUM INTO ?", target); err != nil {
		return control.BackupInfo{}, fmt.Errorf("create database snapshot: %w", err)
	}
	if err := os.Chmod(target, 0o600); err != nil {
		_ = os.Remove(target)
		return control.BackupInfo{}, fmt.Errorf("secure database snapshot: %w", err)
	}
	info, err := s.Inspect(ctx, target)
	if err != nil {
		_ = os.Remove(target)
		return control.BackupInfo{}, err
	}
	info.CreatedAt = createdAt.UTC()
	return info, nil
}

func (s *BackupStore) Inspect(ctx context.Context, path string) (control.BackupInfo, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return control.BackupInfo{}, fmt.Errorf("resolve backup path: %w", err)
	}
	stat, err := os.Stat(absolute)
	if err != nil {
		return control.BackupInfo{}, fmt.Errorf("stat backup: %w", err)
	}
	if !stat.Mode().IsRegular() {
		return control.BackupInfo{}, fmt.Errorf("backup path is not a regular file")
	}
	if stat.Size() == 0 {
		return control.BackupInfo{}, fmt.Errorf("backup file is empty")
	}
	dsn := "file:" + filepath.ToSlash(absolute) + "?mode=ro&immutable=1"
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return control.BackupInfo{}, fmt.Errorf("open backup for inspection: %w", err)
	}
	defer database.Close()
	database.SetMaxOpenConns(1)
	if err := database.PingContext(ctx); err != nil {
		return control.BackupInfo{}, fmt.Errorf("open backup for inspection: %w", err)
	}
	var check string
	if err := database.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&check); err != nil {
		return control.BackupInfo{}, fmt.Errorf("check backup integrity: %w", err)
	}
	if check != "ok" {
		return control.BackupInfo{}, fmt.Errorf("backup integrity check failed: %s", check)
	}
	info := control.BackupInfo{Path: absolute, SizeBytes: stat.Size(), CreatedAt: stat.ModTime().UTC()}
	if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&info.SchemaVersion); err != nil {
		return control.BackupInfo{}, fmt.Errorf("read backup schema version: %w", err)
	}
	var revision sql.NullString
	if err := database.QueryRowContext(ctx, "SELECT revision FROM control_profile WHERE id = 1").Scan(&revision); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return control.BackupInfo{}, fmt.Errorf("read backup profile revision: %w", err)
	}
	if revision.Valid {
		info.Revision = revision.String
	}
	return info, nil
}

func (s *BackupStore) List(ctx context.Context, destinationDir string) ([]control.BackupInfo, error) {
	entries, err := os.ReadDir(destinationDir)
	if errors.Is(err, os.ErrNotExist) {
		return []control.BackupInfo{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read backup directory: %w", err)
	}
	result := make([]control.BackupInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "boxd-") || !strings.HasSuffix(name, ".sqlite") {
			continue
		}
		info, inspectErr := s.Inspect(ctx, filepath.Join(destinationDir, name))
		if inspectErr != nil {
			return nil, fmt.Errorf("inspect backup %s: %w", name, inspectErr)
		}
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}
