package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestConfigItemVisualSchemas(t *testing.T) {
	for _, test := range []struct {
		section string
		path    string
		label   string
	}{
		{sectionRoute, "rules", "Destination ports"},
		{sectionRoute, "rule_set", "Remote URL"},
		{sectionDNS, "servers", "Server port"},
		{sectionDNS, "rules", "Query types"},
	} {
		fields := configItemVisualFields(test.section, test.path)
		found := false
		for _, field := range fields {
			if field.Label == test.label {
				found = true
			}
		}
		if !found {
			t.Errorf("%s/%s is missing %q: %+v", test.section, test.path, test.label, fields)
		}
	}
	if fields := configItemVisualFields(sectionInbounds, ""); len(fields) != 0 {
		t.Fatalf("unexpected inbound item fields: %+v", fields)
	}
}

func TestConfigItemVisualDialogRendersCommonFields(t *testing.T) {
	fake := &fakeBackend{sectionText: `{
  "rules": [
    {"action": "route", "outbound": "proxy", "domain": ["example.com"], "x-unknown": {"keep": true}}
  ]
}`}
	app := newApplication(fake, testConfig(), nil, false)
	state, ok := app.configSectionForRoute("/policy/route")
	if !ok {
		t.Fatal("route section is missing")
	}
	state.itemPath = "rules"
	state.itemDraft = `{"action":"route","outbound":"proxy","domain":["example.com"],"x-unknown":{"keep":true}}`
	state.itemEditOpen = true
	tt := ui.NewTester(app.view, 1440, 900)
	app.router.Push("/policy/route")
	settleTransitions(tt)
	for _, text := range []string{"Common fields", "Action", "Domains", "Destination ports", "Invert match"} {
		if !tt.HasText(text) {
			t.Errorf("item dialog is missing %q: %v", text, tt.Texts())
		}
	}
}

func TestConfigItemVisualCommitsPreserveUnknownFields(t *testing.T) {
	state, item := newConfigItemVisualTestState(t)
	before := state.itemDraft
	app := newApplication(&fakeBackend{}, testConfig(), nil, false)
	app.commitConfigItemStringList(state, item, "domain", "example.com, *.internal")
	if !strings.Contains(state.itemDraft, `"example.com"`) || !strings.Contains(state.itemDraft, `"*.internal"`) {
		t.Fatalf("string list was not written: %s", state.itemDraft)
	}
	app.commitConfigItemNumberList(state, item, "port", "443, 8443-8445")
	if !strings.Contains(state.itemDraft, `443`) || !strings.Contains(state.itemDraft, `"8443-8445"`) {
		t.Fatalf("number list was not written: %s", state.itemDraft)
	}
	app.commitConfigItemVisual(state, item, "invert", true, false)
	if !strings.Contains(state.itemDraft, `"invert": true`) {
		t.Fatalf("boolean was not written: %s", state.itemDraft)
	}
	app.commitConfigItemVisual(state, item, "outbound", nil, true)
	if strings.Contains(state.itemDraft, `"outbound"`) || !strings.Contains(state.itemDraft, `"x-unknown"`) {
		t.Fatalf("delete or unknown-field preservation failed: %s", state.itemDraft)
	}
	if state.itemDraft == before {
		t.Fatal("draft did not change")
	}
}

func TestConfigItemVisualRejectsInvalidValues(t *testing.T) {
	state, item := newConfigItemVisualTestState(t)
	app := newApplication(&fakeBackend{}, testConfig(), nil, false)
	before := state.itemDraft
	for _, value := range []string{"443, nope", "443, abc-def", "443, 8443-8000", "0", "70000"} {
		state.itemErr = ""
		app.commitConfigItemNumberList(state, item, "port", value)
		if state.itemDraft != before || state.itemErr == "" {
			t.Fatalf("invalid number list %q changed draft or missed error: draft=%q err=%q", value, state.itemDraft, state.itemErr)
		}
	}
	for _, value := range []string{"not-a-port", "0", "70000"} {
		state.itemErr = ""
		app.commitConfigItemNumber(state, item, "server_port", value)
		if state.itemDraft != before || state.itemErr == "" {
			t.Fatalf("invalid number %q changed draft or missed error: draft=%q err=%q", value, state.itemDraft, state.itemErr)
		}
	}
}

func TestConfigItemVisualNormalizesPortRanges(t *testing.T) {
	state, item := newConfigItemVisualTestState(t)
	app := newApplication(&fakeBackend{}, testConfig(), nil, false)
	app.commitConfigItemNumberList(state, item, "port", "443, 8443 - 8445")
	if !strings.Contains(state.itemDraft, `"8443-8445"`) || !strings.Contains(state.itemDraft, `443`) {
		t.Fatalf("port range was not normalized: %s", state.itemDraft)
	}
}

func newConfigItemVisualTestState(t *testing.T) (*configSectionState, map[string]any) {
	t.Helper()
	state := newConfigSectionState(configSectionSpec{Key: sectionRoute, Route: "/policy/route", Title: "Route"})
	state.loaded = true
	state.itemPath = "rules"
	state.itemDraft = `{"action":"route","outbound":"proxy","x-unknown":{"keep":true}}`
	state.editorState().Buffer.Set(`{"rules":[` + state.itemDraft + `]}`)
	item, err := parseConfigSectionItem(state.itemDraft)
	if err != nil {
		t.Fatal(err)
	}
	return state, item
}
