package core

import (
	"errors"
	"slices"
	"testing"
)

func TestDefaultPolicyUsesVPNEndpointsAsProxyCandidates(t *testing.T) {
	for _, kind := range []string{"openvpn-client", "openconnect", "wireguard", "tailscale"} {
		t.Run(kind, func(t *testing.T) {
			cfg := map[string]any{"endpoints": []any{map[string]any{"type": kind, "tag": "vpn"}}}
			result, err := NewDefaultOutboundsInstaller().Install(cfg)
			if err != nil {
				t.Fatal(err)
			}
			cfg["outbounds"] = result.Outbounds
			proxy := existingOutbounds(cfg)["proxy"]
			if !slices.Contains(asStringSlice(proxy["outbounds"]), "vpn") {
				t.Fatal("configured VPN endpoint must be a selectable proxy candidate")
			}
			if _, err := NewDefaultDNSInstaller().Install(cfg); err != nil {
				t.Fatalf("DNS detour through VPN endpoint rejected: %v", err)
			}
			if !setupProxyReady(cfg) {
				t.Fatal("VPN endpoint must satisfy the configured proxy step")
			}
		})
	}
}

func TestDefaultPolicyExcludesBridgeAndVPNServer(t *testing.T) {
	cfg := map[string]any{
		"outbounds": []any{map[string]any{"type": "bridge", "tag": "lan"}},
		"endpoints": []any{map[string]any{"type": "openvpn-server", "tag": "server"}},
	}
	result, err := NewDefaultOutboundsInstaller().Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg["outbounds"] = result.Outbounds
	if members := asStringSlice(existingOutbounds(cfg)["proxy"]["outbounds"]); !slices.Equal(members, []string{"block"}) {
		t.Fatalf("non-proxy forwarding must not enter the proxy selector: %v", members)
	}
	if setupProxyReady(cfg) {
		t.Fatal("local bridge or VPN server is not an upstream proxy")
	}
	cfg["outbounds"] = []any{map[string]any{"type": "bridge", "tag": "proxy"}}
	if _, err := NewDefaultDNSInstaller().Install(cfg); !errors.Is(err, ErrDNSProxyUnsafe) {
		t.Fatalf("bridge is not a safe DNS proxy detour: %v", err)
	}
}

func TestConfiguredProxySynchronizationKeepsVPNSelection(t *testing.T) {
	cfg := map[string]any{
		"outbounds": []any{
			map[string]any{"type": "vless", "tag": "node"},
			map[string]any{"type": "selector", "tag": "proxy", "default": "vpn", "outbounds": []string{"vpn"}},
		},
		"endpoints": []any{map[string]any{"type": "openvpn-client", "tag": "vpn"}},
	}
	cfg["outbounds"] = SyncConfiguredProxySelector(cfg, []string{"node"})
	proxy := existingOutbounds(cfg)["proxy"]
	if proxy["default"] != "vpn" || !slices.Equal(asStringSlice(proxy["outbounds"]), []string{"node", "vpn"}) {
		t.Fatalf("synchronizing nodes lost the VPN selection: %#v", proxy)
	}
	cfg["outbounds"] = SyncConfiguredProxySelector(cfg, nil)
	if members := asStringSlice(existingOutbounds(cfg)["proxy"]["outbounds"]); !slices.Equal(members, []string{"vpn", "node"}) {
		t.Fatalf("a valid VPN must remain usable without managed nodes: %v", members)
	}
}

func TestDefaultPolicyRespectsEndpointWithProxyTag(t *testing.T) {
	cfg := map[string]any{"endpoints": []any{map[string]any{"type": "openconnect", "tag": "proxy"}}}
	result, err := NewDefaultOutboundsInstaller().Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg["outbounds"] = result.Outbounds
	if existingOutbounds(cfg)["proxy"]["type"] != "openconnect" || len(result.Outbounds) != 2 {
		t.Fatal("an endpoint proxy tag must not be shadowed by a generated selector")
	}
	if len(SyncConfiguredProxySelector(cfg, nil)) != 2 {
		t.Fatal("node synchronization must preserve the endpoint's proxy tag")
	}
}

func TestDefaultPolicyRecognizesRuleSetMultipleTags(t *testing.T) {
	cfg := policyDefaultsFixture()
	route := cfg["route"].(map[string]any)
	for _, item := range route["rule_set"].([]any) {
		entry := item.(map[string]any)
		entry["tag"] = []string{entry["tag"].(string), "alias-" + entry["tag"].(string)}
	}
	tags := existingRuleSetTags(cfg)
	if !tags["loyalsoldier-direct"] || !tags["alias-loyalsoldier-direct"] || !tags["geoip-cn"] {
		t.Fatalf("multi-tag rule sets were not indexed: %#v", tags)
	}
	result, err := NewDefaultDNSInstaller().Install(cfg)
	if err != nil || policyRuleIndex(t, result.DNS["rules"].([]any), predefinedDNSRule("loyalsoldier-reject")) != 0 {
		t.Fatalf("DNS defaults must use existing rule-set aliases: %v", err)
	}
}

func TestDefaultPolicyDoesNotShadowServerEndpointBlockTag(t *testing.T) {
	cfg := map[string]any{"endpoints": []any{map[string]any{"type": "openvpn-server", "tag": "block"}}}
	result, err := NewDefaultOutboundsInstaller().Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg["outbounds"] = result.Outbounds
	entries := existingOutbounds(cfg)
	if entries["block"]["type"] != "openvpn-server" || entries["block-fallback"]["type"] != "block" {
		t.Fatal("safe fallback must use a tag distinct from an existing endpoint")
	}
	if !slices.Equal(asStringSlice(entries["proxy"]["outbounds"]), []string{"block-fallback"}) {
		t.Fatal("a VPN server must not be used as the fallback proxy")
	}
}
