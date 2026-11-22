package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/xuthus5/boxd/control"
)

// SecurityRepository persists the local administrator password verifier.
type SecurityRepository struct {
	db *sql.DB
}

var _ control.SecurityRepository = (*SecurityRepository)(nil)

func (r *SecurityRepository) Load(ctx context.Context) (control.SecurityCredential, bool, error) {
	var credential control.SecurityCredential
	var updatedAt int64
	err := r.db.QueryRowContext(ctx, `
		SELECT salt, hash, iterations, updated_at
		FROM security_credential WHERE id = 1`,
	).Scan(&credential.Salt, &credential.Hash, &credential.Iterations, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return control.SecurityCredential{}, false, nil
	}
	if err != nil {
		return control.SecurityCredential{}, false, err
	}
	credential.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	return credential, true, nil
}

func (r *SecurityRepository) Save(ctx context.Context, credential control.SecurityCredential) error {
	updatedAt := credential.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO security_credential(id, salt, hash, iterations, updated_at)
		VALUES(1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			salt = excluded.salt,
			hash = excluded.hash,
			iterations = excluded.iterations,
			updated_at = excluded.updated_at`,
		credential.Salt, credential.Hash, credential.Iterations, updatedAt.UTC().UnixMilli(),
	)
	return err
}
