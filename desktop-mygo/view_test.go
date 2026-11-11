package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func newTestApplication(model *appModel) *application {
	app := newApplication(&fakeBackend{}, testConfig(), nil, false)
	if model != nil {
		app.model = model
	}
	return app
}

func TestDashboardView(t *testing.T) {
	app := newTestApplication(seededModel(themeLight))
	tt := ui.NewTester(app.view, 1440, 900)
	for _, text := range []string{"Dashboard", "Kernel controls", "Traffic", "Download", "Upload", "Start", "Stop", "Restart"} {
		if !tt.HasText(text) {
			t.Errorf("dashboard is missing %q; texts=%v", text, tt.Texts())
		}
	}
}

func TestSidebarNavigation(t *testing.T) {
	app := newTestApplication(seededModel(themeLight))
	tt := ui.NewTester(app.view, 1440, 900)
	pages := []struct {
		label string
		path  string
		text  string
	}{
		{label: "Nodes", path: "/nodes", text: "Import node"},
		{label: "Subscriptions", path: "/subscriptions", text: "Healthy"},
		{label: "Connections", path: "/connections", text: "Active"},
		{label: "Logs", path: "/logs", text: "Recent activity"},
		{label: "Config", path: "/config", text: "Raw configuration"},
		{label: "Settings", path: "/settings", text: "Appearance"},
		{label: "Dashboard", path: "/dashboard", text: "Kernel controls"},
	}
	for _, page := range pages {
		if err := tt.Click(page.label); err != nil {
			t.Fatalf("click %s: %v", page.label, err)
		}
		settleTransitions(tt)
		if got := app.router.Path(); got != page.path {
			t.Fatalf("after %s route = %q, want %q", page.label, got, page.path)
		}
		if !tt.HasText(page.text) {
			t.Errorf("page %s is missing %q; texts=%v", page.path, page.text, tt.Texts())
		}
	}
}

func TestEmptyStates(t *testing.T) {
	app := newTestApplication(newAppModel())
	app.model.loaded = true
	tt := ui.NewTester(app.view, 1440, 900)
	if err := tt.Click("Nodes"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("No nodes imported yet") {
		t.Fatalf("nodes empty state missing: %v", tt.Texts())
	}
	if err := tt.Click("Connections"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("No active connections") {
		t.Fatalf("connections empty state missing: %v", tt.Texts())
	}
	if err := tt.Click("Subscriptions"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("No subscriptions") {
		t.Fatalf("subscriptions empty state missing: %v", tt.Texts())
	}
}

func TestInvalidRouteFallsBackToDashboard(t *testing.T) {
	app := newTestApplication(seededModel(themeLight))
	tt := ui.NewTester(app.view, 1440, 900)
	app.router.Push("/not-a-page")
	settleTransitions(tt)
	if got := app.router.Path(); got != "/dashboard" {
		t.Fatalf("invalid route path = %q, want /dashboard", got)
	}
	if !tt.HasText("Kernel controls") {
		t.Fatalf("fallback dashboard is missing: %v", tt.Texts())
	}
}

func TestNodesAreVirtualized(t *testing.T) {
	app := newTestApplication(seededModel(themeLight))
	tt := ui.NewTester(app.view, 1440, 900)
	app.router.Push("/nodes")
	settleTransitions(tt)
	visible := 0
	for _, text := range tt.Texts() {
		if strings.HasPrefix(text, "node-") {
			visible++
		}
	}
	if visible == 0 || visible >= 60 {
		t.Fatalf("visible node rows = %d, want a virtualized subset", visible)
	}
}

func TestConfigurationFailureView(t *testing.T) {
	app := newApplication(nil, testConfig(), errTestStartup, false)
	tt := ui.NewTester(app.view, 1200, 800)
	if !tt.HasText("boxd could not start") || !tt.HasText("startup failed") {
		t.Fatalf("startup failure view texts=%v", tt.Texts())
	}
}

var errTestStartup = &testError{message: "startup failed"}

type testError struct{ message string }

func (e *testError) Error() string { return e.message }

func TestThemeToggleUpdatesPreferences(t *testing.T) {
	app := newTestApplication(seededModel(themeLight))
	tt := ui.NewTester(app.view, 1440, 900)
	if err := tt.Click("Light"); err != nil {
		t.Fatal(err)
	}
	if app.model.prefs.Theme != themeDark {
		t.Fatalf("theme = %q, want %q", app.model.prefs.Theme, themeDark)
	}
	if app.model.busy {
		t.Fatal("theme action left the UI busy")
	}
}

func TestCloseBehaviorWithoutTray(t *testing.T) {
	app := newTestApplication(newAppModel())
	if app.shouldHideOnClose() {
		t.Fatal("close should not hide when the tray is unavailable")
	}
}

func TestNodeResultsSummary(t *testing.T) {
	nodes := seededNodes(9)
	if got := countTested(nodes); got != 9 {
		t.Fatalf("tested = %d, want 9", got)
	}
	if got := countFailed(nodes); got != 1 {
		t.Fatalf("failed = %d, want 1", got)
	}
	if got := countHealthySubs(seededSubscriptions()); got != 2 {
		t.Fatalf("healthy subscriptions = %d, want 2", got)
	}
	if got := countFailedSubs(seededSubscriptions()); got != 1 {
		t.Fatalf("failed subscriptions = %d, want 1", got)
	}
	if !strings.HasPrefix(subscriptionTraffic(seededSubscriptions()[0]), "858.3 MB") {
		t.Fatalf("subscription traffic = %q", subscriptionTraffic(seededSubscriptions()[0]))
	}
}
