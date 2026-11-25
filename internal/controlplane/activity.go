package controlplane

import (
	"context"
	"sync"
	"time"

	"github.com/xuthus5/boxd/control"
)

const (
	defaultActivityCapacity  = 500
	activitySubscriberBuffer = 64
)

// ActivityService owns the bounded application log replay window and fan-out.
// It is intentionally transport-neutral: HTTP maps entries to SSE, while the
// native host consumes the same records directly.
type ActivityService struct {
	mu          sync.Mutex
	entries     []control.ActivityLogEntry
	next        int
	full        bool
	capacity    int
	sequence    uint64
	subscribers map[uint64]chan control.ActivityLogEntry
	nextID      uint64
	now         func() time.Time
}

var _ control.ActivityLogAPI = (*ActivityService)(nil)

// NewActivityService constructs the shared activity log.
func NewActivityService(capacity int) *ActivityService {
	if capacity <= 0 {
		capacity = defaultActivityCapacity
	}
	return &ActivityService{
		entries:     make([]control.ActivityLogEntry, capacity),
		capacity:    capacity,
		subscribers: make(map[uint64]chan control.ActivityLogEntry),
		now:         time.Now,
	}
}

// Recent returns up to limit entries in chronological order.
func (s *ActivityService) Recent(limit int) []control.ActivityLogEntry {
	if limit <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	count := s.next
	if s.full {
		count = s.capacity
	}
	if limit < count {
		count = limit
	}
	result := make([]control.ActivityLogEntry, 0, count)
	start := s.next - count
	if start < 0 {
		start += s.capacity
	}
	for index := 0; index < count; index++ {
		result = append(result, cloneActivityEntry(s.entries[(start+index)%s.capacity]))
	}
	return result
}

// Subscribe registers a non-blocking subscriber. The channel closes when ctx
// ends or the subscriber falls too far behind.
func (s *ActivityService) Subscribe(ctx context.Context) (<-chan control.ActivityLogEntry, error) {
	channel := make(chan control.ActivityLogEntry, activitySubscriberBuffer)
	s.mu.Lock()
	id := s.nextID
	s.nextID++
	s.subscribers[id] = channel
	s.mu.Unlock()
	go func() {
		<-ctx.Done()
		s.mu.Lock()
		if subscriber, exists := s.subscribers[id]; exists {
			delete(s.subscribers, id)
			close(subscriber)
		}
		s.mu.Unlock()
	}()
	return channel, nil
}

// Record appends one entry and fans it out without blocking the caller.
func (s *ActivityService) Record(_ context.Context, entry control.ActivityLogEntry) control.ActivityLogEntry {
	level := entry.Level
	if level == "" {
		level = "info"
	}
	s.mu.Lock()
	s.sequence++
	entry = control.ActivityLogEntry{
		ID:      s.sequence,
		Time:    entry.Time,
		Level:   level,
		Source:  entry.Source,
		Message: entry.Message,
		Fields:  cloneActivityFields(entry.Fields),
	}
	if entry.Time.IsZero() {
		entry.Time = s.now().UTC()
	}
	s.entries[s.next] = entry
	s.next = (s.next + 1) % s.capacity
	if s.next == 0 {
		s.full = true
	}
	for id, subscriber := range s.subscribers {
		if !deliverActivity(subscriber, entry) {
			delete(s.subscribers, id)
			close(subscriber)
		}
	}
	s.mu.Unlock()
	return cloneActivityEntry(entry)
}

func deliverActivity(channel chan control.ActivityLogEntry, entry control.ActivityLogEntry) (open bool) {
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

func cloneActivityEntry(entry control.ActivityLogEntry) control.ActivityLogEntry {
	entry.Fields = cloneActivityFields(entry.Fields)
	return entry
}

func cloneActivityFields(fields map[string]any) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	result := make(map[string]any, len(fields))
	for key, value := range fields {
		result[key] = value
	}
	return result
}
