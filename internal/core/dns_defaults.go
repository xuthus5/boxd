package core

type DNSDefaultsInstaller interface {
	Install(cfg map[string]any) (*DNSDefaultsResult, error)
}

type DNSDefaultsResult struct {
	DNS       map[string]any
	Installed map[string]any
}

type DefaultDNSInstaller struct{}

func NewDefaultDNSInstaller() *DefaultDNSInstaller {
	return &DefaultDNSInstaller{}
}

func (i *DefaultDNSInstaller) Install(cfg map[string]any) (*DNSDefaultsResult, error) {
	outboundTags := existingOutboundTags(cfg)

	detour := "proxy"
	if !outboundTags[detour] {
		if route, _ := cfg["route"].(map[string]any); route != nil {
			if final, _ := route["final"].(string); final != "" && outboundTags[final] {
				detour = final
			}
		}
		if !outboundTags[detour] {
			detour = "direct"
		}
	}

	servers := []any{
		map[string]any{"address": "rcode://success", "tag": "dns-block"},
		map[string]any{"address": "local", "detour": "direct", "tag": "dns-local"},
		map[string]any{"address": "https://223.5.5.5/dns-query", "address_resolver": "dns-local", "detour": "direct", "strategy": "ipv4_only", "tag": "dns-direct"},
		map[string]any{"address": "https://dns.google/dns-query", "address_resolver": "dns-direct", "detour": detour, "strategy": "ipv4_only", "tag": "dns-remote"},
		map[string]any{"address": "fakeip", "strategy": "ipv4_only", "tag": "dns-fake"},
	}

	ruleSets := existingRuleSetTags(cfg)
	rules := []any{
		map[string]any{"domain": []string{"dns.google"}, "server": "dns-direct"},
		map[string]any{"outbound": []string{"any"}, "server": "dns-direct"},
	}

	switch {
	case ruleSets["loyalsoldier-reject"]:
		rules = append(rules, map[string]any{"disable_cache": true, "rule_set": []string{"loyalsoldier-reject"}, "server": "dns-block"})
	case ruleSets["geosite-category-ads-all"]:
		rules = append(rules, map[string]any{"disable_cache": true, "rule_set": []string{"geosite-category-ads-all"}, "server": "dns-block"})
	}

	switch {
	case ruleSets["loyalsoldier-direct"]:
		rules = append(rules, map[string]any{"rule_set": []string{"loyalsoldier-direct"}, "action": "route", "server": "dns-direct"})
	case ruleSets["geosite-cn"]:
		rules = append(rules, map[string]any{"rule_set": []string{"geosite-cn"}, "action": "route", "server": "dns-direct"})
	}

	switch {
	case ruleSets["loyalsoldier-proxy"]:
		rules = append(rules, map[string]any{"rule_set": []string{"loyalsoldier-proxy"}, "action": "route", "server": "dns-remote"})
	case ruleSets["geosite-google-play"]:
		rules = append(rules, map[string]any{"rule_set": []string{"geosite-google-play"}, "action": "route", "server": "dns-remote"})
	}

	dns := map[string]any{
		"servers":           servers,
		"rules":             rules,
		"final":             "dns-remote",
		"independent_cache": true,
		"fakeip": map[string]any{
			"enabled":     true,
			"inet4_range": "198.18.0.0/15",
			"inet6_range": "fc00::/18",
		},
	}

	return &DNSDefaultsResult{
		DNS:       dns,
		Installed: cloneAnyMap(dns),
	}, nil
}
