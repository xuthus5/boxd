package core

import "runtime"

type InboundDefaultsInstaller interface {
	Install(cfg map[string]any) (*InboundDefaultsResult, error)
}

type InboundDefaultsResult struct {
	Inbounds  []any
	Installed []map[string]any
}

type DefaultInboundsInstaller struct{}

func NewDefaultInboundsInstaller() *DefaultInboundsInstaller {
	return &DefaultInboundsInstaller{}
}

// Install 确保常用本地入站模板存在：mixed-in（1080）与 tun-in（auto_route）。
// 已存在同 tag 入站时保留用户配置，仅补缺。
func (i *DefaultInboundsInstaller) Install(cfg map[string]any) (*InboundDefaultsResult, error) {
	existing, _ := cfg["inbounds"].([]any)
	byTag, order, passthrough := indexInbounds(existing)
	ensureInbound(byTag, &order, "mixed-in", mixedInboundTemplate())
	ensureInbound(byTag, &order, "tun-in", tunInboundTemplate())
	return buildInboundDefaultsResult(byTag, order, passthrough), nil
}

func mixedInboundTemplate() map[string]any {
	return map[string]any{
		"type": "mixed", "tag": "mixed-in", "listen": "::", "listen_port": 1080,
	}
}

func tunInboundTemplate() map[string]any {
	// stack: system 仅 Linux 支持；其他平台使用 gvisor（跨平台，无需系统驱动）。
	stack := "system"
	if runtime.GOOS != "linux" {
		stack = "gvisor"
	}
	return map[string]any{
		"type": "tun", "tag": "tun-in", "interface_name": "boxd0",
		"address": []string{"172.19.0.1/30", "fdfe:dcba:9876::1/126"},
		"mtu":     9000, "auto_route": true, "strict_route": true, "stack": stack,
	}
}

func ensureInbound(byTag map[string]map[string]any, order *[]string, tag string, template map[string]any) {
	if _, ok := byTag[tag]; ok {
		return
	}
	byTag[tag] = cloneMap(template)
	*order = append(*order, tag)
}

func indexInbounds(existing []any) (map[string]map[string]any, []string, []any) {
	byTag := make(map[string]map[string]any, len(existing))
	order := make([]string, 0, len(existing)+2)
	passthrough := make([]any, 0)
	for _, item := range existing {
		inbound, ok := item.(map[string]any)
		if !ok || inbound == nil {
			passthrough = append(passthrough, item)
			continue
		}
		tag, _ := inbound["tag"].(string)
		if tag == "" {
			passthrough = append(passthrough, item)
			continue
		}
		byTag[tag] = cloneMap(inbound)
		order = append(order, tag)
	}
	return byTag, order, passthrough
}

func buildInboundDefaultsResult(byTag map[string]map[string]any, order []string, passthrough []any) *InboundDefaultsResult {
	result := make([]any, 0, len(order)+len(passthrough))
	installed := make([]map[string]any, 0, 2)
	seen := make(map[string]struct{}, len(order))
	for _, tag := range order {
		inbound, ok := byTag[tag]
		if !ok {
			continue
		}
		if _, duplicated := seen[tag]; duplicated {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, inbound)
		if tag == "mixed-in" || tag == "tun-in" {
			installed = append(installed, cloneMap(inbound))
		}
	}
	result = append(result, passthrough...)
	return &InboundDefaultsResult{Inbounds: result, Installed: installed}
}
