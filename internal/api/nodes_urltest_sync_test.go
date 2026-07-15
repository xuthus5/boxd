package api

import (
	"bytes"
	"os"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

func TestSyncOutboundsResolvesURLTestOverrides(t *testing.T) {
	nodeManager, subscriptionManager, configPath := prepareResolvedURLTestFixture(t)
	if err := syncOutboundsToConfig(nodeManager, subscriptionManager, configPath); err != nil {
		t.Fatalf("synchronizing outbounds: %v", err)
	}

	config := readConfigMap(t, configPath)
	customGroup := outboundByTag(t, config, "custom")
	if customGroup["url"] != "https://global.example/generate_204" {
		t.Fatalf("custom url = %#v", customGroup["url"])
	}
	if customGroup["interval"] != "30s" {
		t.Fatalf("custom interval = %#v", customGroup["interval"])
	}
	if customGroup["tolerance"] != float64(0) {
		t.Fatalf("custom tolerance = %#v", customGroup["tolerance"])
	}
	assertURLTestGroupMembers(t, customGroup, "custom-node")
	if group := optionalOutboundByTag(config, "disabled"); group != nil {
		t.Fatalf("disabled group should be removed: %#v", group)
	}
	proxy := outboundByTag(t, config, "proxy")
	if proxy["default"] != "custom" {
		t.Fatalf("proxy default = %#v, want custom", proxy["default"])
	}
	members, ok := proxy["outbounds"].([]any)
	if !ok {
		t.Fatalf("proxy outbounds = %#v", proxy["outbounds"])
	}
	foundCustom := false
	for _, member := range members {
		foundCustom = foundCustom || member == "custom"
		if member == "disabled" {
			t.Fatalf("proxy should not reference disabled group: %#v", members)
		}
	}
	if !foundCustom {
		t.Fatalf("proxy should include custom group: %#v", members)
	}
}

func assertURLTestGroupMembers(t *testing.T, group map[string]any, expected ...string) {
	t.Helper()
	members, ok := group["outbounds"].([]any)
	if !ok {
		t.Fatalf("urltest outbounds = %#v", group["outbounds"])
	}
	for _, tag := range expected {
		found := false
		for _, member := range members {
			found = found || member == tag
		}
		if !found {
			t.Fatalf("urltest group missing %q: %#v", tag, members)
		}
	}
}

func prepareResolvedURLTestFixture(
	t *testing.T,
) (*core.NodeManager, *core.SubscriptionManager, string) {
	t.Helper()
	nodeManager, subscriptionManager, settings, configPath := newAPIManagers(t)
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{
			map[string]any{"type": "urltest", "tag": "disabled", "outbounds": []string{"old"}},
		},
		"route": map[string]any{},
	})
	if err := settings.SetURLTestDefaults(model.URLTestDefaults{
		Enabled: true, URL: "https://global.example/generate_204", Interval: "10m", Tolerance: 80,
	}); err != nil {
		t.Fatalf("saving defaults: %v", err)
	}
	customInterval := "30s"
	zeroTolerance := uint16(0)
	addSubscriptionFixture(t, subscriptionManager, subscriptionFixture{
		name: "custom", nodeTag: "custom-node", server: "192.0.2.1",
		overrides: &model.URLTestOverrides{Interval: &customInterval, Tolerance: &zeroTolerance},
	})
	disabled := false
	addSubscriptionFixture(t, subscriptionManager, subscriptionFixture{
		name: "disabled", nodeTag: "disabled-node", server: "192.0.2.2",
		overrides: &model.URLTestOverrides{Enabled: &disabled},
	})
	return nodeManager, subscriptionManager, configPath
}

type subscriptionFixture struct {
	name      string
	nodeTag   string
	server    string
	overrides *model.URLTestOverrides
}

func addSubscriptionFixture(
	t *testing.T,
	manager *core.SubscriptionManager,
	spec subscriptionFixture,
) {
	t.Helper()
	subscription, err := manager.Create(core.SubscriptionParams{
		Name: spec.name, URL: "https://example.com/" + spec.name, IntervalMin: 60, URLTest: spec.overrides,
	})
	if err != nil {
		t.Fatalf("creating subscription: %v", err)
	}
	setSubscriptionOutbounds(t, manager, subscription.ID, []model.Outbound{
		{Tag: spec.nodeTag, Type: "vless", Server: spec.server, Port: 443},
	})
}

