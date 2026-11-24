package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/xuthus5/boxd/control"
)

// RuleSetRepository stores refresh metadata for installed rule sets.
type RuleSetRepository struct {
	db *sql.DB
}

var _ control.RuleSetRepository = (*RuleSetRepository)(nil)

func (r *RuleSetRepository) List(ctx context.Context) ([]control.RuleSetStatus, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT name, etag, last_modified, sha256, size_bytes, updated_at, error
		FROM rule_set_status ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]control.RuleSetStatus, 0)
	for rows.Next() {
		status, err := scanRuleSetStatus(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, status)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *RuleSetRepository) Get(ctx context.Context, name string) (control.RuleSetStatus, bool, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT name, etag, last_modified, sha256, size_bytes, updated_at, error
		FROM rule_set_status WHERE name = ?`, name)
	status, err := scanRuleSetStatus(row)
	if errors.Is(err, sql.ErrNoRows) {
		return control.RuleSetStatus{}, false, nil
	}
	if err != nil {
		return control.RuleSetStatus{}, false, err
	}
	return status, true, nil
}

func (r *RuleSetRepository) Upsert(ctx context.Context, status control.RuleSetStatus) error {
	updatedAt := status.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO rule_set_status(name, source, etag, sha256, size_bytes, updated_at, last_modified, error)
		VALUES(?, '', ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			etag = excluded.etag,
			sha256 = excluded.sha256,
			size_bytes = excluded.size_bytes,
			updated_at = excluded.updated_at,
			last_modified = excluded.last_modified,
			error = excluded.error`,
		status.Name, status.ETag, status.SHA256, status.SizeBytes, updatedAt.UTC().UnixMilli(), status.LastModified, status.Error,
	)
	return err
}

func (r *RuleSetRepository) Delete(ctx context.Context, name string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM rule_set_status WHERE name = ?", name)
	return err
}

func scanRuleSetStatus(row rowScanner) (control.RuleSetStatus, error) {
	var status control.RuleSetStatus
	var updatedAt int64
	if err := row.Scan(&status.Name, &status.ETag, &status.LastModified, &status.SHA256, &status.SizeBytes, &updatedAt, &status.Error); err != nil {
		return control.RuleSetStatus{}, err
	}
	status.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	return status, nil
}
