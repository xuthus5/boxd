package api

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"
)

type outboundSyncFunc func(*core.NodeManager, *core.SubscriptionManager, string) error

var outboundSyncImplementations = []struct {
	name string
	sync outboundSyncFunc
}{
	{name: "http", sync: syncOutboundsToConfig},
	{name: "service", sync: service.SyncOutboundsToConfig},
}

func TestSyncOutboundsFailClosedLifecycle(t *testing.T) {
	for _, implementation := range outboundSyncImplementations {
		t.Run(implementation.name, func(t *testing.T) {
			nodes, subscriptions, _, path := newAPIManagers(t)
			writeConfigFile(t, path, blockingProxyTestConfig())
			if err := implementation.sync(nodes, subscriptions, path); err != nil {
				t.Fatal(err)
			}
			assertSyncProxyMembers(t, path, []string{"block"})
			if err := nodes.Add(safeSyncTestNode("node-a")); err != nil {
				t.Fatal(err)
			}
			if err := implementation.sync(nodes, subscriptions, path); err != nil {
				t.Fatal(err)
			}
			assertSyncProxyMembers(t, path, []string{"node-a"})
			entry := outboundByTag(t, readConfigMap(t, path), "node-a")
			if _, exists := entry["routing_mark"]; exists {
				t.Fatalf("unexpected routing mark: %#v", entry)
			}
			if err := nodes.Delete("node-a"); err != nil {
				t.Fatal(err)
			}
			if err := implementation.sync(nodes, subscriptions, path); err != nil {
				t.Fatal(err)
			}
			assertSyncProxyMembers(t, path, []string{"block"})
		})
	}
}

func TestSyncOutboundsPreservesUserChoices(t *testing.T) {
	for _, implementation := range outboundSyncImplementations {
		t.Run(implementation.name, func(t *testing.T) {
			nodes, subscriptions, _, path := newAPIManagers(t)
			cfg := blockingProxyTestConfig()
			entries := cfg["outbounds"].([]any)
			entries[0].(map[string]any)["routing_mark"] = 64
			proxy := entries[2].(map[string]any)
			proxy["outbounds"] = []string{"node-a", "node-b"}
			proxy["default"] = "node-b"
			proxy["interrupt_exist_connections"] = true
			cfg["outbounds"] = append(entries, map[string]any{"type": "dns", "tag": "dns-out"})
			writeConfigFile(t, path, cfg)
			for _, tag := range []string{"node-a", "node-b"} {
				if err := nodes.Add(safeSyncTestNode(tag)); err != nil {
					t.Fatal(err)
				}
			}
			if err := implementation.sync(nodes, subscriptions, path); err != nil {
				t.Fatal(err)
			}
			after := readConfigMap(t, path)
			proxy = outboundByTag(t, after, "proxy")
			if proxy["default"] != "node-b" || proxy["interrupt_exist_connections"] != true {
				t.Fatalf("proxy preferences lost: %#v", proxy)
			}
			if outboundByTag(t, after, "direct")["routing_mark"] != float64(64) {
				t.Fatal("direct routing mark was changed")
			}
			_ = outboundByTag(t, after, "dns-out")
		})
	}
}

func TestSyncOutboundsPreservesProxyTagType(t *testing.T) {
	for _, implementation := range outboundSyncImplementations {
		t.Run(implementation.name, func(t *testing.T) {
			nodes, subscriptions, _, path := newAPIManagers(t)
			cfg := blockingProxyTestConfig()
			cfg["outbounds"].([]any)[2] = map[string]any{"type": "direct", "tag": "proxy", "routing_mark": 64}
			writeConfigFile(t, path, cfg)
			if err := nodes.Add(safeSyncTestNode("node-a")); err != nil {
				t.Fatal(err)
			}
			if err := implementation.sync(nodes, subscriptions, path); err != nil {
				t.Fatal(err)
			}
			after := readConfigMap(t, path)
			proxy := outboundByTag(t, after, "proxy")
			if proxy["type"] != "direct" || proxy["routing_mark"] != float64(64) {
				t.Fatalf("proxy type or options changed: %#v", proxy)
			}
			assertSyncTopologyValid(t, after)
		})
	}
}

func TestSyncOutboundsPreservesChosenCustomGroup(t *testing.T) {
	for _, implementation := range outboundSyncImplementations {
		t.Run(implementation.name, func(t *testing.T) {
			nodes, subscriptions, _, path := newAPIManagers(t)
			cfg := blockingProxyTestConfig()
			entries := cfg["outbounds"].([]any)
			entries[2].(map[string]any)["outbounds"] = []string{"custom"}
			entries[2].(map[string]any)["default"] = "custom"
			cfg["outbounds"] = append(entries, map[string]any{
				"type": "urltest", "tag": "custom", "outbounds": []string{"node-a"},
			})
			writeConfigFile(t, path, cfg)
			if err := nodes.Add(safeSyncTestNode("node-a")); err != nil {
				t.Fatal(err)
			}
			if err := implementation.sync(nodes, subscriptions, path); err != nil {
				t.Fatal(err)
			}
			proxy := outboundByTag(t, readConfigMap(t, path), "proxy")
			if proxy["default"] != "custom" {
				t.Fatalf("custom group selection was lost: %#v", proxy)
			}
			if err := nodes.Delete("node-a"); err != nil {
				t.Fatal(err)
			}
			if err := implementation.sync(nodes, subscriptions, path); err != nil {
				t.Fatal(err)
			}
			assertSyncProxyMembers(t, path, []string{"block"})
		})
	}
}

func TestSyncOutboundsTreatsEmptyFinalAsUnconfigured(t *testing.T) {
	for _, implementation := range outboundSyncImplementations {
		t.Run(implementation.name, func(t *testing.T) {
			nodes, subscriptions, _, path := newAPIManagers(t)
			cfg := blockingProxyTestConfig()
			cfg["route"].(map[string]any)["final"] = ""
			writeConfigFile(t, path, cfg)
			if err := implementation.sync(nodes, subscriptions, path); err != nil {
				t.Fatal(err)
			}
			route := readConfigMap(t, path)["route"].(map[string]any)
			if route["final"] != "proxy" {
				t.Fatalf("unconfigured final must use proxy, got %#v", route)
			}
		})
	}
}

func blockingProxyTestConfig() map[string]any {
	return map[string]any{
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "block", "tag": "block"},
			map[string]any{"type": "selector", "tag": "proxy", "outbounds": []string{"block"}},
		},
		"route": map[string]any{"final": "proxy"},
	}
}

func safeSyncTestNode(tag string) model.Outbound {
	return model.Outbound{
		Tag: tag, Type: "vless", Server: "192.0.2.1", Port: 443,
		Raw: map[string]any{"uuid": "00000000-0000-0000-0000-000000000001"},
	}
}

func assertSyncProxyMembers(t *testing.T, path string, expected []string) {
	t.Helper()
	cfg := readConfigMap(t, path)
	entry := outboundByTag(t, cfg, "proxy")
	members := make([]string, 0, len(expected))
	for _, value := range entry["outbounds"].([]any) {
		members = append(members, value.(string))
	}
	if !reflect.DeepEqual(expected, members) {
		t.Fatalf("want proxy members %#v, got %#v", expected, members)
	}
	assertSyncTopologyValid(t, cfg)
}

func assertSyncTopologyValid(t *testing.T, cfg map[string]any) {
	t.Helper()
	body, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, issue := range core.AnalyzeConfig(body).Issues {
		if issue.Severity == model.ConfigDiagnosticSeverityError {
			t.Fatalf("invalid synchronized configuration: %#v", issue)
		}
	}
}
