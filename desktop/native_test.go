package main

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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

func TestSetGNOMEProxyNotAvailable(t *testing.T) {
	oldLook := execLookPath
	t.Cleanup(func() { execLookPath = oldLook })
	execLookPath = func(string) (string, error) {
		return "", errors.New("not found")
	}
	if err := setGNOMEProxy(true); err == nil {
		t.Fatal("expected error for missing gsettings")
	}
}

func TestSetGNOMEProxyDisable(t *testing.T) {
	oldLook := execLookPath
	oldRun := runCommand
	t.Cleanup(func() {
		execLookPath = oldLook
		runCommand = oldRun
	})
	var calls []string
	execLookPath = func(string) (string, error) { return "/usr/bin/gsettings", nil }
	runCommand = func(name string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		return nil, nil
	}
	if err := setGNOMEProxy(false); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || !strings.Contains(calls[0], "org.gnome.system.proxy mode none") {
		t.Fatalf("calls = %v", calls)
	}
}

func TestSetGNOMEProxyEnable(t *testing.T) {
	oldLook := execLookPath
	oldRun := runCommand
	t.Cleanup(func() {
		execLookPath = oldLook
		runCommand = oldRun
	})
	var calls []string
	execLookPath = func(string) (string, error) { return "/usr/bin/gsettings", nil }
	runCommand = func(name string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		return nil, nil
	}
	if err := setGNOMEProxy(true); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 7 {
		t.Fatalf("calls = %d, want 7: %v", len(calls), calls)
	}
}

func TestSetGNOMEProxyGsettingsError(t *testing.T) {
	oldLook := execLookPath
	oldRun := runCommand
	t.Cleanup(func() {
		execLookPath = oldLook
		runCommand = oldRun
	})
	execLookPath = func(string) (string, error) { return "/usr/bin/gsettings", nil }
	runCommand = func(name string, args ...string) ([]byte, error) {
		return []byte("schema not found"), errors.New("failed")
	}
	err := setGNOMEProxy(true)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "schema not found" {
		t.Fatalf("err = %q", err.Error())
	}
}

func TestSystemProxyStatusNonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("linux only test")
	}
	n := NewNativeCapabilities(&desktopRuntime{})
	status, err := n.SystemProxyStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status["available"] != false {
		t.Fatalf("available = %v", status["available"])
	}
}

func TestSystemProxyStatusGsettingsMissing(t *testing.T) {
	oldLook := execLookPath
	t.Cleanup(func() { execLookPath = oldLook })
	execLookPath = func(string) (string, error) {
		return "", errors.New("not found")
	}
	n := NewNativeCapabilities(&desktopRuntime{})
	status, err := n.SystemProxyStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status["available"] != false {
		t.Fatalf("available = %v", status["available"])
	}
}

func TestSystemProxyStatusManual(t *testing.T) {
	oldLook := execLookPath
	oldOut := runOutputCommand
	t.Cleanup(func() {
		execLookPath = oldLook
		runOutputCommand = oldOut
	})
	execLookPath = func(string) (string, error) { return "/usr/bin/gsettings", nil }
	runOutputCommand = func(name string, args ...string) ([]byte, error) {
		return []byte("'manual'\n"), nil
	}
	n := NewNativeCapabilities(&desktopRuntime{})
	status, err := n.SystemProxyStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status["enabled"] != true {
		t.Fatalf("enabled = %v", status["enabled"])
	}
}

func TestSystemProxyStatusNone(t *testing.T) {
	oldLook := execLookPath
	oldOut := runOutputCommand
	t.Cleanup(func() {
		execLookPath = oldLook
		runOutputCommand = oldOut
	})
	execLookPath = func(string) (string, error) { return "/usr/bin/gsettings", nil }
	runOutputCommand = func(name string, args ...string) ([]byte, error) {
		return []byte("'none'\n"), nil
	}
	n := NewNativeCapabilities(&desktopRuntime{})
	status, err := n.SystemProxyStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status["enabled"] != false {
		t.Fatalf("enabled = %v", status["enabled"])
	}
}

