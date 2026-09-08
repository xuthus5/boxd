package core

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"go.etcd.io/bbolt"
	bolterrors "go.etcd.io/bbolt/errors"

	"github.com/xuthus5/boxd/internal/model"
)

func newRouteRuleMetadataManager(t *testing.T) *RouteRuleMetadataManager {
	t.Helper()
	db, err := bbolt.Open(filepath.Join(t.TempDir(), "metadata.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil && !errors.Is(err, bolterrors.ErrDatabaseNotOpen) {
			t.Errorf("closing metadata database: %v", err)
		}
	})
	return NewRouteRuleMetadataManager(db)
}

func TestRouteRuleMetadataManagerSaveAndReconcile(t *testing.T) {
	manager := newRouteRuleMetadataManager(t)
	rules := []any{
		map[string]any{"action": "sniff"},
		map[string]any{"protocol": "dns", "action": "hijack-dns"},
	}
	metadata := []model.RouteRuleMetadata{
		{Name: "协议嗅探", Description: "识别协议"},
		{Name: "DNS 请求劫持"},
	}
	if err := manager.Save(rules, metadata); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := manager.List([]any{rules[1], rules[0]})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got[0].Name != "DNS 请求劫持" || got[1].Name != "协议嗅探" || got[1].Description != "识别协议" {
		t.Fatalf("List() = %#v", got)
	}
}

func TestRouteRuleMetadataManagerKeepsMetadataWhenRuleChangesInPlace(t *testing.T) {
	manager := newRouteRuleMetadataManager(t)
	rules := []any{map[string]any{"action": "route", "outbound": "proxy"}}
	metadata := []model.RouteRuleMetadata{{Name: "代理规则", Description: "规则内容可编辑"}}
	if err := manager.Save(rules, metadata); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	changed := []any{map[string]any{"action": "reject"}}
	got, err := manager.List(changed)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got[0] != metadata[0] {
		t.Fatalf("List() = %#v, want %#v", got, metadata)
	}
}

func TestRouteRuleMetadataManagerValidatesInput(t *testing.T) {
	manager := newRouteRuleMetadataManager(t)
	rules := []any{map[string]any{"action": "sniff"}}
	if err := manager.Save(rules, nil); err == nil {
		t.Fatal("Save() should reject a mismatched metadata count")
	}
	metadata := []model.RouteRuleMetadata{{Name: strings.Repeat("x", MaxRouteRuleNameLength+1)}}
	if err := manager.Save(rules, metadata); err == nil {
		t.Fatal("Save() should reject an oversized name")
	}
	metadata = []model.RouteRuleMetadata{{Description: strings.Repeat("x", MaxRouteRuleDescriptionLength+1)}}
	if err := manager.Save(rules, metadata); err == nil {
		t.Fatal("Save() should reject an oversized description")
	}
	if err := manager.Save([]any{make(chan int)}, []model.RouteRuleMetadata{{}}); err == nil {
		t.Fatal("Save() should reject a rule that cannot be encoded")
	}
	if _, err := manager.List([]any{make(chan int)}); err == nil {
		t.Fatal("List() should reject a rule that cannot be encoded")
	}
}

