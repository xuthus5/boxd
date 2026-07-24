package core

import (
	"context"
	"errors"
	"testing"
	"time"

	N "github.com/sagernet/sing/common/network"
)

func TestOutboundDelayPropagatesCallerCancellation(t *testing.T) {
	started := make(chan context.Context, 1)
	previous := runURLTest
	runURLTest = func(testCtx context.Context, _ string, _ N.Dialer) (uint16, error) {
		started <- testCtx
		<-testCtx.Done()
		return 0, testCtx.Err()
	}
	t.Cleanup(func() { runURLTest = previous })

	type contextKey string
	const key contextKey = "box-value"
	instance := newRunningOutboundDelayInstance(context.WithValue(context.Background(), key, "preserved"))
	callerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := instance.OutboundDelay(callerCtx, "proxy", "", time.Second)
		result <- err
	}()

	var testCtx context.Context
	select {
	case testCtx = <-started:
	case <-time.After(time.Second):
		t.Fatal("url test did not start")
	}
	if got := testCtx.Value(key); got != "preserved" {
		t.Fatalf("context value = %v, want preserved", got)
	}

	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("OutboundDelay error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("OutboundDelay did not stop after caller cancellation")
	}
}

func TestOutboundDelayHonorsCallerDeadline(t *testing.T) {
	started := make(chan context.Context, 1)
	previous := runURLTest
	runURLTest = func(testCtx context.Context, _ string, _ N.Dialer) (uint16, error) {
		started <- testCtx
		<-testCtx.Done()
		return 0, testCtx.Err()
	}
	t.Cleanup(func() { runURLTest = previous })

	instance := newRunningOutboundDelayInstance(context.Background())
	callerCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	startedAt := time.Now()
	result := make(chan error, 1)
	go func() {
		_, err := instance.OutboundDelay(callerCtx, "proxy", "", time.Second)
		result <- err
	}()

	select {
	case testCtx := <-started:
		deadline, ok := testCtx.Deadline()
		if !ok || time.Until(deadline) > 200*time.Millisecond {
			t.Fatalf("test context deadline = %v, ok = %v", deadline, ok)
		}
	case <-time.After(time.Second):
		t.Fatal("url test did not start")
	}

	select {
	case err := <-result:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("OutboundDelay error = %v, want context.DeadlineExceeded", err)
		}
		if elapsed := time.Since(startedAt); elapsed > time.Second {
			t.Fatalf("OutboundDelay took %v after caller deadline", elapsed)
		}
	case <-time.After(time.Second):
		t.Fatal("OutboundDelay did not stop after caller deadline")
	}
}

func TestOutboundDelayNormalResult(t *testing.T) {
	previous := runURLTest
	runURLTest = func(testCtx context.Context, _ string, _ N.Dialer) (uint16, error) {
		if testCtx == nil {
			return 0, errors.New("nil test context")
		}
		return 123, nil
	}
	t.Cleanup(func() { runURLTest = previous })

	instance := newRunningOutboundDelayInstance(context.Background())
	delay, err := instance.OutboundDelay(context.Background(), "proxy", "", time.Second)
	if err != nil {
		t.Fatalf("OutboundDelay error = %v", err)
	}
	if delay != 123 {
		t.Fatalf("OutboundDelay delay = %d, want 123", delay)
	}
}

func TestOutboundDelayUsesTimeout(t *testing.T) {
	previous := runURLTest
	runURLTest = func(testCtx context.Context, _ string, _ N.Dialer) (uint16, error) {
		<-testCtx.Done()
		return 0, testCtx.Err()
	}
	t.Cleanup(func() { runURLTest = previous })

	instance := newRunningOutboundDelayInstance(context.Background())
	startedAt := time.Now()
	_, err := instance.OutboundDelay(context.Background(), "proxy", "", 30*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("OutboundDelay error = %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(startedAt); elapsed > time.Second {
		t.Fatalf("OutboundDelay took %v for timeout", elapsed)
	}
}

func TestOutboundDelayRejectsCanceledCallerBeforeProbe(t *testing.T) {
	called := false
	previous := runURLTest
	runURLTest = func(context.Context, string, N.Dialer) (uint16, error) {
		called = true
		return 123, nil
	}
	t.Cleanup(func() { runURLTest = previous })

	callerCtx, cancel := context.WithCancel(context.Background())
	cancel()
	instance := newRunningOutboundDelayInstance(context.Background())
	_, err := instance.OutboundDelay(callerCtx, "proxy", "", time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("OutboundDelay error = %v, want context.Canceled", err)
	}
	if called {
		t.Fatal("url test should not start with a canceled caller context")
	}
}

func TestOutboundDelayOutboundNotFound(t *testing.T) {
	instance := newRunningOutboundDelayInstance(context.Background())
	_, err := instance.OutboundDelay(context.Background(), "missing", "", time.Second)
	if !errors.Is(err, ErrOutboundNotFound) {
		t.Fatalf("OutboundDelay error = %v, want ErrOutboundNotFound", err)
	}
}

func newRunningOutboundDelayInstance(boxCtx context.Context) *SBInstance {
	return &SBInstance{
		running: true,
		box:     newFakeBox(),
		boxCtx:  boxCtx,
	}
}
