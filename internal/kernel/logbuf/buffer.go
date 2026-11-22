// Package logbuf stores the recent kernel logs exposed by Observer.RecentLogs.
package logbuf

import (
	"sync"

	"github.com/xuthus5/boxd/kernel"
)

// Buffer is a fixed-capacity ring buffer.
type Buffer struct {
	mu       sync.RWMutex
	entries  []kernel.LogEntry
	next     int
	capacity int
	full     bool
}

// New creates a buffer with the requested capacity.
func New(capacity int) *Buffer {
	if capacity < 0 {
		capacity = 0
	}
	return &Buffer{entries: make([]kernel.LogEntry, capacity), capacity: capacity}
}

// WriteLog implements kernel.LogSink.
func (b *Buffer) WriteLog(entry kernel.LogEntry) {
	if b.capacity == 0 {
		return
	}
	b.mu.Lock()
	b.entries[b.next] = entry
	b.next = (b.next + 1) % b.capacity
	if b.next == 0 {
		b.full = true
	}
	b.mu.Unlock()
}

// RecentLogs returns up to limit entries in chronological order.
func (b *Buffer) RecentLogs(limit int) []kernel.LogEntry {
	if limit <= 0 || b.capacity == 0 {
		return nil
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	count := b.next
	if b.full {
		count = b.capacity
	}
	if limit < count {
		count = limit
	}
	result := make([]kernel.LogEntry, 0, count)
	start := b.next - count
	if start < 0 {
		start += b.capacity
	}
	for i := 0; i < count; i++ {
		result = append(result, b.entries[(start+i)%b.capacity])
	}
	return result
}
