package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"
)

func TestSetupDraftRequestValidation(t *testing.T) {
	app := newApplication(&fakeBackend{}, testConfig(), nil, false)
	app.setup.status = core.SetupStatus{
		Modules: []core.ConfigModuleStatus{
			{ID: "outbounds", State: "ready", Count: 2},
			{ID: "inbounds", State: "missing"},
		},
		Listeners: []core.SetupListener{{Type: "mixed", Listen: "127.0.0.1", ListenPort: 1080}},
	}
	app.resetSetupDraft()
	if app.setup.inbound != setupInboundPreserve || app.setup.ipv6 != setupIPv6Auto {
		t.Fatalf("draft defaults = inbound %q ipv6 %q", app.setup.inbound, app.setup.ipv6)
	}
	request, err := app.setupRequest()
	if err != nil {
		t.Fatal(err)
	}
	if len(request.Modules) != len(setupModuleIDs) || request.InboundMode != setupInboundPreserve {
		t.Fatalf("setup request = %+v", request)
	}

	delete(app.setup.modules, "inbounds")
	request, err = app.setupRequest()
	if err != nil || request.InboundMode != setupInboundPreserve {
		t.Fatalf("inbounds-off request = %+v err=%v", request, err)
	}
	app.setup.modules = map[string]bool{}
	if _, err := app.setupRequest(); err == nil {
		t.Fatal("empty module selection was accepted")
	}

	app.resetSetupDraft()
	app.setup.status.ConfigError = "invalid JSON"
	if _, err := app.setupRequest(); err == nil {
		t.Fatal("invalid configuration was accepted without reset consent")
	}
	app.setup.reset = true
	request, err = app.setupRequest()
	if err != nil || !request.ResetInvalid {
		t.Fatalf("reset request = %+v err=%v", request, err)
	}
}

func TestSetupStatusFailureRetainsPreviousStatus(t *testing.T) {
	fake := &fakeBackend{setupStatus: core.SetupStatus{ProxyReady: true, KernelRunning: true}}
	app := newApplication(fake, testConfig(), nil, false)
	app.loadSetup(false)
	if !app.setup.loaded || !app.setup.status.ProxyReady {
		t.Fatalf("initial setup = %+v", app.setup)
	}
	fake.actionErr = errors.New("setup unavailable")
	app.loadSetup(true)
	if !app.setup.loaded || !app.setup.status.ProxyReady || !strings.Contains(app.setup.err, "setup unavailable") {
		t.Fatalf("failed refresh = %+v", app.setup)
	}
	fake.actionErr = nil
	fake.setupStatus = core.SetupStatus{KernelRunning: true}
	app.loadSetup(true)
	if !app.setup.status.KernelRunning || app.setup.status.ProxyReady || app.setup.err != "" {
		t.Fatalf("successful refresh = %+v", app.setup)
	}
}

func TestSetupPreviewAndApply(t *testing.T) {
	fake := &fakeBackend{
		setupPlan: &core.SetupPlan{
			SourceHash: "preview-hash", Modules: []string{"outbounds", "inbounds"},
			Warnings: []string{"kernel restart required"}, WillRestart: true,
		},
		setupApplyResult: service.SetupApplyResult{
			ApplyResult: service.ApplyResult{Status: "ok"},
			Modules:     []string{"outbounds", "inbounds"}, ConfigHash: "next-hash",
		},
	}
	app := newApplication(fake, testConfig(), nil, false)
	app.setup.status = core.SetupStatus{Modules: []core.ConfigModuleStatus{{ID: "outbounds", State: "ready"}}}
	app.resetSetupDraft()
	app.setup.dialogOpen = true

	app.previewSetup()
	if app.setup.preview == nil || app.setup.previewing || fake.callCount("setup-preview") != 1 {
		t.Fatalf("preview state = %+v calls=%v", app.setup, fake.calls)
	}
	if len(fake.lastSetupPreview.Modules) != len(setupModuleIDs) {
		t.Fatalf("preview request = %+v", fake.lastSetupPreview)
	}

	app.applySetup()
	if fake.callCount("setup-apply") != 1 || fake.lastSetupApply.SourceHash != "preview-hash" {
		t.Fatalf("apply request = %+v calls=%v", fake.lastSetupApply, fake.calls)
	}
	if app.setup.dialogOpen || app.setup.applying || app.setup.err != "" || app.model.notice != "Modules configured" {
		t.Fatalf("apply state = %+v notice=%q", app.setup, app.model.notice)
	}
	if fake.callCount("load-config") != 1 || fake.callCount("setup-status") != 1 {
		t.Fatalf("post-apply reloads = %v", fake.calls)
	}
}

func TestSetupPreviewAndApplyFailures(t *testing.T) {
	fake := &fakeBackend{setupPreviewErr: errors.New("preview failed")}
	app := newApplication(fake, testConfig(), nil, false)
	app.setup.status = core.SetupStatus{}
	app.resetSetupDraft()
	app.previewSetup()
	if !strings.Contains(app.setup.err, "preview failed") || app.setup.preview != nil {
		t.Fatalf("preview failure = %+v", app.setup)
	}

	fake.setupPreviewErr = nil
	fake.setupPlan = nil
	app.previewSetup()
	if !strings.Contains(app.setup.err, "returned no plan") {
		t.Fatalf("nil preview error = %q", app.setup.err)
	}

	app.setup.preview = nil
	app.applySetup()
	if fake.callCount("setup-apply") != 0 {
		t.Fatalf("apply without preview called backend: %v", fake.calls)
	}

	app.setup.err = ""
	app.setup.preview = &core.SetupPlan{SourceHash: "hash"}
	fake.setupApplyErr = errors.New("apply failed")
	app.applySetup()
	if !strings.Contains(app.setup.err, "apply failed") || app.setup.preview != nil {
		t.Fatalf("apply failure = %+v", app.setup)
	}

	fake.setupApplyErr = nil
	fake.setupApplyResult = service.SetupApplyResult{
		ApplyResult: service.ApplyResult{RolledBack: true, APIError: &model.APIError{Message: "rollback message"}},
	}
	app.setup.preview = &core.SetupPlan{SourceHash: "hash"}
	app.setup.dialogOpen = true
	app.applySetup()
	if !app.setup.dialogOpen || app.setup.err != "rollback message" {
		t.Fatalf("rollback state = %+v", app.setup)
	}
}

