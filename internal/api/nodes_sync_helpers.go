package api

import "github.com/xuthus5/boxd/internal/core"

func isProxyLikeOutboundType(typ string) bool {
	return core.IsProxyLikeOutboundType(typ)
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
