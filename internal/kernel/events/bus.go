// Package events provides the non-blocking event fan-out used by the kernel manager.
package events

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xuthus5/boxd/kernel"
)

const defaultBuffer = 64

// Bus is a bounded in-memory event broker. Slow subscribers lose their oldest
// event instead of blocking engine callbacks.
type Bus struct {
	mu          sync.Mutex
	subscribers map[uint64]*subscriber
	nextID      uint64
	bufferSize  int
	sequence    atomic.Uint64
}

type subscriber struct {
	channel chan kernel.Event
	types   map[kernel.EventType]struct{}
}

// New creates an event bus. A non-positive buffer uses the default size.
func New(bufferSize int) *Bus {
	if bufferSize <= 0 {
		bufferSize = defaultBuffer
	}
	return &Bus{
		subscribers: make(map[uint64]*subscriber),
		bufferSize:  bufferSize,
	}
}

// Publish assigns a monotonic sequence and fans the event out without blocking.
func (b *Bus) Publish(event kernel.Event) {
	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}
	event.Sequence = b.sequence.Add(1)

	b.mu.Lock()
	defer b.mu.Unlock()
	for _, target := range b.subscribers {
		if !target.accepts(event.Type) {
			continue
		}
		deliver(target.channel, event)
	}
}

// Subscribe returns a bounded channel closed when the context is canceled.
func (b *Bus) Subscribe(ctx context.Context, types ...kernel.EventType) (<-chan kernel.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	channel := make(chan kernel.Event, b.bufferSize)
	target := &subscriber{channel: channel, types: typeSet(types)}

	b.mu.Lock()
	id := b.nextID
	b.nextID++
	b.subscribers[id] = target
	b.mu.Unlock()

	go b.waitForCancel(ctx, id, channel)
	return channel, nil
}

func (b *Bus) waitForCancel(ctx context.Context, id uint64, channel chan kernel.Event) {
	<-ctx.Done()
	b.mu.Lock()
	if _, ok := b.subscribers[id]; ok {
		delete(b.subscribers, id)
		close(channel)
	}
	b.mu.Unlock()
}

func (s *subscriber) accepts(eventType kernel.EventType) bool {
	if len(s.types) == 0 {
		return true
	}
	_, ok := s.types[eventType]
	return ok
}

func deliver(channel chan kernel.Event, event kernel.Event) {
	select {
	case channel <- event:
		return
	default:
	}
	select {
	case <-channel:
	default:
	}
	select {
	case channel <- event:
	default:
	}
}

func typeSet(types []kernel.EventType) map[kernel.EventType]struct{} {
	if len(types) == 0 {
		return nil
	}
	set := make(map[kernel.EventType]struct{}, len(types))
	for _, eventType := range types {
		set[eventType] = struct{}{}
	}
	return set
}
