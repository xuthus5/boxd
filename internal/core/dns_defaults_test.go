package core

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

func TestDefaultDNSInstallerInstall(t *testing.T) {
	cfg := policyDefaultsFixture()
	result, err := NewDefaultDNSInstaller().Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.DefaultDomainResolver != "dns-direct" || result.DNS["independent_cache"] != true {
		t.Fatalf("DNS bootstrap/cache defaults = %#v", result)
	}
	if result.DNS["strategy"] != "prefer_ipv4" {
		t.Fatalf("DNS strategy = %v", result.DNS["strategy"])
	}
	cfg["dns"] = result.DNS
	cfg["route"].(map[string]any)["default_domain_resolver"] = result.DefaultDomainResolver
	body, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, issue := range AnalyzeConfig(body).Issues {
		if issue.Severity == model.ConfigDiagnosticSeverityError {
			t.Fatalf("invalid default configuration: %#v", issue)
		}
	}
	rules := result.DNS["rules"].([]any)
	if index := policyRuleIndex(t, rules, predefinedDNSRule("loyalsoldier-reject")); index != 0 {
		t.Fatalf("advertisement rejection index = %d, want first", index)
	}
}

func TestDefaultDNSInstallerUsesRouteAndGeositeFallbacks(t *testing.T) {
	cfg := map[string]any{
		"outbounds": []any{
			map[string]any{"type": "block", "tag": "block"},
			map[string]any{"type": "selector", "tag": "secure", "outbounds": []string{"block"}},
		},
		"route": map[string]any{
			"final": "secure",
			"rule_set": []any{
				policyInlineSet("geosite-cn", "direct.test"),
				policyInlineSet("geosite-google-play", "proxy.test"),
				policyInlineSet("geosite-category-ads-all", "ads.test"),
			},
		},
	}
	result, err := NewDefaultDNSInstaller().Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	remote := result.DNS["servers"].([]any)[1].(map[string]any)
	if remote["detour"] != "secure" {
		t.Fatalf("remote DNS detour = %v, want secure", remote["detour"])
	}
	rules := result.DNS["rules"].([]any)
	for _, want := range []map[string]any{
		predefinedDNSRule("geosite-category-ads-all"),
		routedDNSRule("geosite-google-play", "dns-remote"),
		routedDNSRule("geosite-cn", "dns-direct"),
	} {
		if policyRuleIndex(t, rules, want) < 0 {
			t.Fatalf("missing fallback rule %#v", want)
		}
	}
}

func TestDefaultDNSInstallerUsesOnlyExistingDetours(t *testing.T) {
	cases := []struct {
		name   string
		direct map[string]any
	}{
		{name: "plain direct", direct: map[string]any{"type": "direct", "tag": "direct"}},
		{name: "zero mark", direct: map[string]any{"type": "direct", "tag": "direct", "routing_mark": 0}},
		{name: "custom direct", direct: map[string]any{"type": "direct", "tag": "direct", "bind_interface": "lo"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			cfg := policyDefaultsFixture()
			cfg["outbounds"].([]any)[0] = test.direct
			result, err := NewDefaultDNSInstaller().Install(cfg)
			if err != nil {
				t.Fatal(err)
			}
			direct := result.DNS["servers"].([]any)[0].(map[string]any)
			if _, exists := direct["detour"]; exists {
				t.Fatal("DNS direct transport must use its own underlay dialer")
			}
		})
	}
}

func TestDefaultDNSInstallerClassifiesUnavailableProxy(t *testing.T) {
	cases := []struct {
		name string
		cfg  map[string]any
		want error
	}{
		{name: "empty config", cfg: nil, want: ErrDNSProxyRequired},
		{name: "direct final", cfg: map[string]any{
			"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}},
			"route":     map[string]any{"final": "direct"},
		}, want: ErrDNSProxyRequired},
		{name: "invalid proxy", cfg: map[string]any{
			"outbounds": []any{map[string]any{"type": "selector", "tag": "proxy"}},
		}, want: ErrDNSProxyUnsafe},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewDefaultDNSInstaller().Install(test.cfg)
			if !errors.Is(err, test.want) {
				t.Fatalf("Install error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestDefaultDNSInstallerChecksNestedProxyGroups(t *testing.T) {
	cases := []struct {
		name      string
		member    map[string]any
		wantError bool
	}{
		{name: "block placeholder", member: map[string]any{"type": "block", "tag": "node"}},
		{name: "proxy node", member: map[string]any{"type": "socks", "tag": "node", "server": "192.0.2.1", "server_port": 1080}},
		{name: "direct leaf", member: map[string]any{"type": "direct", "tag": "node"}, wantError: true},
		{name: "DNS leaf", member: map[string]any{"type": "dns", "tag": "node"}, wantError: true},
		{name: "missing type", member: map[string]any{"tag": "node"}, wantError: true},
		{name: "cycle", member: map[string]any{"type": "selector", "tag": "node", "outbounds": []string{"proxy"}}, wantError: true},
		{name: "missing reference", member: map[string]any{"type": "urltest", "tag": "node", "outbounds": []string{"missing"}}, wantError: true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			cfg := map[string]any{"outbounds": []any{
				map[string]any{"type": "selector", "tag": "proxy", "outbounds": []any{"auto"}},
				map[string]any{"type": "urltest", "tag": "auto", "outbounds": []string{"node"}},
				test.member,
			}}
			_, err := NewDefaultDNSInstaller().Install(cfg)
			if (err != nil) != test.wantError {
				t.Fatalf("Install error = %v, want error %v", err, test.wantError)
			}
		})
	}
}
