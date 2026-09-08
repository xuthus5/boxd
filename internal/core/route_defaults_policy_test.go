package core

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRouteDefaultsProtectDNSBeforeCustomRules(t *testing.T) {
	cfg := policyDefaultsFixture()
	custom := map[string]any{"outbound": "direct"}
	cfg["route"].(map[string]any)["rules"] = []any{custom}
	result, err := NewDefaultRouteInstaller().Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	prelude := []any{
		map[string]any{"port": 53, "action": "hijack-dns"},
		map[string]any{"action": "sniff"},
		map[string]any{"protocol": "dns", "action": "hijack-dns"},
		map[string]any{"rule_set": []string{"loyalsoldier-reject"}, "action": "reject"},
	}
	if len(result.Rules) < len(prelude) || !reflect.DeepEqual(prelude, result.Rules[:len(prelude)]) {
		t.Fatalf("DNS/advertisement prelude = %#v", result.Rules)
	}
	if index := policyRuleIndex(t, result.Rules, custom); index != len(prelude) {
		t.Fatalf("custom rule index = %d, want directly after safety prelude", index)
	}
}

func TestRouteDefaultsResolveAfterDomainDecisions(t *testing.T) {
	result, err := NewDefaultRouteInstaller().Install(policyDefaultsFixture())
	if err != nil {
		t.Fatal(err)
	}
	ordered := []map[string]any{
		{"rule_set": []string{"loyalsoldier-proxy"}, "outbound": "proxy"},
		{"rule_set": []string{"loyalsoldier-direct"}, "outbound": "direct"},
		{"action": "resolve"},
		{"rule_set": []string{"geoip-cn"}, "outbound": "direct"},
	}
	previous := -1
	for _, rule := range ordered {
		index := policyRuleIndex(t, result.Rules, rule)
		if index <= previous {
			t.Fatalf("rule %#v at index %d must follow index %d", rule, index, previous)
		}
		previous = index
	}
}

func TestRouteDefaultsPreserveConditionalRules(t *testing.T) {
	cfg := policyDefaultsFixture()
	custom := []any{
		map[string]any{"action": "sniff", "domain_suffix": []string{"special.test"}},
		map[string]any{"rule_set": []string{"loyalsoldier-direct"}, "invert": true, "outbound": "direct"},
		map[string]any{"network": "udp", "port": 443, "domain": []string{"special.test"}, "action": "reject"},
	}
	cfg["route"].(map[string]any)["rules"] = custom
	result, err := NewDefaultRouteInstaller().Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(custom, result.Rules[4:4+len(custom)]) {
		t.Fatalf("conditional custom rules changed: %#v", result.Rules)
	}
	if policyRuleIndex(t, result.Rules, map[string]any{"action": "sniff"}) != 1 {
		t.Fatal("conditional sniff incorrectly suppressed the unrestricted default")
	}
}

func TestRouteDefaultsReplaceExactLegacyTemplates(t *testing.T) {
	cfg := policyDefaultsFixture()
	legacyQUIC := map[string]any{"network": "udp", "port": 443, "action": "reject"}
	legacyAds := map[string]any{"rule_set": []string{"loyalsoldier-reject"}, "outbound": "block"}
	cfg["route"].(map[string]any)["rules"] = []any{legacyQUIC, legacyAds}
	result, err := NewDefaultRouteInstaller().Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, oldRule := range []map[string]any{legacyQUIC, legacyAds} {
		if policyRuleIndex(t, result.Rules, oldRule) >= 0 {
			t.Fatalf("obsolete default retained: %#v", oldRule)
		}
	}
}

func TestRouteDefaultsAreIdempotentAcrossJSON(t *testing.T) {
	cfg := policyDefaultsFixture()
	installer := NewDefaultRouteInstaller()
	first, err := installer.Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg["route"].(map[string]any)["rules"] = first.Rules
	body, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &cfg); err != nil {
		t.Fatal(err)
	}
	second, err := installer.Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, firstErr := json.Marshal(first.Rules)
	secondJSON, secondErr := json.Marshal(second.Rules)
	if firstErr != nil || secondErr != nil {
		t.Fatalf("marshal results: %v / %v", firstErr, secondErr)
	}
	if string(firstJSON) != string(secondJSON) || len(second.Installed) != 0 {
		t.Fatalf("second install changed policy or added defaults: %#v", second)
	}
}

