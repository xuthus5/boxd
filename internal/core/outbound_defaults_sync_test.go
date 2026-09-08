package core

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSyncProxySelectorBlocksWithoutMembers(t *testing.T) {
	outbounds := SyncProxySelector(nil, nil)
	byTag := defaultOutboundTestIndex(outbounds)
	if len(outbounds) != 2 || byTag["block"]["type"] != "block" {
		t.Fatalf("want block and proxy, got %#v", outbounds)
	}
	if !reflect.DeepEqual([]string{"block"}, byTag["proxy"]["outbounds"]) {
		t.Fatalf("unexpected proxy: %#v", byTag["proxy"])
	}
	if _, exists := byTag["proxy"]["default"]; exists {
		t.Fatal("empty proxy must not retain a stale default")
	}
}

func TestSyncProxySelectorPreservesInputAndSelection(t *testing.T) {
	existing := []any{
		map[string]any{"type": "vless", "tag": "a"},
		map[string]any{"type": "vless", "tag": "b"},
		map[string]any{
			"type": "selector", "tag": "proxy", "outbounds": []string{"a", "b"},
			"default": "b", "interrupt_exist_connections": true,
		},
	}
	before, err := json.Marshal(existing)
	if err != nil {
		t.Fatal(err)
	}
	result := SyncProxySelector(existing, []string{"a", "b"})
	proxy := defaultOutboundTestIndex(result)["proxy"]
	if proxy["default"] != "b" || proxy["interrupt_exist_connections"] != true {
		t.Fatalf("user choices changed: %#v", proxy)
	}
	after, err := json.Marshal(existing)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("input mutated: %s", after)
	}
}

func TestSyncProxySelectorPreservesSelectedCustomGroup(t *testing.T) {
	existing := []any{
		map[string]any{"type": "vless", "tag": "a"},
		map[string]any{"type": "urltest", "tag": "custom", "outbounds": []string{"a"}},
		map[string]any{
			"type": "selector", "tag": "proxy", "outbounds": []any{"custom"}, "default": "custom",
		},
	}
	result := SyncProxySelector(existing, []string{"a"})
	proxy := defaultOutboundTestIndex(result)["proxy"]
	if proxy["default"] != "custom" || !reflect.DeepEqual([]string{"a", "custom"}, proxy["outbounds"]) {
		t.Fatalf("selected custom group changed: %#v", proxy)
	}
}

func TestSyncProxySelectorPreservesAllExistingCandidates(t *testing.T) {
	existing := []any{
		map[string]any{"type": "vless", "tag": "a"},
		map[string]any{"type": "vless", "tag": "b"},
		map[string]any{"type": "selector", "tag": "custom", "outbounds": []string{"b"}},
		map[string]any{
			"type": "selector", "tag": "proxy", "outbounds": []string{"a", "b", "custom", "missing"},
			"default": "a",
		},
	}
	result := defaultOutboundTestIndex(SyncProxySelector(existing, []string{"a"}))
	want := []string{"a", "b", "custom"}
	if !reflect.DeepEqual(want, result["proxy"]["outbounds"]) {
		t.Fatalf("want existing candidates %#v, got %#v", want, result["proxy"])
	}
}

func TestSyncProxySelectorPreservesNonSelector(t *testing.T) {
	existing := []any{
		map[string]any{"type": "direct", "tag": "proxy", "routing_mark": 64},
	}
	if got := SyncProxySelector(existing, nil); !reflect.DeepEqual(existing, got) {
		t.Fatalf("want preserved custom proxy %#v, got %#v", existing, got)
	}
}

func TestSyncProxySelectorPreservesCustomTypeAfterNodeRemoval(t *testing.T) {
	existing := []any{
		map[string]any{"type": "urltest", "tag": "proxy", "outbounds": []string{"removed"}, "interval": "5m"},
	}
	result := defaultOutboundTestIndex(SyncProxySelector(existing, nil))
	proxy := result["proxy"]
	if proxy["type"] != "urltest" || proxy["interval"] != "5m" ||
		!reflect.DeepEqual([]string{"block"}, proxy["outbounds"]) {
		t.Fatalf("invalid custom proxy after node removal: %#v", proxy)
	}
}

func TestSyncProxySelectorPrunesRemovedGroupMembers(t *testing.T) {
	existing := []any{
		map[string]any{"type": "vless", "tag": "a"},
		map[string]any{
			"type": "urltest", "tag": "custom", "outbounds": []any{"missing", "a", "a", 1, ""},
			"default": "missing",
		},
		map[string]any{"type": "selector", "tag": "empty", "outbounds": []string{"missing"}},
		map[string]any{"type": "selector", "tag": "invalid", "outbounds": nil},
		nil,
	}
	result := defaultOutboundTestIndex(SyncProxySelector(existing, []string{"a"}))
	if !reflect.DeepEqual([]string{"a"}, result["custom"]["outbounds"]) {
		t.Fatalf("custom members are invalid: %#v", result["custom"])
	}
	if _, exists := result["custom"]["default"]; exists {
		t.Fatal("removed default must not remain")
	}
	for _, tag := range []string{"empty", "invalid"} {
		if !reflect.DeepEqual([]string{"block"}, result[tag]["outbounds"]) {
			t.Fatalf("empty group %s must block: %#v", tag, result[tag])
		}
	}
}

func TestSyncProxySelectorDoesNotTrustBlockTagType(t *testing.T) {
	existing := []any{map[string]any{"type": "direct", "tag": "block", "routing_mark": 64}}
	result := defaultOutboundTestIndex(SyncProxySelector(existing, nil))
	if result["block"]["type"] != "direct" || result["block"]["routing_mark"] != 64 {
		t.Fatal("existing block tag options changed")
	}
	if result["block-fallback"]["type"] != "block" ||
		!reflect.DeepEqual([]string{"block-fallback"}, result["proxy"]["outbounds"]) {
		t.Fatalf("fallback must use an actual blocking outbound: %#v", result)
	}
}

func TestDefaultOutboundsInstallerRetainsNonstandardEntries(t *testing.T) {
	existing := []any{nil, "unknown", map[string]any{"type": "direct"}}
	result, err := NewDefaultOutboundsInstaller().Install(map[string]any{"outbounds": existing})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(existing, result.Outbounds[:len(existing)]) {
		t.Fatalf("existing entries were removed: %#v", result.Outbounds)
	}
}
