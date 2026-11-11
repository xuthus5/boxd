package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestConfigVisualSchemaAndValues(t *testing.T) {
	if len(configVisualFields(sectionLog)) != 4 || len(configVisualFields(sectionNTP)) == 0 {
		t.Fatalf("visual schemas are missing: log=%d ntp=%d", len(configVisualFields(sectionLog)), len(configVisualFields(sectionNTP)))
	}
	object, ok := decodeSectionObject(`{"log":{"level":"info","timestamp":true}}`)
	if !ok {
		t.Fatal("object did not decode")
	}
	setJSONPath(object, "log.level", "debug")
	setJSONPath(object, "log.cache_file.enabled", true)
	if jsonPathString(object, "log.level") != "debug" || !jsonPathBool(object, "log.cache_file.enabled") {
		t.Fatalf("object = %+v", object)
	}
	deleteJSONPath(object, "log.level")
	if jsonPathValue(object, "log.level") != nil {
		t.Fatalf("level was not deleted: %+v", object)
	}
}

func TestConfigVisualFormRendersAndCommits(t *testing.T) {
	fake := &fakeBackend{sectionText: "{\n  \"level\": \"info\",\n  \"timestamp\": true\n}\n"}
	app := newApplication(fake, testConfig(), nil, false)
	state, ok := app.configSectionForRoute("/advanced/log")
	if !ok {
		t.Fatal("log route was not registered")
	}
	app.loadConfigSection(state)
	tt := ui.NewTester(app.view, 1440, 900)
	app.router.Push("/advanced/log")
	settleTransitions(tt)
	if !tt.HasText("Visual editor") || !tt.HasText("Log level") {
		t.Fatalf("visual form is missing: %v", tt.Texts())
	}
	object, ok := decodeSectionObject(state.editorState().Buffer.String())
	if !ok {
		t.Fatal("editor buffer is not an object")
	}
	app.commitVisualText(state, object, "level", "debug")
	if !strings.Contains(state.editorState().Buffer.String(), `"debug"`) || !state.editorState().Buffer.Modified() {
		t.Fatalf("committed buffer = %q modified=%v", state.editorState().Buffer.String(), state.editorState().Buffer.Modified())
	}
}

func TestConfigVisualFormRejectsInvalidNumberAndPreservesUnknownFields(t *testing.T) {
	fake := &fakeBackend{sectionText: "{\n  \"level\": \"info\",\n  \"x-unknown\": {\"keep\": true}\n}\n"}
	app := newApplication(fake, testConfig(), nil, false)
	state, ok := app.configSectionForRoute("/advanced/log")
	if !ok {
		t.Fatal("log route was not registered")
	}
	app.loadConfigSection(state)
	before := state.editorState().Buffer.String()
	object, ok := decodeSectionObject(before)
	if !ok {
		t.Fatal("editor buffer is not an object")
	}
	app.commitVisualNumber(state, object, "sample_rate", "not-a-number")
	if state.editorState().Buffer.String() != before {
		t.Fatalf("invalid number changed buffer: %q", state.editorState().Buffer.String())
	}
	if app.model.actionError == "" {
		t.Fatal("invalid number did not set an action error")
	}
	app.commitVisualText(state, object, "level", "debug")
	if !strings.Contains(state.editorState().Buffer.String(), `"x-unknown"`) ||
		!strings.Contains(state.editorState().Buffer.String(), `"keep": true`) {
		t.Fatalf("unknown fields were lost: %q", state.editorState().Buffer.String())
	}
}
