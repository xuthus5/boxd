package nodelink

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/xuthus5/boxd/control"
)

func decodeURLNode(scheme, link string) (control.Node, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return control.Node{}, linkError("nodes.decode_link", "parse "+scheme+" link: "+err.Error())
	}
	port, err := linkPort(parsed.Host)
	if err != nil {
		return control.Node{}, err
	}
	node := control.Node{
		Name:     linkName(parsed),
		Protocol: scheme,
		Server:   parsed.Hostname(),
		Port:     port,
		Options:  make(map[string]any),
	}
	query := parsed.Query()
	password := urlUserSecret(parsed)
	switch scheme {
	case "vless":
		node.Protocol = "vless"
		node.Options["uuid"] = parsed.User.Username()
		if flow := firstQuery(query, "flow"); flow != "" {
			node.Options["flow"] = flow
		}
	case "trojan":
		node.Protocol = "trojan"
		node.Options["password"] = password
	case "hysteria2", "hy2":
		node.Protocol = "hysteria2"
		node.Options["password"] = password
		if obfs := firstQuery(query, "obfs"); obfs != "" {
			obfsOptions := map[string]any{"type": obfs}
			if obfsPassword := firstQuery(query, "obfs-password", "obfs_password", "obfsPassword"); obfsPassword != "" {
				obfsOptions["password"] = obfsPassword
			}
			node.Options["obfs"] = obfsOptions
		}
	case "socks", "socks5":
		node.Protocol = "socks"
		setBasicAuth(node.Options, parsed)
	case "http", "https":
		node.Protocol = "http"
		setBasicAuth(node.Options, parsed)
		if parsed.Scheme == "https" {
			node.Options["tls"] = map[string]any{"enabled": true}
		}
	}
	if tlsOptions := decodeTLS(query, scheme == "trojan" || scheme == "hysteria2" || scheme == "hy2" || scheme == "https"); tlsOptions != nil {
		node.Options["tls"] = tlsOptions
	}
	transport, err := decodeTransport(query)
	if err != nil {
		return control.Node{}, err
	}
	if transport != nil {
		node.Options["transport"] = transport
	}
	if len(node.Options) == 0 {
		node.Options = nil
	}
	return node, nil
}

func decodeShadowsocks(body string) (control.Node, error) {
	payload, name := splitFragment(body)
	payload, rawQuery := splitQuery(payload)
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return control.Node{}, linkError("nodes.decode_link", "parse shadowsocks query: "+err.Error())
	}
	var method, password, hostPort string
	if at := strings.LastIndex(payload, "@"); at >= 0 {
		userInfo := payload[:at]
		hostPort = payload[at+1:]
		if decoded, decodeErr := decodeBase64String(userInfo); decodeErr == nil && strings.Contains(decoded, ":") {
			method, password = splitCredential(decoded)
		} else {
			unescaped, unescapeErr := url.QueryUnescape(userInfo)
			if unescapeErr != nil {
				return control.Node{}, linkError("nodes.decode_link", "decode shadowsocks user info: "+unescapeErr.Error())
			}
			method, password = splitCredential(unescaped)
		}
	} else {
		decoded, decodeErr := decodeBase64String(payload)
		if decodeErr != nil {
			return control.Node{}, linkError("nodes.decode_link", "decode shadowsocks link: "+decodeErr.Error())
		}
		at := strings.LastIndex(decoded, "@")
		if at < 0 {
			return control.Node{}, linkError("nodes.decode_link", "shadowsocks link has no server")
		}
		method, password = splitCredential(decoded[:at])
		hostPort = decoded[at+1:]
	}
	if method == "" || password == "" {
		return control.Node{}, linkError("nodes.decode_link", "shadowsocks method and password are required")
	}
	host, port, err := splitHostPort(hostPort)
	if err != nil {
		return control.Node{}, err
	}
	options := map[string]any{"method": method, "password": password}
	if plugin := firstQuery(query, "plugin"); plugin != "" {
		parts := strings.SplitN(plugin, ";", 2)
		options["plugin"] = parts[0]
		if len(parts) == 2 && parts[1] != "" {
			options["plugin_opts"] = parts[1]
		}
	}
	return control.Node{
		Name:     decodeName(name, host),
		Protocol: "shadowsocks",
		Server:   host,
		Port:     port,
		Options:  options,
	}, nil
}

