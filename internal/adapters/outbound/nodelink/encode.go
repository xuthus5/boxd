package nodelink

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/xuthus5/boxd/control"
)

func encodeShadowsocks(node control.Node) (string, error) {
	method := optionString(node.Options, "method")
	password := optionString(node.Options, "password")
	if method == "" || password == "" {
		return "", linkError("nodes.encode_link", "shadowsocks method and password are required")
	}
	userInfo := base64.RawURLEncoding.EncodeToString([]byte(method + ":" + password))
	query := url.Values{}
	if plugin := optionString(node.Options, "plugin"); plugin != "" {
		if pluginOptions := optionString(node.Options, "plugin_opts", "pluginOptions"); pluginOptions != "" {
			plugin += ";" + pluginOptions
		}
		query.Set("plugin", plugin)
	}
	result := "ss://" + userInfo + "@" + net.JoinHostPort(node.Server, strconv.Itoa(int(node.Port)))
	if encoded := query.Encode(); encoded != "" {
		result += "?" + encoded
	}
	return result + linkFragment(node), nil
}

func encodeVMess(node control.Node) (string, error) {
	uuid := optionString(node.Options, "uuid")
	if uuid == "" {
		return "", linkError("nodes.encode_link", "vmess UUID is required")
	}
	transport := optionMap(node.Options, "transport")
	tlsOptions := optionMap(node.Options, "tls")
	fields := map[string]any{
		"v":    "2",
		"ps":   node.Name,
		"add":  node.Server,
		"port": strconv.Itoa(int(node.Port)),
		"id":   uuid,
		"aid":  strconv.Itoa(optionInt(node.Options, "alter_id")),
		"scy":  optionString(node.Options, "security"),
		"net":  optionString(transport, "type"),
		"type": "none",
		"host": transportHost(transport),
		"path": optionString(transport, "path"),
	}
	if optionBool(tlsOptions, "enabled") {
		fields["tls"] = "tls"
		fields["sni"] = optionString(tlsOptions, "server_name", "sni")
		if alpn := optionStringSlice(tlsOptions, "alpn"); len(alpn) > 0 {
			fields["alpn"] = strings.Join(alpn, ",")
		}
		if utls := optionMap(tlsOptions, "utls"); utls != nil {
			fields["fp"] = optionString(utls, "fingerprint")
		}
	}
	body, err := json.Marshal(fields)
	if err != nil {
		return "", linkError("nodes.encode_link", "marshal vmess payload: "+err.Error())
	}
	return "vmess://" + base64.RawStdEncoding.EncodeToString(body), nil
}

func encodeURLNode(node control.Node, scheme string) (string, error) {
	parsed := &url.URL{
		Scheme:   scheme,
		Host:     net.JoinHostPort(node.Server, strconv.Itoa(int(node.Port))),
		Fragment: node.Name,
	}
	query := url.Values{}
	tlsOptions := optionMap(node.Options, "tls")
	transport := optionMap(node.Options, "transport")
	switch scheme {
	case "vless":
		uuid := optionString(node.Options, "uuid")
		if uuid == "" {
			return "", linkError("nodes.encode_link", "vless UUID is required")
		}
		parsed.User = url.User(uuid)
		if flow := optionString(node.Options, "flow"); flow != "" {
			query.Set("flow", flow)
		}
	case "trojan":
		password := optionString(node.Options, "password")
		if password == "" {
			return "", linkError("nodes.encode_link", "trojan password is required")
		}
		parsed.User = url.User(password)
	case "hysteria2":
		password := optionString(node.Options, "password")
		if password == "" {
			return "", linkError("nodes.encode_link", "hysteria2 password is required")
		}
		parsed.User = url.User(password)
		if obfs := optionMap(node.Options, "obfs"); obfs != nil {
			if obfsType := optionString(obfs, "type"); obfsType != "" {
				query.Set("obfs", obfsType)
			}
			if obfsPassword := optionString(obfs, "password"); obfsPassword != "" {
				query.Set("obfs-password", obfsPassword)
			}
		}
	case "socks5", "http", "https":
		username := optionString(node.Options, "username")
		password, _ := node.Options["password"].(string)
		if username != "" {
			if password != "" {
				parsed.User = url.UserPassword(username, password)
			} else {
				parsed.User = url.User(username)
			}
		}
		if scheme == "https" || (scheme == "http" && optionBool(tlsOptions, "enabled")) {
			parsed.Scheme = "https"
		}
	}
	encodeTLS(query, tlsOptions, scheme == "trojan" || scheme == "hysteria2")
	encodeTransport(query, transport)
	if encoded := query.Encode(); encoded != "" {
		parsed.RawQuery = encoded
	}
	return parsed.String(), nil
}