func TestRouteDefaultsRecognizeEquivalentSerializedBuiltins(t *testing.T) {
	cfg := policyDefaultsFixture()
	cfg["route"].(map[string]any)["rules"] = []any{
		map[string]any{"action": "route", "network": []any{"icmp"}, "outbound": "direct"},
		map[string]any{"type": "default", "action": "route", "rule_set": "loyalsoldier-direct", "outbound": "direct"},
		map[string]any{"action": "reject", "network": []any{"udp"}, "port": []any{443}},
	}
	result, err := NewDefaultRouteInstaller().Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rules) != 13 {
		t.Fatalf("equivalent serialized builtins were retained: %#v", result.Rules)
	}
}

func TestRouteDefaultsReturnInvalidRuleErrors(t *testing.T) {
	cases := []struct {
		name string
		rule any
	}{
		{name: "unencodable rule", rule: map[string]any{"action": make(chan int)}},
		{name: "unknown action", rule: map[string]any{"action": "nonexistent-action"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			cfg := policyDefaultsFixture()
			cfg["route"].(map[string]any)["rules"] = []any{test.rule}
			if _, err := NewDefaultRouteInstaller().Install(cfg); err == nil {
				t.Fatal("invalid existing rule was silently accepted")
			}
		})
	}
}

func TestRouteDefaultsRetainsLegacyBypassWithoutReplacement(t *testing.T) {
	cfg := policyDefaultsFixture()
	cfg["outbounds"].([]any)[0].(map[string]any)["tag"] = "bypass"
	legacy := []any{
		map[string]any{"ip_is_private": true, "outbound": "bypass"},
		map[string]any{"network": "icmp", "outbound": "bypass"},
		outboundRuleSet("loyalsoldier-direct", "bypass"),
		outboundRuleSet("geoip-cn", "bypass"),
	}
	cfg["route"].(map[string]any)["rules"] = legacy
	result, err := NewDefaultRouteInstaller().Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rules) < 4+len(legacy) || !reflect.DeepEqual(legacy, result.Rules[4:4+len(legacy)]) {
		t.Fatalf("working bypass routes without replacements changed: %#v", result.Rules)
	}
}

func TestRouteDefaultsReportMissingPrivatePhase(t *testing.T) {
	cfg := policyDefaultsFixture()
	installer := NewDefaultRouteInstaller()
	first, err := installer.Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	privateRule := map[string]any{"ip_is_private": true, "outbound": "direct"}
	index := policyRuleIndex(t, first.Rules, privateRule)
	if index < 0 {
		t.Fatal("missing private IP route")
	}
	oldRules := append(make([]any, 0, len(first.Rules)-1), first.Rules[:index]...)
	oldRules = append(oldRules, first.Rules[index+1:]...)
	cfg["route"].(map[string]any)["rules"] = oldRules
	result, err := installer.Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Installed) != 1 || !reflect.DeepEqual(privateRule, result.Installed[0]) {
		t.Fatalf("missing private phase installation = %#v, want one private IP rule", result.Installed)
	}
}

func TestRouteDefaultsReplaceLegacyBypassWhenDirectAvailable(t *testing.T) {
	cfg := policyDefaultsFixture()
	cfg["outbounds"] = append(cfg["outbounds"].([]any), map[string]any{"type": "direct", "tag": "bypass"})
	legacy := []any{
		map[string]any{"ip_is_private": true, "outbound": "bypass"},
		map[string]any{"network": "icmp", "outbound": "bypass"},
		outboundRuleSet("loyalsoldier-direct", "bypass"),
		outboundRuleSet("geoip-cn", "bypass"),
	}
	cfg["route"].(map[string]any)["rules"] = legacy
	result, err := NewDefaultRouteInstaller().Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range legacy {
		old := value.(map[string]any)
		if policyRuleIndex(t, result.Rules, old) >= 0 {
			t.Fatalf("replaceable legacy route retained: %#v", old)
		}
		replacement := cloneAnyMap(old)
		replacement["outbound"] = "direct"
		if policyRuleIndex(t, result.Rules, replacement) < 0 {
			t.Fatalf("missing replacement for legacy route: %#v", old)
		}
	}
}
