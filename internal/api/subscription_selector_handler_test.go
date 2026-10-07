package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

type fakeSubscriptionInstance struct {
	restarts   int
	selections []string
}

type restartOnlyInstance struct {
	restarts int
}

func (f *restartOnlyInstance) Restart() error {
	f.restarts++
	return nil
}

type failingSelectorInstance struct {
	err error
}

func (f *failingSelectorInstance) Restart() error { return nil }

func (f *failingSelectorInstance) SelectOutbound(_, _ string) error { return f.err }

func (f *fakeSubscriptionInstance) Restart() error {
	f.restarts++
	return nil
}

func (f *fakeSubscriptionInstance) SelectOutbound(groupTag, outTag string) error {
	f.selections = append(f.selections, groupTag+"/"+outTag)
	return nil
}

func newSelectorSubscriptionFixture(
	t *testing.T,
) (*core.NodeManager, *core.SubscriptionManager, *model.Subscription, string) {
	t.Helper()
	nodeManager, subscriptionManager, _, configPath := newAPIManagers(t)
	writeConfigFile(t, configPath, map[string]any{"outbounds": []any{}, "route": map[string]any{}})
	disabled := false
	subscription, err := subscriptionManager.Create(core.SubscriptionParams{
		Name: "sub-a", URL: "https://example.com/sub-a", IntervalMin: 60,
		URLTest: &model.URLTestOverrides{Enabled: &disabled},
	})
	if err != nil {
		t.Fatalf("creating subscription: %v", err)
	}
	setSubscriptionOutbounds(t, subscriptionManager, subscription.ID, []model.Outbound{
		{Tag: "node-a", Type: "vless", Server: "192.0.2.1", Port: 443},
		{Tag: "node-b", Type: "trojan", Server: "192.0.2.2", Port: 443},
	})
	return nodeManager, subscriptionManager, subscription, configPath
}

func TestSubscriptionHandlerSetSelectorWritesDefault(t *testing.T) {
	nodeManager, subscriptionManager, subscription, configPath := newSelectorSubscriptionFixture(t)
	instance := &fakeSubscriptionInstance{}
	handler := NewSubscriptionHandler(subscriptionManager, nodeManager, configPath, instance)

	recorder := httptest.NewRecorder()
	request := withURLParam(jsonRequest(
		http.MethodPut,
		"/api/subscriptions/"+subscription.ID+"/selector",
		`{"tag":"node-b"}`,
	), "id", subscription.ID)
	handler.SetSelector(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	updated := decodeBody[model.Subscription](t, recorder)
	if updated.Selected != "node-b" {
		t.Fatalf("returned selection = %q", updated.Selected)
	}
	if stored := subscriptionManager.Get(subscription.ID); stored == nil || stored.Selected != "node-b" {
		t.Fatalf("stored subscription = %#v", stored)
	}

	group := outboundByTag(t, readConfigMap(t, configPath), "sub-a")
	if group["type"] != "selector" {
		t.Fatalf("group type = %#v", group["type"])
	}
	if group["default"] != "node-b" {
		t.Fatalf("group default = %#v", group["default"])
	}
	if len(instance.selections) != 1 || instance.selections[0] != "sub-a/node-b" {
		t.Fatalf("runtime selections = %#v", instance.selections)
	}
}

func TestSubscriptionHandlerSetSelectorClearsDefault(t *testing.T) {
	nodeManager, subscriptionManager, subscription, configPath := newSelectorSubscriptionFixture(t)
	if _, err := subscriptionManager.SetSelected(subscription.ID, "node-b"); err != nil {
		t.Fatalf("preselecting: %v", err)
	}
	handler := NewSubscriptionHandler(subscriptionManager, nodeManager, configPath)

	recorder := httptest.NewRecorder()
	request := withURLParam(jsonRequest(
		http.MethodPut,
		"/api/subscriptions/"+subscription.ID+"/selector",
		`{"tag":""}`,
	), "id", subscription.ID)
	handler.SetSelector(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	group := outboundByTag(t, readConfigMap(t, configPath), "sub-a")
	if _, exists := group["default"]; exists {
		t.Fatalf("cleared group still has default: %#v", group["default"])
	}
}

func TestSubscriptionHandlerSetSelectorValidation(t *testing.T) {
	nodeManager, subscriptionManager, subscription, configPath := newSelectorSubscriptionFixture(t)
	handler := NewSubscriptionHandler(subscriptionManager, nodeManager, configPath)

	cases := []struct {
		name       string
		id         string
		body       string
		wantStatus int
	}{
		{name: "unknown node", id: subscription.ID, body: `{"tag":"missing"}`, wantStatus: http.StatusBadRequest},
		{name: "missing subscription", id: "missing", body: `{"tag":"node-a"}`, wantStatus: http.StatusNotFound},
		{name: "invalid body", id: subscription.ID, body: `{`, wantStatus: http.StatusBadRequest},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := withURLParam(jsonRequest(
				http.MethodPut, "/api/subscriptions/"+testCase.id+"/selector", testCase.body,
			), "id", testCase.id)
			handler.SetSelector(recorder, request)
			if recorder.Code != testCase.wantStatus {
				t.Fatalf("status = %d want = %d body=%s", recorder.Code, testCase.wantStatus, recorder.Body.String())
			}
		})
	}
}

