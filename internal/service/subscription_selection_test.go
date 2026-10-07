package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

type fakeSelectorInstance struct {
	restarts   int
	selections []string
}

func (f *fakeSelectorInstance) Restart() error {
	f.restarts++
	return nil
}

func (f *fakeSelectorInstance) SelectOutbound(groupTag, outTag string) error {
	f.selections = append(f.selections, groupTag+"/"+outTag)
	return nil
}

type failingSelectorInstance struct {
	err error
}

func (f *failingSelectorInstance) Restart() error { return nil }

func (f *failingSelectorInstance) SelectOutbound(_, _ string) error { return f.err }

func seedServiceSubscription(
	t *testing.T,
	manager *core.SubscriptionManager,
	name string,
	outbounds []model.Outbound,
) *model.Subscription {
	t.Helper()
	disabled := false
	subscription, err := manager.Create(core.SubscriptionParams{
		Name: name, URL: "https://example.com/" + name, IntervalMin: 60,
		URLTest: &model.URLTestOverrides{Enabled: &disabled},
	})
	if err != nil {
		t.Fatalf("creating subscription: %v", err)
	}
	writeStoredSubscription(t, manager, subscription.ID, outbounds, "")
	return manager.Get(subscription.ID)
}

func writeStoredSubscription(
	t *testing.T,
	manager *core.SubscriptionManager,
	id string,
	outbounds []model.Outbound,
	selected string,
) {
	t.Helper()
	subscription := manager.Get(id)
	if subscription == nil {
		t.Fatalf("subscription %q not found", id)
	}
	subscription.Outbounds = outbounds
	subscription.Selected = selected
	data, err := json.Marshal(subscription)
	if err != nil {
		t.Fatalf("encoding subscription: %v", err)
	}
	if err := manager.DB().Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte("subscriptions")).Put([]byte(id), data)
	}); err != nil {
		t.Fatalf("saving subscription: %v", err)
	}
}

func selectorGroupDefault(t *testing.T, configPath, tag string) (string, bool) {
	t.Helper()
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	config := map[string]any{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("decoding config: %v", err)
	}
	outbounds, _ := config["outbounds"].([]any)
	for _, entry := range outbounds {
		outbound, _ := entry.(map[string]any)
		if outbound["tag"] != tag {
			continue
		}
		if outbound["type"] != "selector" {
			t.Fatalf("group %q type = %#v", tag, outbound["type"])
		}
		value, exists := outbound["default"]
		selected, _ := value.(string)
		return selected, exists
	}
	t.Fatalf("group %q not found in %#v", tag, outbounds)
	return "", false
}

func TestSubscriptionServiceSetSelectedSyncsConfigAndRuntime(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeTestConfig(t, configPath, map[string]any{
		"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}},
		"route":     map[string]any{},
	})
	nodeManager := core.NewNodeManager(db)
	subscriptionManager := core.NewSubscriptionManager(db, t.TempDir())
	subscription := seedServiceSubscription(t, subscriptionManager, "sub-a", []model.Outbound{
		{Tag: "node-a", Type: "vless", Server: "192.0.2.1", Port: 443},
		{Tag: "node-b", Type: "trojan", Server: "192.0.2.2", Port: 443},
	})
	instance := &fakeSelectorInstance{}
	service := NewSubscriptionService(subscriptionManager, nodeManager, configPath, instance)

	if err := service.SetSelected(context.Background(), subscription.ID, "node-b"); err != nil {
		t.Fatalf("SetSelected() error = %v", err)
	}

	selected, exists := selectorGroupDefault(t, configPath, subscription.Name)
	if !exists || selected != "node-b" {
		t.Fatalf("selector default = %q (exists=%v)", selected, exists)
	}
	stored := subscriptionManager.Get(subscription.ID)
	if stored == nil || stored.Selected != "node-b" {
		t.Fatalf("stored selection = %#v", stored)
	}
	if len(instance.selections) != 1 || instance.selections[0] != "sub-a/node-b" {
		t.Fatalf("runtime selections = %#v", instance.selections)
	}

	if err := service.SetSelected(context.Background(), subscription.ID, ""); err != nil {
		t.Fatalf("clearing selection: %v", err)
	}
	if _, exists := selectorGroupDefault(t, configPath, subscription.Name); exists {
		t.Fatal("cleared selection should drop the default field")
	}
}

func TestSubscriptionServiceSetSelectedRejectsUnknownNode(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeTestConfig(t, configPath, map[string]any{"outbounds": []any{}, "route": map[string]any{}})
	nodeManager := core.NewNodeManager(db)
	subscriptionManager := core.NewSubscriptionManager(db, t.TempDir())
	subscription := seedServiceSubscription(t, subscriptionManager, "sub-a", []model.Outbound{
		{Tag: "node-a", Type: "vless", Server: "192.0.2.1", Port: 443},
	})
	service := NewSubscriptionService(subscriptionManager, nodeManager, configPath, nil)

	if err := service.SetSelected(context.Background(), subscription.ID, "missing"); err == nil {
		t.Fatal("unknown node should be rejected")
	}
	if err := service.SetSelected(context.Background(), "missing", "node-a"); err == nil {
		t.Fatal("missing subscription should be rejected")
	}
	if stored := subscriptionManager.Get(subscription.ID); stored == nil || stored.Selected != "" {
		t.Fatalf("selection = %#v", stored)
	}
}

