package manager

import (
	"context"
	"time"

	"github.com/xuthus5/boxd/kernel"
)

// RecentLogs implements kernel.Observer.
func (m *Manager) RecentLogs(limit int) []kernel.LogEntry {
	return m.logs.RecentLogs(limit)
}

// TrafficSnapshot implements kernel.Observer.
func (m *Manager) TrafficSnapshot() kernel.TrafficSnapshot {
	engine, _ := m.currentEngine()
	if engine == nil {
		return kernel.TrafficSnapshot{At: time.Now().UTC()}
	}
	return engine.TrafficSnapshot()
}

// Subscribe implements kernel.Observer.
func (m *Manager) Subscribe(ctx context.Context, types ...kernel.EventType) (<-chan kernel.Event, error) {
	return m.events.Subscribe(ctx, types...)
}
