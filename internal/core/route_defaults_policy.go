package core

const (
	defaultDNSPort = 53
	legacyQUICPort = 443
)

type defaultRouteRules struct {
	prelude []map[string]any
	// safety 为私网/ICMP 旁路，始终排在全局模式之前，Global 模式不会把局域网流量送入代理。
	safety []map[string]any
	// modes 为 clash_mode 全局模式，排在自定义规则之前，使全局直连/代理覆盖已有路由规则。
	modes  []map[string]any
	policy []map[string]any
	legacy []map[string]any
}

func buildDefaultRouteRules(cfg map[string]any) defaultRouteRules {
	sets := existingRuleSetTags(cfg)
	outbounds := existingOutboundTags(cfg)
	prelude := []map[string]any{
		{"port": defaultDNSPort, "action": "hijack-dns"},
		{"action": "sniff"},
		{"protocol": "dns", "action": "hijack-dns"},
	}
	if tag := preferredRuleSet(sets, "loyalsoldier-reject", "geosite-category-ads-all"); tag != "" {
		prelude = append(prelude, map[string]any{"rule_set": []string{tag}, "action": "reject"})
	}
	safety, modes, policy := defaultRoutePolicy(sets, outbounds)
	return defaultRouteRules{
		prelude: prelude,
		safety:  safety,
		modes:   modes,
		policy:  policy,
		legacy:  legacyDefaultRouteRules(outbounds),
	}
}

// defaultRoutePolicy 按用途拆分为三段：safety 先于 modes，两者都先于自定义规则。
func defaultRoutePolicy(sets, outbounds map[string]bool) (safety, modes, policy []map[string]any) {
	if outbounds["direct"] {
		// 已知私网 IP 在 Global 模式前旁路，解析得到的私网地址仍由后面的规则处理。
		safety = append(safety,
			map[string]any{"ip_is_private": true, "outbound": "direct"},
			map[string]any{"network": "icmp", "outbound": "direct"},
		)
		modes = append(modes, map[string]any{"clash_mode": "Direct", "outbound": "direct"})
	}
	if outbounds["proxy"] {
		modes = append(modes, map[string]any{"clash_mode": "Global", "outbound": "proxy"})
		if tag := preferredRuleSet(sets, "loyalsoldier-proxy", "geosite-google-play"); tag != "" {
			policy = append(policy, outboundRuleSet(tag, "proxy"))
		}
	}
	if !outbounds["direct"] {
		return safety, modes, policy
	}
	if tag := preferredRuleSet(sets, "loyalsoldier-direct", "geosite-cn"); tag != "" {
		policy = append(policy, outboundRuleSet(tag, "direct"))
	}
	// 域名策略已经决定的连接无需解析；其余连接按 DNS 分流解析后再做 IP 匹配。
	policy = append(policy,
		map[string]any{"action": "resolve"},
		map[string]any{"ip_is_private": true, "outbound": "direct"},
	)
	if sets["geoip-cn"] {
		policy = append(policy, outboundRuleSet("geoip-cn", "direct"))
	}
	return safety, modes, policy
}

func preferredRuleSet(sets map[string]bool, primary, fallback string) string {
	if sets[primary] {
		return primary
	}
	if sets[fallback] {
		return fallback
	}
	return ""
}

func outboundRuleSet(tag, outbound string) map[string]any {
	return map[string]any{"rule_set": []string{tag}, "outbound": outbound}
}

// legacyDefaultRouteRules 仅识别历史内置的完整模板，且保留没有可用替代的直连规则。
func legacyDefaultRouteRules(outbounds map[string]bool) []map[string]any {
	rules := []map[string]any{{"network": "udp", "port": legacyQUICPort, "action": "reject"}}
	if outbounds["direct"] {
		for _, outbound := range []string{"direct", "bypass"} {
			rules = append(rules,
				map[string]any{"ip_is_private": true, "outbound": outbound},
				map[string]any{"network": "icmp", "outbound": outbound},
			)
			for _, tag := range []string{"loyalsoldier-direct", "geosite-cn", "geoip-cn"} {
				rules = append(rules, outboundRuleSet(tag, outbound))
			}
		}
	}
	for _, tag := range []string{"loyalsoldier-reject", "geosite-category-ads-all"} {
		rules = append(rules, outboundRuleSet(tag, "block"))
	}
	for _, tag := range []string{"loyalsoldier-proxy", "geosite-google-play"} {
		rules = append(rules, outboundRuleSet(tag, "proxy"))
	}
	return rules
}
