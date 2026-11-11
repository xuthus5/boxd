package main

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/xuthus5/boxd/internal/model"
)

func TestNodeManagementActions(t *testing.T) {
	fake := &fakeBackend{}
	app := newApplication(fake, testConfig(), nil, false)
	app.openAddNode()
	app.nodeForm.link = "vless://example"
	app.submitNodeForm()
	if fake.callCount("import-node") != 1 || app.nodeForm.open {
		t.Fatalf("import state = calls %v open %v", fake.calls, app.nodeForm.open)
	}
	app.model.nodes = seededNodes(200)
	app.testAllNodes()
	if got := len(fake.lastBatch.Items); got != 128 {
		t.Fatalf("batch size = %d, want 128", got)
	}
	if fake.lastBatch.Concurrency != 8 {
		t.Fatalf("batch concurrency = %d", fake.lastBatch.Concurrency)
	}
}

func TestNodeEditDeleteAndValidation(t *testing.T) {
	fake := &fakeBackend{}
	app := newApplication(fake, testConfig(), nil, false)
	node := nodeView{Tag: "edge", Type: "vless", Server: "example.com", Port: 443, Source: "import", Config: map[string]any{"type": "vless"}}
	app.openEditNode(node)
	app.nodeForm.server = "new.example.com"
	app.submitNodeForm()
	if fake.callCount("update-node") != 1 || fake.lastOldTag != "edge" || fake.lastNode.Server != "new.example.com" {
		t.Fatalf("update state = %+v", fake)
	}
	app.askDeleteNode(node)
	if !app.deleteForm.open || app.deleteForm.kind != "node" {
		t.Fatalf("delete form = %+v", app.deleteForm)
	}
	app.confirmDelete()
	if fake.callCount("delete-node") != 1 {
		t.Fatalf("delete calls = %v", fake.calls)
	}
	app.nodeForm.beginEdit(node)
	app.nodeForm.config = "{"
	app.submitNodeForm()
	if app.nodeForm.err == "" {
		t.Fatal("invalid JSON was accepted")
	}
	app.nodeForm.config = "{} trailing"
	app.submitNodeForm()
	if app.nodeForm.err == "" {
		t.Fatal("trailing JSON data was accepted")
	}
}

func TestSubscriptionManagementActions(t *testing.T) {
	fake := &fakeBackend{}
	app := newApplication(fake, testConfig(), nil, false)
	app.openAddSubscription()
	app.subForm.name = "Primary"
	app.subForm.url = "https://example.com/sub"
	app.submitSubscriptionForm()
	if fake.callCount("create-subscription") != 1 || app.subForm.open {
		t.Fatalf("create state = %v open %v", fake.calls, app.subForm.open)
	}
	sub := model.Subscription{ID: "sub-1", Name: "Primary", URL: "https://example.com/sub", IntervalMin: 30}
	app.openEditSubscription(sub)
	app.subForm.interval = 45
	app.submitSubscriptionForm()
	if fake.callCount("update-subscription") != 1 || fake.lastSubID != "sub-1" || fake.lastSub.IntervalMin != 45 {
		t.Fatalf("update state = %+v", fake)
	}
	app.openSubscriptionSelector(model.Subscription{ID: "sub-1", Outbounds: []model.Outbound{{Tag: "a"}, {Tag: "b"}}})
	app.selectorForm.selected = "b"
	app.submitSubscriptionSelector()
	if fake.lastSelect != "b" || fake.lastSubID != "sub-1" || app.selectorForm.open {
		t.Fatalf("selector state = %+v", fake)
	}
	app.refreshAllSubscriptions()
	app.askDeleteSubscription(sub)
	app.confirmDelete()
	if fake.callCount("refresh-all-subscriptions") != 1 || fake.callCount("delete-subscription") != 1 {
		t.Fatalf("calls = %v", fake.calls)
	}
}

