package core

import (
	"maps"
	"runtime"
)

type OutboundDefaultsInstaller interface {
	Install(cfg map[string]any) (*OutboundDefaultsResult, error)
}

type OutboundDefaultsResult struct {
	Outbounds []any
	Final     string
	Installed []map[string]any
}

type DefaultOutboundsInstaller struct{}

const defaultOutboundCount = 3

// supportRoutingMark 指示当前平台是否支持出站 routing_mark（SO_MARK 仅 Linux）。
// 包级变量便于测试注入非 Linux 平台。
var supportRoutingMark = func() bool { return runtime.GOOS == "linux" }()

// SupportRoutingMark 返回当前平台是否支持出站 routing_mark（SO_MARK）。
func SupportRoutingMark() bool { return supportRoutingMark }

func NewDefaultOutboundsInstaller() *DefaultOutboundsInstaller {
	return &DefaultOutboundsInstaller{}
}

// Install 仅补齐缺失出站；已有类型、选项和手动分组保持不变。
func (i *DefaultOutboundsInstaller) Install(cfg map[string]any) (*OutboundDefaultsResult, error) {
	existing, _ := cfg["outbounds"].([]any)
	outbounds := make([]any, 0, len(existing)+defaultOutboundCount)
	byTag := existingOutbounds(cfg)
	for _, item := range existing {
		entry, ok := item.(map[string]any)
		if !ok || entry == nil {
			outbounds = append(outbounds, item)
			continue
		}
		outbounds = append(outbounds, cloneMap(entry))
	}
	installed := make([]map[string]any, 0, defaultOutboundCount)
	for _, tag := range []string{"direct", "block"} {
		if byTag[tag] != nil {
			continue
		}
		entry := map[string]any{"type": tag, "tag": tag}
		outbounds = append(outbounds, entry)
		installed = append(installed, cloneMap(entry))
	}
	if byTag["proxy"] == nil {
		sync := newOutboundGroupSync(outbounds)
		sync.includeEgresses(byTag)
		members := defaultProxyMembers(configuredEgresses(cfg))
		if len(members) == 0 {
			members = []string{sync.blockingTag()}
		}
		entry := map[string]any{"type": "selector", "tag": "proxy", "outbounds": members}
		sync.replace(entry)
		for _, item := range sync.outbounds[len(outbounds):] {
			added, _ := item.(map[string]any)
			installed = append(installed, cloneMap(added))
		}
		outbounds = sync.outbounds
	}
	return &OutboundDefaultsResult{Outbounds: outbounds, Final: "proxy", Installed: installed}, nil
}

func defaultProxyMembers(outbounds []any) []string {
	members := make([]string, 0, len(outbounds))
	seen := make(map[string]bool, len(outbounds))
	for _, item := range outbounds {
		entry, _ := item.(map[string]any)
		tag, _ := entry["tag"].(string)
		if tag != "" && !seen[tag] && isProxyCandidate(entry) {
			members = append(members, tag)
			seen[tag] = true
		}
	}
	return members
}

func isProxyCandidate(ob map[string]any) bool {
	typ, _ := ob["type"].(string)
	switch typ {
	case "", "direct", "bridge", "block", "dns", "selector", "urltest", "openvpn-server":
		return false
	default:
		return true
	}
}

// configuredEgresses 将 VPN 端点与普通出站一起作为可引用的出口。
func configuredEgresses(cfg map[string]any) []any {
	outbounds, _ := cfg["outbounds"].([]any)
	endpoints, _ := cfg["endpoints"].([]any)
	result := make([]any, 0, len(outbounds)+len(endpoints))
	result = append(result, outbounds...)
	return append(result, endpoints...)
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	maps.Copy(out, in)
	return out
}
