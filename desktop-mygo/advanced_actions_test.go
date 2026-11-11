package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/xuthus5/boxd/internal/model"
)

func TestConfigSectionActions(t *testing.T) {
	fake := &fakeBackend{sectionText: "{\n  \"level\": \"info\"\n}\n"}
	app := newApplication(fake, testConfig(), nil, false)
	state, ok := app.configSectionForRoute("/advanced/log")
	if !ok {
		t.Fatal("log section route was not registered")
	}
	app.loadConfigSection(state)
	if !state.loaded || state.loading || !strings.Contains(state.editorState().Buffer.String(), "info") {
		t.Fatalf("section state = %+v text=%q", state, state.editorState().Buffer.String())
	}
	app.formatConfigSection(state)
	if !strings.Contains(state.editorState().Buffer.String(), "\n  \"level\"") {
		t.Fatalf("formatted section = %q", state.editorState().Buffer.String())
	}
	app.validateConfigSection(state)
	if fake.callCount("validate-config-section") != 1 || fake.lastSection != sectionLog {
		t.Fatalf("validate calls = %+v", fake.calls)
	}
	app.applyConfigSection(state)
	if fake.callCount("apply-config-section") != 1 || state.editorState().Buffer.Modified() {
		t.Fatalf("apply calls = %+v modified=%v", fake.calls, state.editorState().Buffer.Modified())
	}
	fake.actionErr = errors.New("apply failed")
	state.editorState().Buffer.Set(`{"level":"debug"}`)
	app.applyConfigSection(state)
	if !strings.Contains(app.model.actionError, "apply failed") || !state.editorState().Buffer.Modified() {
		t.Fatalf("failure state = error %q modified %v", app.model.actionError, state.editorState().Buffer.Modified())
	}
}

func TestConfigSectionDefaultsAndDiagnostics(t *testing.T) {
	fake := &fakeBackend{}
	app := newApplication(fake, testConfig(), nil, false)
	state, ok := app.configSectionForRoute("/policy/dns")
	if !ok {
		t.Fatal("dns route was not registered")
	}
	state.attempted = true
	app.installConfigSectionDefaults(state)
	if fake.callCount("install-config-defaults") != 1 || fake.callCount("load-config-section") != 1 {
		t.Fatalf("defaults calls = %+v", fake.calls)
	}
	fake.diagnostics = model.ConfigDiagnostics{
		Status:  model.ConfigDiagnosticsWarning,
		Summary: model.ConfigDiagnosticsSummary{Errors: 1, Warnings: 2},
		Counts:  model.ConfigDiagnosticsCounts{Inbounds: 1, Outbounds: 2, RouteRules: 3},
	}
	app.loadConfigDiagnostics()
	if !app.config.diagnosticsLoaded || app.config.diagnostics.Status != model.ConfigDiagnosticsWarning {
		t.Fatalf("diagnostics = %+v", app.config)
	}
}

func TestAdvancedActionFailureStates(t *testing.T) {
	fake := &fakeBackend{sectionErr: errors.New("load failed")}
	app := newApplication(fake, testConfig(), nil, false)
	state, ok := app.configSectionForRoute("/advanced/log")
	if !ok {
		t.Fatal("log section route was not registered")
	}
	app.loadConfigSection(state)
	if !strings.Contains(app.model.actionError, "load failed") || state.loaded || state.loading {
		t.Fatalf("load failure = error %q state %+v", app.model.actionError, state)
	}

	fake.actionErr = errors.New("rule-set save failed")
	app.model.actionError = ""
	app.ruleSets.savedAuto = model.RuleSetAutoUpdate{Interval: "24h"}
	app.ruleSets.draftAuto = app.ruleSets.savedAuto
	app.ruleSets.autoLoaded = true
	app.setRuleSetAutoEnabled(true)
	if !strings.Contains(app.model.actionError, "rule-set save failed") || app.ruleSets.savedAuto.Enabled || app.ruleSets.draftAuto.Enabled {
		t.Fatalf("rule-set rollback = error %q state %+v", app.model.actionError, app.ruleSets)
	}

	app.model.actionError = ""
	app.securityForm.current = "old"
	app.securityForm.next = "new"
	app.securityForm.confirm = "new"
	app.savePassword()
	if !strings.Contains(app.model.actionError, "rule-set save failed") || app.securityForm.next != "new" {
		t.Fatalf("password failure = error %q form %+v", app.model.actionError, app.securityForm)
	}
}

