package core

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSBInstanceTrafficTracker(t *testing.T) {
	instance := NewSBInstance(filepath.Join(t.TempDir(), "missing.json"), NewLogWriter(5))

	// 未启动时 TrafficTracker 应返回 nil
	tracker := instance.TrafficTracker()
	if tracker != nil {
		t.Error("expected nil TrafficTracker when not running")
	}
}

func TestSBInstanceRestartStopError(t *testing.T) {
	instance := NewSBInstance(filepath.Join(t.TempDir(), "missing.json"), NewLogWriter(5))

	// Stop 在未运行时返回 nil，Restart 再调 Start 也会失败（config 不存在）
	err := instance.Restart()
	if err == nil {
		t.Fatal("expected error from Restart with missing config")
	}
}

func TestSBInstanceFlushDNSNotRunning(t *testing.T) {
	instance := NewSBInstance(filepath.Join(t.TempDir(), "missing.json"), NewLogWriter(5))
	if err := instance.FlushDNS(); err != ErrNotRunning {
		t.Errorf("FlushDNS() err = %v, want ErrNotRunning", err)
	}
}

func TestSBInstanceFlushFakeIPNotRunning(t *testing.T) {
	instance := NewSBInstance(filepath.Join(t.TempDir(), "missing.json"), NewLogWriter(5))
	if err := instance.FlushFakeIP(); err != ErrNotRunning {
		t.Errorf("FlushFakeIP() err = %v, want ErrNotRunning", err)
	}
}

func TestSBInstanceOutboundDelayNotRunning(t *testing.T) {
	instance := NewSBInstance(filepath.Join(t.TempDir(), "missing.json"), NewLogWriter(5))
	_, err := instance.OutboundDelay(context.TODO(), "test", "", 0)
	if err != ErrNotRunning {
		t.Errorf("OutboundDelay() err = %v, want ErrNotRunning", err)
	}
}
