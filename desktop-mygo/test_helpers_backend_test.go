package main

import (
	"context"

	"github.com/xuthus5/boxd/internal/model"

	"github.com/xuthus5/boxd/desktop-mygo/internal/runtime"
)

func (f *fakeBackend) SetKernelAutostart(context.Context, bool) error {
	f.record("kernel-autostart")
	return f.actionErr
}

func (f *fakeBackend) SetUIPreferences(_ context.Context, prefs model.UIPreferences) error {
	f.record("preferences")
	f.mu.Lock()
	f.lastPrefs = prefs
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) SetTestURL(context.Context, string) error {
	f.record("test-url")
	return f.actionErr
}

func (f *fakeBackend) ChangePassword(_ context.Context, currentPassword, newPassword string) error {
	f.record("change-password")
	f.mu.Lock()
	f.lastCurrent = currentPassword
	f.lastNew = newPassword
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) CreateBackup(_ context.Context, path string) error {
	f.record("create-backup")
	f.mu.Lock()
	f.lastPath = path
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) ImportLink(context.Context, string) error {
	f.record("import-link")
	return f.actionErr
}

func (f *fakeBackend) SupportBundle(context.Context) (supportBundleCollection, error) {
	f.record("support-bundle")
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.supportBundle, f.supportBundleErr
}

func (f *fakeBackend) record(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, name)
}

func (f *fakeBackend) callCount(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, call := range f.calls {
		if call == name {
			count++
		}
	}
	return count
}

func testConfig() runtime.Config {
	return runtime.Config{
		Mode: "embedded", DataDir: "/tmp/boxd-test", ConfigPath: "/tmp/boxd-test/config.json",
		RefreshInterval: 1,
	}
}
