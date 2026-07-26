package core

type DNSDefaultsInstaller interface {
	Install(cfg map[string]any) (*DNSDefaultsResult, error)
}

type DNSDefaultsResult struct {
	DNS                   map[string]any
	Installed             map[string]any
	DefaultDomainResolver string
}

type DefaultDNSInstaller struct{}

func NewDefaultDNSInstaller() *DefaultDNSInstaller {
	return &DefaultDNSInstaller{}
}

func (i *DefaultDNSInstaller) Install(cfg map[string]any) (*DNSDefaultsResult, error) {
	detour := selectDNSDetour(cfg)
	ruleSets := existingRuleSetTags(cfg)
	servers := defaultDNSServers(detour)
	rules := defaultDNSRules(ruleSets)
	dns := map[string]any{
		"servers":           servers,
		"strategy":          "ipv4_only",
		"rules":             rules,
		"final":             "dns-remote",
		"independent_cache": true,
	}

	return &DNSDefaultsResult{
		DNS:                   dns,
		Installed:             cloneAnyMap(dns),
		DefaultDomainResolver: "dns-direct",
	}, nil
}

func selectDNSDetour(cfg map[string]any) string {
	outboundTags := existingOutboundTags(cfg)
	if outboundTags["proxy"] {
		return "proxy"
	}
	route, _ := cfg["route"].(map[string]any)
	final, _ := route["final"].(string)
	if final != "" && outboundTags[final] {
		return final
	}
	return "direct"
}

func defaultDNSServers(detour string) []any {
	return []any{
		map[string]any{"type": "local", "detour": "direct", "tag": "dns-local"},
		map[string]any{"type": "https", "server": "223.5.5.5", "detour": "direct", "tag": "dns-direct"},
		map[string]any{
			"type": "https", "server": "dns.google", "domain_resolver": "dns-direct",
			"detour": detour, "tag": "dns-remote",
		},
		map[string]any{"type": "fakeip", "inet4_range": "198.18.0.0/15", "inet6_range": "fc00::/18", "tag": "dns-fake"},
	}
}

func defaultDNSRules(ruleSets map[string]bool) []any {
	rules := []any{
		map[string]any{"domain": []string{"dns.google"}, "server": "dns-direct"},
	}

	switch {
	case ruleSets["loyalsoldier-reject"]:
		rules = append(rules, predefinedDNSRule("loyalsoldier-reject"))
	case ruleSets["geosite-category-ads-all"]:
		rules = append(rules, predefinedDNSRule("geosite-category-ads-all"))
	}

	switch {
	case ruleSets["loyalsoldier-direct"]:
		rules = append(rules, routedDNSRule("loyalsoldier-direct", "dns-direct"))
	case ruleSets["geosite-cn"]:
		rules = append(rules, routedDNSRule("geosite-cn", "dns-direct"))
	}

	switch {
	case ruleSets["loyalsoldier-proxy"]:
		rules = append(rules, routedDNSRule("loyalsoldier-proxy", "dns-remote"))
	case ruleSets["geosite-google-play"]:
		rules = append(rules, routedDNSRule("geosite-google-play", "dns-remote"))
	}
	return rules
}

func predefinedDNSRule(ruleSet string) map[string]any {
	return map[string]any{
		"rule_set": []string{ruleSet},
		"action":   "predefined",
		"rcode":    "NOERROR",
	}
}

func routedDNSRule(ruleSet, server string) map[string]any {
	return map[string]any{
		"rule_set": []string{ruleSet},
		"action":   "route",
		"server":   server,
	}
}
