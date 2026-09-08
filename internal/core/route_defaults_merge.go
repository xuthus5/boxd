package core

import (
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
	result := &RouteDefaultsResult{
		Rules:     make([]any, 0, len(defaults.prelude)+len(custom)+len(defaults.policy)),
		Installed: make([]map[string]any, 0),
	}
	if err := appendDefaultRouteRules(result, defaults.prelude, existingKeys); err != nil {
		return nil, err
	}
	result.Rules = append(result.Rules, custom...)
	if err := appendDefaultRouteRules(result, defaults.policy, existingKeys); err != nil {
		return nil, err
	}
	return result, nil
}

func knownDefaultRouteRules(defaults defaultRouteRules) (map[string]bool, error) {
	known := make(map[string]bool)
	for _, group := range [][]map[string]any{defaults.prelude, defaults.policy, defaults.legacy} {
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
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse route rule: %w", err)
	}
	canonical, err := json.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("normalize route rule: %w", err)
	}
	return string(canonical), nil
}
