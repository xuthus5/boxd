package api

import (
	"encoding/json"
	"fmt"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

var managedNodeTypes = map[string]bool{
	"vless": true, "vmess": true, "trojan": true, "shadowsocks": true,
	"hysteria": true, "hysteria2": true, "tuic": true, "shadowtls": true, "anytls": true,
}

func syncOutboundsToConfig(
	nodeManager *core.NodeManager,
	subManager *core.SubscriptionManager,
	configPath string,
) error {
	cfg, previousConfig, err := readSyncConfig(configPath)
	if err != nil {
		return err
	}
	settings := core.NewSettingsManager(subManager.DB())
	defaults, err := settings.URLTestDefaults()
	if err != nil {
		return err
	}
	previousGroups, err := settings.URLTestManagedGroups()
	if err != nil {
		return err
	}

	subscriptions, err := subManager.List()
	if err != nil {
		return err
	}
	existing := existingOutbounds(cfg)
	existingByTag := indexExistingOutbounds(existing)
	managedGroups := managedURLTestTags(previousGroups, subscriptions)
	outbounds := preserveExistingOutbounds(existing, managedGroups)
	outbounds, err = appendManagedOutbounds(outbounds, nodeManager.List(), existingByTag)
	if err != nil {
		return err
	}
	for _, subscription := range subscriptions {
		outbounds, err = appendManagedOutbounds(outbounds, subscription.Outbounds, existingByTag)
		if err != nil {
			return err
		}
	}
	proxyTags := collectProxyTags(outbounds)
	builder := urlTestGroupBuilder{defaults: defaults, existingByTag: existingByTag}
	outbounds, groupTags := builder.append(outbounds, subscriptions)
	outbounds = upsertProxySelector(outbounds, groupTags, proxyTags)
	cfg["outbounds"] = outbounds
	ensureRouteFinal(cfg)
	commit := syncCommit{path: configPath, previous: previousConfig, groups: settings}
	return commit.write(cfg, groupTags)
}

func existingOutbounds(config map[string]any) []any {
	outbounds, ok := config["outbounds"].([]any)
	if !ok {
		return []any{}
	}
	return outbounds
}

func indexExistingOutbounds(existing []any) map[string]map[string]any {
	byTag := make(map[string]map[string]any)
	for _, outbound := range existing {
		entry, _ := outbound.(map[string]any)
		tag, _ := entry["tag"].(string)
		if tag != "" {
			byTag[tag] = cloneAnyMap(entry)
		}
	}
	return byTag
}

func managedURLTestTags(previous []string, subscriptions []model.Subscription) map[string]bool {
	tags := make(map[string]bool, len(previous)+len(subscriptions))
	for _, tag := range previous {
		if tag != "" {
			tags[tag] = true
		}
	}
	for _, subscription := range subscriptions {
		if subscription.Name != "" {
			tags[subscription.Name] = true
		}
	}
	return tags
}

func preserveExistingOutbounds(existing []any, managedGroups map[string]bool) []any {
	outbounds := []any{map[string]any{"type": "direct", "tag": "direct"}}
	for _, outbound := range existing {
		entry, _ := outbound.(map[string]any)
		if shouldReplaceExistingOutbound(entry, managedGroups) {
			continue
		}
		outbounds = append(outbounds, outbound)
	}
	return outbounds
}

func shouldReplaceExistingOutbound(entry map[string]any, managedGroups map[string]bool) bool {
	if entry == nil {
		return true
	}
	typeName, _ := entry["type"].(string)
	tag, _ := entry["tag"].(string)
	if managedNodeTypes[typeName] || typeName == "direct" || typeName == "dns" {
		return true
	}
	if typeName == "urltest" && managedGroups[tag] {
		return true
	}
	return typeName == "selector" && tag == "proxy"
}

func appendManagedOutbounds(
	outbounds []any,
	managed []model.Outbound,
	existingByTag map[string]map[string]any,
) ([]any, error) {
	for _, outbound := range managed {
		entry, err := buildManagedOutbound(existingByTag[outbound.Tag], outbound)
		if err != nil {
			return nil, err
		}
		outbounds = append(outbounds, entry)
	}
	return outbounds, nil
}

func buildManagedOutbound(existing map[string]any, outbound model.Outbound) (map[string]any, error) {
	entry := cloneAnyMap(existing)
	if entry == nil {
		entry = map[string]any{}
	}
	entry["type"] = outbound.Type
	entry["tag"] = outbound.Tag
	entry["server"] = outbound.Server
	entry["server_port"] = outbound.Port
	if outbound.Raw != nil {
		rawJSON, err := json.Marshal(outbound.Raw)
		if err != nil {
			return nil, fmt.Errorf("encoding outbound %q raw config: %w", outbound.Tag, err)
		}
		rawMap := map[string]any{}
		if err := json.Unmarshal(rawJSON, &rawMap); err != nil {
			return nil, fmt.Errorf("decoding outbound %q raw config: %w", outbound.Tag, err)
		}
		for key, value := range rawMap {
			entry[key] = value
		}
	}
	if isProxyLikeOutboundType(outbound.Type) {
		if _, ok := entry["routing_mark"]; !ok {
			entry["routing_mark"] = 128
		}
	}
	return entry, nil
}

func collectProxyTags(outbounds []any) []string {
	tags := []string{}
	for _, outbound := range outbounds {
		entry, _ := outbound.(map[string]any)
		tag, _ := entry["tag"].(string)
		typeName, _ := entry["type"].(string)
		if tag != "" && isProxySelectorCandidate(typeName) {
			tags = append(tags, tag)
		}
	}
	return tags
}

func isProxySelectorCandidate(typeName string) bool {
	switch typeName {
	case "direct", "block", "dns", "selector", "urltest":
		return false
	default:
		return true
	}
}

type urlTestGroupBuilder struct {
	defaults      model.URLTestDefaults
	existingByTag map[string]map[string]any
}

func (b urlTestGroupBuilder) append(
	outbounds []any,
	subscriptions []model.Subscription,
) ([]any, []string) {
	groupTags := []string{}
	for _, subscription := range subscriptions {
		memberTags := subscriptionProxyTags(subscription)
		resolved := core.ResolveURLTest(b.defaults, subscription.URLTest)
		if len(memberTags) == 0 || !resolved.Enabled {
			continue
		}
		entry := cloneAnyMap(b.existingByTag[subscription.Name])
		if entry == nil {
			entry = map[string]any{}
		}
		entry["type"] = "urltest"
		entry["tag"] = subscription.Name
		entry["outbounds"] = memberTags
		entry["url"] = resolved.URL
		entry["interval"] = resolved.Interval
		entry["tolerance"] = resolved.Tolerance
		outbounds = append(outbounds, entry)
		groupTags = append(groupTags, subscription.Name)
	}
	return outbounds, groupTags
}

func subscriptionProxyTags(subscription model.Subscription) []string {
	tags := []string{}
	for _, outbound := range subscription.Outbounds {
		if isProxyLikeOutboundType(outbound.Type) && outbound.Tag != "" {
			tags = append(tags, outbound.Tag)
		}
	}
	return tags
}

func upsertProxySelector(outbounds []any, groupTags, proxyTags []string) []any {
	members := append([]string{}, groupTags...)
	members = append(members, proxyTags...)
	if len(members) == 0 {
		return outbounds
	}
	defaultTag := members[0]
	for index, outbound := range outbounds {
		entry, _ := outbound.(map[string]any)
		if entry != nil && entry["type"] == "selector" && entry["tag"] == "proxy" {
			entry["outbounds"] = members
			entry["default"] = defaultTag
			outbounds[index] = entry
			return outbounds
		}
	}
	return append(outbounds, map[string]any{
		"type": "selector", "tag": "proxy", "outbounds": members, "default": defaultTag,
	})
}

func ensureRouteFinal(config map[string]any) {
	route, _ := config["route"].(map[string]any)
	if route == nil {
		route = map[string]any{}
		config["route"] = route
	}
	if _, ok := route["final"]; !ok {
		route["final"] = "proxy"
	}
}