func TestRouteRuleMetadataManagerStorageErrors(t *testing.T) {
	manager := newRouteRuleMetadataManager(t)
	if got, err := manager.List(nil); err != nil || len(got) != 0 {
		t.Fatalf("List() = %#v, %v", got, err)
	}
	if err := manager.db.Update(func(tx *bbolt.Tx) error {
		bucket, createErr := tx.CreateBucketIfNotExists(routeRuleMetadataBucket)
		if createErr != nil {
			return createErr
		}
		return bucket.Put(routeRuleMetadataKey, []byte("{"))
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.List(nil); err == nil {
		t.Fatal("List() should reject corrupt stored metadata")
	}
	if err := manager.ApplyDefaultNames(nil); err == nil {
		t.Fatal("ApplyDefaultNames() should propagate storage errors")
	}
	if err := manager.db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := manager.Save(nil, nil); err == nil {
		t.Fatal("Save() should fail after the database is closed")
	}
}

func TestDefaultRouteRuleNameVariants(t *testing.T) {
	tests := []struct {
		name string
		rule map[string]any
		want string
	}{
		{name: "DNS port hijack", rule: map[string]any{"port": 53, "action": "hijack-dns"}, want: "DNS 请求劫持"},
		{name: "resolve", rule: map[string]any{"action": "resolve"}, want: "域名解析后分流"},
		{name: "direct mode", rule: map[string]any{"clash_mode": "Direct"}, want: "直连模式"},
		{name: "global mode", rule: map[string]any{"clash_mode": "Global"}, want: "全局代理模式"},
		{name: "geosite ads", rule: map[string]any{"rule_set": []any{"geosite-category-ads-all"}}, want: "广告流量拦截"},
		{name: "loyalsoldier direct", rule: map[string]any{"rule_set": []string{"loyalsoldier-direct"}}, want: "中国域名直连"},
		{name: "empty string rules", rule: map[string]any{"rule_set": []string{}}, want: ""},
		{name: "empty any rules", rule: map[string]any{"rule_set": []any{}}, want: ""},
		{name: "invalid any rules", rule: map[string]any{"rule_set": []any{1}}, want: ""},
		{name: "unknown", rule: map[string]any{"action": "route"}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultRouteRuleName(tt.rule); got != tt.want {
				t.Errorf("defaultRouteRuleName() = %q, want %q", got, tt.want)
			}
		})
	}
	if !errors.Is(validateMetadataError(t), ErrInvalidRouteRuleMetadata) {
		t.Fatal("validation errors should wrap ErrInvalidRouteRuleMetadata")
	}
}

func validateMetadataError(t *testing.T) error {
	t.Helper()
	manager := newRouteRuleMetadataManager(t)
	return manager.Save(nil, []model.RouteRuleMetadata{{}})
}

func TestRouteRuleMetadataManagerApplyDefaultNames(t *testing.T) {
	manager := newRouteRuleMetadataManager(t)
	rules := []any{
		map[string]any{"action": "sniff"},
		map[string]any{"protocol": "dns", "action": "hijack-dns"},
		map[string]any{"ip_is_private": true, "outbound": "direct"},
		map[string]any{"network": "icmp", "outbound": "direct"},
		map[string]any{"network": "udp", "port": 443, "action": "reject"},
		map[string]any{"rule_set": []string{"loyalsoldier-reject"}, "outbound": "block"},
		map[string]any{"rule_set": []string{"geosite-cn"}, "outbound": "direct"},
		map[string]any{"rule_set": []string{"geoip-cn"}, "outbound": "direct"},
		map[string]any{"rule_set": []string{"loyalsoldier-proxy"}, "outbound": "proxy"},
		map[string]any{"rule_set": []string{"geosite-google-play"}, "outbound": "proxy"},
	}
	initial := make([]model.RouteRuleMetadata, len(rules))
	initial[0].Name = "自定义嗅探名称"
	if err := manager.Save(rules, initial); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := manager.ApplyDefaultNames(rules); err != nil {
		t.Fatalf("ApplyDefaultNames() error = %v", err)
	}
	got, err := manager.List(rules)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	expected := []string{
		"自定义嗅探名称", "DNS 请求劫持", "私有地址直连", "ICMP 流量直连", "QUIC 流量拦截",
		"广告流量拦截", "中国域名直连", "中国 IP 直连", "代理列表流量走代理", "Google Play 流量走代理",
	}
	for index, name := range expected {
		if got[index].Name != name {
			t.Errorf("metadata[%d].Name = %q, want %q", index, got[index].Name, name)
		}
	}
}

func TestRouteRuleMetadataManagerInitializesDefaultNamesOnce(t *testing.T) {
	manager := newRouteRuleMetadataManager(t)
	rules := []any{map[string]any{"action": "sniff"}}
	if err := manager.InitializeDefaultNames(rules); err != nil {
		t.Fatalf("InitializeDefaultNames() error = %v", err)
	}
	metadata, err := manager.List(rules)
	if err != nil || metadata[0].Name != "协议嗅探" {
		t.Fatalf("List() = %#v, %v", metadata, err)
	}
	if err := manager.Save(rules, []model.RouteRuleMetadata{{}}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := manager.InitializeDefaultNames(rules); err != nil {
		t.Fatalf("second InitializeDefaultNames() error = %v", err)
	}
	metadata, err = manager.List(rules)
	if err != nil || metadata[0].Name != "" {
		t.Fatalf("second List() = %#v, %v", metadata, err)
	}
}
