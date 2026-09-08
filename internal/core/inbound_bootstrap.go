package core

func initialInbounds(enableTUN bool) []any {
	inbounds := []any{mixedInboundTemplate()}
	if enableTUN {
		inbounds = append(inbounds, tunInboundTemplate())
	}
	return inbounds
}

// ConfigureDefaultTUNRouting 为自动路由 TUN 补齐防回环的出口检测。
// 既有明确的接口或路由标记设置仍由用户控制。
func ConfigureDefaultTUNRouting(cfg map[string]any) {
	if !hasAutomaticTUN(cfg) {
		return
	}
	route, _ := cfg["route"].(map[string]any)
	if route == nil {
		route = map[string]any{}
	}
	for _, key := range []string{"auto_detect_interface", "default_interface", "default_mark"} {
		if _, exists := route[key]; exists {
			return
		}
	}
	route["auto_detect_interface"] = true
	cfg["route"] = route
}

func hasAutomaticTUN(cfg map[string]any) bool {
	inbounds, _ := cfg["inbounds"].([]any)
	for _, item := range inbounds {
		inbound, _ := item.(map[string]any)
		if inbound["type"] == "tun" && inbound["auto_route"] == true {
			return true
		}
	}
	return false
}
