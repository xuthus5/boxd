package core

import (
	"fmt"
	"strconv"
	"strings"
)

func clashTLS(proxy map[string]any) map[string]any {
	enabled := asBool(proxy["tls"]) || firstString(proxy, "tls") == "tls" || firstString(proxy, "sni", "servername") != ""
	if reality, ok := proxy["reality-opts"].(map[string]any); ok && reality != nil {
		enabled = true
	}
	if !enabled {
		return nil
	}
	tls := map[string]any{"enabled": true}
	if sni := firstString(proxy, "servername", "sni"); sni != "" {
		tls["server_name"] = sni
	}
	if asBool(proxy["skip-cert-verify"]) || asBool(proxy["skip_cert_verify"]) {
		tls["insecure"] = true
	}
	if alpn := asStringSlice(proxy["alpn"]); len(alpn) > 0 {
		tls["alpn"] = alpn
	}
	if reality, ok := proxy["reality-opts"].(map[string]any); ok && reality != nil {
		tls["reality"] = map[string]any{
			"enabled":    true,
			"public_key": firstString(reality, "public-key", "public_key"),
			"short_id":   firstString(reality, "short-id", "short_id"),
		}
	}
	return tls
}

func clashTransport(proxy map[string]any) map[string]any {
	network := strings.ToLower(firstString(proxy, "network", "net"))
	if network == "" || network == "tcp" {
		return nil
	}
	transport := map[string]any{"type": network}
	switch network {
	case "ws", "websocket":
		transport["type"] = "ws"
		if path := firstString(proxy, "ws-path", "path"); path != "" {
			transport["path"] = path
		}
		if host := firstString(proxy, "ws-host", "host"); host != "" {
			transport["headers"] = map[string]any{"Host": host}
		}
		if opts, ok := proxy["ws-opts"].(map[string]any); ok {
			if path := firstString(opts, "path"); path != "" {
				transport["path"] = path
			}
			if headers, ok := opts["headers"].(map[string]any); ok {
				transport["headers"] = headers
			}
		}
	case "grpc":
		if name := firstString(proxy, "grpc-service-name", "serviceName"); name != "" {
			transport["service_name"] = name
		}
		if opts, ok := proxy["grpc-opts"].(map[string]any); ok {
			if name := firstString(opts, "grpc-service-name", "serviceName"); name != "" {
				transport["service_name"] = name
			}
		}
	case "h2", "http":
		if path := firstString(proxy, "path"); path != "" {
			transport["path"] = path
		}
		if host := firstString(proxy, "host"); host != "" {
			transport["host"] = []string{host}
		}
	}
	return transport
}

func asStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s := asString(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	default:
		return nil
	}
}

func asString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case fmt.Stringer:
		return v.String()
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	default:
		return ""
	}
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if s := asString(values[key]); s != "" {
			return s
		}
	}
	return ""
}

func asInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	default:
		return 0
	}
}

func asBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true") || strings.EqualFold(v, "yes") || v == "1"
	case int:
		return v != 0
	case float64:
		return v != 0
	default:
		return false
	}
}
