package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/xuthus5/boxd/control"
)

// SettingsRepository implements the control-plane settings port.
type SettingsRepository struct {
	db *sql.DB
}

var _ control.SettingsRepository = (*SettingsRepository)(nil)

// List returns all settings ordered by key.
func (r *SettingsRepository) List(ctx context.Context) ([]control.Setting, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT key, value, updated_at FROM settings ORDER BY key")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]control.Setting, 0)
	for rows.Next() {
		setting, err := scanSetting(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, setting)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// Get returns one setting and whether it exists.
func (r *SettingsRepository) Get(ctx context.Context, key string) (control.Setting, bool, error) {
	var updatedAt int64
	var setting control.Setting
	err := r.db.QueryRowContext(ctx,
		"SELECT key, value, updated_at FROM settings WHERE key = ?", key,
	).Scan(&setting.Key, &setting.Value, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return control.Setting{}, false, nil
	}
	if err != nil {
		return control.Setting{}, false, err
	}
	setting.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	return setting, true, nil
}

// Upsert inserts or replaces a setting.
func (r *SettingsRepository) Upsert(ctx context.Context, setting control.Setting) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO settings(key, value, updated_at) VALUES(?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		setting.Key, setting.Value, setting.UpdatedAt.UTC().UnixMilli(),
	)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSetting(row rowScanner) (control.Setting, error) {
	var setting control.Setting
	var updatedAt int64
	if err := row.Scan(&setting.Key, &setting.Value, &updatedAt); err != nil {
		return control.Setting{}, err
	}
	setting.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	return setting, nil
}