func decodeVMess(body string) (control.Node, error) {
	payload, name := splitFragment(body)
	decoded, err := decodeBase64String(payload)
	if err != nil {
		return control.Node{}, linkError("nodes.decode_link", "decode vmess payload: "+err.Error())
	}
	var fields map[string]any
	if err := json.Unmarshal([]byte(decoded), &fields); err != nil {
		return control.Node{}, linkError("nodes.decode_link", "parse vmess JSON: "+err.Error())
	}
	server := stringField(fields, "add", "server")
	portValue, err := intField(fields, "port")
	if err != nil || portValue <= 0 || portValue > 65535 {
		return control.Node{}, linkError("nodes.decode_link", "vmess port is invalid")
	}
	uuid := stringField(fields, "id", "uuid")
	if server == "" || uuid == "" {
		return control.Node{}, linkError("nodes.decode_link", "vmess server and UUID are required")
	}
	options := map[string]any{"uuid": uuid}
	if security := stringField(fields, "scy", "security"); security != "" && security != "auto" {
		options["security"] = security
	}
	if alterID, alterErr := intField(fields, "aid", "alter_id"); alterErr == nil && alterID > 0 {
		options["alter_id"] = alterID
	}
	query := url.Values{}
	query.Set("type", stringField(fields, "net", "network"))
	query.Set("host", stringField(fields, "host"))
	query.Set("path", stringField(fields, "path"))
	if strings.EqualFold(stringField(fields, "tls"), "tls") {
		query.Set("security", "tls")
	}
	query.Set("sni", stringField(fields, "sni"))
	query.Set("alpn", stringField(fields, "alpn"))
	query.Set("fp", stringField(fields, "fp"))
	if tlsOptions := decodeTLS(query, false); tlsOptions != nil {
		options["tls"] = tlsOptions
	}
	transport, err := decodeTransport(query)
	if err != nil {
		return control.Node{}, err
	}
	if transport != nil {
		options["transport"] = transport
	}
	return control.Node{
		Name:     decodeName(stringField(fields, "ps", "name"), decodeName(name, server)),
		Protocol: "vmess",
		Server:   server,
		Port:     uint16(portValue),
		Options:  options,
	}, nil
}

func decodeTransport(query url.Values) (map[string]any, error) {
	network := strings.ToLower(firstQuery(query, "type", "network", "net"))
	path := firstQuery(query, "path")
	host := firstQuery(query, "host")
	switch network {
	case "", "tcp", "raw":
		return nil, nil
	case "ws", "websocket":
		options := map[string]any{"type": "ws"}
		if path != "" {
			options["path"] = path
		}
		if host != "" {
			options["headers"] = map[string]any{"Host": host}
		}
		return options, nil
	case "grpc", "gun":
		options := map[string]any{"type": "grpc"}
		if serviceName := firstQuery(query, "serviceName", "service_name", "path"); serviceName != "" {
			options["service_name"] = serviceName
		}
		return options, nil
	case "h2", "http":
		options := map[string]any{"type": "http"}
		if path != "" {
			options["path"] = path
		}
		if host != "" {
			options["host"] = []string{host}
		}
		return options, nil
	case "httpupgrade":
		options := map[string]any{"type": "httpupgrade"}
		if path != "" {
			options["path"] = path
		}
		if host != "" {
			options["host"] = host
		}
		return options, nil
	default:
		return nil, linkError("nodes.decode_link", fmt.Sprintf("unsupported transport %q", network))
	}
}

