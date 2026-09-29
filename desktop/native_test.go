package main

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

// fakeAutostart 记录自启注册调用，供原生能力单测注入。
type fakeAutostart struct {
	enabled   bool
	enableErr error
	args      []string
	enables   int
	disables  int
}

func (f *fakeAutostart) IsEnabled() (bool, error) { return f.enabled, nil }

func (f *fakeAutostart) EnableWithOptions(opts application.AutostartOptions) error {
	f.enables++
	f.args = opts.Arguments
	f.enabled = true
	return f.enableErr
}

func (f *fakeAutostart) Disable() error {
	f.disables++
	f.enabled = false
	return nil
}

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

func TestHasHiddenFlag(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{name: "empty", args: nil, want: false},
		{name: "unrelated", args: []string{"--desktop-mode", "remote"}, want: false},
		{name: "present", args: []string{"--hidden"}, want: true},
		{name: "present among others", args: []string{"--foo", "--hidden", "--bar"}, want: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasHiddenFlag(tt.args); got != tt.want {
				t.Fatalf("hasHiddenFlag(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestSetAutostartRegistersHiddenFlag(t *testing.T) {
	fake := &fakeAutostart{}
	rt := &desktopRuntime{autostart: fake}
	n := NewNativeCapabilities(rt)
	if err := n.SetAutostart(context.Background(), true); err != nil {
		t.Fatalf("SetAutostart(true) error = %v", err)
	}
	if fake.enables != 1 {
		t.Fatalf("enable calls = %d, want 1", fake.enables)
	}
	if len(fake.args) != 1 || fake.args[0] != autostartHiddenFlag {
		t.Fatalf("autostart args = %v, want [%s]", fake.args, autostartHiddenFlag)
	}
}

func TestSetAutostartDisable(t *testing.T) {
	fake := &fakeAutostart{enabled: true}
	rt := &desktopRuntime{autostart: fake}
	n := NewNativeCapabilities(rt)
	if err := n.SetAutostart(context.Background(), false); err != nil {
		t.Fatalf("SetAutostart(false) error = %v", err)
	}
	if fake.disables != 1 || fake.enabled {
		t.Fatalf("disable calls = %d, enabled = %v", fake.disables, fake.enabled)
	}
}

func TestEnsureAutostartHiddenArg(t *testing.T) {
	t.Run("nil runtime", func(t *testing.T) {
		if err := NewNativeCapabilities(nil).ensureAutostartHiddenArg(); err != nil {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("disabled is untouched", func(t *testing.T) {
		fake := &fakeAutostart{}
		rt := &desktopRuntime{autostart: fake}
		if err := NewNativeCapabilities(rt).ensureAutostartHiddenArg(); err != nil {
			t.Fatalf("error = %v", err)
		}
		if fake.enables != 0 {
			t.Fatalf("enable calls = %d, want 0", fake.enables)
		}
	})

	t.Run("enabled is re-registered with flag", func(t *testing.T) {
		fake := &fakeAutostart{enabled: true}
		rt := &desktopRuntime{autostart: fake}
		if err := NewNativeCapabilities(rt).ensureAutostartHiddenArg(); err != nil {
			t.Fatalf("error = %v", err)
		}
		if fake.enables != 1 {
			t.Fatalf("enable calls = %d, want 1", fake.enables)
		}
		if len(fake.args) != 1 || fake.args[0] != autostartHiddenFlag {
			t.Fatalf("autostart args = %v, want [%s]", fake.args, autostartHiddenFlag)
		}
	})

	t.Run("enable error is propagated", func(t *testing.T) {
		fake := &fakeAutostart{enabled: true, enableErr: errors.New("boom")}
		rt := &desktopRuntime{autostart: fake}
		if err := NewNativeCapabilities(rt).ensureAutostartHiddenArg(); err == nil {
			t.Fatal("expected error")
		}
	})
}
