package core

import (
	"encoding/json"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

func TestIsProxyLikeOutboundType(t *testing.T) {
	for _, typ := range []string{"vless", "vmess", "trojan", "shadowsocks", "hysteria2", "anytls", "ssh", "tor"} {
		if !IsProxyLikeOutboundType(typ) {
			t.Fatalf("IsProxyLikeOutboundType(%q) = false, want true", typ)
		}
	}
	for _, typ := range []string{"", "direct", "block", "dns", "selector", "urltest", "wireguard"} {
		if IsProxyLikeOutboundType(typ) {
			t.Fatalf("IsProxyLikeOutboundType(%q) = true, want false", typ)
		}
	}
}

func TestSubscriptionSelectableTagsKeepsOrderAndSkipsGroups(t *testing.T) {
	subscription := model.Subscription{Outbounds: []model.Outbound{
		{Tag: "node-a", Type: "vless"},
		{Tag: "group", Type: "selector"},
		{Tag: "", Type: "vmess"},
		{Tag: "node-b", Type: "trojan"},
	}}
	tags := SubscriptionSelectableTags(subscription)
	if len(tags) != 2 || tags[0] != "node-a" || tags[1] != "node-b" {
		t.Fatalf("tags = %#v", tags)
	}
}

func TestValidateSubscriptionSelection(t *testing.T) {
	subscription := model.Subscription{Name: "sub", Outbounds: []model.Outbound{
		{Tag: "node-a", Type: "vless"},
		{Tag: "group", Type: "selector"},
	}}
	if err := ValidateSubscriptionSelection(subscription, ""); err != nil {
		t.Fatalf("empty selection should clear: %v", err)
	}
	if err := ValidateSubscriptionSelection(subscription, "node-a"); err != nil {
		t.Fatalf("known node rejected: %v", err)
	}
	if err := ValidateSubscriptionSelection(subscription, "group"); err == nil {
		t.Fatal("group member should be rejected")
	}
	if err := ValidateSubscriptionSelection(subscription, "missing"); err == nil {
		t.Fatal("unknown node should be rejected")
	}
}

func TestResolveSubscriptionSelectorDefault(t *testing.T) {
	members := []string{"node-a", "node-b"}
	if got := ResolveSubscriptionSelectorDefault(members, "node-b"); got != "node-b" {
		t.Fatalf("default = %q", got)
	}
	if got := ResolveSubscriptionSelectorDefault(members, ""); got != "" {
		t.Fatalf("empty selection = %q", got)
	}
	if got := ResolveSubscriptionSelectorDefault(members, "missing"); got != "" {
		t.Fatalf("stale selection = %q", got)
	}
}

func TestResolveSubscriptionSelectorTarget(t *testing.T) {
	members := []string{"node-a", "node-b"}
	cases := []struct {
		name     string
		selected string
		want     string
	}{
		{name: "selected", selected: "node-b", want: "node-b"},
		{name: "empty falls back", selected: "", want: "node-a"},
		{name: "stale falls back", selected: "removed", want: "node-a"},
		{name: "empty members", want: ""},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			caseMembers := members
			if testCase.name == "empty members" {
				caseMembers = nil
			}
			if got := ResolveSubscriptionSelectorTarget(caseMembers, testCase.selected); got != testCase.want {
				t.Fatalf("target = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestManagedSubscriptionSelectorTargets(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()
	manager := NewSubscriptionManager(db, t.TempDir())
	disabled := false
	selector, err := manager.Create(SubscriptionParams{
		Name: "selector", URL: "https://example.com/selector",
		URLTest: &model.URLTestOverrides{Enabled: &disabled},
	})
	if err != nil {
		t.Fatal(err)
	}
	writeCoreSubscriptionOutbounds(t, manager, selector.ID, []model.Outbound{
		{Tag: "node-a", Type: "vless"},
		{Tag: "node-b", Type: "trojan"},
	}, "node-b")
	empty, err := manager.Create(SubscriptionParams{
		Name: "empty", URL: "https://example.com/empty",
		URLTest: &model.URLTestOverrides{Enabled: &disabled},
	})
	if err != nil {
		t.Fatal(err)
	}
	writeCoreSubscriptionOutbounds(t, manager, empty.ID, []model.Outbound{
		{Tag: "node-c", Type: "shadowsocks"},
	}, "")
	enabled := true
	urltest, err := manager.Create(SubscriptionParams{
		Name: "url-test", URL: "https://example.com/url-test",
		URLTest: &model.URLTestOverrides{Enabled: &enabled},
	})
	if err != nil {
		t.Fatal(err)
	}
	writeCoreSubscriptionOutbounds(t, manager, urltest.ID, []model.Outbound{
		{Tag: "node-d", Type: "vmess"},
	}, "")
	noMembers, err := manager.Create(SubscriptionParams{
		Name: "no-members", URL: "https://example.com/no-members",
		URLTest: &model.URLTestOverrides{Enabled: &disabled},
	})
	if err != nil {
		t.Fatal(err)
	}
	writeCoreSubscriptionOutbounds(t, manager, noMembers.ID, nil, "")

	targets, err := ManagedSubscriptionSelectorTargets(manager)
	if err != nil {
		t.Fatalf("ManagedSubscriptionSelectorTargets() error = %v", err)
	}
	if targets["selector"] != "node-b" || targets["empty"] != "node-c" {
		t.Fatalf("targets = %#v", targets)
	}
	if _, exists := targets["url-test"]; exists {
		t.Fatalf("urltest group must be skipped: %#v", targets)
	}
	if _, exists := targets["no-members"]; exists {
		t.Fatalf("empty group must be skipped: %#v", targets)
	}
	if got, err := ManagedSubscriptionSelectorTargets(nil); err != nil || len(got) != 0 {
		t.Fatalf("nil manager = %#v, %v", got, err)
	}
}

func TestManagedSubscriptionSelectorTargetsErrors(t *testing.T) {
	db, cleanup := setupSubDB(t)
	manager := NewSubscriptionManager(db, t.TempDir())
	cleanup()
	if _, err := ManagedSubscriptionSelectorTargets(manager); err == nil {
		t.Fatal("closed database should fail")
	}
}

func writeCoreSubscriptionOutbounds(
	t *testing.T,
	manager *SubscriptionManager,
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
		return tx.Bucket(subBucket).Put([]byte(id), data)
	}); err != nil {
		t.Fatalf("saving subscription: %v", err)
	}
}

func TestSubscriptionManagerSetSelected(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	manager := NewSubscriptionManager(db, t.TempDir())
	created, err := manager.Create(SubscriptionParams{
		Name: "sub", URL: "https://example.com/sub", IntervalMin: 60,
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := manager.SetSelected(created.ID, "node-a")
	if err != nil {
		t.Fatalf("SetSelected() error = %v", err)
	}
	if updated.Selected != "node-a" {
		t.Fatalf("returned selection = %q", updated.Selected)
	}
	stored := manager.Get(created.ID)
	if stored == nil || stored.Selected != "node-a" {
		t.Fatalf("stored subscription = %#v", stored)
	}

	cleared, err := manager.SetSelected(created.ID, "")
	if err != nil {
		t.Fatalf("clearing selection: %v", err)
	}
	if cleared.Selected != "" {
		t.Fatalf("cleared selection = %q", cleared.Selected)
	}

	if _, err := manager.SetSelected("missing", "node-a"); err == nil {
		t.Fatal("missing subscription should fail")
	}
}

func TestSubscriptionManagerSetSelectedRejectsCorruptRecord(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	manager := NewSubscriptionManager(db, t.TempDir())
	if err := manager.DB().Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(subBucket).Put([]byte("broken"), []byte("{"))
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.SetSelected("broken", "node-a"); err == nil {
		t.Fatal("corrupt subscription should fail")
	}
}
