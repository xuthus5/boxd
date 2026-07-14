package core

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/xuthus5/boxui/internal/model"
)

// parseWireGuard 解析 wireguard:// 链接
// 格式：wireguard://<base64-private-key>@<server>:<port>?public_key=<peer-pk>&address=<addr>&mtu=<mtu>#tag
func parseWireGuard(u *url.URL) (*model.ImportResult, error) {
	privateKey := u.User.String()
	if privateKey == "" {
		return nil, fmt.Errorf("missing private key in wireguard link")
	}

	server := u.Hostname()
	portStr := u.Port()
	port, _ := strconv.Atoi(portStr)
	if port == 0 {
		port = 51820
	}

	q := u.Query()
	publicKey := q.Get("public_key")
	address := q.Get("address")
	if address == "" {
		address = "10.0.0.2/32"
	}
	mtuStr := q.Get("mtu")
	mtu, _ := strconv.Atoi(mtuStr)

	peer := map[string]any{
		"address":     server,
		"port":        port,
		"public_key":  publicKey,
		"allowed_ips": []string{"0.0.0.0/0", "::/0"},
	}
	if psk := q.Get("pre_shared_key"); psk != "" {
		peer["pre_shared_key"] = psk
	}

	config := map[string]any{
		"type":        "wireguard",
		"private_key": privateKey,
		"peers":       []any{peer},
		"address":     []string{address},
	}
	if mtu > 0 {
		config["mtu"] = mtu
	}

	tag := nodeName(u, fmt.Sprintf("wg-%s-%d", server, port))
	return &model.ImportResult{
		Tag:    tag,
		Type:   "wireguard",
		Server: server,
		Port:   port,
		Config: config,
	}, nil
}

// parseTUIC 解析 tuic:// 链接
// 格式：tuic://<uuid>:<password>@<server>:<port>?congestion_control=bbr&udp_relay_mode=quic&sni=example.com#tag
func parseTUIC(u *url.URL) (*model.ImportResult, error) {
	uuid := u.User.Username()
	password, _ := u.User.Password()
	if uuid == "" {
		return nil, fmt.Errorf("missing uuid in tuic link")
	}

	server := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 443
	}

	q := u.Query()
	config := map[string]any{
		"type":        "tuic",
		"server":      server,
		"server_port": port,
		"uuid":        uuid,
		"password":    password,
	}
	if cc := q.Get("congestion_control"); cc != "" {
		config["congestion_control"] = cc
	}
	if mode := q.Get("udp_relay_mode"); mode != "" {
		config["udp_relay_mode"] = mode
	}

	sni := q.Get("sni")
	if sni == "" {
		sni = server
	}
	tlsCfg := map[string]any{"enabled": true, "server_name": sni}
	if alpn := q.Get("alpn"); alpn != "" {
		tlsCfg["alpn"] = []string{alpn}
	}
	if q.Get("insecure") == "1" {
		tlsCfg["insecure"] = true
	}
	config["tls"] = tlsCfg

	tag := nodeName(u, fmt.Sprintf("tuic-%s-%d", server, port))
	return &model.ImportResult{
		Tag:    tag,
		Type:   "tuic",
		Server: server,
		Port:   port,
		Config: config,
	}, nil
}

// parseAnyTLS 解析 anytls:// 链接
// 格式：anytls://<password>@<server>:<port>?sni=example.com&insecure=0#tag
func parseAnyTLS(u *url.URL) (*model.ImportResult, error) {
	password := u.User.String()
	if password == "" {
		return nil, fmt.Errorf("missing password in anytls link")
	}

	server := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 443
	}

	q := u.Query()
	config := map[string]any{
		"type":        "anytls",
		"server":      server,
		"server_port": port,
		"password":    password,
	}

	sni := q.Get("sni")
	if sni == "" {
		sni = server
	}
	tlsCfg := map[string]any{"enabled": true, "server_name": sni}
	if q.Get("insecure") == "1" {
		tlsCfg["insecure"] = true
	}
	config["tls"] = tlsCfg

	tag := nodeName(u, fmt.Sprintf("anytls-%s-%d", server, port))
	return &model.ImportResult{
		Tag:    tag,
		Type:   "anytls",
		Server: server,
		Port:   port,
		Config: config,
	}, nil
}

// parseShadowTLS 解析 shadowtls:// 链接
// 格式：shadowtls://<password>@<server>:<port>?version=3&sni=example.com#tag
func parseShadowTLS(u *url.URL) (*model.ImportResult, error) {
	password := u.User.String()
	if password == "" {
		return nil, fmt.Errorf("missing password in shadowtls link")
	}

	server := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 443
	}

	q := u.Query()
	version, _ := strconv.Atoi(q.Get("version"))
	if version == 0 {
		version = 3
	}

	config := map[string]any{
		"type":        "shadowtls",
		"server":      server,
		"server_port": port,
		"version":     version,
		"password":    password,
	}

	sni := q.Get("sni")
	if sni == "" {
		sni = server
	}
	config["tls"] = map[string]any{"enabled": true, "server_name": sni}

	tag := nodeName(u, fmt.Sprintf("shadowtls-%s-%d", server, port))
	return &model.ImportResult{
		Tag:    tag,
		Type:   "shadowtls",
		Server: server,
		Port:   port,
		Config: config,
	}, nil
}
