package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestFormatConfigJSON(t *testing.T) {
	got, err := formatConfigJSON(`{"log":{"level":"info"},"n":1}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "\n  \"log\"") || !strings.HasSuffix(got, "\n") {
		t.Fatalf("formatted JSON = %q", got)
	}
	if _, err := formatConfigJSON(`{"log":`); err == nil {
		t.Fatal("invalid JSON was accepted")
	}
	if _, err := formatConfigJSON(`{"a":1} trailing`); err == nil {
		t.Fatal("trailing JSON was accepted")
	}
}

func TestConfigPageLoadsEditor(t *testing.T) {
	fake := &fakeBackend{configText: "{\n  \"log\": {\"level\": \"info\"}\n}\n"}
	app := newApplication(fake, testConfig(), nil, false)
	app.loadConfigSync()
	if !app.config.loaded || app.config.loading {
		t.Fatalf("config state = loaded %v loading %v", app.config.loaded, app.config.loading)
	}
	if got := app.configEditorState().Buffer.String(); !strings.Contains(got, "\"log\"") {
		t.Fatalf("editor text = %q", got)
	}
	tt := ui.NewTester(app.view, 1440, 900)
	app.router.Push("/config")
	settleTransitions(tt)
	for _, text := range []string{"Config page title", "Raw configuration", "Reload", "Format", "Validate", "Apply"} {
		if !tt.HasText(text) {
			t.Errorf("config page is missing %q", text)
		}
	}
}

func TestConfigActions(t *testing.T) {
	fake := &fakeBackend{}
	app := newApplication(fake, testConfig(), nil, false)
	app.configEditorState().Buffer.Reset(`{"log":{"level":"info"}}`)

	app.validateConfig()
	if fake.callCount("validate-config") != 1 || fake.configBody == "" {
		t.Fatalf("validate calls = %#v", fake.calls)
	}
	app.applyConfig()
	if fake.callCount("apply-config") != 1 {
		t.Fatalf("apply calls = %#v", fake.calls)
	}
	if app.configEditorState().Buffer.Modified() {
		t.Fatal("applied config remained marked as modified")
	}
	if app.model.notice != "Configuration applied" {
		t.Fatalf("notice = %q", app.model.notice)
	}
}

func TestConfigLoadFailureAndRetryGuard(t *testing.T) {
	fake := &fakeBackend{configErr: errors.New("config read failed")}
	app := newApplication(fake, testConfig(), nil, false)
	app.loadConfigSync()
	if !app.config.attempted || app.config.loaded || !strings.Contains(app.model.actionError, "config read failed") {
		t.Fatalf("state = %+v error=%q", app.config, app.model.actionError)
	}
	tt := ui.NewTester(app.view, 1440, 900)
	app.router.Push("/config")
	settleTransitions(tt)
	if got := fake.callCount("load-config"); got != 1 {
		t.Fatalf("load count = %d, want 1", got)
	}
}
