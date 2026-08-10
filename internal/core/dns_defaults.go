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

type dnsDefaultDetours struct {
	direct string
	remote string
}

func NewDefaultDNSInstaller() *DefaultDNSInstaller {
	return &DefaultDNSInstaller{}
}

func (i *DefaultDNSInstaller) Install(cfg map[string]any) (*DNSDefaultsResult, error) {
	detours := selectDNSDetours(cfg)
	ruleSets := existingRuleSetTags(cfg)
	servers := defaultDNSServers(detours)
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

func selectDNSDetours(cfg map[string]any) dnsDefaultDetours {
	outbounds := existingOutbounds(cfg)
	detours := dnsDefaultDetours{}
	if ob := outbounds["direct"]; ob != nil && !isEmptyDirectOutbound(ob) {
		detours.direct = "direct"
	}
	if ob := outbounds["proxy"]; ob != nil && !isEmptyDirectOutbound(ob) {
		detours.remote = "proxy"
		return detours
	}
	route, _ := cfg["route"].(map[string]any)
	final, _ := route["final"].(string)
	if ob := outbounds[final]; final != "" && ob != nil && !isEmptyDirectOutbound(ob) {
		detours.remote = final
		return detours
	}
	detours.remote = detours.direct
	return detours
}

// existingOutbounds 返回按 tag 索引的既有出站配置。
func existingOutbounds(cfg map[string]any) map[string]map[string]any {
	result := make(map[string]map[string]any)
	outbounds, _ := cfg["outbounds"].([]any)
	for _, item := range outbounds {
		if m, ok := item.(map[string]any); ok {
			if tag, _ := m["tag"].(string); tag != "" {
				result[tag] = m
			}
		}
	}
	return result
}

// isEmptyDirectOutbound 判断出站是否为无任何拨号选项的空 direct。
// sing-box 内核会拒绝 DNS detour 指向空 direct，等价于默认 direct 拨号，因此应省略 detour。
func isEmptyDirectOutbound(ob map[string]any) bool {
	if ob["type"] != "direct" {
		return false
	}
	for key := range ob {
		switch key {
		case "tag", "type":
		default:
			return false
		}
	}
	return true
}

func defaultDNSServers(detours dnsDefaultDetours) []any {
	return []any{
		withDNSDetour(map[string]any{"type": "local", "tag": "dns-local"}, detours.direct),
		withDNSDetour(map[string]any{"type": "https", "server": "223.5.5.5", "tag": "dns-direct"}, detours.direct),
		withDNSDetour(map[string]any{
			"type": "https", "server": "dns.google",
			"domain_resolver": "dns-direct", "tag": "dns-remote",
		}, detours.remote),
		map[string]any{"type": "fakeip", "inet4_range": "198.18.0.0/15", "inet6_range": "fc00::/18", "tag": "dns-fake"},
	}
}

func withDNSDetour(server map[string]any, detour string) map[string]any {
	if detour != "" {
		server["detour"] = detour
	}
	return server
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
