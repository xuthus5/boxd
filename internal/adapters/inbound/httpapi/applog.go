package httpapi

import (
	"context"
	"sync"
	"time"
)

const appLogSubscriberBuffer = 64

// appLogEntry is the browser-facing shape of an application log line.
type appLogEntry struct {
	ID      uint64         `json:"id"`
	Time    time.Time      `json:"time"`
	Level   string         `json:"level"`
	Source  string         `json:"source"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

// logHub keeps a bounded replay window and fans new lines out without blocking
// request handlers. Sensitive request bodies and headers never enter this hub.
type logHub struct {
	mu          sync.Mutex
	entries     []appLogEntry
	next        int
	full        bool
	capacity    int
	sequence    uint64
	subscribers map[uint64]chan appLogEntry
	nextID      uint64
}

func newLogHub(capacity int) *logHub {
	if capacity <= 0 {
		capacity = 500
	}
	return &logHub{
		entries:     make([]appLogEntry, capacity),
		capacity:    capacity,
		subscribers: make(map[uint64]chan appLogEntry),
	}
}

func (h *logHub) add(level, source, message string, fields map[string]any) {
	if level == "" {
		level = "info"
	}
	h.mu.Lock()
	h.sequence++
	entry := appLogEntry{
		ID:      h.sequence,
		Time:    time.Now().UTC(),
		Level:   level,
		Source:  source,
		Message: message,
		Fields:  cloneFields(fields),
	}
	h.entries[h.next] = entry
	h.next = (h.next + 1) % h.capacity
	if h.next == 0 {
		h.full = true
	}
	for id, subscriber := range h.subscribers {
		if !deliverLog(subscriber, entry) {
			delete(h.subscribers, id)
			close(subscriber)
		}
	}
	h.mu.Unlock()
}

func (h *logHub) recent(limit int) []appLogEntry {
	if limit <= 0 {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	count := h.next
	if h.full {
		count = h.capacity
	}
	if limit < count {
		count = limit
	}
	result := make([]appLogEntry, 0, count)
	start := h.next - count
	if start < 0 {
		start += h.capacity
	}
	for index := 0; index < count; index++ {
		result = append(result, h.entries[(start+index)%h.capacity])
	}
	return result
}

func (h *logHub) subscribe(ctx context.Context) <-chan appLogEntry {
	channel := make(chan appLogEntry, appLogSubscriberBuffer)
	h.mu.Lock()
	id := h.nextID
	h.nextID++
	h.subscribers[id] = channel
	h.mu.Unlock()
	go func() {
		<-ctx.Done()
		h.mu.Lock()
		if subscriber, exists := h.subscribers[id]; exists {
			delete(h.subscribers, id)
			close(subscriber)
		}
		h.mu.Unlock()
	}()
	return channel
}

func deliverLog(channel chan appLogEntry, entry appLogEntry) (open bool) {
	defer func() {
		if recover() != nil {
			open = false
		}
	}()
	select {
	case channel <- entry:
		return true
	default:
		return false
	}
}

func cloneFields(fields map[string]any) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	result := make(map[string]any, len(fields))
	for key, value := range fields {
		result[key] = value
	}
	return result
}
