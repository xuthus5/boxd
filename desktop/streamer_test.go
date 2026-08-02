package main

import (
	"sync"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/service"
)

// fakeEmitter 捕获事件便于断言。
type fakeEmitter struct {
	mu     sync.Mutex
	events map[string][]any
}

func newFakeEmitter() *fakeEmitter {
	return &fakeEmitter{events: make(map[string][]any)}
}

func (f *fakeEmitter) Emit(name string, data ...any) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events[name] = append(f.events[name], data...)
	return true
}

func (f *fakeEmitter) count(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events[name])
}

func (f *fakeEmitter) last(name string) any {
	f.mu.Lock()
	defer f.mu.Unlock()
	events := f.events[name]
	if len(events) == 0 {
		return nil
	}
	return events[len(events)-1]
}

func TestStreamerNilEmitter(t *testing.T) {
	rt := &desktopRuntime{svc: service.New(service.Deps{})}
	s := newEventStreamer(nil, rt)
	// nil emitter 不应 panic
	s.emit("boxd:test", map[string]string{"a": "b"})
}

func TestStreamerNilRT(t *testing.T) {
	s := newEventStreamer(newFakeEmitter(), nil)
	s.Start()
	s.Stop()
	time.Sleep(50 * time.Millisecond)
}

func TestStreamerPushesTrafficAndConnections(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	emitter := newFakeEmitter()
	s := newEventStreamer(emitter, rt)
	s.Start()
	defer s.Stop()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if emitter.count(TrafficEventName) > 0 && emitter.count(ConnectionsEventName) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if emitter.count(TrafficEventName) == 0 {
		t.Fatal("no traffic event emitted")
	}
	if emitter.count(ConnectionsEventName) == 0 {
		t.Fatal("no connections event emitted")
	}
	if traffic := emitter.last(TrafficEventName); traffic == nil {
		t.Fatal("traffic event data is nil")
	}
}

func TestStreamerPushesLogs(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	emitter := newFakeEmitter()
	s := newEventStreamer(emitter, rt)
	s.Start()
	defer s.Stop()

	// 写入一条内核日志，应被推送
	rt.svc.Deps.KernelLogWriter.WriteAppEntry("info", "test kernel log message")
	rt.svc.Deps.AppLogWriter.WriteAppEntry("warn", "test app log message")

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if emitter.count(KernelLogEventName) > 0 && emitter.count(AppLogEventName) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if emitter.count(KernelLogEventName) == 0 {
		t.Fatal("no kernel log event emitted")
	}
	if emitter.count(AppLogEventName) == 0 {
		t.Fatal("no app log event emitted")
	}
}

func TestStreamerStopIsIdempotent(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	s := newEventStreamer(newFakeEmitter(), rt)
	s.Start()
	s.Stop()
	s.Stop()
}

func TestStreamerNilWriters(t *testing.T) {
	rt := &desktopRuntime{svc: service.New(service.Deps{})}
	s := newEventStreamer(newFakeEmitter(), rt)
	s.Start()
	time.Sleep(50 * time.Millisecond)
	s.Stop()
}

func TestAppEventEmitterNil(t *testing.T) {
	e := appEventEmitter{app: nil}
	if e.Emit("boxd:x", map[string]string{"a": "b"}) {
		t.Fatal("expected false for nil app")
	}
}

func TestNewEventStreamer(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	s := NewEventStreamer(nil, rt)
	if s == nil {
		t.Fatal("streamer is nil")
	}
	// nil app 时 emit 应安全
	s.emit("boxd:test", map[string]string{"a": "b"})
}
