package main

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

func TestNativeRuntimeInfo(t *testing.T) {
	rt := &desktopRuntime{cfg: desktopConfig{Mode: "embedded", RemoteURL: "http://127.0.0.1:9091"}}
	n := NewNativeCapabilities(rt)
	info := n.Runtime(context.Background())
	if info["mode"] != "embedded" {
		t.Fatalf("mode = %v", info["mode"])
	}
	if info["remote_url"] != "http://127.0.0.1:9091" {
		t.Fatalf("remote_url = %v", info["remote_url"])
	}
	if info["platform"] != runtime.GOOS {
		t.Fatalf("platform = %v", info["platform"])
	}
}

func TestNativeRuntimeInfoNil(t *testing.T) {
	n := NewNativeCapabilities(nil)
	info := n.Runtime(context.Background())
	if info["mode"] != "embedded" {
		t.Fatalf("mode = %v", info["mode"])
	}
}

func TestNativeAutostartNotAvailable(t *testing.T) {
	rt := &desktopRuntime{}
	n := NewNativeCapabilities(rt)
	if enabled, err := n.IsAutostartEnabled(context.Background()); err != nil || enabled {
		t.Fatalf("enabled=%v err=%v", enabled, err)
	}
	if err := n.SetAutostart(context.Background(), true); err == nil {
		t.Fatal("expected error for nil autostart")
	}
	if n.autostartEnabled() {
		t.Fatal("expected autostart disabled")
	}
}

func TestNotifyLinuxCallsSend(t *testing.T) {
	original := sendNotification
	var gotTitle, gotMsg string
	sendNotification = func(title, message string) error {
		gotTitle, gotMsg = title, message
		return nil
	}
	t.Cleanup(func() { sendNotification = original })
	if err := notifyLinux("t", "m"); err != nil {
		t.Fatal(err)
	}
	if gotTitle != "t" || gotMsg != "m" {
		t.Fatalf("title=%q msg=%q", gotTitle, gotMsg)
	}
}

func TestNotifyLinuxPropagatesError(t *testing.T) {
	original := sendNotification
	sendNotification = func(_, _ string) error {
		return errors.New("dbus error")
	}
	t.Cleanup(func() { sendNotification = original })
	if err := notifyLinux("t", "m"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSendNotificationNoService(t *testing.T) {
	original := registeredNotificationService
	registeredNotificationService = nil
	t.Cleanup(func() { registeredNotificationService = original })
	if err := sendNotification("t", "m"); err != nil {
		t.Fatalf("expected silent degradation, got %v", err)
	}
}

func TestGetWailsNotificationService(t *testing.T) {
	original := registeredNotificationService
	registeredNotificationService = nil
	t.Cleanup(func() { registeredNotificationService = original })
	if svc := getWailsNotificationService(); svc != nil {
		t.Fatal("expected nil service")
	}
}

func TestNotifyViaServicePanicSafe(t *testing.T) {
	original := registeredNotificationService
	registeredNotificationService = notifications.New()
	t.Cleanup(func() { registeredNotificationService = original })
	// 服务已创建但 Startup 未执行（dbus 未连接），SendNotification 应被 recover 为错误而非 panic。
	if err := sendNotification("t", "m"); err == nil {
		t.Fatal("expected dbus-uninitialized send to return error")
	}
}
