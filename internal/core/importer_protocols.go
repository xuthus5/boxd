package core

import (
	"fmt"
	"net/url"

	"github.com/xuthus5/boxd/internal/model"
)

// parseWireGuard 解析 wireguard:// 链接
// 格式：wireguard://<base64-private-key>@<server>:<port>?public_key=<peer-pk>&address=<addr>&mtu=<mtu>#tag
func parseWireGuard(u *url.URL) (*model.ImportResult, error) {
	privateKey, err := userInfoValue(u)
	if err != nil {
		return nil, err
	}
	if privateKey == "" {
		return nil, fmt.Errorf("missing private key in wireguard link")
	}

	server, port, err := parseModernLinkServer(u, "wireguard", 51820)
	if err != nil {
		return nil, err
	}
	config, err := buildWireGuardLinkConfig(u, server, port, privateKey)
	if err != nil {
		return nil, err
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

func buildWireGuardLinkConfig(u *url.URL, server string, port int, privateKey string) (map[string]any, error) {

	q := u.Query()
	publicKey := q.Get("public_key")
	address := q.Get("address")
	if address == "" {
		address = "10.0.0.2/32"
	}
	mtuStr := q.Get("mtu")
	mtu, err := parseModernNonNegativeInt(mtuStr, "wireguard mtu")
	if mtuStr != "" && err != nil {
		return nil, err
	}

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
	return config, nil
}

// parseTUIC 解析 tuic:// 链接
// 格式：tuic://<uuid>:<password>@<server>:<port>?congestion_control=bbr&udp_relay_mode=quic&sni=example.com#tag
func parseTUIC(u *url.URL) (*model.ImportResult, error) {
	uuid, password, err := parseTUICCredentials(u)
	if err != nil {
		return nil, err
	}
	query := u.Query()
	uuid = firstNonEmpty(uuid, query.Get("uuid"))
	password = firstNonEmpty(password, query.Get("password"), query.Get("auth"))
	if uuid == "" {
		return nil, fmt.Errorf("missing uuid in tuic link")
	}

	server, port, err := parseModernLinkServer(u, "tuic", 443)
	if err != nil {
		return nil, err
	}
	config := map[string]any{
		"type":        "tuic",
		"server":      server,
		"server_port": port,
		"uuid":        uuid,
		"password":    password,
	}
	options, err := buildTUICLinkOptions(query)
	if err != nil {
		return nil, fmt.Errorf("invalid tuic options: %w", err)
	}
	for key, value := range options {
		config[key] = value
	}
	tlsConfig, err := buildModernLinkTLS(query, server)
	if err != nil {
		return nil, fmt.Errorf("invalid tuic TLS: %w", err)
	}
	config["tls"] = tlsConfig

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
	password, err := userInfoValue(u)
	if err != nil {
		return nil, err
	}
	query := u.Query()
	password = firstNonEmpty(password, query.Get("password"), query.Get("auth"))
	if password == "" {
		return nil, fmt.Errorf("missing password in anytls link")
	}
	server, port, err := parseModernLinkServer(u, "anytls", 443)
	if err != nil {
		return nil, err
	}
	config := map[string]any{
		"type":        "anytls",
		"server":      server,
		"server_port": port,
		"password":    password,
	}

	options, err := buildAnyTLSLinkOptions(query)
	if err != nil {
		return nil, fmt.Errorf("invalid anytls options: %w", err)
	}
	for key, value := range options {
		config[key] = value
	}
	tlsConfig, err := buildModernLinkTLS(query, server)
	if err != nil {
		return nil, fmt.Errorf("invalid anytls TLS: %w", err)
	}
	config["tls"] = tlsConfig

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
	password, err := userInfoValue(u)
	if err != nil {
		return nil, err
	}
	query := u.Query()
	password = firstNonEmpty(password, query.Get("password"), query.Get("auth"))
	if password == "" {
		return nil, fmt.Errorf("missing password in shadowtls link")
	}
	server, port, err := parseModernLinkServer(u, "shadowtls", 443)
	if err != nil {
		return nil, err
	}
	version, err := parseShadowTLSVersion(query)
	if err != nil {
		return nil, fmt.Errorf("invalid shadowtls options: %w", err)
	}

	config := map[string]any{
		"type":        "shadowtls",
		"server":      server,
		"server_port": port,
		"version":     version,
		"password":    password,
	}

	tlsConfig, err := buildModernLinkTLS(query, server)
	if err != nil {
		return nil, fmt.Errorf("invalid shadowtls TLS: %w", err)
	}
	config["tls"] = tlsConfig

	tag := nodeName(u, fmt.Sprintf("shadowtls-%s-%d", server, port))
	return &model.ImportResult{
		Tag:    tag,
		Type:   "shadowtls",
		Server: server,
		Port:   port,
		Config: config,
	}, nil
}
