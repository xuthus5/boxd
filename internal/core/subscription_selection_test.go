package core

import (
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