func TestSyncOutboundsDropsStaleSubscriptionSelection(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeTestConfig(t, configPath, map[string]any{"outbounds": []any{}, "route": map[string]any{}})
	nodeManager := core.NewNodeManager(db)
	subscriptionManager := core.NewSubscriptionManager(db, t.TempDir())
	disabled := false
	subscription, err := subscriptionManager.Create(core.SubscriptionParams{
		Name: "sub-a", URL: "https://example.com/sub-a", IntervalMin: 60,
		URLTest: &model.URLTestOverrides{Enabled: &disabled},
	})
	if err != nil {
		t.Fatalf("creating subscription: %v", err)
	}
	writeStoredSubscription(t, subscriptionManager, subscription.ID, []model.Outbound{
		{Tag: "node-a", Type: "vless", Server: "192.0.2.1", Port: 443},
	}, "removed-node")

	if err := SyncOutboundsToConfig(nodeManager, subscriptionManager, configPath); err != nil {
		t.Fatalf("SyncOutboundsToConfig() error = %v", err)
	}
	if _, exists := selectorGroupDefault(t, configPath, "sub-a"); exists {
		t.Fatal("stale selection should not be written as default")
	}
}

func TestSubscriptionServiceSetSelectedRollsBackOnSyncFailure(t *testing.T) {
	db := newTestDB(t)
	nodeManager := core.NewNodeManager(db)
	subscriptionManager := core.NewSubscriptionManager(db, t.TempDir())
	subscription := seedServiceSubscription(t, subscriptionManager, "sub-a", []model.Outbound{
		{Tag: "node-a", Type: "vless", Server: "192.0.2.1", Port: 443},
		{Tag: "node-b", Type: "trojan", Server: "192.0.2.2", Port: 443},
	})
	if _, err := subscriptionManager.SetSelected(subscription.ID, "node-a"); err != nil {
		t.Fatalf("preselecting: %v", err)
	}
	configPath := filepath.Join(t.TempDir(), "missing.json")
	service := NewSubscriptionService(subscriptionManager, nodeManager, configPath, nil)

	if err := service.SetSelected(context.Background(), subscription.ID, "node-b"); err == nil {
		t.Fatal("config sync failure should surface")
	}
	if stored := subscriptionManager.Get(subscription.ID); stored == nil || stored.Selected != "node-a" {
		t.Fatalf("selection should roll back: %#v", stored)
	}
}

func TestSubscriptionServiceSetSelectedToleratesRuntimeVariants(t *testing.T) {
	cases := []struct {
		name     string
		instance restartable
	}{
		{name: "nil instance", instance: nil},
		{name: "not selectable", instance: &fakeRestart{}},
		{name: "not running", instance: &failingSelectorInstance{err: core.ErrNotRunning}},
		{name: "switch failure", instance: &failingSelectorInstance{err: errors.New("switch failed")}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			db := newTestDB(t)
			configPath := filepath.Join(t.TempDir(), "config.json")
			writeTestConfig(t, configPath, map[string]any{"outbounds": []any{}, "route": map[string]any{}})
			nodeManager := core.NewNodeManager(db)
			subscriptionManager := core.NewSubscriptionManager(db, t.TempDir())
			subscription := seedServiceSubscription(t, subscriptionManager, "sub-a", []model.Outbound{
				{Tag: "node-a", Type: "vless", Server: "192.0.2.1", Port: 443},
			})
			service := NewSubscriptionService(subscriptionManager, nodeManager, configPath, testCase.instance)
			if err := service.SetSelected(context.Background(), subscription.ID, "node-a"); err != nil {
				t.Fatalf("SetSelected() error = %v", err)
			}
		})
	}
}

func TestSubscriptionServiceSetSelectedReportsWriteFailure(t *testing.T) {
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
	writeStoredSubscription(t, writableManager, subscription.ID, []model.Outbound{
		{Tag: "node-a", Type: "vless", Server: "192.0.2.1", Port: 443},
	}, "")
	if err := db.Close(); err != nil {
		t.Fatalf("closing database: %v", err)
	}
	readOnly, err := bbolt.Open(dbPath, 0600, &bbolt.Options{ReadOnly: true, Timeout: time.Second})
	if err != nil {
		t.Fatalf("opening read-only database: %v", err)
	}
	t.Cleanup(func() { _ = readOnly.Close() })

	configPath := filepath.Join(t.TempDir(), "config.json")
	writeTestConfig(t, configPath, map[string]any{"outbounds": []any{}, "route": map[string]any{}})
	service := NewSubscriptionService(
		core.NewSubscriptionManager(readOnly, dir),
		core.NewNodeManager(readOnly),
		configPath,
		&fakeSelectorInstance{},
	)
	if err := service.SetSelected(context.Background(), subscription.ID, "node-a"); err == nil {
		t.Fatal("write failure should surface")
	}
}
