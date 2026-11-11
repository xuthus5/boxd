package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

func TestKernelActions(t *testing.T) {
	fake := &fakeBackend{}
	app := newApplication(fake, testConfig(), nil, false)

	app.startKernelAction("")
	app.stopKernelAction()
	app.restartKernelAction()
	if fake.callCount("start") != 1 || fake.callCount("stop") != 1 || fake.callCount("restart") != 1 {
		t.Fatalf("calls = %#v", fake.calls)
	}
	if app.model.busy || app.model.notice != "Kernel restarted" {
		t.Fatalf("action state = busy %v notice %q", app.model.busy, app.model.notice)
	}
}

func TestItemActionsAndFailureRollback(t *testing.T) {
	fake := &fakeBackend{}
	app := newApplication(fake, testConfig(), nil, false)
	app.model.testURL = "https://old.example/"
	app.model.kernelAutostart = false
	app.model.prefs = model.UIPreferences{Theme: themeLight, Language: "zh", MinimumLogLevel: "all"}

	fake.actionErr = errors.New("backend unavailable")
	app.setTestURL("https://new.example/")
	if app.model.testURL != "https://old.example/" || !strings.Contains(app.model.actionError, "backend unavailable") {
		t.Fatalf("test URL rollback = %q error=%q", app.model.testURL, app.model.actionError)
	}
	app.setKernelAutostart(true)
	if app.model.kernelAutostart {
		t.Fatal("kernel autostart was not rolled back")
	}
	next := app.model.prefs
	next.Theme = themeDark
	app.updateUIPreferences(next)
	if app.model.prefs.Theme != themeLight {
		t.Fatalf("preferences were not rolled back: %+v", app.model.prefs)
	}

	fake.actionErr = nil
	app.setTestURL("https://new.example/")
	app.setKernelAutostart(true)
	app.updateUIPreferences(next)
	if app.model.testURL != "https://new.example/" || !app.model.kernelAutostart || app.model.prefs.Theme != themeDark {
		t.Fatalf("successful settings state = %+v", app.model)
	}
	if fake.callCount("test-url") != 2 || fake.callCount("kernel-autostart") != 2 || fake.callCount("preferences") != 2 {
		t.Fatalf("setting calls = %#v", fake.calls)
	}
}

func TestSettingsIgnoreChangesWhileBusy(t *testing.T) {
	fake := &fakeBackend{}
	app := newApplication(fake, testConfig(), nil, false)
	app.model.testURL = "https://old.example/"
	app.model.kernelAutostart = false
	app.model.busy = true
	app.setTestURL("https://new.example/")
	app.setKernelAutostart(true)
	if app.model.testURL != "https://old.example/" || app.model.kernelAutostart {
		t.Fatalf("busy settings changed: url=%q autostart=%v", app.model.testURL, app.model.kernelAutostart)
	}
	if fake.callCount("test-url") != 0 || fake.callCount("kernel-autostart") != 0 {
		t.Fatalf("busy settings called backend: %#v", fake.calls)
	}
}

func TestSubscriptionAndConnectionActions(t *testing.T) {
	fake := &fakeBackend{}
	app := newApplication(fake, testConfig(), nil, false)
	app.refreshSubscription("sub-1")
	app.closeConnection(42)
	if fake.callCount("refresh-subscription") != 1 || fake.callCount("close-connection") != 1 {
		t.Fatalf("calls = %#v", fake.calls)
	}
	if app.model.notice != "Connection closed" {
		t.Fatalf("notice = %q", app.model.notice)
	}
}

func TestHandleURLWithoutWindow(t *testing.T) {
	fake := &fakeBackend{}
	app := newApplication(fake, testConfig(), nil, false)
	app.handleURL("boxd://import?link=vless%3A%2F%2Fexample")
	if fake.callCount("import-link") != 1 {
		t.Fatalf("valid URL did not import: %#v", fake.calls)
	}

	app.handleURL("https://example.test")
	if !strings.Contains(app.model.actionError, "unsupported deep link scheme") {
		t.Fatalf("invalid URL error = %q", app.model.actionError)
	}
}

func TestFinishActionRollback(t *testing.T) {
	app := newApplication(&fakeBackend{}, testConfig(), nil, false)
	app.model.busy = true
	app.model.busyLabel = "working"
	app.finishAction(errors.New("boom"), "unused", func() { app.model.notice = "rolled back" })
	if app.model.busy || app.model.notice != "rolled back" || app.model.actionError == "" {
		t.Fatalf("finish state = %+v", app.model)
	}
}