func TestConfigHistoryActions(t *testing.T) {
	event := model.ConfigApplyEvent{ID: "snap-1", Source: "desktop", Status: "applied", Restorable: true, Size: 42}
	fake := &fakeBackend{history: []model.ConfigApplyEvent{event}, snapshotText: "{\n  \"log\": {}\n}\n"}
	app := newApplication(fake, testConfig(), nil, false)
	app.loadConfigHistory(false)
	if !app.configHistory.loaded || len(app.configHistory.events) != 1 {
		t.Fatalf("history state = %+v", app.configHistory)
	}
	app.openConfigSnapshot(event)
	if !app.configHistory.previewOpen || !strings.Contains(app.configHistory.previewState().Buffer.String(), "log") {
		t.Fatalf("preview state = %+v", app.configHistory)
	}
	app.askRestoreConfigSnapshot(event.ID)
	if !app.deleteForm.open || app.deleteForm.kind != "config-restore" || app.configHistory.previewOpen {
		t.Fatalf("restore form = %+v preview=%v", app.deleteForm, app.configHistory.previewOpen)
	}
	app.confirmDelete()
	if fake.callCount("restore-config") != 1 {
		t.Fatalf("restore calls = %+v", fake.calls)
	}
}

func TestRuleSetActions(t *testing.T) {
	fake := &fakeBackend{
		ruleSets:    []model.RuleSetStatusItem{{Tag: "geosite", Type: "remote", Updatable: true}},
		ruleSetAuto: model.RuleSetAutoUpdate{Enabled: false, Interval: "24h"},
	}
	app := newApplication(fake, testConfig(), nil, false)
	app.loadRuleSets(false)
	if !app.ruleSets.loaded || !app.ruleSets.autoLoaded || len(app.ruleSets.items) != 1 {
		t.Fatalf("rule-set state = %+v", app.ruleSets)
	}
	app.updateRuleSets([]string{"geosite"})
	if fake.lastRuleSet.Tags[0] != "geosite" {
		t.Fatalf("update request = %+v", fake.lastRuleSet)
	}
	app.setRuleSetAutoEnabled(true)
	if !app.ruleSets.savedAuto.Enabled || !fake.lastAuto.Enabled {
		t.Fatalf("auto update state = %+v fake=%+v", app.ruleSets, fake.lastAuto)
	}
	app.saveRuleSetAutoInterval("12h")
	if app.ruleSets.savedAuto.Interval != "12h" || fake.lastAuto.Interval != "12h" {
		t.Fatalf("auto interval state = %+v fake=%+v", app.ruleSets, fake.lastAuto)
	}
}

func TestPasswordFormValidationAndSave(t *testing.T) {
	fake := &fakeBackend{}
	app := newApplication(fake, testConfig(), nil, false)
	app.securityForm.current = "old"
	app.securityForm.next = "new"
	app.securityForm.confirm = "different"
	app.savePassword()
	if !strings.Contains(app.model.actionError, "do not match") || fake.callCount("change-password") != 0 {
		t.Fatalf("mismatch state = error %q calls %v", app.model.actionError, fake.calls)
	}
	app.securityForm.confirm = "new"
	app.savePassword()
	if fake.lastCurrent != "old" || fake.lastNew != "new" || app.securityForm.next != "" {
		t.Fatalf("password state = form %+v fake=%+v", app.securityForm, fake)
	}
}

func TestAdvancedPagesRender(t *testing.T) {
	fake := &fakeBackend{
		sectionText: "{\n  \"level\": \"info\"\n}\n",
		diagnostics: model.ConfigDiagnostics{Status: model.ConfigDiagnosticsHealthy},
		history:     []model.ConfigApplyEvent{{ID: "1", Source: "desktop", Status: "applied", Restorable: true}},
		ruleSets:    []model.RuleSetStatusItem{{Tag: "geo", Type: "remote", Updatable: true}},
		ruleSetAuto: model.RuleSetAutoUpdate{Interval: "24h"},
	}
	app := newApplication(fake, testConfig(), nil, false)
	app.model.loaded = true
	tt := ui.NewTester(app.view, 1440, 900)
	for _, page := range []struct {
		path string
		text string
	}{
		{"/proxy/inbounds", "Inbounds page title"},
		{"/policy/route", "Route page title"},
		{"/advanced/log", "Log page title"},
		{"/advanced/history", "Config history"},
		{"/advanced/rule-sets", "Rule sets"},
	} {
		app.router.Push(page.path)
		settleTransitions(tt)
		if !tt.HasText(page.text) {
			t.Errorf("page %s is missing %q; texts=%v", page.path, page.text, tt.Texts())
		}
	}
	app.router.Push("/settings")
	settleTransitions(tt)
	for _, text := range []string{"Account security", "Backup", "Export backup"} {
		if !tt.HasText(text) {
			t.Errorf("settings is missing %q", text)
		}
	}
}
