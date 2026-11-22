package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

// ProfileRepository stores the managed desired state as one atomic snapshot.
type ProfileRepository struct {
	db *sql.DB
}

var _ control.ProfileRepository = (*ProfileRepository)(nil)

// Load returns the current desired-state snapshot.
func (r *ProfileRepository) Load(ctx context.Context) (control.DesiredState, error) {
	var revision string
	var body []byte
	var updatedAt int64
	err := r.db.QueryRowContext(ctx,
		"SELECT revision, state_json, updated_at FROM control_profile WHERE id = 1",
	).Scan(&revision, &body, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return control.DesiredState{}, kernel.ErrConfigNotFound
	}
	if err != nil {
		return control.DesiredState{}, err
	}
	var state control.DesiredState
	if err := json.Unmarshal(body, &state); err != nil {
		return control.DesiredState{}, kernel.WrapError(kernel.ErrCodeInternal, "profile.load", "decode desired state", err)
	}
	state.Revision = revision
	state.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	return state, nil
}

// Save writes a complete snapshot atomically.
func (r *ProfileRepository) Save(ctx context.Context, state control.DesiredState) error {
	if state.Revision == "" {
		return kernel.NewError(kernel.ErrCodeInvalidConfig, "profile.save", "profile revision is required")
	}
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = time.Now().UTC()
	}
	body, err := json.Marshal(state)
	if err != nil {
		return kernel.WrapError(kernel.ErrCodeInternal, "profile.save", "encode desired state", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO control_profile(id, revision, state_json, updated_at)
		VALUES(1, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			revision = excluded.revision,
			state_json = excluded.state_json,
			updated_at = excluded.updated_at`,
		state.Revision, body, state.UpdatedAt.UTC().UnixMilli(),
	)
	return err
}