func TestSyncOutboundsAllowsSubscriptionURLTestWhenGlobalDisabled(t *testing.T) {
	nodeManager, subscriptionManager, settings, configPath := newAPIManagers(t)
	writeConfigFile(t, configPath, map[string]any{"outbounds": []any{}, "route": map[string]any{}})
	defaults := core.DefaultURLTestDefaults()
	defaults.Enabled = false
	if err := settings.SetURLTestDefaults(defaults); err != nil {
		t.Fatalf("saving defaults: %v", err)
	}

	enabled := true
	subscription, err := subscriptionManager.Create(core.SubscriptionParams{
		Name:        "enabled",
		URL:         "https://example.com/enabled",
		IntervalMin: 60,
		URLTest:     &model.URLTestOverrides{Enabled: &enabled},
	})
	if err != nil {
		t.Fatalf("creating subscription: %v", err)
	}
	setSubscriptionOutbounds(t, subscriptionManager, subscription.ID, []model.Outbound{
		{Tag: "enabled-node", Type: "vless", Server: "192.0.2.3", Port: 443},
	})

	if err := syncOutboundsToConfig(nodeManager, subscriptionManager, configPath); err != nil {
		t.Fatalf("synchronizing outbounds: %v", err)
	}
	config := readConfigMap(t, configPath)
	group := outboundByTag(t, config, "enabled")
	if group["url"] != defaults.URL || group["interval"] != defaults.Interval {
		t.Fatalf("enabled group = %#v", group)
	}
}

func TestSyncOutboundsRemovesManagedURLTestGroupAfterRename(t *testing.T) {
	nodeManager, subscriptionManager, configPath, subscription := prepareManagedURLTestLifecycle(t)
	if err := subscriptionManager.Update(subscription.ID, core.SubscriptionParams{
		Name: "new-name", URL: subscription.URL, IntervalMin: subscription.IntervalMin,
	}); err != nil {
		t.Fatalf("renaming subscription: %v", err)
	}
	if err := syncOutboundsToConfig(nodeManager, subscriptionManager, configPath); err != nil {
		t.Fatalf("synchronizing renamed subscription: %v", err)
	}
	config := readConfigMap(t, configPath)
	if group := optionalOutboundByTag(config, "old-name"); group != nil {
		t.Fatalf("old managed group should be removed: %#v", group)
	}
	if group := optionalOutboundByTag(config, "new-name"); group == nil {
		t.Fatal("renamed managed group was not created")
	}
	assertProxyDoesNotContain(t, config, "old-name")
}

func TestSyncOutboundsRemovesManagedURLTestGroupAfterDelete(t *testing.T) {
	nodeManager, subscriptionManager, configPath, subscription := prepareManagedURLTestLifecycle(t)
	if err := subscriptionManager.Delete(subscription.ID); err != nil {
		t.Fatalf("deleting subscription: %v", err)
	}
	if err := syncOutboundsToConfig(nodeManager, subscriptionManager, configPath); err != nil {
		t.Fatalf("synchronizing deleted subscription: %v", err)
	}
	if group := optionalOutboundByTag(readConfigMap(t, configPath), "old-name"); group != nil {
		t.Fatalf("deleted managed group should be removed: %#v", group)
	}
}

func prepareManagedURLTestLifecycle(
	t *testing.T,
) (*core.NodeManager, *core.SubscriptionManager, string, *model.Subscription) {
	t.Helper()
	nodeManager, subscriptionManager, _, configPath := newAPIManagers(t)
	writeConfigFile(t, configPath, map[string]any{"outbounds": []any{}, "route": map[string]any{}})
	subscription, err := subscriptionManager.Create(core.SubscriptionParams{
		Name: "old-name", URL: "https://example.com/sub", IntervalMin: 60,
	})
	if err != nil {
		t.Fatalf("creating subscription: %v", err)
	}
	setSubscriptionOutbounds(t, subscriptionManager, subscription.ID, []model.Outbound{
		{Tag: "managed-node", Type: "vless", Server: "192.0.2.4", Port: 443},
	})

	if err := syncOutboundsToConfig(nodeManager, subscriptionManager, configPath); err != nil {
		t.Fatalf("initial synchronization: %v", err)
	}
	if group := optionalOutboundByTag(readConfigMap(t, configPath), "old-name"); group == nil {
		t.Fatal("initial managed group was not created")
	}
	return nodeManager, subscriptionManager, configPath, subscription
}

func assertProxyDoesNotContain(t *testing.T, config map[string]any, unwanted string) {
	t.Helper()
	proxy := outboundByTag(t, config, "proxy")
	members, ok := proxy["outbounds"].([]any)
	if !ok {
		t.Fatalf("proxy outbounds = %#v", proxy["outbounds"])
	}
	for _, member := range members {
		if member == unwanted {
			t.Fatalf("proxy still references old managed group: %#v", members)
		}
	}
}

func TestSyncOutboundsLeavesConfigUnchangedWhenSubscriptionDataIsInvalid(t *testing.T) {
	nodeManager, subscriptionManager, _, configPath := newAPIManagers(t)
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{map[string]any{"type": "wireguard", "tag": "manual"}},
		"route":     map[string]any{"final": "direct"},
	})
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading initial config: %v", err)
	}
	if err := subscriptionManager.DB().Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte("subscriptions")).Put([]byte("broken"), []byte("{"))
	}); err != nil {
		t.Fatalf("saving invalid subscription: %v", err)
	}

	if err := syncOutboundsToConfig(nodeManager, subscriptionManager, configPath); err == nil {
		t.Fatal("expected subscription decode error")
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading config after failed sync: %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("config changed after failed sync\nbefore: %s\nafter: %s", before, after)
	}
}
