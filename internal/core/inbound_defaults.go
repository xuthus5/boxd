package core

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

// Install 仅补齐 mixed 入站；TUN 必须由用户显式选择。
func (i *DefaultInboundsInstaller) Install(cfg map[string]any) (*InboundDefaultsResult, error) {
	return BuildInboundProfile(cfg, InboundProfileOptions{Mode: "preserve"}, DetectNetworkCapabilities())
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
	for _, tag := range order {
		inbound, ok := byTag[tag]
		if !ok {
			continue
		}
		result = append(result, inbound)
		if tag == "mixed-in" || tag == "tun-in" {
			installed = append(installed, cloneMap(inbound))
		}
	}
	result = append(result, passthrough...)
	return &InboundDefaultsResult{Inbounds: result, Installed: installed}
}