func TestSubscriptionURLTestPolicyPreserved(t *testing.T) {
	fake := &fakeBackend{}
	app := newApplication(fake, testConfig(), nil, false)
	disabled := false
	app.openEditSubscription(model.Subscription{
		ID: "1", Name: "Selector", URL: "https://example.com/sub", IntervalMin: 30,
		URLTest: &model.URLTestOverrides{Enabled: &disabled},
	})
	if app.subForm.urlTestMode != urlTestModeDisabled {
		t.Fatalf("url-test mode = %q", app.subForm.urlTestMode)
	}
	app.submitSubscriptionForm()
	if fake.lastSub.URLTest == nil || fake.lastSub.URLTest.Enabled == nil || *fake.lastSub.URLTest.Enabled {
		t.Fatalf("disabled policy was not preserved: %+v", fake.lastSub.URLTest)
	}

	app.openAddSubscription()
	app.subForm.name = "URL test"
	app.subForm.url = "https://example.com/urltest"
	app.subForm.urlTestMode = urlTestModeEnabled
	app.subForm.testURL = "https://example.com/probe"
	app.subForm.testInterval = "2m"
	app.subForm.tolerance = 20
	app.submitSubscriptionForm()
	overrides := fake.lastSub.URLTest
	if overrides == nil || overrides.Enabled == nil || !*overrides.Enabled ||
		overrides.Interval == nil || *overrides.Interval != "2m" ||
		overrides.Tolerance == nil || *overrides.Tolerance != 20 {
		t.Fatalf("enabled overrides = %+v", overrides)
	}
}

func TestConnectionFilterAndBulkClose(t *testing.T) {
	fake := &fakeBackend{}
	app := newApplication(fake, testConfig(), nil, false)
	app.model.connections = seededConnections()
	app.connFilter = "github"
	matched := app.filteredConnections()
	if len(matched) != 1 || matched[0].ID != 41 {
		t.Fatalf("matched = %+v", matched)
	}
	app.askCloseConnections(matched, true)
	app.confirmDelete()
	if len(fake.lastFilters.IDs) != 1 || fake.lastFilters.IDs[0] != "41" {
		t.Fatalf("filters = %+v", fake.lastFilters)
	}
	app.askCloseConnection(matched[0])
	app.confirmDelete()
	if fake.callCount("close-connection") != 1 {
		t.Fatalf("calls = %v", fake.calls)
	}
}

func TestConnectionCloseAllAndErrorRecovery(t *testing.T) {
	fake := &fakeBackend{}
	app := newApplication(fake, testConfig(), nil, false)
	app.model.connections = seededConnections()
	app.askCloseConnections(nil, false)
	if !app.deleteForm.open || app.deleteForm.kind != "connections" || len(app.deleteForm.ids) != 0 {
		t.Fatalf("close-all form = %+v", app.deleteForm)
	}
	app.confirmDelete()
	if fake.callCount("close-connections") != 1 || len(fake.lastFilters.IDs) != 0 {
		t.Fatalf("close-all filters = %+v", fake.lastFilters)
	}

	fake.actionErr = errors.New("close failed")
	app.askCloseConnection(app.model.connections[0])
	app.confirmDelete()
	if app.model.actionError == "" || app.model.busy || app.deleteForm.open {
		t.Fatalf("error state = busy %v error %q form %+v", app.model.busy, app.model.actionError, app.deleteForm)
	}
}

func TestManagementPageControls(t *testing.T) {
	app := newTestApplication(seededModel(themeLight))
	tt := ui.NewTester(app.view, 1440, 900)
	app.router.Push("/nodes")
	settleTransitions(tt)
	for _, text := range []string{"Import node", "Test all", "Source", "Actions"} {
		if !tt.HasText(text) {
			t.Errorf("nodes page missing %q", text)
		}
	}
	app.router.Push("/subscriptions")
	settleTransitions(tt)
	for _, text := range []string{"Add subscription", "Refresh all", "Exit", "Select"} {
		if !tt.HasText(text) {
			t.Errorf("subscriptions page missing %q", text)
		}
	}
	app.openAddNode()
	settleTransitions(tt)
	if !tt.HasText("Import proxy link") {
		t.Fatalf("node dialog missing: %v", tt.Texts())
	}
}
