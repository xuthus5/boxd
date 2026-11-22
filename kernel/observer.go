package kernel

import "context"

// LogSink receives structured engine logs without blocking the engine.
type LogSink interface {
	WriteLog(entry LogEntry)
}

// EventSink receives engine events through a non-blocking boundary.
type EventSink interface {
	Publish(event Event)
}

// Observer exposes recent logs, traffic, and asynchronous events.
type Observer interface {
	RecentLogs(limit int) []LogEntry
	TrafficSnapshot() TrafficSnapshot
	Subscribe(ctx context.Context, types ...EventType) (<-chan Event, error)
}