func TestSetupHelpers(t *testing.T) {
	status := core.SetupStatus{Modules: []core.ConfigModuleStatus{
		{ID: "outbounds", State: "ready", Count: 2},
		{ID: "route", State: "invalid", Count: 1},
		{ID: "dns", State: "missing"},
	}}
	if ready, total := setupReadyModules(status); ready != 1 || total != 3 {
		t.Fatalf("ready modules = %d/%d", ready, total)
	}
	if setupTone("ready") != "ready" || setupTone("invalid") != "invalid" || setupTone("missing") != "missing" {
		t.Fatal("setup tone mapping is incorrect")
	}
	if setupModuleLabel("rule_sets") != "Rule sets" || setupModuleLabel("custom_module") != "custom module" {
		t.Fatal("setup module labels are incorrect")
	}
	if !strings.Contains(setupModuleChoiceLabel(status, "outbounds"), "ready · 2") {
		t.Fatalf("module label = %q", setupModuleChoiceLabel(status, "outbounds"))
	}
	if !strings.Contains(setupModuleChoiceLabel(status, "inbounds"), "missing · 0") {
		t.Fatalf("missing module label = %q", setupModuleChoiceLabel(status, "inbounds"))
	}
	if readyLabel(true) != "ready" || readyLabel(false) != "not ready" {
		t.Fatal("ready labels are incorrect")
	}
	if capabilityLabel(true, "") != "available" || capabilityLabel(false, "") != "unavailable" ||
		capabilityLabel(false, "blocked") != "unavailable (blocked)" {
		t.Fatal("capability labels are incorrect")
	}
	if !strings.Contains(setupWarningLabel("proxy_nodes_required"), "Add a node") ||
		setupWarningLabel("custom_warning") != "custom_warning" {
		t.Fatal("setup warning labels are incorrect")
	}
}

func TestSetupDashboardDialogAndApplyFlow(t *testing.T) {
	fake := &fakeBackend{
		setupStatus: core.SetupStatus{
			Capabilities: core.NetworkCapabilities{Platform: "linux", TUNAvailable: true, IPv6Available: true, DefaultListen: "127.0.0.1"},
			Modules: []core.ConfigModuleStatus{
				{ID: "outbounds", State: "ready", Count: 2},
				{ID: "rule_sets", State: "ready", Count: 1},
				{ID: "inbounds", State: "missing"},
				{ID: "dns", State: "ready", Count: 1},
				{ID: "route", State: "missing"},
				{ID: "experimental", State: "ready", Count: 1},
			},
			Listeners:  []core.SetupListener{{Type: "mixed", Listen: "127.0.0.1", ListenPort: 1080}},
			ProxyReady: true,
		},
		setupPlan: &core.SetupPlan{
			SourceHash: "hash", Modules: []string{"outbounds", "inbounds"},
			Warnings: []string{"kernel restart required"}, WillRestart: true,
		},
		setupApplyResult: service.SetupApplyResult{ApplyResult: service.ApplyResult{Status: "ok"}},
	}
	app := newTestApplication(newAppModel())
	app.backend = fake
	app.model.loaded = true
	tt := ui.NewTester(app.view, 1440, 900)
	for _, text := range []string{"Setup", "Configure modules", "4/6 ready", "Listeners"} {
		if !tt.HasText(text) {
			t.Fatalf("dashboard setup is missing %q: %v", text, tt.Texts())
		}
	}
	if err := tt.Click("Configure modules"); err != nil {
		t.Fatal(err)
	}
	settleTransitions(tt)
	for _, text := range []string{"Inbound mode", "IPv6", "Preview", "Apply"} {
		if !tt.HasText(text) {
			t.Fatalf("setup dialog is missing %q: %v", text, tt.Texts())
		}
	}
	if err := tt.Click("Preview"); err != nil {
		t.Fatal(err)
	}
	settleTransitions(tt)
	if !tt.HasText("Preview ready") || !tt.HasText("kernel restart required") {
		t.Fatalf("preview is not rendered: %v", tt.Texts())
	}
	if err := tt.Click("Apply"); err != nil {
		t.Fatal(err)
	}
	settleTransitions(tt)
	if app.setup.dialogOpen || app.model.notice != "Modules configured" {
		t.Fatalf("apply did not close dialog: %+v notice=%q", app.setup, app.model.notice)
	}
}

func TestSetupDialogUnavailableState(t *testing.T) {
	fake := &fakeBackend{actionErr: errors.New("setup unavailable")}
	app := newTestApplication(newAppModel())
	app.backend = fake
	app.model.loaded = true
	tt := ui.NewTester(app.view, 1440, 900)
	if !tt.HasText("Setup status is unavailable") {
		t.Fatalf("setup unavailable state missing: %v", tt.Texts())
	}
	app.openSetupDialog()
	settleTransitions(tt)
	if !tt.HasText("Setup status is unavailable") || !tt.HasText("Cancel") {
		t.Fatalf("setup dialog unavailable state missing: %v", tt.Texts())
	}
}
