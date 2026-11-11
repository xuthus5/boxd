package main

import (
	"fmt"
	"strings"
)

type configItemPath struct {
	Path    string
	Title   string
	Default string
}

func configSectionItemPaths(section string) []configItemPath {
	switch section {
	case sectionInbounds, sectionOutbounds, sectionEndpoints, sectionServices,
		sectionNetworkNamespaces, sectionHTTPClients, sectionCertificateProvider:
		return []configItemPath{{Title: "items", Default: "{\n  \"tag\": \"\",\n  \"type\": \"\"\n}\n"}}
	case sectionRoute:
		return []configItemPath{
			{Path: "rules", Title: "rules", Default: "{\n  \"action\": \"route\"\n}\n"},
			{Path: "rule_set", Title: "rule sets", Default: "{\n  \"type\": \"inline\"\n}\n"},
		}
	case sectionDNS:
		return []configItemPath{
			{Path: "servers", Title: "servers", Default: "{\n  \"type\": \"udp\"\n}\n"},
			{Path: "rules", Title: "rules", Default: "{\n  \"action\": \"route\"\n}\n"},
		}
	default:
		return nil
	}
}

func configItemCardTitle(state *configSectionState, itemPath configItemPath) string {
	if itemPath.Path == "" {
		return state.spec.Title + " items"
	}
	return state.spec.Title + " " + itemPath.Title
}

func configItemPathTitle(state *configSectionState, path string) string {
	for _, itemPath := range configSectionItemPaths(state.spec.Key) {
		if itemPath.Path == path {
			return itemPath.Title
		}
	}
	return "item"
}

func configItemDialogLabel(state *configSectionState, path string) string {
	if path == "" {
		return strings.TrimSuffix(state.spec.Title, "s") + " item"
	}
	return strings.TrimSuffix(configItemPathTitle(state, path), "s")
}

func configItemLabel(item map[string]any) string {
	for _, key := range []string{"tag", "action", "type"} {
		if value := strings.TrimSpace(jsonPathString(item, key)); value != "" {
			return value
		}
	}
	return "(untitled)"
}

func configItemDetail(item map[string]any) string {
	for _, key := range []string{"outbound", "server", "listen", "address", "path", "domain"} {
		if value := strings.TrimSpace(jsonPathString(item, key)); value != "" {
			return key + "=" + value
		}
	}
	return fmt.Sprintf("%d field(s)", len(item))
}

func configItemKey(state *configSectionState, path, action string, index int) string {
	key := "config-item-" + action + "-" + state.spec.Key
	if path != "" {
		key += "-" + strings.ReplaceAll(path, "_", "-")
	}
	if index >= 0 {
		return fmt.Sprintf("%s-%d", key, index)
	}
	return key
}
