package core

import (
	"fmt"
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
	config, outType, err := buildClashConfig(typ, proxy, server, port)
	if err != nil {
		return nil, err
	}
	config["tag"] = name
	return &model.Outbound{Tag: name, Type: outType, Server: server, Port: port, Raw: config}, nil
}

func buildClashConfig(typ string, proxy map[string]any, server string, port int) (map[string]any, string, error) {
	switch typ {
	case "ss", "shadowsocks":
		return clashShadowsocks(proxy, server, port), "shadowsocks", nil
	case "trojan":
		return clashTrojan(proxy, server, port), "trojan", nil
	case "vmess":
		return clashVMess(proxy, server, port), "vmess", nil
	case "vless":
		return clashVLESS(proxy, server, port), "vless", nil
	case "hysteria2", "hy2":
		return clashHysteria2(proxy, server, port), "hysteria2", nil
	case "tuic":
		return clashTUIC(proxy, server, port), "tuic", nil
	case "http":
		return clashHTTP(proxy, server, port), "http", nil
	case "socks", "socks5":
		return clashSocks(proxy, server, port), "socks", nil
	default:
		return nil, "", fmt.Errorf("unsupported clash type %s", typ)
	}
}
