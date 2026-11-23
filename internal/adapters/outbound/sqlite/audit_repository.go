package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/xuthus5/boxd/control"
)

// AuditRepository persists control-plane audit events.
type AuditRepository struct {
	db *sql.DB
}

var _ control.AuditRepository = (*AuditRepository)(nil)

func (r *AuditRepository) Append(ctx context.Context, event control.AuditEvent) (int64, error) {
	detail := []byte("{}")
	if len(event.Detail) > 0 {
		encoded, err := json.Marshal(event.Detail)
		if err != nil {
			return 0, err
		}
		detail = encoded
	}
	createdAt := event.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_events(action, actor, source, detail_json, created_at)
		VALUES(?, ?, ?, ?, ?)`,
		event.Action, event.Actor, event.Source, string(detail), createdAt.UTC().UnixMilli(),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *AuditRepository) List(ctx context.Context, filter control.AuditFilter) ([]control.AuditEvent, error) {
	query := strings.Builder{}
	query.WriteString("SELECT id, action, actor, source, detail_json, created_at FROM audit_events WHERE 1 = 1")
	args := make([]any, 0, 8)
	if filter.Action != "" {
		query.WriteString(" AND action = ?")
		args = append(args, filter.Action)
	}
	if filter.Actor != "" {
		query.WriteString(" AND actor = ?")
		args = append(args, filter.Actor)
	}
	if filter.Source != "" {
		query.WriteString(" AND source = ?")
		args = append(args, filter.Source)
	}
	if !filter.Since.IsZero() {
		query.WriteString(" AND created_at >= ?")
		args = append(args, filter.Since.UTC().UnixMilli())
	}
	if !filter.Until.IsZero() {
		query.WriteString(" AND created_at <= ?")
		args = append(args, filter.Until.UTC().UnixMilli())
	}
	query.WriteString(" ORDER BY created_at DESC, id DESC LIMIT ?")
	args = append(args, filter.Limit)

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]control.AuditEvent, 0)
	for rows.Next() {
		var event control.AuditEvent
		var detail string
		var createdAt int64
		if err := rows.Scan(&event.ID, &event.Action, &event.Actor, &event.Source, &detail, &createdAt); err != nil {
			return nil, err
		}
		if detail != "" {
			if err := json.Unmarshal([]byte(detail), &event.Detail); err != nil {
				return nil, err
			}
		}
		event.CreatedAt = time.UnixMilli(createdAt).UTC()
		result = append(result, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
