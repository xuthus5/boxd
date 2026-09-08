package core

import "testing"

func TestDefaultRouteInstallerInstall(t *testing.T) {
	result, err := NewDefaultRouteInstaller().Install(policyDefaultsFixture())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rules) != 13 || len(result.Installed) != 13 {
		t.Fatalf("rules/installed = %d/%d, want 13/13", len(result.Rules), len(result.Installed))
	}
	for _, want := range []map[string]any{
		{"clash_mode": "Direct", "outbound": "direct"},
		{"clash_mode": "Global", "outbound": "proxy"},
	} {
		if policyRuleIndex(t, result.Rules, want) < 0 {
			t.Fatalf("missing mode route %#v", want)
		}
	}
}

func TestDefaultRouteInstallerGeositeFallback(t *testing.T) {
	cfg := policyDefaultsFixture()
	cfg["route"].(map[string]any)["rule_set"] = []any{
		policyInlineSet("geosite-cn", "direct.test"),
		policyInlineSet("geosite-google-play", "proxy.test"),
		policyInlineSet("geosite-category-ads-all", "ads.test"),
		map[string]any{"tag": "geoip-cn"},
	}
	result, err := NewDefaultRouteInstaller().Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, tag := range []string{"geosite-cn", "geoip-cn", "geosite-google-play", "geosite-category-ads-all"} {
		found := false
		for _, value := range result.Rules {
			if ruleSetTag(value.(map[string]any)) == tag {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing fallback set %s", tag)
		}
	}
}

func TestDefaultRouteInstallerMinimalOutbounds(t *testing.T) {
	cases := []struct {
		name      string
		outbounds []any
		count     int
	}{
		{name: "no outbounds", count: 3},
		{name: "direct only", outbounds: []any{map[string]any{"type": "direct", "tag": "direct"}}, count: 8},
		{name: "proxy only", outbounds: []any{map[string]any{"type": "block", "tag": "proxy"}}, count: 4},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			result, err := NewDefaultRouteInstaller().Install(map[string]any{"outbounds": test.outbounds})
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Rules) != test.count {
				t.Fatalf("rules = %d, want %d", len(result.Rules), test.count)
			}
		})
	}
}

func TestDefaultRouteInstallerDedup(t *testing.T) {
	cfg := policyDefaultsFixture()
	cfg["route"].(map[string]any)["rules"] = []any{
		map[string]any{"action": "sniff"},
		map[string]any{"action": "sniff"},
	}
	result, err := NewDefaultRouteInstaller().Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rules) != 13 || len(result.Installed) != 12 {
		t.Fatalf("rules/installed = %d/%d, want 13/12", len(result.Rules), len(result.Installed))
	}
}

func TestStringValue(t *testing.T) {
	if got := stringValue("hello"); got != "hello" {
		t.Errorf("got %q, want 'hello'", got)
	}
	if got := stringValue(123); got != "" {
		t.Errorf("got %q, want ''", got)
	}
	if got := stringValue(nil); got != "" {
		t.Errorf("got %q, want ''", got)
	}
}

func TestPortValue(t *testing.T) {
	if got := portValue("443"); got != "443" {
		t.Errorf("string got %q, want 443", got)
	}
	if got := portValue(443); got != "443" {
		t.Errorf("int got %q, want 443", got)
	}
	if got := portValue(int64(443)); got != "443" {
		t.Errorf("int64 got %q, want 443", got)
	}
	if got := portValue(float64(443)); got != "443" {
		t.Errorf("float64 got %q, want 443", got)
	}
	if got := portValue(nil); got != "" {
		t.Errorf("nil got %q, want ''", got)
	}
	if got := portValue(false); got != "" {
		t.Errorf("bool got %q, want ''", got)
	}
}
