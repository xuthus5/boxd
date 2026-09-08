package core

import (
	"errors"
	"reflect"
	"testing"
)

type dependencyDNSInstaller func(map[string]any) (*DNSDefaultsResult, error)

func (f dependencyDNSInstaller) Install(cfg map[string]any) (*DNSDefaultsResult, error) {
	return f(cfg)
}

type dependencyOutboundInstaller func(map[string]any) (*OutboundDefaultsResult, error)

func (f dependencyOutboundInstaller) Install(cfg map[string]any) (*OutboundDefaultsResult, error) {
	return f(cfg)
}

func TestPrepareDNSDefaultsCreatesBlockedProxy(t *testing.T) {
	cfg := map[string]any{}
	result, err := PrepareDNSDefaults(cfg, NewDefaultDNSInstaller(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.DNS["final"] != "dns-remote" {
		t.Fatalf("unexpected default DNS: %#v", result.DNS)
	}
	if cfg["route"].(map[string]any)["final"] != "proxy" {
		t.Fatal("adding the proxy must not leave an implicit direct final")
	}
	outbounds := existingOutbounds(cfg)
	if outbounds["block"]["type"] != "block" {
		t.Fatal("missing fail-closed outbound")
	}
	if !reflect.DeepEqual([]string{"block"}, asStringSlice(outbounds["proxy"]["outbounds"])) {
		t.Fatalf("empty proxy must stay blocked: %#v", outbounds["proxy"])
	}
}

func TestPrepareDNSDefaultsPreservesExplicitUnsafeProxy(t *testing.T) {
	cfg := map[string]any{"outbounds": []any{
		map[string]any{"type": "direct", "tag": "direct"},
		map[string]any{"type": "selector", "tag": "proxy", "outbounds": []string{"direct"}},
	}}
	_, err := PrepareDNSDefaults(cfg, NewDefaultDNSInstaller(), nil)
	if !errors.Is(err, ErrDNSProxyUnsafe) {
		t.Fatalf("want an actionable unsafe-proxy error, got %v", err)
	}
	if len(existingOutbounds(cfg)) != 2 {
		t.Fatal("explicit user policy was changed")
	}
}

func TestPrepareDNSDefaultsPropagatesInstallErrors(t *testing.T) {
	installErr := errors.New("installer failed")
	for _, tt := range []struct {
		name string
		dns  dependencyDNSInstaller
		out  dependencyOutboundInstaller
	}{
		{name: "DNS installer", dns: func(map[string]any) (*DNSDefaultsResult, error) { return nil, installErr }},
		{
			name: "outbound installer",
			dns:  func(map[string]any) (*DNSDefaultsResult, error) { return nil, ErrDNSProxyRequired },
			out:  func(map[string]any) (*OutboundDefaultsResult, error) { return nil, installErr },
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := PrepareDNSDefaults(map[string]any{}, tt.dns, tt.out); !errors.Is(err, installErr) {
				t.Fatalf("installer error was lost: %v", err)
			}
		})
	}
}

func TestEnsureDefaultDNSForRouting(t *testing.T) {
	cfg := map[string]any{}
	if err := EnsureDefaultDNSForRouting(cfg); err != nil {
		t.Fatal(err)
	}
	dns := cfg["dns"].(map[string]any)
	route := cfg["route"].(map[string]any)
	if dns["final"] != "dns-remote" || route["default_domain_resolver"] != "dns-direct" {
		t.Fatalf("routing needs explicit DNS and bootstrap resolution: %#v", cfg)
	}
	custom := map[string]any{"servers": []any{map[string]any{"type": "local", "tag": "custom"}}}
	cfg["dns"] = custom
	if err := EnsureDefaultDNSForRouting(cfg); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(custom, cfg["dns"]) {
		t.Fatal("explicit DNS configuration must be preserved")
	}
}

func TestEnsureDefaultDNSForRoutingReportsUnsafeProxy(t *testing.T) {
	cfg := map[string]any{"outbounds": []any{map[string]any{"tag": "proxy", "type": "direct"}}}
	if err := EnsureDefaultDNSForRouting(cfg); !errors.Is(err, ErrDNSProxyUnsafe) {
		t.Fatalf("unsafe proxy error was lost: %v", err)
	}
}

func TestDefaultProxyFinalPreservesExplicitPolicy(t *testing.T) {
	for _, tt := range []struct {
		name  string
		final string
		want  string
	}{
		{name: "empty uses proxy", want: "proxy"},
		{name: "explicit direct remains direct", final: "direct", want: "direct"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := map[string]any{
				"route":     map[string]any{"final": tt.final},
				"outbounds": []any{map[string]any{"type": "block", "tag": "proxy"}},
			}
			if _, err := PrepareDNSDefaults(cfg, NewDefaultDNSInstaller(), nil); err != nil {
				t.Fatal(err)
			}
			if got := cfg["route"].(map[string]any)["final"]; got != tt.want {
				t.Fatalf("route final: want %q, got %v", tt.want, got)
			}
		})
	}
}
