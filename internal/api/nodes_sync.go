package api

import (
	"encoding/json"
	"os"

	"github.com/xuthus5/boxui/internal/core"
)

func syncOutboundsToConfig(nodeManager *core.NodeManager, subManager *core.SubscriptionManager, configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	existingByTag := make(map[string]map[string]any)
	subNames := make(map[string]bool)
	for _, sub := range subManager.List() {
		subNames[sub.Name] = true
	}
	if existing, ok := cfg["outbounds"].([]any); ok {
		for _, ob := range existing {
			obm, _ := ob.(map[string]any)
			if obm == nil {
				continue
			}
			tag, _ := obm["tag"].(string)
			if tag == "" {
				continue
			}
			existingByTag[tag] = cloneAnyMap(obm)
		}
	}

	outbounds := []any{
		map[string]any{"type": "direct", "tag": "direct"},
	}

	managedNodeTypes := map[string]bool{
		"vless": true, "vmess": true, "trojan": true, "shadowsocks": true,
		"hysteria": true, "hysteria2": true, "tuic": true, "shadowtls": true, "anytls": true,
	}
	if existing, ok := cfg["outbounds"].([]any); ok {
		for _, ob := range existing {
			obm, _ := ob.(map[string]any)
			if obm == nil {
				continue
			}
			t, _ := obm["type"].(string)
			tag, _ := obm["tag"].(string)
			if managedNodeTypes[t] {
				continue
			}
			// 跳过已有的 direct/dns（已预置在 outbounds 初始列表）
			if t == "direct" || t == "dns" {
				continue
			}
			// 跳过旧的 urltest 分组（回收）
			if t == "urltest" && subNames[tag] {
				continue
			}
			// 跳过旧的 proxy selector（重建）
			if t == "selector" && tag == "proxy" {
				continue
			}
			outbounds = append(outbounds, ob)
		}
	}

	for _, n := range nodeManager.List() {
		ob := cloneAnyMap(existingByTag[n.Tag])
		if ob == nil {
			ob = map[string]any{}
		}
		ob["type"] = n.Type
		ob["tag"] = n.Tag
		ob["server"] = n.Server
		ob["server_port"] = n.Port
		if n.Raw != nil {
			rawJSON, err := json.Marshal(n.Raw)
			if err == nil {
				var rawMap map[string]any
				if err := json.Unmarshal(rawJSON, &rawMap); err == nil {
					for k, v := range rawMap {
						ob[k] = v
					}
				}
			}
		}
		if isProxyLikeOutboundType(n.Type) {
			if _, ok := ob["routing_mark"]; !ok {
				ob["routing_mark"] = 128
			}
		}
		outbounds = append(outbounds, ob)
	}

	for _, sub := range subManager.List() {
		for _, ob := range sub.Outbounds {
			entry := cloneAnyMap(existingByTag[ob.Tag])
			if entry == nil {
				entry = map[string]any{}
			}
			entry["type"] = ob.Type
			entry["tag"] = ob.Tag
			entry["server"] = ob.Server
			entry["server_port"] = ob.Port
			if ob.Raw != nil {
				rawJSON, err := json.Marshal(ob.Raw)
				if err == nil {
					var rawMap map[string]any
					if err := json.Unmarshal(rawJSON, &rawMap); err == nil {
						for k, val := range rawMap {
							entry[k] = val
						}
					}
				}
			}
			if isProxyLikeOutboundType(ob.Type) {
				if _, ok := entry["routing_mark"]; !ok {
					entry["routing_mark"] = 128
				}
			}
			outbounds = append(outbounds, entry)
		}
	}

	// 收集所有代理节点 tag（排除 direct/block/dns/selector/urltest）
	var allProxyTags []string
	for _, ob := range outbounds {
		obm, _ := ob.(map[string]any)
		if obm == nil {
			continue
		}
		tag, _ := obm["tag"].(string)
		t, _ := obm["type"].(string)
		if t == "direct" || t == "block" || t == "dns" || t == "selector" || t == "urltest" {
			continue
		}
		if tag != "" {
			allProxyTags = append(allProxyTags, tag)
		}
	}

	// 为每个订阅创建 urltest 分组，并收集分组 tag
	var subGroupTags []string
	for _, sub := range subManager.List() {
		var subTags []string
		for _, ob := range sub.Outbounds {
			if isProxyLikeOutboundType(ob.Type) && ob.Tag != "" {
				subTags = append(subTags, ob.Tag)
			}
		}
		if len(subTags) == 0 {
			continue
		}
		groupTag := sub.Name
		entry := cloneAnyMap(existingByTag[groupTag])
		if entry == nil {
			entry = map[string]any{}
		}
		entry["type"] = "urltest"
		entry["tag"] = groupTag
		entry["outbounds"] = subTags
		entry["url"] = "https://www.gstatic.com/generate_204"
		entry["interval"] = "3m"
		entry["tolerance"] = 50
		outbounds = append(outbounds, entry)
		subGroupTags = append(subGroupTags, groupTag)
	}

	// 构造 proxy selector 的 outbounds：分组 + 所有代理节点
	var proxyOutbounds []string
	proxyOutbounds = append(proxyOutbounds, subGroupTags...)
	proxyOutbounds = append(proxyOutbounds, allProxyTags...)

	// 更新或创建 proxy selector
	if len(proxyOutbounds) > 0 {
		found := false
		for i, ob := range outbounds {
			obm, _ := ob.(map[string]any)
			if obm != nil && obm["type"] == "selector" && obm["tag"] == "proxy" {
				obm["outbounds"] = proxyOutbounds
				if len(subGroupTags) > 0 {
					obm["default"] = subGroupTags[0]
				} else {
					obm["default"] = allProxyTags[0]
				}
				outbounds[i] = obm
				found = true
				break
			}
		}
		if !found {
			proxyDef := map[string]any{
				"type":      "selector",
				"tag":       "proxy",
				"outbounds": proxyOutbounds,
			}
			if len(subGroupTags) > 0 {
				proxyDef["default"] = subGroupTags[0]
			} else {
				proxyDef["default"] = allProxyTags[0]
			}
			outbounds = append(outbounds, proxyDef)
		}
	}

	cfg["outbounds"] = outbounds

	if cfg["route"] == nil {
		cfg["route"] = map[string]any{}
	}
	route, _ := cfg["route"].(map[string]any)
	if route == nil {
		route = map[string]any{}
		cfg["route"] = route
	}
	if _, ok := route["final"]; !ok {
		route["final"] = "proxy"
	}

	written, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, written, 0600)
}

func isProxyLikeOutboundType(typ string) bool {
	switch typ {
	case "vless", "vmess", "trojan", "shadowsocks", "hysteria", "hysteria2", "tuic", "shadowtls", "anytls", "ssh", "tor":
		return true
	default:
		return false
	}
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
