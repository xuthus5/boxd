package core

import (
	"errors"
	"fmt"
)

var (
	ErrDNSProxyRequired = errors.New("default DNS requires a proxy outbound")
	ErrDNSProxyUnsafe   = errors.New("default DNS proxy must not contain direct fallback or invalid outbound references")
)

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
	proxy, err := selectDNSProxy(cfg)
	if err != nil {
		return nil, err
	}
	ruleSets := existingRuleSetTags(cfg)
	dns := map[string]any{
		"servers":           defaultDNSServers(proxy),
		"strategy":          "prefer_ipv4",
		"rules":             defaultDNSRules(ruleSets),
		"final":             "dns-remote",
		"independent_cache": true,
	}

	return &DNSDefaultsResult{
		DNS:                   dns,
		Installed:             cloneAnyMap(dns),
		DefaultDomainResolver: "dns-direct",
	}, nil
}

func selectDNSProxy(cfg map[string]any) (string, error) {
	outbounds := existingOutbounds(cfg)
	tag := "proxy"
	if outbounds[tag] == nil {
		route, _ := cfg["route"].(map[string]any)
		tag, _ = route["final"].(string)
		if outbound := outbounds[tag]; outbound == nil || outbound["type"] == "direct" {
			return "", ErrDNSProxyRequired
		}
	}
	if !dnsProxyIsSafe(outbounds, tag, make(map[string]bool)) {
		return "", fmt.Errorf("%w: %s", ErrDNSProxyUnsafe, tag)
	}
	return tag, nil
}

func dnsProxyIsSafe(outbounds map[string]map[string]any, tag string, visiting map[string]bool) bool {
	outbound := outbounds[tag]
	if outbound == nil || visiting[tag] {
		return false
	}
	visiting[tag] = true
	defer delete(visiting, tag)
	switch stringValue(outbound["type"]) {
	case "", "direct", "dns":
		return false
	case "selector", "urltest":
		members := asStringSlice(outbound["outbounds"])
		if len(members) == 0 {
			return false
		}
		for _, member := range members {
			if !dnsProxyIsSafe(outbounds, member, visiting) {
				return false
			}
		}
	}
	return true
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

func defaultDNSServers(proxy string) []any {
	return []any{
		map[string]any{"type": "https", "server": "223.5.5.5", "tag": "dns-direct"},
		map[string]any{
			"type": "https", "server": "8.8.8.8", "tag": "dns-remote", "detour": proxy,
			"tls": map[string]any{"server_name": "dns.google"},
		},
	}
}

func defaultDNSRules(ruleSets map[string]bool) []any {
	rules := make([]any, 0)
	if tag := preferredRuleSet(ruleSets, "loyalsoldier-reject", "geosite-category-ads-all"); tag != "" {
		rules = append(rules, predefinedDNSRule(tag))
	}
	rules = append(rules,
		map[string]any{"clash_mode": "Direct", "server": "dns-direct"},
		map[string]any{"clash_mode": "Global", "server": "dns-remote"},
	)
	// 两个列表存在交集，DNS 与路由保持相同的代理优先级。
	if tag := preferredRuleSet(ruleSets, "loyalsoldier-proxy", "geosite-google-play"); tag != "" {
		rules = append(rules, routedDNSRule(tag, "dns-remote"))
	}
	if tag := preferredRuleSet(ruleSets, "loyalsoldier-direct", "geosite-cn"); tag != "" {
		rules = append(rules, routedDNSRule(tag, "dns-direct"))
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
