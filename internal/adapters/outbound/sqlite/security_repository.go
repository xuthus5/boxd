package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/xuthus5/boxd/internal/webauth"
)

// WebAuthRepository persists the browser administrator password verifier.
// It is a Web-ingress adapter and is not exposed through control.Application.
type WebAuthRepository struct {
	db *sql.DB
}

var _ webauth.Repository = (*WebAuthRepository)(nil)

func (r *WebAuthRepository) Load(ctx context.Context) (webauth.Credential, bool, error) {
	var credential webauth.Credential
	var updatedAt int64
	err := r.db.QueryRowContext(ctx, `
		SELECT salt, hash, iterations, updated_at
		FROM security_credential WHERE id = 1`,
	).Scan(&credential.Salt, &credential.Hash, &credential.Iterations, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return webauth.Credential{}, false, nil
	}
	if err != nil {
		return webauth.Credential{}, false, err
	}
	credential.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	return credential, true, nil
}

func (r *WebAuthRepository) Save(ctx context.Context, credential webauth.Credential) error {
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
