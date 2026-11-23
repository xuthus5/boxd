package controlplane_test

import (
	"context"
	"sync"
	"testing"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/internal/controlplane"
	"github.com/xuthus5/boxd/kernel"
)

func TestAuditRecordsMutationsAndFiltersByIdentity(t *testing.T) {
	ctx := context.Background()
	audit := &memoryAuditRepository{}
	application := newApplicationWithDependencies(t, controlplane.Dependencies{AuditRepository: audit})

	actorContext := control.WithAuditIdentity(ctx, "alice", "test-host")
	if _, err := application.Egress().SaveNode(actorContext, control.Node{
		ID: "hk", Name: "HK", Protocol: "shadowsocks", Server: "127.0.0.1", Port: 8388,
		Options: map[string]any{"method": "aes-128-gcm", "password": "secret"},
	}); err != nil {
		t.Fatalf("SaveNode() error = %v", err)
	}
	events, err := application.Audit().List(ctx, control.AuditFilter{Action: "egress.save_node"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(events) != 1 || events[0].Actor != "alice" || events[0].Source != "test-host" || events[0].ID == 0 {
		t.Fatalf("audit events = %+v", events)
	}
	if events[0].Detail["revision"] == "" {
		t.Fatalf("audit detail = %+v", events[0].Detail)
	}
	if _, err := application.Audit().Record(ctx, control.AuditEvent{}); !hasCode(err, kernel.ErrCodeInvalidConfig) {
		t.Fatalf("Record(empty) error = %v, want invalid config", err)
	}
	extra, err := application.Audit().Record(ctx, control.AuditEvent{Action: "host.started"})
	if err != nil || extra.Actor != "local" || extra.Source != "control-plane" {
		t.Fatalf("Record() = %+v, %v", extra, err)
	}
}

type memoryAuditRepository struct {
	mu     sync.Mutex
	events []control.AuditEvent
}

func (r *memoryAuditRepository) Append(_ context.Context, event control.AuditEvent) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	event.ID = int64(len(r.events) + 1)
	r.events = append(r.events, event)
	return event.ID, nil
}

func (r *memoryAuditRepository) List(_ context.Context, filter control.AuditFilter) ([]control.AuditEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]control.AuditEvent, 0)
	for index := len(r.events) - 1; index >= 0; index-- {
		event := r.events[index]
		if filter.Action != "" && event.Action != filter.Action {
			continue
		}
		if filter.Actor != "" && event.Actor != filter.Actor {
			continue
		}
		if filter.Source != "" && event.Source != filter.Source {
			continue
		}
		result = append(result, event)
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}
	return result, nil
}
