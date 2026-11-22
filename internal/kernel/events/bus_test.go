package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/kernel/events"
	"github.com/xuthus5/boxd/kernel"
)

func TestBusFiltersAndClosesOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	bus := events.New(2)
	channel, err := bus.Subscribe(ctx, kernel.EventKernelState)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	bus.Publish(kernel.Event{Type: kernel.EventKernelLog})
	bus.Publish(kernel.Event{Type: kernel.EventKernelState})

	select {
	case event := <-channel:
		if event.Type != kernel.EventKernelState {
			t.Fatalf("event type = %s", event.Type)
		}
		if event.Sequence == 0 || event.Time.IsZero() {
			t.Fatal("event was not normalized")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
	cancel()
	select {
	case _, ok := <-channel:
		if ok {
			t.Fatal("channel remained open after cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("channel was not closed")
	}
}

func TestBusDropsOldestWhenSubscriberIsSlow(t *testing.T) {
	bus := events.New(1)
	channel, err := bus.Subscribe(context.Background())
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	bus.Publish(kernel.Event{Type: kernel.EventKernelLog, Payload: 1})
	bus.Publish(kernel.Event{Type: kernel.EventKernelLog, Payload: 2})
	event := <-channel
	if event.Payload != 2 {
		t.Fatalf("payload = %v, want newest event", event.Payload)
	}
}
