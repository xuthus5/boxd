package core

import (
	"reflect"
	"testing"
)

func TestDefaultOutboundsInstallerFailsClosedWithoutNodes(t *testing.T) {
	result, err := NewDefaultOutboundsInstaller().Install(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Outbounds) != 3 {
		t.Fatalf("want 3 default outbounds, got %#v", result.Outbounds)
	}
	byTag := defaultOutboundTestIndex(result.Outbounds)
	if !reflect.DeepEqual([]string{"block"}, byTag["proxy"]["outbounds"]) {
		t.Fatalf("want blocking proxy fallback, got %#v", byTag["proxy"])
	}
	if _, exists := byTag["direct"]["routing_mark"]; exists {
		t.Fatalf("unexpected routing mark: %#v", byTag["direct"])
	}
}

func TestDefaultOutboundsInstallerAddsOnlyReferencedGroups(t *testing.T) {
	result, err := NewDefaultOutboundsInstaller().Install(map[string]any{
		"outbounds": []any{map[string]any{"type": "vless", "tag": "node-a"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Outbounds) != 4 {
		t.Fatalf("want node and 3 default outbounds, got %#v", result.Outbounds)
	}
	byTag := defaultOutboundTestIndex(result.Outbounds)
	if !reflect.DeepEqual([]string{"node-a"}, byTag["proxy"]["outbounds"]) {
		t.Fatalf("want only node-a, got %#v", byTag["proxy"])
	}
}

func TestDefaultOutboundsInstallerExcludesNonProxyCandidates(t *testing.T) {
	existing := []any{
		map[string]any{"type": "direct", "tag": "direct"},
		map[string]any{"type": "block", "tag": "block"},
		map[string]any{"type": "dns", "tag": "dns-out"},
		map[string]any{"type": "urltest", "tag": "custom", "outbounds": []string{"node-a"}},
		map[string]any{"type": "vless", "tag": "node-a"},
	}
	result, err := NewDefaultOutboundsInstaller().Install(map[string]any{"outbounds": existing})
	if err != nil {
		t.Fatal(err)
	}
	proxy := defaultOutboundTestIndex(result.Outbounds)["proxy"]
	if !reflect.DeepEqual([]string{"node-a"}, proxy["outbounds"]) {
		t.Fatalf("want only node-a, got %#v", proxy)
	}
}

func TestDefaultOutboundsInstallerPreservesExplicitEntries(t *testing.T) {
	existing := []any{
		map[string]any{"type": "direct", "tag": "direct", "routing_mark": 64},
		map[string]any{"type": "urltest", "tag": "proxy", "outbounds": []string{"node-a"}, "interval": "5m"},
		map[string]any{"type": "selector", "tag": "auto", "outbounds": []string{"node-a"}},
		map[string]any{"type": "dns", "tag": "dns-out"},
		map[string]any{"type": "vless", "tag": "node-a"},
	}
	result, err := NewDefaultOutboundsInstaller().Install(map[string]any{"outbounds": existing})
	if err != nil {
		t.Fatal(err)
	}
	byTag := defaultOutboundTestIndex(result.Outbounds)
	for _, item := range existing {
		entry := item.(map[string]any)
		tag := entry["tag"].(string)
		if !reflect.DeepEqual(entry, byTag[tag]) {
			t.Fatalf("want preserved %s = %#v, got %#v", tag, entry, byTag[tag])
		}
	}
	second, err := NewDefaultOutboundsInstaller().Install(map[string]any{"outbounds": result.Outbounds})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Outbounds, second.Outbounds) {
		t.Fatalf("reinstall changed configuration: %#v", second.Outbounds)
	}
}

func TestDefaultOutboundsInstallerHandlesReservedBlockTag(t *testing.T) {
	existing := []any{map[string]any{"type": "direct", "tag": "block"}}
	result, err := NewDefaultOutboundsInstaller().Install(map[string]any{"outbounds": existing})
	if err != nil {
		t.Fatal(err)
	}
	byTag := defaultOutboundTestIndex(result.Outbounds)
	if byTag["block"]["type"] != "direct" || byTag["block-fallback"]["type"] != "block" {
		t.Fatalf("invalid blocking fallback: %#v", result.Outbounds)
	}
	if len(result.Installed) != len(result.Outbounds)-len(existing) {
		t.Fatalf("installed count does not reflect added outbounds: %#v", result.Installed)
	}
}

func defaultOutboundTestIndex(outbounds []any) map[string]map[string]any {
	byTag := make(map[string]map[string]any)
	for _, item := range outbounds {
		entry, _ := item.(map[string]any)
		tag, _ := entry["tag"].(string)
		byTag[tag] = entry
	}
	return byTag
}
