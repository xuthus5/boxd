package controlplane

import (
	"context"
	"strings"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

const (
	defaultAuditLimit  = 100
	maxAuditLimit      = 1000
	defaultAuditActor  = "local"
	defaultAuditSource = "control-plane"
)

// AuditService persists and queries control-plane audit events.
type AuditService struct {
	repository control.AuditRepository
	now        func() time.Time
}

var _ control.AuditAPI = (*AuditService)(nil)

func NewAuditService(repository control.AuditRepository) *AuditService {
	return &AuditService{repository: repository, now: time.Now}
}

func (s *AuditService) List(ctx context.Context, filter control.AuditFilter) ([]control.AuditEvent, error) {
	if s.repository == nil {
		return nil, capabilityUnavailable("audit.list", "audit repository")
	}
	if filter.Limit <= 0 {
		filter.Limit = defaultAuditLimit
	}
	if filter.Limit > maxAuditLimit {
		filter.Limit = maxAuditLimit
	}
	events, err := s.repository.List(ctx, filter)
	if err != nil {
		return nil, wrapStoreError("audit.list", "list audit events", err)
	}
	return events, nil
}

func (s *AuditService) Record(ctx context.Context, event control.AuditEvent) (control.AuditEvent, error) {
	if s.repository == nil {
		return control.AuditEvent{}, capabilityUnavailable("audit.record", "audit repository")
	}
	event.Action = strings.TrimSpace(event.Action)
	if event.Action == "" {
		return control.AuditEvent{}, kernel.NewError(kernel.ErrCodeInvalidConfig, "audit.record", "audit action is required")
	}
	actor, source := control.AuditIdentityFromContext(ctx)
	if strings.TrimSpace(event.Actor) == "" {
		event.Actor = actor
	}
	if event.Actor == "" {
		event.Actor = defaultAuditActor
	}
	if strings.TrimSpace(event.Source) == "" {
		event.Source = source
	}
	if event.Source == "" {
		event.Source = defaultAuditSource
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = s.now().UTC()
	}
	id, err := s.repository.Append(ctx, event)
	if err != nil {
		return control.AuditEvent{}, wrapStoreError("audit.record", "append audit event", err)
	}
	event.ID = id
	return event, nil
}

func recordAudit(ctx context.Context, recorder control.AuditRecorder, action string, detail map[string]any) {
	if recorder == nil {
		return
	}
	_, _ = recorder.Record(ctx, control.AuditEvent{Action: action, Detail: detail})
}
