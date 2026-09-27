package core

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sagernet/sing-box/option"
)

func mergeDefaultRouteRules(existing []any, defaults defaultRouteRules) (*RouteDefaultsResult, error) {
	known, err := knownDefaultRouteRules(defaults)
	if err != nil {
		return nil, err
	}
	custom, existingKeys, err := customRouteRules(existing, known)
	if err != nil {
		return nil, err
	}
	// 私网/ICMP 旁路保持高于全局模式的优先级；其余自定义规则让位于全局模式。
	bypass, custom := splitBypassRules(custom)
	result := &RouteDefaultsResult{
		Rules:     make([]any, 0, len(defaults.prelude)+len(bypass)+len(defaults.safety)+len(defaults.modes)+len(custom)+len(defaults.policy)),
		Installed: make([]map[string]any, 0),
	}
	if err := appendDefaultRouteRules(result, defaults.prelude, existingKeys); err != nil {
		return nil, err
	}
	result.Rules = append(result.Rules, bypass...)
	if err := appendDefaultRouteRules(result, defaults.safety, existingKeys); err != nil {
		return nil, err
	}
	if err := appendDefaultRouteRules(result, defaults.modes, existingKeys); err != nil {
		return nil, err
	}
	result.Rules = append(result.Rules, custom...)
	if err := appendDefaultRouteRules(result, defaults.policy, existingKeys); err != nil {
		return nil, err
	}
	return result, nil
}

func splitBypassRules(rules []any) (bypass, rest []any) {
	for _, value := range rules {
		if rule, ok := value.(map[string]any); ok && isBypassRule(rule) {
			bypass = append(bypass, value)
			continue
		}
		rest = append(rest, value)
	}
	return bypass, rest
}

func isBypassRule(rule map[string]any) bool {
	if private, _ := rule["ip_is_private"].(bool); private {
		return true
	}
	for _, network := range asStringSlice(rule["network"]) {
		if network == "icmp" {
			return true
		}
	}
	return false
}

func knownDefaultRouteRules(defaults defaultRouteRules) (map[string]bool, error) {
	known := make(map[string]bool)
	for _, group := range [][]map[string]any{defaults.prelude, defaults.safety, defaults.modes, defaults.policy, defaults.legacy} {
		for _, rule := range group {
			key, err := routeRuleKey(rule)
			if err != nil {
				return nil, err
			}
			known[key] = true
		}
	}
	return known, nil
}

func customRouteRules(existing []any, known map[string]bool) ([]any, map[string]int, error) {
	custom := make([]any, 0)
	keys := make(map[string]int, len(existing))
	for _, value := range existing {
		key, err := routeRuleKey(value)
		if err != nil {
			return nil, nil, err
		}
		keys[key]++
		if !known[key] {
			custom = append(custom, value)
		}
	}
	return custom, keys, nil
}

func appendDefaultRouteRules(result *RouteDefaultsResult, rules []map[string]any, existing map[string]int) error {
	for _, rule := range rules {
		key, err := routeRuleKey(rule)
		if err != nil {
			return err
		}
		result.Rules = append(result.Rules, cloneAnyMap(rule))
		if existing[key] > 0 {
			existing[key]--
			continue
		}
		result.Installed = append(result.Installed, cloneAnyMap(rule))
	}
	return nil
}

func routeRuleKey(rule any) (string, error) {
	body, err := json.Marshal(rule)
	if err != nil {
		return "", fmt.Errorf("encode route rule: %w", err)
	}
	// 使用当前内核的完整结构规范化标量/数组与省略的 route action，保留所有匹配条件。
	var parsed option.Rule
	if err := parsed.UnmarshalJSONContext(context.Background(), body); err != nil {
		return "", fmt.Errorf("parse route rule: %w", err)
	}
	canonical, err := json.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("normalize route rule: %w", err)
	}
	return string(canonical), nil
}
