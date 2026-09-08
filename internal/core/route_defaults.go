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
	existing, _ := route["rules"].([]any)
	return mergeDefaultRouteRules(existing, buildDefaultRouteRules(cfg))
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
