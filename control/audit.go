package control

import (
	"context"
	"time"
)

// AuditEvent is an immutable record of a control-plane operation.
type AuditEvent struct {
	ID        int64          `json:"id"`
	Action    string         `json:"action"`
	Actor     string         `json:"actor"`
	Source    string         `json:"source"`
	Detail    map[string]any `json:"detail,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// AuditFilter narrows an audit query. Zero values disable the corresponding
// condition.
type AuditFilter struct {
	Action string
	Actor  string
	Source string
	Since  time.Time
	Until  time.Time
	Limit  int
}

// AuditAPI exposes audit queries and manual event recording. Control-plane
// mutations are recorded automatically; hosts may add authentication or
// transport-specific events through Record.
type AuditAPI interface {
	List(ctx context.Context, filter AuditFilter) ([]AuditEvent, error)
	Record(ctx context.Context, event AuditEvent) (AuditEvent, error)
}

// AuditRecorder is the write-only subset used by internal services.
type AuditRecorder interface {
	Record(ctx context.Context, event AuditEvent) (AuditEvent, error)
}

type auditIdentityContextKey struct{}

// WithAuditIdentity attaches a trusted actor and source to a context. Host
// adapters should set this after authentication and before calling the control
// plane.
func WithAuditIdentity(ctx context.Context, actor, source string) context.Context {
	return context.WithValue(ctx, auditIdentityContextKey{}, auditIdentity{actor: actor, source: source})
}

type auditIdentity struct {
	actor  string
	source string
}

func AuditIdentityFromContext(ctx context.Context) (string, string) {
	identity, ok := ctx.Value(auditIdentityContextKey{}).(auditIdentity)
	if !ok {
		return "", ""
	}
	return identity.actor, identity.source
}
