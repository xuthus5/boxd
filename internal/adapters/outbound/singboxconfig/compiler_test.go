package singboxconfig_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/internal/adapters/outbound/singboxconfig"
)

func TestCompilerRendersStableManagedConfig(t *testing.T) {
	state := control.DefaultDesiredState()
	state.Nodes = []control.Node{{
		ID:       "hk-01",
		Name:     "Hong Kong",
		Protocol: "shadowsocks",
		Server:   "127.0.0.1",
		Port:     8388,
		Options: map[string]any{
			"method":   "aes-128-gcm",
			"password": "secret",
		},
	}}
	state.Groups = []control.Group{{
		ID:       "smart",
		Name:     "Smart",
		Kind:     control.GroupKindURLTest,
		Members:  []string{"hk-01"},
		Selected: "hk-01",
	}}
	state.Chains = []control.ProxyChain{{
		ID:   "relay",
		Name: "Relay",
		Hops: []control.ProxyHop{
			{Kind: control.EgressKindNode, ID: "hk-01"},
			{Kind: control.EgressKindGroup, ID: "smart"},
		},
	}}
	state.Routes.DefaultEgress = control.EgressRef{Kind: control.EgressKindChain, ID: "relay"}
	state.Routes.Presets.AdBlock = true
	state.Routes.Rules = []control.RouteRule{{
		ID:     "media",
		Name:   "Media",
		Action: control.RouteActionEgress,
		Egress: control.EgressRef{Kind: control.EgressKindGroup, ID: "smart"},
		Match: control.RouteMatch{
			DomainSuffixes: []string{".netflix.com"},
			ProcessNames:   []string{"chrome.exe"},
		},
	}}

	compiler := singboxconfig.New()
	first, err := compiler.Compile(context.Background(), state)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	second, err := compiler.Compile(context.Background(), state)
	if err != nil {
		t.Fatalf("Compile() second error = %v", err)
	}
	if first.Revision == "" || first.Revision != second.Revision {
		t.Fatalf("revision = %q and %q, want stable non-empty revision", first.Revision, second.Revision)
	}

	var document map[string]any
	if err := json.Unmarshal(first.Body, &document); err != nil {
		t.Fatalf("generated config is not JSON: %v", err)
	}
	outbounds := document["outbounds"].([]any)
	if !hasOutboundTag(outbounds, control.NodeTag("hk-01")) || !hasOutboundTag(outbounds, control.GroupTag("smart")) || !hasOutboundTag(outbounds, control.ChainTag("relay")) {
		t.Fatalf("outbounds = %#v, want node, group, and chain tags", outbounds)
	}
	route := document["route"].(map[string]any)
	if route["final"] != control.ChainTag("relay") {
		t.Fatalf("route.final = %v, want chain tag", route["final"])
	}
	if route["find_process"] != true {
		t.Fatalf("route.find_process = %v, want true", route["find_process"])
	}
	if len(route["rule_set"].([]any)) != 3 {
		t.Fatalf("route.rule_set = %#v, want 3 definitions", route["rule_set"])
	}
	dns := document["dns"].(map[string]any)
	if dns["final"] != "dns-foreign" {
		t.Fatalf("dns.final = %v, want dns-foreign", dns["final"])
	}
}

func hasOutboundTag(outbounds []any, tag string) bool {
	for _, item := range outbounds {
		outbound, ok := item.(map[string]any)
		if ok && outbound["tag"] == tag {
			return true
		}
	}
	return false
}

func TestCompilerUsesManagedRuleSetDefinitions(t *testing.T) {
	state := control.DefaultDesiredState()
	state.Routes.Presets = control.RoutePresets{}
	state.RuleSets = append(state.RuleSets, control.RuleSetDefinition{
		Name: "custom-ads", URL: "https://example.com/custom-ads.srs", Format: control.RuleSetFormatSource, Enabled: true,
	})
	state.Routes.Rules = []control.RouteRule{{
		ID: "ads", Name: "Ads", Action: control.RouteActionReject,
		Match: control.RouteMatch{RuleSets: []string{"custom-ads"}},
	}}
	document, err := singboxconfig.New().Compile(context.Background(), state)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	var compiled map[string]any
	if err := json.Unmarshal(document.Body, &compiled); err != nil {
		t.Fatalf("generated config is not JSON: %v", err)
	}
	route := compiled["route"].(map[string]any)
	ruleSets := route["rule_set"].([]any)
	if len(ruleSets) != 1 {
		t.Fatalf("rule_set = %#v, want one custom definition", ruleSets)
	}
	custom := ruleSets[0].(map[string]any)
	if custom["tag"] != "custom-ads" || custom["url"] != "https://example.com/custom-ads.srs" || custom["format"] != "source" {
		t.Fatalf("custom rule set = %#v", custom)
	}
}
