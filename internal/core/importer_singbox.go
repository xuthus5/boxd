package core

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

type vmessLinkConfig struct {
	Add           string          `json:"add"`
	Port          json.RawMessage `json:"port"`
	ID            string          `json:"id"`
	Aid           json.RawMessage `json:"aid"`
	Scy           string          `json:"scy"`
	Net           string          `json:"net"`
	TLS           string          `json:"tls"`
	Host          string          `json:"host"`
	Path          string          `json:"path"`
	PS            string          `json:"ps"`
	SNI           string          `json:"sni"`
	ServerName    string          `json:"servername"`
	AllowInsecure json.RawMessage `json:"allowInsecure"`
	ALPN          string          `json:"alpn"`
	Fingerprint   string          `json:"fp"`
}

func parseVmess(raw string) (*model.ImportResult, error) {
	b64 := strings.TrimPrefix(raw, "vmess://")
	data, err := decodeLinkBase64(b64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode vmess link")
	}

	var vmess vmessLinkConfig
	if err := json.Unmarshal(data, &vmess); err != nil {
		return nil, fmt.Errorf("failed to parse vmess JSON")
	}
	port, alterID, insecure, err := parseVmessScalarFields(vmess)
	if err != nil {
		return nil, err
	}
	if vmess.Add == "" || vmess.ID == "" {
		return nil, fmt.Errorf("invalid vmess link: missing server or uuid")
	}
	transport, err := buildV2RayTransport(vmess.Net, vmess.Path, vmess.Host, "")
	if err != nil {
		return nil, fmt.Errorf("invalid vmess transport: %w", err)
	}

	tag := vmess.PS
	if tag == "" {
		tag = fmt.Sprintf("vmess-%s-%d", vmess.Add, port)
	}
	config := map[string]any{
		"type":        "vmess",
		"server":      vmess.Add,
		"server_port": port,
		"uuid":        vmess.ID,
		"alter_id":    alterID,
	}
	if vmess.Scy != "" {
		config["security"] = vmess.Scy
	}
	if transport != nil {
		config["transport"] = transport
	}
	serverName := firstNonEmpty(vmess.SNI, vmess.ServerName, vmess.Host)
	if tls := buildTLSConfig(strings.EqualFold(vmess.TLS, "tls"), serverName, insecure, vmess.ALPN); tls != nil {
		if vmess.Fingerprint != "" {
			tls["utls"] = map[string]any{"enabled": true, "fingerprint": vmess.Fingerprint}
		}
		config["tls"] = tls
	}

	return &model.ImportResult{Tag: tag, Type: "vmess", Server: vmess.Add, Port: port, Config: config}, nil
}

func parseVmessScalarFields(vmess vmessLinkConfig) (int, int, bool, error) {
	port, err := parseLinkJSONInt(vmess.Port, "vmess server port")
	if err != nil || port < 1 || port > 65535 {
		return 0, 0, false, fmt.Errorf("invalid vmess server port")
	}
	alterID, err := parseLinkJSONInt(vmess.Aid, "vmess alter id")
	if err != nil || alterID < 0 {
		return 0, 0, false, fmt.Errorf("invalid vmess alter id")
	}
	insecure, err := parseLinkJSONBool(vmess.AllowInsecure, "vmess allowInsecure")
	if err != nil {
		return 0, 0, false, err
	}
	return port, alterID, insecure, nil
}

func buildV2RayTransport(network, path, host, serviceName string) (map[string]any, error) {
	switch strings.ToLower(strings.TrimSpace(network)) {
	case "", "tcp", "none":
		return nil, nil
	case "ws", "websocket":
		transport := map[string]any{"type": "ws"}
		if path != "" {
			transport["path"] = path
		}
		if host != "" {
			transport["headers"] = map[string]any{"Host": host}
		}
		return transport, nil
	case "http", "h2":
		transport := map[string]any{"type": "http"}
		if path != "" {
			transport["path"] = path
		}
		if host != "" {
			transport["host"] = []string{host}
		}
		return transport, nil
	case "grpc":
		transport := map[string]any{"type": "grpc"}
		if serviceName == "" {
			serviceName = path
		}
		if serviceName != "" {
			transport["service_name"] = serviceName
		}
		return transport, nil
	case "quic":
		return map[string]any{"type": "quic"}, nil
	case "httpupgrade":
		transport := map[string]any{"type": "httpupgrade"}
		if path != "" {
			transport["path"] = path
		}
		if host != "" {
			transport["host"] = host
		}
		return transport, nil
	default:
		return nil, fmt.Errorf("unsupported transport %q", network)
	}
}

func buildLinkTransport(query url.Values) (map[string]any, error) {
	network := firstNonEmpty(query.Get("type"), query.Get("network"))
	serviceName := firstNonEmpty(query.Get("serviceName"), query.Get("service_name"))
	return buildV2RayTransport(network, query.Get("path"), query.Get("host"), serviceName)
}

func buildLinkTLSConfig(query url.Values, defaultEnabled bool) map[string]any {
	security := strings.ToLower(strings.TrimSpace(firstNonEmpty(query.Get("security"), query.Get("tls"))))
	enabled := defaultEnabled || security == "tls" || security == "reality" || security == "xtls" || security == "1" || security == "true" || hasLinkTLSOptions(query)
	if security == "none" {
		enabled = false
	}
	if !enabled {
		return nil
	}

	serverName := firstNonEmpty(query.Get("sni"), query.Get("servername"), query.Get("server_name"))
	tlsConfig := buildTLSConfig(true, serverName, linkValueIsTrue(query, "allowInsecure", "allow_insecure", "insecure", "skip-cert-verify", "skip_cert_verify"), query.Get("alpn"))
	if fingerprint := query.Get("fp"); fingerprint != "" {
		tlsConfig["utls"] = map[string]any{"enabled": true, "fingerprint": fingerprint}
	}
	if publicKey := query.Get("pbk"); publicKey != "" || security == "reality" {
		reality := map[string]any{"enabled": true}
		if publicKey != "" {
			reality["public_key"] = publicKey
		}
		if shortID := query.Get("sid"); shortID != "" {
			reality["short_id"] = shortID
		}
		tlsConfig["reality"] = reality
	}
	return tlsConfig
}

func hasLinkTLSOptions(query url.Values) bool {
	for _, key := range []string{"tls", "sni", "servername", "server_name", "fp", "pbk", "sid", "alpn", "allowInsecure", "allow_insecure", "insecure", "skip-cert-verify", "skip_cert_verify"} {
		if query.Get(key) != "" {
			return true
		}
	}
	return false
}

func linkValueIsTrue(query url.Values, keys ...string) bool {
	for _, key := range keys {
		switch strings.ToLower(strings.TrimSpace(query.Get(key))) {
		case "1", "true", "yes", "on":
			return true
		}
	}
	return false
}

func buildTLSConfig(enabled bool, serverName string, insecure bool, alpn string) map[string]any {
	if !enabled {
		return nil
	}
	config := map[string]any{"enabled": true}
	if serverName != "" {
		config["server_name"] = serverName
	}
	if insecure {
		config["insecure"] = true
	}
	if protocols := splitLinkList(alpn); len(protocols) > 0 {
		config["alpn"] = protocols
	}
	return config
}

func splitLinkList(value string) []string {
	values := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' })
	return values
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
