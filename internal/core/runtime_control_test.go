package core

import (
	"context"
	"errors"
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
