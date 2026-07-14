package core

import (
	"maps"
	"strconv"
)

type RouteDefaultsInstaller interface {
	Install(cfg map[string]any) (*RouteDefaultsResult, error)
}

type RouteDefaultsResult struct {
	Rules     []any
	Installed []map[string]any
}

type DefaultRouteInstaller struct{}

func NewDefaultRouteInstaller() *DefaultRouteInstaller {
	return &DefaultRouteInstaller{}
}

func (i *DefaultRouteInstaller) Install(cfg map[string]any) (*RouteDefaultsResult, error) {
	route, _ := cfg["route"].(map[string]any)
	var existing []any
	if route != nil {
		existing, _ = route["rules"].([]any)
	}
	ruleSets := existingRuleSetTags(cfg)
	outbounds := existingOutboundTags(cfg)
	directTag := "direct"
	if outbounds["bypass"] {
		directTag = "bypass"
	}

	defaults := make([]map[string]any, 0, 12)
	// 1. 嗅探：协议探测后再按后续规则分流。
	defaults = append(defaults, map[string]any{"action": "sniff"})
	// 2. 劫持 DNS：把 DNS 请求交给内核 DNS 模块处理。
	defaults = append(defaults, map[string]any{"protocol": "dns", "action": "hijack-dns"})
	if outbounds[directTag] {
		// 3. 绕过局域网 / 私有 IP 地址。
		defaults = append(defaults, map[string]any{"ip_is_private": true, "outbound": directTag})
	}
	if outbounds[directTag] {
		// 4. 绕过 ICMP：让 ICMP（ping）走直连出站，避免被拦截或代理。
		defaults = append(defaults, map[string]any{"network": "icmp", "outbound": directTag})
	}
	if outbounds["block"] {
		// 5. 屏蔽 QUIC：拒绝 UDP 443（QUIC），强制回落到 TCP，便于分流与审计。
		defaults = append(defaults, map[string]any{"network": "udp", "port": 443, "action": "reject"})
		// 6. 屏蔽广告：优先使用 Loyalsoldier reject 规则集，回退到 geosite 广告规则集。
		switch {
		case ruleSets["loyalsoldier-reject"]:
			defaults = append(defaults, map[string]any{"rule_set": []string{"loyalsoldier-reject"}, "outbound": "block"})
		case ruleSets["geosite-category-ads-all"]:
			defaults = append(defaults, map[string]any{"rule_set": []string{"geosite-category-ads-all"}, "outbound": "block"})
		}
	}
	if outbounds[directTag] {
		// 7. 中国直连：优先 Loyalsoldier direct，回退 geosite-cn。
		switch {
		case ruleSets["loyalsoldier-direct"]:
			defaults = append(defaults, map[string]any{"rule_set": []string{"loyalsoldier-direct"}, "outbound": directTag})
		case ruleSets["geosite-cn"]:
			defaults = append(defaults, map[string]any{"rule_set": []string{"geosite-cn"}, "outbound": directTag})
		}
		// 8. 中国 IP 直连：使用独立的 geoip-cn 规则集。
		if ruleSets["geoip-cn"] {
			defaults = append(defaults, map[string]any{"rule_set": []string{"geoip-cn"}, "outbound": directTag})
		}
	}
	if outbounds["proxy"] {
		// 9. 代理：优先 Loyalsoldier proxy，回退 geosite-google-play。
		switch {
		case ruleSets["loyalsoldier-proxy"]:
			defaults = append(defaults, map[string]any{"rule_set": []string{"loyalsoldier-proxy"}, "outbound": "proxy"})
		case ruleSets["geosite-google-play"]:
			defaults = append(defaults, map[string]any{"rule_set": []string{"geosite-google-play"}, "outbound": "proxy"})
		}
	}

	merged := make([]any, 0, len(existing)+len(defaults))
	seen := make(map[string]struct{}, len(existing)+len(defaults))
	for _, item := range existing {
		if m, ok := item.(map[string]any); ok {
			merged = append(merged, item)
			seen[ruleSignature(m)] = struct{}{}
			continue
		}
		merged = append(merged, item)
	}
	installed := make([]map[string]any, 0, len(defaults))
	for _, rule := range defaults {
		sig := ruleSignature(rule)
		if _, ok := seen[sig]; ok {
			continue
		}
		seen[sig] = struct{}{}
		merged = append(merged, cloneAnyMap(rule))
		installed = append(installed, cloneAnyMap(rule))
	}

	return &RouteDefaultsResult{Rules: merged, Installed: installed}, nil
}

func existingRuleSetTags(cfg map[string]any) map[string]bool {
	result := make(map[string]bool)
	route, _ := cfg["route"].(map[string]any)
	if route == nil {
		return result
	}
	sets, _ := route["rule_set"].([]any)
	for _, item := range sets {
		if m, ok := item.(map[string]any); ok {
			if tag, _ := m["tag"].(string); tag != "" {
				result[tag] = true
			}
		}
	}
	return result
}

func existingOutboundTags(cfg map[string]any) map[string]bool {
	result := make(map[string]bool)
	outbounds, _ := cfg["outbounds"].([]any)
	for _, item := range outbounds {
		if m, ok := item.(map[string]any); ok {
			if tag, _ := m["tag"].(string); tag != "" {
				result[tag] = true
			}
		}
	}
	return result
}

func ruleSignature(rule map[string]any) string {
	if network, _ := rule["network"].(string); network != "" {
		return "network:" + network + "->" + portValue(rule["port"]) + "->" + stringValue(rule["action"]) + "->" + stringValue(rule["outbound"])
	}
	if networks, ok := rule["network"].([]any); ok && len(networks) > 0 {
		if first, _ := networks[0].(string); first != "" {
			return "network:" + first + "->" + portValue(rule["port"]) + "->" + stringValue(rule["action"]) + "->" + stringValue(rule["outbound"])
		}
	}
	if protocol, _ := rule["protocol"].(string); protocol != "" {
		return "protocol:" + protocol + "->" + stringValue(rule["outbound"]) + "->" + stringValue(rule["action"])
	}
	if private, _ := rule["ip_is_private"].(bool); private {
		return "ip_is_private->" + stringValue(rule["outbound"])
	}
	if sets, ok := rule["rule_set"].([]string); ok && len(sets) > 0 {
		return "rule_set:" + sets[0] + "->" + stringValue(rule["outbound"])
	}
	if sets, ok := rule["rule_set"].([]any); ok && len(sets) > 0 {
		if first, _ := sets[0].(string); first != "" {
			return "rule_set:" + first + "->" + stringValue(rule["outbound"])
		}
	}
	return stringValue(rule["outbound"])
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func portValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return ""
	}
}

func cloneAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	maps.Copy(out, in)
	return out
}
