package core

import "testing"

func TestRouteDefaultsRuntimeDomainAndIPPolicy(t *testing.T) {
	cases := []struct {
		name, domain, mode, outbound string
		remoteLookup                 bool
	}{
		{name: "unknown uses proxy", domain: "unknown.test", outbound: "proxy", remoteLookup: true},
		{name: "IP fallback resolves first", domain: "cn-only.test", outbound: "direct", remoteLookup: true},
		{name: "resolved private destination", domain: "private-only.test", outbound: "direct", remoteLookup: true},
		{name: "proxy domain", domain: "proxy.test", outbound: "proxy"},
		{name: "direct domain", domain: "direct.test", outbound: "direct"},
		{name: "overlap prefers proxy", domain: "overlap.test", outbound: "proxy"},
		{name: "advertisements rejected", domain: "ads.test"},
		{name: "Direct mode", domain: "unknown.test", mode: "Direct", outbound: "direct"},
		{name: "Global mode", domain: "direct.test", mode: "Global", outbound: "proxy"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			runtime := newPolicyRuntime(t, policyDefaultsFixture())
			if test.mode != "" {
				runtime.mode.mode = test.mode
			}
			_ = runtime.route(t, test.domain)
			if runtime.trace.outbound != test.outbound {
				t.Fatalf("selected outbound = %q, want %q; destination=%v resolved=%v", runtime.trace.outbound,
					test.outbound, runtime.trace.metadata.Destination, runtime.trace.metadata.DestinationAddresses)
			}
			if test.remoteLookup && (runtime.remote.calls.Load() == 0 || runtime.direct.calls.Load() != 0) {
				t.Fatalf("IP rule resolution bypassed DNS policy: direct=%d remote=%d",
					runtime.direct.calls.Load(), runtime.remote.calls.Load())
			}
		})
	}
}

func TestRouteDefaultsRuntimeResolveFailureDoesNotDial(t *testing.T) {
	runtime := newPolicyRuntime(t, policyDefaultsFixture())
	runtime.remote.fail = true
	if err := runtime.route(t, "unknown.test"); err == nil {
		t.Fatal("failed DNS resolution should fail the route")
	}
	if runtime.trace.outbound != "" || runtime.direct.calls.Load() != 0 {
		t.Fatalf("DNS failure selected %q or leaked direct DNS", runtime.trace.outbound)
	}
}

func TestRouteDefaultsRuntimePrivateIPsBypassGlobal(t *testing.T) {
	cases := []struct {
		name, address string
	}{
		{name: "IPv4 LAN", address: "10.1.2.3"},
		{name: "IPv6 LAN", address: "fd00::1"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			runtime := newPolicyRuntime(t, policyDefaultsFixture())
			runtime.mode.mode = "Global"
			_ = runtime.route(t, test.address)
			if runtime.trace.outbound != "direct" {
				t.Fatalf("Global private destination selected %q, want direct", runtime.trace.outbound)
			}
			if runtime.direct.calls.Load() != 0 || runtime.remote.calls.Load() != 0 {
				t.Fatal("literal private IP should bypass DNS resolution")
			}
		})
	}
}

func TestRouteDefaultsRuntimeRetainsLegacyBypassWithoutDirect(t *testing.T) {
	cfg := policyDefaultsFixture()
	cfg["outbounds"].([]any)[0].(map[string]any)["tag"] = "bypass"
	cfg["route"].(map[string]any)["rules"] = []any{
		map[string]any{"ip_is_private": true, "outbound": "bypass"},
	}
	runtime := newPolicyRuntime(t, cfg)
	runtime.mode.mode = "Global"
	_ = runtime.route(t, "10.1.2.3")
	if runtime.trace.outbound != "bypass" {
		t.Fatalf("existing LAN route selected %q, want bypass", runtime.trace.outbound)
	}
}
