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

// supportRoutingMark 指示当前平台是否支持出站 routing_mark（SO_MARK 仅 Linux）。
// 包级变量便于测试注入非 Linux 平台。
var supportRoutingMark = func() bool { return runtime.GOOS == "linux" }()

// SupportRoutingMark 返回当前平台是否支持出站 routing_mark（SO_MARK）。
func SupportRoutingMark() bool { return supportRoutingMark }

func NewDefaultOutboundsInstaller() *DefaultOutboundsInstaller {
	return &DefaultOutboundsInstaller{}
}

func (i *DefaultOutboundsInstaller) Install(cfg map[string]any) (*OutboundDefaultsResult, error) {
	existing, _ := cfg["outbounds"].([]any)

	outboundsByTag := make(map[string]map[string]any, len(existing))
	order := make([]string, 0, len(existing))
	proxyCandidates := make([]string, 0)

	for _, item := range existing {
		ob, ok := item.(map[string]any)
		if !ok || ob == nil {
			continue
		}
		tag, _ := ob["tag"].(string)
		if tag == "" {
			continue
		}
		outboundsByTag[tag] = cloneMap(ob)
		order = append(order, tag)
		if isProxyCandidate(ob) {
			proxyCandidates = append(proxyCandidates, tag)
		}
	}

	ensureBuiltin := func(tag, typ string) {
		if _, ok := outboundsByTag[tag]; ok {
			return
		}
		outboundsByTag[tag] = map[string]any{"tag": tag, "type": typ}
		order = append(order, tag)
	}
	ensureBuiltin("direct", "direct")
	ensureBuiltin("bypass", "direct")
	ensureBuiltin("block", "block")
	delete(outboundsByTag, "dns-out")
	// routing_mark（SO_MARK 策略路由标记）仅 Linux 支持，其他平台会致内核启动失败。
	if supportRoutingMark {
		if direct, ok := outboundsByTag["direct"]; ok {
			direct["routing_mark"] = 128
		}
		if bypass, ok := outboundsByTag["bypass"]; ok {
			bypass["routing_mark"] = 128
		}
	}

	if len(proxyCandidates) == 0 {
		proxyCandidates = []string{"direct"}
	}

	upsertGroup := func(tag, typ string, members []string) {
		group, ok := outboundsByTag[tag]
		if !ok {
			group = map[string]any{"tag": tag}
			order = append(order, tag)
			group["type"] = typ
			group["outbounds"] = members
		} else {
			// 保留用户既有组的 default/outbounds，仅同步类型，避免破坏 selector 默认项导致回滚。
			group["type"] = typ
		}
		outboundsByTag[tag] = group
	}

	upsertGroup("proxy", "selector", proxyCandidates)
	if len(proxyCandidates) > 0 && (len(proxyCandidates) != 1 || proxyCandidates[0] != "direct") {
		upsertGroup("auto", "urltest", proxyCandidates)
	}
	upsertGroup("whitelist", "selector", []string{"bypass", "proxy"})
	upsertGroup("blacklist", "selector", []string{"block", "proxy"})

	result := make([]any, 0, len(order))
	installed := make([]map[string]any, 0, 6)
	seen := make(map[string]struct{}, len(order))
	for _, tag := range order {
		ob, ok := outboundsByTag[tag]
		if !ok {
			continue
		}
		if _, duplicated := seen[tag]; duplicated {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, ob)
		switch tag {
		case "direct", "bypass", "block", "proxy", "auto", "whitelist", "blacklist":
			installed = append(installed, cloneMap(ob))
		}
	}

	return &OutboundDefaultsResult{
		Outbounds: result,
		Final:     "proxy",
		Installed: installed,
	}, nil
}

func isProxyCandidate(ob map[string]any) bool {
	typ, _ := ob["type"].(string)
	switch typ {
	case "", "direct", "block", "dns", "selector", "urltest":
		return false
	default:
		return true
	}
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	maps.Copy(out, in)
	return out
}