func TestSystemProxyStatusOutputError(t *testing.T) {
	oldLook := execLookPath
	oldOut := runOutputCommand
	t.Cleanup(func() {
		execLookPath = oldLook
		runOutputCommand = oldOut
	})
	execLookPath = func(string) (string, error) { return "/usr/bin/gsettings", nil }
	runOutputCommand = func(name string, args ...string) ([]byte, error) {
		return nil, errors.New("failed")
	}
	n := NewNativeCapabilities(&desktopRuntime{})
	status, err := n.SystemProxyStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status["enabled"] != false {
		t.Fatalf("enabled = %v", status["enabled"])
	}
}

func TestNativeSetSystemProxyNonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("linux only test")
	}
	n := NewNativeCapabilities(&desktopRuntime{})
	if err := n.SetSystemProxy(context.Background(), true); err == nil {
		t.Fatal("expected error on non-linux")
	}
}

func TestNativeDataDirConfigPath(t *testing.T) {
	rt := &desktopRuntime{cfg: desktopConfig{DataDir: "/tmp/data", ConfigPath: "/tmp/config.json"}}
	n := NewNativeCapabilities(rt)
	if dir, err := n.DataDir(context.Background()); err != nil || dir != "/tmp/data" {
		t.Fatalf("dir=%q err=%v", dir, err)
	}
	if path, err := n.ConfigPath(context.Background()); err != nil || path != "/tmp/config.json" {
		t.Fatalf("path=%q err=%v", path, err)
	}
}

func TestNativeDataDirNilRT(t *testing.T) {
	n := NewNativeCapabilities(nil)
	if _, err := n.DataDir(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	if _, err := n.ConfigPath(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestNativeResolvePath(t *testing.T) {
	n := NewNativeCapabilities(&desktopRuntime{})
	abs, err := n.ResolvePath(context.Background(), "config.json")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(abs) {
		t.Fatalf("not absolute: %q", abs)
	}
}

func TestNativeResolvePathEmpty(t *testing.T) {
	n := NewNativeCapabilities(&desktopRuntime{})
	if _, err := n.ResolvePath(context.Background(), "  "); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestNativeNotify(t *testing.T) {
	original := sendNotification
	var called bool
	sendNotification = func(_, _ string) error {
		called = true
		return nil
	}
	t.Cleanup(func() { sendNotification = original })
	n := NewNativeCapabilities(&desktopRuntime{})
	if err := n.Notify(context.Background(), "title", "message"); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected sendNotification to be called")
	}
}

func TestNativeSetSystemProxy(t *testing.T) {
	oldLook := execLookPath
	oldRun := runCommand
	t.Cleanup(func() {
		execLookPath = oldLook
		runCommand = oldRun
	})
	execLookPath = func(string) (string, error) { return "/usr/bin/gsettings", nil }
	runCommand = func(name string, args ...string) ([]byte, error) { return nil, nil }
	n := NewNativeCapabilities(&desktopRuntime{})
	if err := n.SetSystemProxy(context.Background(), true); err != nil {
		t.Fatal(err)
	}
}

func TestNativeSystemProxyStatus(t *testing.T) {
	oldLook := execLookPath
	oldOut := runOutputCommand
	t.Cleanup(func() {
		execLookPath = oldLook
		runOutputCommand = oldOut
	})
	execLookPath = func(string) (string, error) { return "/usr/bin/gsettings", nil }
	runOutputCommand = func(name string, args ...string) ([]byte, error) {
		return []byte("'none'\n"), nil
	}
	n := NewNativeCapabilities(&desktopRuntime{})
	if _, err := n.SystemProxyStatus(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRunCommandTimeout(t *testing.T) {
	// 用 sleep 验证超时终止（5s 超时内应快速返回）。
	start := time.Now()
	_, err := runCommand("sleep", "10")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if time.Since(start) > 6*time.Second {
		t.Fatalf("timeout took too long: %v", time.Since(start))
	}
}

func TestRunCommandSuccess(t *testing.T) {
	output, err := runCommand("true")
	if err != nil {
		t.Fatal(err)
	}
	if len(output) != 0 {
		t.Fatalf("unexpected output %q", output)
	}
}