func encodeTLS(query url.Values, tlsOptions map[string]any, enabledByDefault bool) {
	if tlsOptions == nil {
		if enabledByDefault {
			query.Set("security", "tls")
		}
		return
	}
	enabled := optionBool(tlsOptions, "enabled")
	if !enabled && !enabledByDefault {
		query.Set("security", "none")
		return
	}
	security := "tls"
	if reality := optionMap(tlsOptions, "reality"); optionBool(reality, "enabled") {
		security = "reality"
		if publicKey := optionString(reality, "public_key"); publicKey != "" {
			query.Set("pbk", publicKey)
		}
		if shortID := optionString(reality, "short_id"); shortID != "" {
			query.Set("sid", shortID)
		}
	}
	query.Set("security", security)
	if serverName := optionString(tlsOptions, "server_name", "sni"); serverName != "" {
		query.Set("sni", serverName)
	}
	if optionBool(tlsOptions, "insecure") {
		query.Set("allowInsecure", "1")
	}
	if alpn := optionStringSlice(tlsOptions, "alpn"); len(alpn) > 0 {
		query.Set("alpn", strings.Join(alpn, ","))
	}
	if utls := optionMap(tlsOptions, "utls"); utls != nil {
		if fingerprint := optionString(utls, "fingerprint"); fingerprint != "" {
			query.Set("fp", fingerprint)
		}
	}
}

func encodeTransport(query url.Values, transport map[string]any) {
	if transport == nil {
		return
	}
	transportType := optionString(transport, "type")
	if transportType == "" || transportType == "tcp" {
		return
	}
	query.Set("type", transportType)
	if path := optionString(transport, "path"); path != "" {
		query.Set("path", path)
	}
	if host := transportHost(transport); host != "" {
		query.Set("host", host)
	}
	if serviceName := optionString(transport, "service_name"); serviceName != "" {
		query.Set("serviceName", serviceName)
	}
}

func transportHost(transport map[string]any) string {
	if transport == nil {
		return ""
	}
	if host := optionString(transport, "host"); host != "" {
		return host
	}
	headers := optionMap(transport, "headers")
	return optionString(headers, "Host", "host")
}

func linkFragment(node control.Node) string {
	if node.Name == "" {
		return ""
	}
	return "#" + url.PathEscape(node.Name)
}

func optionString(options map[string]any, keys ...string) string {
	for _, key := range keys {
		value, exists := options[key]
		if !exists || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if typed != "" {
				return typed
			}
		case json.Number:
			return typed.String()
		case float64:
			return strconv.FormatFloat(typed, 'f', -1, 64)
		case int:
			return strconv.Itoa(typed)
		case int64:
			return strconv.FormatInt(typed, 10)
		}
	}
	return ""
}

func optionBool(options map[string]any, keys ...string) bool {
	for _, key := range keys {
		value, exists := options[key]
		if !exists {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed
		case string:
			return parseBool(typed)
		}
	}
	return false
}

func optionInt(options map[string]any, keys ...string) int {
	value := optionString(options, keys...)
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func optionMap(options map[string]any, key string) map[string]any {
	if options == nil {
		return nil
	}
	value, exists := options[key]
	if !exists {
		return nil
	}
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[string]string:
		result := make(map[string]any, len(typed))
		for itemKey, itemValue := range typed {
			result[itemKey] = itemValue
		}
		return result
	default:
		return nil
	}
}

func optionStringSlice(options map[string]any, key string) []string {
	if options == nil {
		return nil
	}
	switch typed := options[key].(type) {
	case []string:
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if value, ok := item.(string); ok && value != "" {
				result = append(result, value)
			}
		}
		return result
	default:
		return nil
	}
}
