package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestConfigSectionItemsEditAddAndDelete(t *testing.T) {
	fake := &fakeBackend{sectionText: `[
  {"tag":"mixed-in","type":"mixed","listen":"0.0.0.0","listen_port":1080,"x-keep":true},
  {"tag":"http-in","type":"http","listen":"127.0.0.1","listen_port":8080}
]`}
	app := newApplication(fake, testConfig(), nil, false)
	state, ok := app.configSectionForRoute("/proxy/inbounds")
	if !ok {
		t.Fatal("inbounds route was not registered")
	}
	app.loadConfigSection(state)
	tt := ui.NewTester(app.view, 1440, 900)
	app.router.Push("/proxy/inbounds")
	settleTransitions(tt)
	for _, text := range []string{"Inbounds items", "2 configured item(s)", "mixed-in", "http-in", "Add item"} {
		if !tt.HasText(text) {
			t.Errorf("items card is missing %q: %v", text, tt.Texts())
		}
	}

	app.openConfigSectionItemEditor(state, -1)
	settleTransitions(tt)
	if !tt.HasText("Save item") {
		t.Fatalf("item editor did not open: %v", tt.Texts())
	}
	state.itemDraft = `{"tag":"socks-in","type":"socks","listen":"127.0.0.1","listen_port":1081}`
	app.saveConfigSectionItem(state)
	items, ok := configSectionObjectItems(state)
	if !ok || len(items) != 3 || jsonPathString(items[2], "tag") != "socks-in" {
		t.Fatalf("items after add = %+v, ok=%v", items, ok)
	}
	if !state.editorState().Buffer.Modified() || strings.Contains(state.itemErr, "must be") {
		t.Fatalf("state after add = %+v", state)
	}

	app.openConfigSectionItemEditor(state, 0)
	state.itemDraft = `{"tag":"mixed-in","type":"mixed","listen":"127.0.0.1","listen_port":1080,"x-keep":true}`
	app.saveConfigSectionItem(state)
	items, ok = configSectionObjectItems(state)
	if !ok || jsonPathString(items[0], "listen") != "127.0.0.1" || !jsonPathBool(items[0], "x-keep") {
		t.Fatalf("items after edit = %+v, ok=%v", items, ok)
	}

	app.askDeleteConfigSectionItem(state, 1, "http-in")
	if !app.deleteForm.open || app.deleteForm.kind != "config-section-item" {
		t.Fatalf("delete form = %+v", app.deleteForm)
	}
	app.confirmDelete()
	items, ok = configSectionObjectItems(state)
	if !ok || len(items) != 2 {
		t.Fatalf("items after delete = %+v, ok=%v", items, ok)
	}
}

func TestConfigSectionItemValidation(t *testing.T) {
	fake := &fakeBackend{sectionText: `[{"tag":"mixed-in","type":"mixed"}]`}
	app := newApplication(fake, testConfig(), nil, false)
	state, _ := app.configSectionForRoute("/proxy/inbounds")
	app.loadConfigSection(state)
	before := state.editorState().Buffer.String()
	for _, draft := range []string{`[]`, `null`, `{"tag":`} {
		state.itemDraft = draft
		app.saveConfigSectionItem(state)
		if state.itemErr == "" || state.editorState().Buffer.String() != before {
			t.Fatalf("invalid item %q changed state: err=%q buffer=%q", draft, state.itemErr, state.editorState().Buffer.String())
		}
	}
	item, err := parseConfigSectionItem(`{"tag":"mixed-in","nested":{"keep":true}}`)
	if err != nil || jsonPathString(item, "tag") != "mixed-in" {
		t.Fatalf("parsed item = %+v err=%v", item, err)
	}
}

func TestConfigSectionItemsIgnoreNonArraySections(t *testing.T) {
	fake := &fakeBackend{sectionText: `{"level":"info"}`}
	app := newApplication(fake, testConfig(), nil, false)
	state, _ := app.configSectionForRoute("/advanced/log")
	app.loadConfigSection(state)
	if _, ok := configSectionObjectItems(state); ok {
		t.Fatal("object section was treated as an item array")
	}
}
