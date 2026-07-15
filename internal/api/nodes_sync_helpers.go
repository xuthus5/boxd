package api

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
	for key, value := range in {
		out[key] = value
	}
	return out
}
