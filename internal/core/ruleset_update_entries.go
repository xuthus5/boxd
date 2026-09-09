package core

import (
	"maps"
	"strings"
)

const ruleSetHTTPPolicyKey = "_boxd_http_policy"

type ruleSetHTTPPolicy struct {
	tag     string
	source  string
	managed bool
}

// 展开内核的 {tag} 模板，原始配置与内联客户端对象保持不变。
func expandConfiguredRuleSetTags(entry map[string]any) []map[string]any {
	tags := stringValues(entry["tag"])
	if values, ok := entry["tag"].([]string); ok {
		tags = values
	}
	expanded := make([]map[string]any, 0, len(tags))
	seen := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		item := maps.Clone(entry)
		item["tag"] = tag
		for _, field := range []string{"path", "url", "initial_path"} {
			if value, exists := item[field].(string); exists {
				item[field] = strings.ReplaceAll(value, "{tag}", tag)
			}
		}
		expanded = append(expanded, item)
	}
	return expanded
}

func configuredRuleSetHTTPPolicy(cfg, entry map[string]any) ruleSetHTTPPolicy {
	if stringValue(entry["type"]) != "remote" {
		return ruleSetHTTPPolicy{}
	}
	if tag, ok := entry["http_client"].(string); ok && tag != "" {
		return ruleSetHTTPPolicy{tag: tag, source: "rule_set", managed: true}
	}
	if _, inline := entry["http_client"].(map[string]any); inline {
		return ruleSetHTTPPolicy{source: "inline", managed: true}
	}
	if tag := stringValue(entry["download_detour"]); tag != "" {
		return ruleSetHTTPPolicy{tag: tag, source: "legacy_detour", managed: true}
	}
	return defaultRuleSetHTTPPolicy(cfg)
}

func defaultRuleSetHTTPPolicy(cfg map[string]any) ruleSetHTTPPolicy {
	route := objectValue(cfg["route"])
	if tag := stringValue(route["default_http_client"]); tag != "" {
		return ruleSetHTTPPolicy{tag: tag, source: "route_default", managed: true}
	}
	clients, _ := cfg["http_clients"].([]any)
	if len(clients) > 0 {
		return ruleSetHTTPPolicy{tag: stringValue(objectValue(clients[0])["tag"]), source: "global_default", managed: true}
	}
	if !ruleSetDefaultOutboundIsPlainDirect(cfg) {
		tag := stringValue(route["final"])
		if outbounds := diagnosticEntries(cfg, "outbounds"); tag == "" && len(outbounds) > 0 {
			tag = outbounds[0].tag
		}
		return ruleSetHTTPPolicy{tag: tag, source: "default_outbound", managed: true}
	}
	return ruleSetHTTPPolicy{}
}

func ruleSetDefaultOutboundIsPlainDirect(cfg map[string]any) bool {
	outbounds := diagnosticEntries(cfg, "outbounds")
	tag := stringValue(objectValue(cfg["route"])["final"])
	if tag == "" {
		if len(outbounds) == 0 {
			return true
		}
		tag = outbounds[0].tag
	}
	entries := append(outbounds, diagnosticEntries(cfg, "endpoints")...)
	entry, found := diagnosticEntryByTag(entries, tag)
	if !found || entry.typeName != "direct" {
		return false
	}
	for key := range objectAtPath(cfg, entry.path) {
		if key != "tag" && key != "type" {
			return false
		}
	}
	return true
}

func ruleSetEntryHTTPPolicy(entry map[string]any) ruleSetHTTPPolicy {
	if policy, ok := entry[ruleSetHTTPPolicyKey].(ruleSetHTTPPolicy); ok {
		return policy
	}
	return configuredRuleSetHTTPPolicy(nil, entry)
}

func ruleSetEntries(cfg map[string]any) []map[string]any {
	route, _ := cfg["route"].(map[string]any)
	if route == nil {
		return nil
	}
	raw, _ := route["rule_set"].([]any)
	out := make([]map[string]any, 0, len(raw))
	seen := make(map[string]bool)
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		for _, entry := range expandConfiguredRuleSetTags(m) {
			tag := stringValue(entry["tag"])
			if seen[tag] {
				continue
			}
			seen[tag] = true
			entry[ruleSetHTTPPolicyKey] = configuredRuleSetHTTPPolicy(cfg, entry)
			out = append(out, entry)
		}
	}
	return out
}

func selectRuleSets(entries []map[string]any, req RuleSetUpdateRequest) []map[string]any {
	tagFilter := ruleSetSelectionFilter(req.Tags, strings.TrimSpace)
	typeFilter := ruleSetSelectionFilter(req.Types, func(value string) string { return strings.ToLower(strings.TrimSpace(value)) })
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		tag, _ := entry["tag"].(string)
		typ, _ := entry["type"].(string)
		if typ == "" {
			typ = "inline"
		}
		if len(tagFilter) > 0 {
			if _, ok := tagFilter[tag]; !ok {
				continue
			}
		}
		if len(typeFilter) > 0 {
			if _, ok := typeFilter[typ]; !ok {
				continue
			}
		}
		out = append(out, entry)
	}
	return out
}

func ruleSetSelectionFilter(values []string, normalize func(string) string) map[string]struct{} {
	filter := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value = normalize(value); value != "" {
			filter[value] = struct{}{}
		}
	}
	return filter
}
