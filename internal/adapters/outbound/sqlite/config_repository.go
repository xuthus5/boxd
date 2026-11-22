package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/xuthus5/boxd/kernel"
)

const (
	currentRevisionKey  = "current_revision"
	defaultHistoryLimit = 50
)

// ConfigRepository implements kernel.ConfigStore with a single current pointer
// and immutable configuration history.
type ConfigRepository struct {
	db *sql.DB
}

var _ kernel.ConfigStore = (*ConfigRepository)(nil)
var _ kernel.ConfigRemover = (*ConfigRepository)(nil)

// Load returns the current configuration.
func (r *ConfigRepository) Load(ctx context.Context) (kernel.ConfigDocument, error) {
	var revision string
	err := r.db.QueryRowContext(ctx,
		"SELECT value FROM config_metadata WHERE key = ?",
		currentRevisionKey,
	).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		return kernel.ConfigDocument{}, kernel.ErrConfigNotFound
	}
	if err != nil {
		return kernel.ConfigDocument{}, err
	}

	var document kernel.ConfigDocument
	document.Revision = revision
	err = r.db.QueryRowContext(ctx,
		"SELECT format, body FROM config_history WHERE revision = ?",
		revision,
	).Scan(&document.Format, &document.Body)
	if errors.Is(err, sql.ErrNoRows) {
		return kernel.ConfigDocument{}, kernel.NewError(kernel.ErrCodeInternal, "config.load", "current revision is missing from history")
	}
	if err != nil {
		return kernel.ConfigDocument{}, err
	}
	document.Body = append([]byte(nil), document.Body...)
	return document, nil
}

// SaveAtomic updates the current pointer and appends immutable history. Saving
// an existing identical revision is idempotent so rollback can reuse it.
func (r *ConfigRepository) SaveAtomic(ctx context.Context, document kernel.ConfigDocument) error {
	if document.Revision == "" {
		return kernel.NewError(kernel.ErrCodeInvalidConfig, "config.save", "revision is required")
	}
	transaction, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback() }()

	var existingFormat string
	var existingBody []byte
	queryErr := transaction.QueryRowContext(ctx,
		"SELECT format, body FROM config_history WHERE revision = ?",
		document.Revision,
	).Scan(&existingFormat, &existingBody)
	switch {
	case queryErr == nil:
		if existingFormat != document.Format || !bytes.Equal(existingBody, document.Body) {
			return kernel.NewError(kernel.ErrCodeConflict, "config.save", "revision already has different content")
		}
	case errors.Is(queryErr, sql.ErrNoRows):
		if _, err := transaction.ExecContext(ctx, `
			INSERT INTO config_history(revision, source, mode, summary, format, body, created_at)
			VALUES(?, ?, ?, ?, ?, ?, ?)`,
			document.Revision, "control", "managed", "", document.Format, document.Body, time.Now().UTC().UnixMilli(),
		); err != nil {
			return err
		}
	default:
		return queryErr
	}

	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO config_metadata(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		currentRevisionKey, document.Revision,
	); err != nil {
		return err
	}
	return transaction.Commit()
}

// History returns newest revisions first.
func (r *ConfigRepository) History(ctx context.Context, limit int) ([]kernel.ConfigRevision, error) {
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT revision, source, summary, created_at
		FROM config_history ORDER BY created_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]kernel.ConfigRevision, 0, limit)
	for rows.Next() {
		var revision kernel.ConfigRevision
		var createdAt int64
		if err := rows.Scan(&revision.Revision, &revision.Source, &revision.Summary, &createdAt); err != nil {
			return nil, err
		}
		revision.CreatedAt = time.UnixMilli(createdAt).UTC()
		result = append(result, revision)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteCurrent clears the current pointer without erasing history.
func (r *ConfigRepository) DeleteCurrent(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM config_metadata WHERE key = ?", currentRevisionKey)
	return err
}