func TestSyncOutboundsWritesSubscriptionSelectorDefault(t *testing.T) {
	nodeManager, subscriptionManager, subscription, configPath := newSelectorSubscriptionFixture(t)
	if _, err := subscriptionManager.SetSelected(subscription.ID, "node-b"); err != nil {
		t.Fatalf("preselecting: %v", err)
	}
	if err := syncOutboundsToConfig(nodeManager, subscriptionManager, configPath); err != nil {
		t.Fatalf("syncOutboundsToConfig() error = %v", err)
	}
	group := outboundByTag(t, readConfigMap(t, configPath), "sub-a")
	if group["default"] != "node-b" {
		t.Fatalf("group default = %#v", group["default"])
	}
}

func TestSyncOutboundsDropsDefaultForURLTestGroup(t *testing.T) {
	nodeManager, subscriptionManager, settings, configPath := newAPIManagers(t)
	writeConfigFile(t, configPath, map[string]any{"outbounds": []any{}, "route": map[string]any{}})
	if err := settings.SetURLTestDefaults(core.DefaultURLTestDefaults()); err != nil {
		t.Fatalf("saving defaults: %v", err)
	}
	enabled := true
	subscription, err := subscriptionManager.Create(core.SubscriptionParams{
		Name: "sub-a", URL: "https://example.com/sub-a", IntervalMin: 60,
		URLTest: &model.URLTestOverrides{Enabled: &enabled},
	})
	if err != nil {
		t.Fatalf("creating subscription: %v", err)
	}
	setSubscriptionOutbounds(t, subscriptionManager, subscription.ID, []model.Outbound{
		{Tag: "node-a", Type: "vless", Server: "192.0.2.1", Port: 443},
		{Tag: "node-b", Type: "trojan", Server: "192.0.2.2", Port: 443},
	})
	if _, err := subscriptionManager.SetSelected(subscription.ID, "node-b"); err != nil {
		t.Fatalf("preselecting: %v", err)
	}

	if err := syncOutboundsToConfig(nodeManager, subscriptionManager, configPath); err != nil {
		t.Fatalf("syncOutboundsToConfig() error = %v", err)
	}
	group := outboundByTag(t, readConfigMap(t, configPath), "sub-a")
	if group["type"] != "urltest" {
		t.Fatalf("group type = %#v", group["type"])
	}
	if _, exists := group["default"]; exists {
		t.Fatalf("urltest group should not carry default: %#v", group["default"])
	}
}

func TestSubscriptionHandlerSetSelectorRollsBackOnSyncFailure(t *testing.T) {
	nodeManager, subscriptionManager, subscription, _ := newSelectorSubscriptionFixture(t)
	if _, err := subscriptionManager.SetSelected(subscription.ID, "node-a"); err != nil {
		t.Fatalf("preselecting: %v", err)
	}
	handler := NewSubscriptionHandler(subscriptionManager, nodeManager, filepath.Join(t.TempDir(), "missing.json"))

	recorder := httptest.NewRecorder()
	request := withURLParam(jsonRequest(
		http.MethodPut, "/api/subscriptions/"+subscription.ID+"/selector", `{"tag":"node-b"}`,
	), "id", subscription.ID)
	handler.SetSelector(recorder, request)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if stored := subscriptionManager.Get(subscription.ID); stored == nil || stored.Selected != "node-a" {
		t.Fatalf("selection should roll back: %#v", stored)
	}
}

func TestSubscriptionHandlerSetSelectorToleratesRuntimeInstanceVariants(t *testing.T) {
	cases := []struct {
		name     string
		instance restartableInstance
	}{
		{name: "not selectable", instance: &restartOnlyInstance{}},
		{name: "not running", instance: &failingSelectorInstance{err: core.ErrNotRunning}},
		{name: "switch failure", instance: &failingSelectorInstance{err: errors.New("switch failed")}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			nodeManager, subscriptionManager, subscription, configPath := newSelectorSubscriptionFixture(t)
			handler := NewSubscriptionHandler(subscriptionManager, nodeManager, configPath, testCase.instance)
			recorder := httptest.NewRecorder()
			request := withURLParam(jsonRequest(
				http.MethodPut, "/api/subscriptions/"+subscription.ID+"/selector", `{"tag":"node-b"}`,
			), "id", subscription.ID)
			handler.SetSelector(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestSubscriptionHandlerSetSelectorReportsWriteFailure(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "readonly.db")
	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	writableManager := core.NewSubscriptionManager(db, dir)
	disabled := false
	subscription, err := writableManager.Create(core.SubscriptionParams{
		Name: "sub-a", URL: "https://example.com/sub-a", IntervalMin: 60,
		URLTest: &model.URLTestOverrides{Enabled: &disabled},
	})
	if err != nil {
		t.Fatalf("creating subscription: %v", err)
	}
	setSubscriptionOutbounds(t, writableManager, subscription.ID, []model.Outbound{
		{Tag: "node-a", Type: "vless", Server: "192.0.2.1", Port: 443},
	})
	if err := db.Close(); err != nil {
		t.Fatalf("closing database: %v", err)
	}
	readOnly, err := bbolt.Open(dbPath, 0600, &bbolt.Options{ReadOnly: true, Timeout: time.Second})
	if err != nil {
		t.Fatalf("opening read-only database: %v", err)
	}
	t.Cleanup(func() { _ = readOnly.Close() })

	configPath := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, configPath, map[string]any{"outbounds": []any{}, "route": map[string]any{}})
	handler := NewSubscriptionHandler(
		core.NewSubscriptionManager(readOnly, dir),
		core.NewNodeManager(readOnly),
		configPath,
	)
	recorder := httptest.NewRecorder()
	request := withURLParam(jsonRequest(
		http.MethodPut, "/api/subscriptions/"+subscription.ID+"/selector", `{"tag":"node-a"}`,
	), "id", subscription.ID)
	handler.SetSelector(recorder, request)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
}
