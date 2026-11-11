package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestCompareConfigJSON(t *testing.T) {
	items, err := compareConfigJSON(
		`{"log":{"level":"info"},"dns":{"servers":["a"]},"route":{"final":"old"}}`,
		`{"log":{"level":"debug","timestamp":true},"dns":{"servers":["a","b"]},"route":{}}`,
	)
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]configDiffKind, len(items))
	for _, item := range items {
		got[item.Path] = item.Kind
	}
	for path, kind := range map[string]configDiffKind{
		"log.level":      configDiffChanged,
		"log.timestamp":  configDiffAdded,
		"dns.servers[1]": configDiffAdded,
		"route.final":    configDiffRemoved,
	} {
		if got[path] != kind {
			t.Errorf("diff %s = %q, want %q; items=%+v", path, got[path], kind, items)
		}
	}
}

func TestCompareConfigJSONNoSemanticChangeAndErrors(t *testing.T) {
	items, err := compareConfigJSON(`{"log":{"level":"info"}}`, "{\n  \"log\": {\"level\": \"info\"}\n}\n")
	if err != nil || len(items) != 0 {
		t.Fatalf("formatting produced diff: items=%+v err=%v", items, err)
	}
	if _, err := compareConfigJSON(`{"log":`, `{}`); err == nil {
		t.Fatal("invalid baseline was accepted")
	}
	if _, err := compareConfigJSON(`{}`, `{"log":`); err == nil || !strings.Contains(err.Error(), "not valid JSON") {
		t.Fatalf("invalid draft error = %v", err)
	}
}

func TestConfigDiffAndResetView(t *testing.T) {
	fake := &fakeBackend{configText: "{\n  \"log\": {\"level\": \"info\"}\n}\n"}
	app := newApplication(fake, testConfig(), nil, false)
	app.loadConfigSync()
	tt := ui.NewTester(app.view, 1440, 900)
	app.router.Push("/config")
	settleTransitions(tt)
	if !tt.HasText("No changes compared with the loaded configuration.") {
		t.Fatalf("empty diff is missing: %v", tt.Texts())
	}
	app.configEditorState().Buffer.Reset(`{"log":{"level":"debug"}}`)
	app.invalidate()
	settleTransitions(tt)
	if !tt.HasText("log.level") || !tt.HasText("CHANGED") {
		t.Fatalf("changed diff is missing: %v", tt.Texts())
	}
	app.resetConfigDraft()
	if app.configEditorState().Buffer.String() != app.config.baseline || app.configEditorState().Buffer.Modified() {
		t.Fatalf("reset buffer = %q modified=%v", app.configEditorState().Buffer.String(), app.configEditorState().Buffer.Modified())
	}
}

func TestConfigSectionDiffAndReset(t *testing.T) {
	fake := &fakeBackend{sectionText: "{\n  \"level\": \"info\"\n}\n"}
	app := newApplication(fake, testConfig(), nil, false)
	state, ok := app.configSectionForRoute("/advanced/log")
	if !ok {
		t.Fatal("log route was not registered")
	}
	app.loadConfigSection(state)
	tt := ui.NewTester(app.view, 1440, 900)
	app.router.Push("/advanced/log")
	settleTransitions(tt)
	if !tt.HasText("Log diff") || !tt.HasText("No changes compared with the loaded configuration.") {
		t.Fatalf("section diff is missing: %v", tt.Texts())
	}
	state.editorState().Buffer.Set(`{"level":"debug"}`)
	app.invalidate()
	settleTransitions(tt)
	if !tt.HasText("level") || !tt.HasText("CHANGED") || !configSectionCanReset(state) {
		t.Fatalf("section change is missing: %v", tt.Texts())
	}
	app.resetConfigSection(state)
	if state.editorState().Buffer.String() != state.baseline || state.editorState().Buffer.Modified() {
		t.Fatalf("section reset = %q modified=%v", state.editorState().Buffer.String(), state.editorState().Buffer.Modified())
	}
}
