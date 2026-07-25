package core

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func hysteria2Password(u *url.URL) (string, error) {
	password, err := userInfoValue(u)
	if err != nil {
		return "", err
	}
	if password != "" {
		return password, nil
	}
	query := u.Query()
	return firstNonEmpty(query.Get("auth"), query.Get("password")), nil
}

func parseHysteria2Port(u *url.URL) (int, error) {
	portText := u.Port()
	if portText == "" {
		return 443, nil
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid hysteria2 server port %q", portText)
	}
	return port, nil
}

func buildHysteria2Options(query url.Values, server string) (map[string]any, error) {
	tls, err := buildHysteria2TLS(query, server)
	if err != nil {
		return nil, err
	}
	options := map[string]any{"tls": tls}
	ports, err := parseHysteria2Ports(query)
	if err != nil {
		return nil, err
	}
	if len(ports) > 0 {
		options["server_ports"] = ports
	}
	if interval := firstHysteria2Value(query, "hop_interval", "hop-interval", "hopInterval"); interval != "" {
		options["hop_interval"], err = normalizeHysteria2Duration(interval)
		if err != nil {
			return nil, err
		}
	}
	if err := addHysteria2Bandwidth(options, query); err != nil {
		return nil, err
	}
	obfs, err := parseHysteria2Obfs(query)
	if err != nil {
		return nil, err
	}
	if obfs != nil {
		options["obfs"] = obfs
	}
	network, err := parseHysteria2Network(query)
	if err != nil {
		return nil, err
	}
	if network != nil {
		options["network"] = network
	}
	return options, nil
}

func addHysteria2Bandwidth(options map[string]any, query url.Values) error {
	fields := []struct {
		name  string
		query []string
	}{
		{name: "up_mbps", query: []string{"upmbps", "up_mbps", "up-mbps"}},
		{name: "down_mbps", query: []string{"downmbps", "down_mbps", "down-mbps"}},
	}
	for _, field := range fields {
		value, present, err := parseHysteria2Mbps(query, field.query...)
		if err != nil {
			return fmt.Errorf("%s: %w", field.name, err)
		}
		if present {
			options[field.name] = value
		}
	}
	return nil
}

func buildHysteria2TLS(query url.Values, server string) (map[string]any, error) {
	serverName := firstHysteria2Value(query, "sni", "servername", "server_name")
	if serverName == "" {
		serverName = server
	}
	insecure, err := parseHysteria2Bool(query, "insecure", "skip-cert-verify", "skip_cert_verify")
	if err != nil {
		return nil, err
	}
	tls := buildTLSConfig(true, serverName, insecure, firstHysteria2Value(query, "alpn"))
	pinValue := firstHysteria2Value(query, "pinSHA256", "pin_sha256", "pin-sha256", "certificate_public_key_sha256")
	hashes, err := normalizeCertificatePublicKeySHA256(pinValue)
	if err != nil {
		return nil, err
	}
	if len(hashes) > 0 {
		tls["certificate_public_key_sha256"] = hashes
	}
	return tls, nil
}

func parseHysteria2Ports(query url.Values) ([]string, error) {
	raw := firstHysteria2Value(query, "mport", "server_ports", "server-ports")
	if raw == "" {
		return nil, nil
	}
	parts := splitLinkList(raw)
	ports := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized, err := normalizeHysteria2PortRange(part)
		if err != nil {
			return nil, err
		}
		ports = append(ports, normalized)
	}
	return ports, nil
}

func normalizeHysteria2PortRange(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("invalid server port range %q", raw)
	}
	if strings.Contains(value, "-") {
		if strings.Count(value, "-") != 1 || strings.HasPrefix(value, "-") || strings.HasSuffix(value, "-") {
			return "", fmt.Errorf("invalid server port range %q", raw)
		}
		value = strings.Replace(value, "-", ":", 1)
	}
	if !strings.Contains(value, ":") {
		port, err := strconv.Atoi(value)
		if err != nil || port < 1 || port > 65535 {
			return "", fmt.Errorf("invalid server port range %q", raw)
		}
		return fmt.Sprintf("%d:%d", port, port), nil
	}
	parts := strings.Split(value, ":")
	if len(parts) != 2 || parts[0] == "" && parts[1] == "" {
		return "", fmt.Errorf("invalid server port range %q", raw)
	}
	parsedParts := make([]int, len(parts))
	for index, part := range parts {
		if part == "" {
			continue
		}
		port, err := strconv.Atoi(part)
		if err != nil || port < 1 || port > 65535 {
			return "", fmt.Errorf("invalid server port range %q", raw)
		}
		parsedParts[index] = port
	}
	if parts[0] != "" && parts[1] != "" {
		start, end := parsedParts[0], parsedParts[1]
		if start > end {
			return "", fmt.Errorf("invalid server port range %q", raw)
		}
	}
	return value, nil
}

func normalizeHysteria2Duration(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if _, err := time.ParseDuration(value); err != nil {
		if _, integerErr := strconv.Atoi(value); integerErr == nil {
			value += "s"
		} else {
			return "", fmt.Errorf("invalid hop interval %q", raw)
		}
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration < 0 {
		return "", fmt.Errorf("invalid hop interval %q", raw)
	}
	return value, nil
}

func parseHysteria2Mbps(query url.Values, keys ...string) (int, bool, error) {
	raw := firstHysteria2Value(query, keys...)
	if raw == "" {
		return 0, false, nil
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0, true, fmt.Errorf("invalid bandwidth %q", raw)
	}
	return value, true, nil
}

func parseHysteria2Obfs(query url.Values) (map[string]any, error) {
	typ := strings.ToLower(strings.TrimSpace(firstHysteria2Value(query, "obfs")))
	password := firstHysteria2Value(query, "obfs-password", "obfs_password")
	if typ == "" {
		if password != "" {
			return nil, fmt.Errorf("obfs password requires obfs=salamander")
		}
		return nil, nil
	}
	if typ != "salamander" {
		return nil, fmt.Errorf("unsupported obfs type %q", typ)
	}
	if password == "" {
		return nil, fmt.Errorf("missing obfs password")
	}
	return map[string]any{"type": typ, "password": password}, nil
}

func parseHysteria2Network(query url.Values) (any, error) {
	raw := firstHysteria2Value(query, "network")
	if raw == "" {
		return nil, nil
	}
	items := splitLinkList(raw)
	if len(items) == 0 {
		return nil, fmt.Errorf("network cannot be empty")
	}
	networks := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		network := strings.ToLower(strings.TrimSpace(item))
		if network != "tcp" && network != "udp" {
			return nil, fmt.Errorf("unsupported network %q", item)
		}
		if _, exists := seen[network]; exists {
			continue
		}
		seen[network] = struct{}{}
		networks = append(networks, network)
	}
	if len(networks) == 1 {
		return networks[0], nil
	}
	return networks, nil
}

func parseHysteria2Bool(query url.Values, keys ...string) (bool, error) {
	raw := firstHysteria2Value(query, keys...)
	if raw == "" {
		return false, nil
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q", raw)
	}
}

func firstHysteria2Value(query url.Values, keys ...string) string {
	for _, key := range keys {
		if value := query.Get(key); value != "" {
			return value
		}
	}
	return ""
}
