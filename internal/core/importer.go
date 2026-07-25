package core

import (
	"fmt"
	"net/url"

	"github.com/xuthus5/boxd/internal/model"
)

func nodeName(u *url.URL, fallback string) string {
	if u.Fragment != "" {
		if decoded, err := url.QueryUnescape(u.Fragment); err == nil && decoded != "" {
			return decoded
		}
		return u.Fragment
	}
	return fallback
}

func decodeUserInfo(value string) (string, error) {
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return "", fmt.Errorf("invalid user info: %w", err)
	}
	return decoded, nil
}

func userInfoValue(u *url.URL) (string, error) {
	if u == nil || u.User == nil {
		return "", nil
	}
	return decodeUserInfo(u.User.String())
}

func ParseProxyLink(link string) (*model.ImportResult, error) {
	if isHysteria2Link(link) {
		return parseHysteria2RawLink(link)
	}
	u, err := url.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("invalid link: %w", err)
	}

	switch u.Scheme {
	case "vmess":
		return parseVmess(link)
	case "ss":
		return parseSS(u)
	case "trojan":
		return parseTrojan(u)
	case "ssr":
		return parseSSRLink(link)
	case "vless":
		return parseVless(u)
	case "hysteria":
		return parseHysteria(u)
	case "hysteria2", "hy2":
		return parseHysteria2(u)
	case "wireguard", "wg":
		return parseWireGuard(u)
	case "tuic":
		return parseTUIC(u)
	case "anytls":
		return parseAnyTLS(u)
	case "shadowtls":
		return parseShadowTLS(u)
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
}

func parseSS(u *url.URL) (*model.ImportResult, error) {
	linkData, err := parseShadowsocksLink(u)
	if err != nil {
		return nil, err
	}
	plugin, pluginOptions := shadowsocksPluginConfig(u.Query())
	config := map[string]any{
		"type":        "shadowsocks",
		"server":      linkData.server,
		"server_port": linkData.port,
		"method":      linkData.method,
		"password":    linkData.password,
	}
	if plugin != "" {
		config["plugin"] = plugin
	}
	if pluginOptions != "" {
		config["plugin_opts"] = pluginOptions
	}
	tag := nodeName(u, fmt.Sprintf("ss-%s-%d", linkData.server, linkData.port))

	return &model.ImportResult{
		Tag:    tag,
		Type:   "shadowsocks",
		Server: linkData.server,
		Port:   linkData.port,
		Config: config,
	}, nil
}

func parseTrojan(u *url.URL) (*model.ImportResult, error) {
	password, err := userInfoValue(u)
	if err != nil {
		return nil, err
	}
	if password == "" {
		return nil, fmt.Errorf("missing password in trojan link")
	}
	server, port, err := parseModernLinkServer(u, "trojan", 443)
	if err != nil {
		return nil, err
	}
	query := u.Query()
	transport, err := buildLinkTransport(query)
	if err != nil {
		return nil, fmt.Errorf("invalid trojan transport: %w", err)
	}

	tag := nodeName(u, fmt.Sprintf("trojan-%s-%d", server, port))
	config := map[string]any{
		"type":        "trojan",
		"server":      server,
		"server_port": port,
		"password":    password,
	}
	if transport != nil {
		config["transport"] = transport
	}
	if tls := buildLinkTLSConfig(query, true); tls != nil {
		config["tls"] = tls
	}
	return &model.ImportResult{
		Tag:    tag,
		Type:   "trojan",
		Server: server,
		Port:   port,
		Config: config,
	}, nil
}

func parseVless(u *url.URL) (*model.ImportResult, error) {
	uuid, err := userInfoValue(u)
	if err != nil {
		return nil, err
	}
	if uuid == "" {
		return nil, fmt.Errorf("missing uuid in vless link")
	}
	server, port, err := parseModernLinkServer(u, "vless", 443)
	if err != nil {
		return nil, err
	}
	q := u.Query()

	flow := q.Get("flow")
	transport, err := buildLinkTransport(q)
	if err != nil {
		return nil, fmt.Errorf("invalid vless transport: %w", err)
	}

	config := map[string]any{
		"type":        "vless",
		"server":      server,
		"server_port": port,
		"uuid":        uuid,
	}

	if flow != "" {
		config["flow"] = flow
	}

	if transport != nil {
		config["transport"] = transport
	}
	if tls := buildLinkTLSConfig(q, false); tls != nil {
		config["tls"] = tls
	}

	tag := nodeName(u, fmt.Sprintf("vless-%s-%d", server, port))
	return &model.ImportResult{
		Tag:    tag,
		Type:   "vless",
		Server: server,
		Port:   port,
		Config: config,
	}, nil
}

func parseHysteria2(u *url.URL) (*model.ImportResult, error) {
	password, err := hysteria2Password(u)
	if err != nil {
		return nil, fmt.Errorf("invalid hysteria2 credentials: %w", err)
	}
	server := u.Hostname()
	if server == "" {
		return nil, fmt.Errorf("invalid hysteria2 link: missing server")
	}
	port, err := parseHysteria2Port(u)
	if err != nil {
		return nil, err
	}
	options, err := buildHysteria2Options(u.Query(), server)
	if err != nil {
		return nil, fmt.Errorf("invalid hysteria2 options: %w", err)
	}
	config := map[string]any{
		"type":        "hysteria2",
		"server":      server,
		"server_port": port,
		"password":    password,
	}
	for key, value := range options {
		config[key] = value
	}

	tag := nodeName(u, fmt.Sprintf("hysteria2-%s-%d", server, port))
	return &model.ImportResult{
		Tag:    tag,
		Type:   "hysteria2",
		Server: server,
		Port:   port,
		Config: config,
	}, nil
}
