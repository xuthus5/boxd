package core

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// 内核未运行时，所有运行时维护方法应返回 ErrNotRunning，不做任何实际拨号。
func TestRuntimeControlNotRunning(t *testing.T) {
	inst := NewSBInstance("/nonexistent/config.json", NewLogWriter(10))

	if err := inst.FlushDNS(); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("FlushDNS err = %v, want ErrNotRunning", err)
	}
	if err := inst.FlushFakeIP(); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("FlushFakeIP err = %v, want ErrNotRunning", err)
	}
	if d, err := inst.OutboundDelay(context.Background(), "x", "", time.Second); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("OutboundDelay err = %v, want ErrNotRunning", err)
	} else if d != 0 {
		t.Fatalf("OutboundDelay delay = %d, want 0", d)
	}
}

func TestRuntimeControlRunningWithoutOptionalServices(t *testing.T) {
	inst := NewSBInstance("", nil)
	inst.running = true
	inst.boxCtx = context.Background()
	if err := inst.FlushDNS(); err != nil {
		t.Fatalf("FlushDNS() error = %v", err)
	}
	if err := inst.FlushFakeIP(); !errors.Is(err, ErrFeatureNotEnabled) {
		t.Fatalf("FlushFakeIP() error = %v, want ErrFeatureNotEnabled", err)
	}
}

func TestRuntimeControlSentinels(t *testing.T) {
	// 确保哨兵错误与既有序列不冲突且可被 errors.Is 识别。
	if !errors.Is(ErrFeatureNotEnabled, ErrFeatureNotEnabled) {
		t.Fatal("ErrFeatureNotEnabled self-match failed")
	}
	if !errors.Is(ErrOutboundNotFound, ErrOutboundNotFound) {
		t.Fatal("ErrOutboundNotFound self-match failed")
	}
	if errors.Is(ErrFeatureNotEnabled, ErrNotRunning) {
		t.Fatal("ErrFeatureNotEnabled should not match ErrNotRunning")
	}
}

func TestOutboundDelayNotRunning(t *testing.T) {
	s := &SBInstance{}
	if _, err := s.OutboundDelay(context.Background(), "x", "", time.Second); err == nil {
		t.Fatal("expected not running")
	}
	if err := s.FlushDNS(); err == nil {
		t.Fatal("expected not running flush dns")
	}
	if err := s.FlushFakeIP(); err == nil {
		t.Fatal("expected not running flush fakeip")
	}
}

type fakeClashMode struct {
	mode string
	list []string
}

func (f *fakeClashMode) Mode() string { return f.mode }

func (f *fakeClashMode) ModeList() []string { return append([]string(nil), f.list...) }

func (f *fakeClashMode) SetMode(mode string) {
	for _, item := range f.list {
		if strings.EqualFold(item, mode) {
			f.mode = item
			return
		}
	}
}

func TestClashModeNotRunning(t *testing.T) {
	inst := NewSBInstance("/nonexistent/config.json", NewLogWriter(10))
	if _, err := inst.ClashMode(); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("ClashMode err = %v", err)
	}
	if _, err := inst.SetClashMode("Rule"); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("SetClashMode err = %v", err)
	}
}

func TestClashModeFeatureNotEnabled(t *testing.T) {
	inst := NewSBInstance("", nil)
	inst.running = true
	inst.boxCtx = context.Background()
	if _, err := inst.ClashMode(); !errors.Is(err, ErrFeatureNotEnabled) {
		t.Fatalf("ClashMode err = %v", err)
	}
	if _, err := inst.SetClashMode("Rule"); !errors.Is(err, ErrFeatureNotEnabled) {
		t.Fatalf("SetClashMode err = %v", err)
	}
}

func TestClashModeGetAndSet(t *testing.T) {
	fake := &fakeClashMode{mode: "Rule", list: []string{"Rule", "Global", "Direct"}}
	previous := clashModeFromContext
	clashModeFromContext = func(context.Context) clashModeControl { return fake }
	t.Cleanup(func() { clashModeFromContext = previous })

	inst := NewSBInstance("", nil)
	inst.running = true
	inst.boxCtx = context.Background()

	status, err := inst.ClashMode()
	if err != nil {
		t.Fatal(err)
	}
	if status.Mode != "Rule" || len(status.ModeList) != 3 {
		t.Fatalf("status = %#v", status)
	}

	status, err = inst.SetClashMode("global")
	if err != nil {
		t.Fatal(err)
	}
	if status.Mode != "Global" {
		t.Fatalf("mode = %q", status.Mode)
	}

	if _, err := inst.SetClashMode(""); !errors.Is(err, ErrInvalidMode) {
		t.Fatalf("empty mode err = %v", err)
	}
	if _, err := inst.SetClashMode("Script"); !errors.Is(err, ErrInvalidMode) {
		t.Fatalf("unknown mode err = %v", err)
	}
}
