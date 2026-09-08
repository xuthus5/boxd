package core

import (
	"encoding/json"
	"path/filepath"
	"testing"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

func TestSyncProxySelectorPreservesCachedSelection(t *testing.T) {
	for _, tt := range []struct {
		name     string
		selected string
		members  []string
	}{
		{name: "custom group", selected: "custom", members: []string{"node-a", "node-b"}},
		{name: "individual node", selected: "node-b", members: []string{"node-a", "custom"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := cachedProxySyncConfig(t.TempDir())
			first := startCachedProxyRuntime(t, cfg)
			if first.selector.Now() != "node-a" {
				t.Fatal("fixture must start with the configured default")
			}
			if !first.selector.SelectOutbound(tt.selected) {
				t.Fatalf("cannot select %q", tt.selected)
			}
			first.close(t)
			cfg["outbounds"] = SyncProxySelector(cfg["outbounds"].([]any), tt.members)
			second := startCachedProxyRuntime(t, cfg)
			if got := second.selector.Now(); got != tt.selected {
				t.Fatalf("want cached selection %q after sync and reopen, got %q", tt.selected, got)
			}
		})
	}
}

func TestSyncProxySelectorDropsCachedBlockingPlaceholder(t *testing.T) {
	cfg := cachedProxySyncConfig(t.TempDir())
	outbounds := []any{
		map[string]any{"type": "block", "tag": "block"},
		map[string]any{"type": "block", "tag": "node-a"},
		map[string]any{
			"type": "selector", "tag": "proxy", "outbounds": []string{"node-a", "block"}, "default": "node-a",
		},
	}
	cfg["outbounds"] = outbounds
	first := startCachedProxyRuntime(t, cfg)
	if !first.selector.SelectOutbound("block") {
		t.Fatal("cannot persist a blocking choice")
	}
	first.close(t)
	withoutNodes := []any{outbounds[0], outbounds[2]}
	cfg["outbounds"] = SyncProxySelector(withoutNodes, nil)
	blocked := startCachedProxyRuntime(t, cfg)
	if blocked.selector.Now() != "block" {
		t.Fatal("removing the final node must block")
	}
	blocked.close(t)
	withNode := append(cfg["outbounds"].([]any), outbounds[1])
	cfg["outbounds"] = SyncProxySelector(withNode, []string{"node-a"})
	connected := startCachedProxyRuntime(t, cfg)
	if got := connected.selector.Now(); got != "node-a" {
		t.Fatalf("first node must replace the cached block placeholder, got %q", got)
	}
}

func cachedProxySyncConfig(dataDir string) map[string]any {
	return map[string]any{
		"outbounds": []any{
			map[string]any{"type": "block", "tag": "node-a"},
			map[string]any{"type": "block", "tag": "node-b"},
			map[string]any{"type": "selector", "tag": "custom", "outbounds": []string{"node-b"}},
			map[string]any{
				"type": "selector", "tag": "proxy", "outbounds": []string{"node-a", "node-b", "custom"},
				"default": "node-a",
			},
		},
		"route": map[string]any{"final": "proxy"},
		"log":   map[string]any{"disabled": true},
		"experimental": map[string]any{
			"cache_file": map[string]any{"enabled": true, "path": filepath.Join(dataDir, "cache.db")},
		},
	}
}

type cachedProxyRuntime struct {
	instance *box.Box
	selector interface {
		Now() string
		SelectOutbound(string) bool
	}
}

func startCachedProxyRuntime(t *testing.T, cfg map[string]any) *cachedProxyRuntime {
	t.Helper()
	ctx := include.Context(t.Context())
	body, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var options option.Options
	if err := options.UnmarshalJSONContext(ctx, body); err != nil {
		t.Fatal(err)
	}
	instance, err := box.New(box.Options{Context: ctx, Options: options})
	if err != nil {
		t.Fatal(err)
	}
	runtime := &cachedProxyRuntime{instance: instance}
	t.Cleanup(func() { runtime.close(t) })
	if err := instance.Start(); err != nil {
		t.Fatal(err)
	}
	outbound, ok := instance.Outbound().Outbound("proxy")
	if !ok {
		t.Fatal("proxy outbound is missing")
	}
	runtime.selector, ok = outbound.(interface {
		Now() string
		SelectOutbound(string) bool
	})
	if !ok {
		t.Fatal("proxy outbound is not selectable")
	}
	return runtime
}

func (r *cachedProxyRuntime) close(t *testing.T) {
	t.Helper()
	if r.instance == nil {
		return
	}
	instance := r.instance
	r.instance = nil
	if err := instance.Close(); err != nil {
		t.Errorf("close cached proxy runtime: %v", err)
	}
}