func decodeTLS(query url.Values, enabledByDefault bool) map[string]any {
	security := strings.ToLower(firstQuery(query, "security"))
	enabled := enabledByDefault
	if security == "none" || security == "false" {
		enabled = false
	}
	if security == "tls" || security == "reality" {
		enabled = true
	}
	if firstQuery(query, "tls") != "" {
		enabled = parseBool(firstQuery(query, "tls"))
	}
	sni := firstQuery(query, "sni", "serverName", "server_name", "peer")
	if !enabled && sni == "" && firstQuery(query, "pbk", "public_key") == "" {
		return nil
	}
	tlsOptions := map[string]any{"enabled": true}
	if sni != "" {
		tlsOptions["server_name"] = sni
	}
	if parseBool(firstQuery(query, "allowInsecure", "allow_insecure", "insecure")) {
		tlsOptions["insecure"] = true
	}
	if alpn := firstQuery(query, "alpn"); alpn != "" {
		tlsOptions["alpn"] = strings.Split(alpn, ",")
	}
	if fingerprint := firstQuery(query, "fp", "fingerprint"); fingerprint != "" {
		tlsOptions["utls"] = map[string]any{"enabled": true, "fingerprint": fingerprint}
	}
	if security == "reality" || firstQuery(query, "pbk", "public_key") != "" {
		reality := map[string]any{"enabled": true}
		if publicKey := firstQuery(query, "pbk", "public_key"); publicKey != "" {
			reality["public_key"] = publicKey
		}
		if shortID := firstQuery(query, "sid", "short_id"); shortID != "" {
			reality["short_id"] = shortID
		}
		tlsOptions["reality"] = reality
	}
	return tlsOptions
}

func linkPort(host string) (uint16, error) {
	_, portText, err := net.SplitHostPort(host)
	if err != nil {
		return 0, linkError("nodes.decode_link", "node link has no valid host and port: "+err.Error())
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil || port == 0 {
		return 0, linkError("nodes.decode_link", "node link port is invalid")
	}
	return uint16(port), nil
}

func splitHostPort(value string) (string, uint16, error) {
	host, portText, err := net.SplitHostPort(value)
	if err != nil {
		return "", 0, linkError("nodes.decode_link", "node link has no valid host and port: "+err.Error())
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil || port == 0 {
		return "", 0, linkError("nodes.decode_link", "node link port is invalid")
	}
	if host == "" {
		return "", 0, linkError("nodes.decode_link", "node link server is required")
	}
	return host, uint16(port), nil
}

func splitFragment(value string) (string, string) {
	if index := strings.Index(value, "#"); index >= 0 {
		return value[:index], value[index+1:]
	}
	return value, ""
}

func splitQuery(value string) (string, string) {
	if index := strings.Index(value, "?"); index >= 0 {
		return value[:index], value[index+1:]
	}
	return value, ""
}

func splitCredential(value string) (string, string) {
	method, password, found := strings.Cut(value, ":")
	if !found {
		return "", ""
	}
	return method, password
}

func decodeBase64String(value string) (string, error) {
	value = strings.TrimSpace(value)
	encodings := []*base64.Encoding{
		base64.RawURLEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.StdEncoding,
	}
	var lastErr error
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(value)
		if err == nil {
			return string(decoded), nil
		}
		lastErr = err
	}
	return "", lastErr
}

func linkName(parsed *url.URL) string {
	return decodeName(parsed.Fragment, parsed.Hostname())
}

func decodeName(value, fallback string) string {
	if value == "" {
		return fallback
	}
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	if strings.TrimSpace(decoded) == "" {
		return fallback
	}
	return decoded
}

func urlUserSecret(parsed *url.URL) string {
	if parsed.User == nil {
		return ""
	}
	if password, ok := parsed.User.Password(); ok {
		return password
	}
	return parsed.User.Username()
}

func setBasicAuth(options map[string]any, parsed *url.URL) {
	if parsed.User == nil {
		return
	}
	if username := parsed.User.Username(); username != "" {
		options["username"] = username
	}
	if password, ok := parsed.User.Password(); ok {
		options["password"] = password
	}
}

func firstQuery(query url.Values, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func stringField(fields map[string]any, keys ...string) string {
	for _, key := range keys {
		value, exists := fields[key]
		if !exists || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return typed
			}
		case json.Number:
			return typed.String()
		case float64:
			return strconv.FormatFloat(typed, 'f', -1, 64)
		}
	}
	return ""
}

func intField(fields map[string]any, keys ...string) (int, error) {
	value := stringField(fields, keys...)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}
