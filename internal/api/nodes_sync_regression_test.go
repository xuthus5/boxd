package api

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/xuthus5/boxui/internal/model"
)

func TestSyncOutboundsToConfigPreservesNonProxyOutbounds(t *testing.T) {
	nodeManager, subscriptionManager, _, configPath := newAPIManagers(t)
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "wireguard", "tag": "wg"},
		},
		"route": map[string]any{},
	})
	if err := nodeManager.Add(model.Outbound{
		Tag: "node-a", Type: "trojan", Server: "example.com", Port: 443,
		Raw: map[string]any{"password": "secret"},
	}); err != nil {
		t.Fatal(err)
	}

	if err := syncOutboundsToConfig(nodeManager, subscriptionManager, configPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	config := map[string]any{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if outbounds := config["outbounds"].([]any); len(outbounds) < 4 {
		t.Fatalf("outbounds length = %d, want at least 4", len(outbounds))
	}
	for _, expected := range []string{`"wg"`, `"password"`, `"tag": "direct"`} {
		if !strings.Contains(string(data), expected) {
			t.Fatalf("config missing %s: %s", expected, data)
		}
	}
}

func TestSyncOutboundsToConfigKeepsExistingProxySelector(t *testing.T) {
	nodeManager, subscriptionManager, _, configPath := newAPIManagers(t)
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{map[string]any{"type": "wireguard", "tag": "wg"}},
		"route":     map[string]any{"final": "direct"},
	})
	if err := nodeManager.Add(model.Outbound{
		Tag: "proxy", Type: "selector", Raw: map[string]any{"outbounds": []string{"wg"}},
	}); err != nil {
		t.Fatal(err)
	}

	if err := syncOutboundsToConfig(nodeManager, subscriptionManager, configPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), `"proxy"`) != 1 {
		t.Fatalf("config should contain one proxy selector: %s", data)
	}
	if !strings.Contains(string(data), `"final": "direct"`) {
		t.Fatalf("route final should be preserved: %s", data)
	}
}

func TestSyncOutboundsToConfigPreservesCustomGroupsAndBuiltins(t *testing.T) {
	nodeManager, subscriptionManager, _, configPath := newAPIManagers(t)
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "block", "tag": "block"},
			map[string]any{"type": "selector", "tag": "whitelist", "outbounds": []string{"direct", "proxy"}},
			map[string]any{"type": "urltest", "tag": "auto", "outbounds": []string{"node-a"}},
		},
		"route": map[string]any{"final": "proxy"},
	})
	if err := nodeManager.Add(model.Outbound{Tag: "node-a", Type: "trojan", Server: "example.com", Port: 443}); err != nil {
		t.Fatal(err)
	}
	if err := syncOutboundsToConfig(nodeManager, subscriptionManager, configPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"tag": "block"`, `"tag": "whitelist"`, `"tag": "auto"`} {
		if !strings.Contains(string(data), expected) {
			t.Fatalf("config missing preserved outbound %s: %s", expected, data)
		}
	}
}

func TestSyncOutboundsToConfigPreservesExistingManagedOutboundFields(t *testing.T) {
	nodeManager, subscriptionManager, _, configPath := newAPIManagers(t)
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{map[string]any{
			"type": "vless", "tag": "node-a", "server": "old.example", "server_port": 443,
			"routing_mark": 128, "domain_strategy": "prefer_ipv4", "packet_encoding": "",
		}},
		"route": map[string]any{},
	})
	if err := nodeManager.Add(model.Outbound{
		Tag: "node-a", Type: "vless", Server: "example.com", Port: 8443,
		Raw: map[string]any{"uuid": "u", "tls": map[string]any{"enabled": true}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := syncOutboundsToConfig(nodeManager, subscriptionManager, configPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"routing_mark": 128`, `"domain_strategy": "prefer_ipv4"`, `"packet_encoding": ""`} {
		if !strings.Contains(string(data), expected) {
			t.Fatalf("config missing preserved field %s: %s", expected, data)
		}
	}
}
