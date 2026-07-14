package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/xuthus5/boxui/internal/model"
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

func ParseProxyLink(link string) (*model.ImportResult, error) {
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
		return parseSSR(u)
	case "vless":
		return parseVless(u)
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

func parseVmess(raw string) (*model.ImportResult, error) {
	b64 := strings.TrimPrefix(raw, "vmess://")
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		data, err = base64.RawURLEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode vmess link")
		}
	}

	var vmess struct {
		Add  string `json:"add"`
		Port int    `json:"port"`
		ID   string `json:"id"`
		Aid  int    `json:"aid"`
		Net  string `json:"net"`
		TLS  string `json:"tls"`
		Host string `json:"host"`
		Path string `json:"path"`
		PS   string `json:"ps"`
	}

	if err := json.Unmarshal(data, &vmess); err != nil {
		return nil, fmt.Errorf("failed to parse vmess JSON")
	}

	tag := vmess.PS
	if tag == "" {
		tag = fmt.Sprintf("vmess-%s-%d", vmess.Add, vmess.Port)
	}

	return &model.ImportResult{
		Tag:    tag,
		Type:   "vmess",
		Server: vmess.Add,
		Port:   vmess.Port,
		Config: map[string]any{
			"type":        "vmess",
			"server":      vmess.Add,
			"server_port": vmess.Port,
			"uuid":        vmess.ID,
			"alter_id":    vmess.Aid,
			"network":     vmess.Net,
			"tls":         vmess.TLS == "tls",
			"ws-opts": map[string]any{
				"path": vmess.Path,
				"headers": map[string]string{
					"Host": vmess.Host,
				},
			},
		},
	}, nil
}

func parseSS(u *url.URL) (*model.ImportResult, error) {
	userInfo := u.User
	if userInfo == nil {
		return nil, fmt.Errorf("missing credentials in ss link")
	}

	server := u.Hostname()
	portStr := u.Port()
	port, _ := strconv.Atoi(portStr)

	parts := strings.SplitN(server, ":", 2)
	if len(parts) == 2 {
		server = parts[0]
		port, _ = strconv.Atoi(parts[1])
	}

	pass, _ := userInfo.Password()
	tag := nodeName(u, fmt.Sprintf("ss-%s-%d", server, port))

	return &model.ImportResult{
		Tag:    tag,
		Type:   "shadowsocks",
		Server: server,
		Port:   port,
		Config: map[string]any{
			"type":        "shadowsocks",
			"server":      server,
			"server_port": port,
			"method":      userInfo.Username(),
			"password":    pass,
		},
	}, nil
}

func parseTrojan(u *url.URL) (*model.ImportResult, error) {
	password := u.User.String()
	server := u.Hostname()
	portStr := u.Port()
	port, _ := strconv.Atoi(portStr)

	tag := nodeName(u, fmt.Sprintf("trojan-%s-%d", server, port))
	return &model.ImportResult{
		Tag:    tag,
		Type:   "trojan",
		Server: server,
		Port:   port,
		Config: map[string]any{
			"type":        "trojan",
			"server":      server,
			"server_port": port,
			"password":    password,
		},
	}, nil
}

func parseSSR(u *url.URL) (*model.ImportResult, error) {
	server := u.Hostname()
	portStr := u.Port()
	port, _ := strconv.Atoi(portStr)

	tag := nodeName(u, fmt.Sprintf("ssr-%s-%d", server, port))
	return &model.ImportResult{
		Tag:    tag,
		Type:   "shadowsocksr",
		Server: server,
		Port:   port,
		Config: map[string]any{
			"type":        "shadowsocksr",
			"server":      server,
			"server_port": port,
		},
	}, nil
}

func parseVless(u *url.URL) (*model.ImportResult, error) {
	uuid := u.User.String()
	server := u.Hostname()
	portStr := u.Port()
	port, _ := strconv.Atoi(portStr)
	q := u.Query()

	flow := q.Get("flow")
	security := q.Get("security")
	sni := q.Get("sni")
	fp := q.Get("fp")
	pbk := q.Get("pbk")
	sid := q.Get("sid")
	network := q.Get("type")

	config := map[string]any{
		"type":        "vless",
		"server":      server,
		"server_port": port,
		"uuid":        uuid,
	}

	if flow != "" {
		config["flow"] = flow
	}

	if security == "reality" || sni != "" || pbk != "" {
		tlsCfg := map[string]any{
			"enabled":     true,
			"server_name": sni,
		}

		if fp != "" {
			tlsCfg["utls"] = map[string]any{
				"enabled":     true,
				"fingerprint": fp,
			}
		}

		if pbk != "" {
			realityCfg := map[string]any{"enabled": true, "public_key": pbk}
			if sid != "" {
				realityCfg["short_id"] = sid
			}
			tlsCfg["reality"] = realityCfg
		}

		config["tls"] = tlsCfg
	}

	if network != "" {
		config["network"] = network
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
	password := u.User.String()
	server := u.Hostname()
	portStr := u.Port()
	port, _ := strconv.Atoi(portStr)

	tag := nodeName(u, fmt.Sprintf("hysteria2-%s-%d", server, port))
	return &model.ImportResult{
		Tag:    tag,
		Type:   "hysteria2",
		Server: server,
		Port:   port,
		Config: map[string]any{
			"type":        "hysteria2",
			"server":      server,
			"server_port": port,
			"password":    password,
		},
	}, nil
}
