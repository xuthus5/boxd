package core

import (
	"fmt"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v3"

	"github.com/xuthus5/boxd/internal/model"
)

// parseClashSubscription 解析常见 Clash / Meta proxies 列表。
// 仅转换 boxd 已支持的协议子集；无法识别的节点会被跳过。
func parseClashSubscription(body []byte) []model.Outbound {
	var doc struct {
		Proxies []map[string]any `yaml:"proxies"`
	}
	if err := yaml.Unmarshal(body, &doc); err != nil || len(doc.Proxies) == 0 {
		return nil
	}
	outbounds := make([]model.Outbound, 0, len(doc.Proxies))
	for _, proxy := range doc.Proxies {
		result, err := clashProxyToOutbound(proxy)
		if err != nil {
			continue
		}
		outbounds = append(outbounds, *result)
	}
	if len(outbounds) == 0 {
		return nil
	}
	return outbounds
}

func clashProxyToOutbound(proxy map[string]any) (*model.Outbound, error) {
	typ := strings.ToLower(strings.TrimSpace(asString(proxy["type"])))
	name := strings.TrimSpace(asString(proxy["name"]))
	server := strings.TrimSpace(asString(proxy["server"]))
	port := asInt(proxy["port"])
	if typ == "" || server == "" || port <= 0 {
		return nil, fmt.Errorf("incomplete clash proxy")
	}
	if name == "" {
		name = fmt.Sprintf("%s-%s-%d", typ, server, port)
	}

	var config map[string]any
	switch typ {
	case "ss", "shadowsocks":
		config = map[string]any{
			"type":        "shadowsocks",
			"server":      server,
			"server_port": port,
			"method":      firstString(proxy, "cipher", "method"),
			"password":    asString(proxy["password"]),
		}
		typ = "shadowsocks"
	case "trojan":
		config = map[string]any{
			"type":        "trojan",
			"server":      server,
			"server_port": port,
			"password":    asString(proxy["password"]),
		}
		if sni := firstString(proxy, "sni", "servername"); sni != "" {
			config["tls"] = map[string]any{"enabled": true, "server_name": sni}
		}
	case "vmess":
		config = map[string]any{
			"type":        "vmess",
			"server":      server,
			"server_port": port,
			"uuid":        firstString(proxy, "uuid", "id"),
			"alter_id":    asInt(proxy["alterId"]),
			"security":    firstString(proxy, "cipher", "security"),
		}
		if network := firstString(proxy, "network", "net"); network != "" {
			config["transport"] = map[string]any{"type": network}
		}
		if asBool(proxy["tls"]) || firstString(proxy, "tls") == "tls" {
			tls := map[string]any{"enabled": true}
			if sni := firstString(proxy, "servername", "sni"); sni != "" {
				tls["server_name"] = sni
			}
			config["tls"] = tls
		}
	case "vless":
		config = map[string]any{
			"type":        "vless",
			"server":      server,
			"server_port": port,
			"uuid":        firstString(proxy, "uuid", "id"),
		}
		if flow := asString(proxy["flow"]); flow != "" {
			config["flow"] = flow
		}
		if network := firstString(proxy, "network", "net"); network != "" {
			config["transport"] = map[string]any{"type": network}
		}
		if asBool(proxy["tls"]) || firstString(proxy, "tls") != "" || firstString(proxy, "sni", "servername") != "" {
			tls := map[string]any{"enabled": true}
			if sni := firstString(proxy, "servername", "sni"); sni != "" {
				tls["server_name"] = sni
			}
			if reality, ok := proxy["reality-opts"].(map[string]any); ok {
				tls["reality"] = map[string]any{
					"enabled":    true,
					"public_key": firstString(reality, "public-key", "public_key"),
					"short_id":   firstString(reality, "short-id", "short_id"),
				}
			}
			config["tls"] = tls
		}
	case "hysteria2", "hy2":
		config = map[string]any{
			"type":        "hysteria2",
			"server":      server,
			"server_port": port,
			"password":    firstString(proxy, "password", "auth"),
		}
		if sni := firstString(proxy, "sni", "servername"); sni != "" {
			config["tls"] = map[string]any{"enabled": true, "server_name": sni}
		}
		typ = "hysteria2"
	case "tuic":
		config = map[string]any{
			"type":        "tuic",
			"server":      server,
			"server_port": port,
			"uuid":        asString(proxy["uuid"]),
			"password":    asString(proxy["password"]),
		}
		if sni := firstString(proxy, "sni", "servername"); sni != "" {
			config["tls"] = map[string]any{"enabled": true, "server_name": sni}
		}
	default:
		return nil, fmt.Errorf("unsupported clash type %s", typ)
	}
	config["tag"] = name
	return &model.Outbound{
		Tag:    name,
		Type:   typ,
		Server: server,
		Port:   port,
		Raw:    config,
	}, nil
}

func asString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(asString(values[key])); value != "" {
			return value
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
		n, _ := strconv.Atoi(strings.TrimSpace(v))
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
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			return true
		}
	case int:
		return v != 0
	case float64:
		return v != 0
	}
	return false
}
