package core

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func parseTUICCredentials(u *url.URL) (string, string, error) {
	if u == nil {
		return "", "", fmt.Errorf("invalid tuic link")
	}
	if u.User == nil {
		return "", "", nil
	}
	credentials, err := userInfoValue(u)
	if err != nil {
		return "", "", fmt.Errorf("invalid tuic credentials: %w", err)
	}
	uuid, password, _ := strings.Cut(credentials, ":")
	return uuid, password, nil
}

func parseModernLinkServer(u *url.URL, scheme string, defaultPort int) (string, int, error) {
	if u == nil || u.Hostname() == "" {
		return "", 0, fmt.Errorf("invalid %s link: missing server", scheme)
	}
	server := u.Hostname()
	portText := u.Port()
	if portText == "" {
		return server, defaultPort, nil
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("invalid %s server port %q", scheme, portText)
	}
	return server, port, nil
}

func buildModernLinkTLS(query url.Values, server string) (map[string]any, error) {
	if _, _, err := parseModernBool(query, "allowInsecure", "allow_insecure", "insecure", "skip-cert-verify", "skip_cert_verify"); err != nil {
		return nil, err
	}
	tlsConfig := buildLinkTLSConfig(query, true)
	if tlsConfig == nil {
		return nil, fmt.Errorf("TLS cannot be disabled for this protocol")
	}
	if _, exists := tlsConfig["server_name"]; !exists {
		tlsConfig["server_name"] = server
	}
	pinValue := firstHysteria2Value(query, "pinSHA256", "pin_sha256", "pin-sha256", "certificate_public_key_sha256")
	hashes, err := normalizeCertificatePublicKeySHA256(pinValue)
	if err != nil {
		return nil, err
	}
	if len(hashes) > 0 {
		tlsConfig["certificate_public_key_sha256"] = hashes
	}
	return tlsConfig, nil
}

func buildTUICLinkOptions(query url.Values) (map[string]any, error) {
	options := make(map[string]any)
	if value := strings.ToLower(firstHysteria2Value(query, "congestion_control", "congestion-controller")); value != "" {
		if value != "cubic" && value != "new_reno" && value != "bbr" {
			return nil, fmt.Errorf("unsupported tuic congestion control %q", value)
		}
		options["congestion_control"] = value
	}
	if value := strings.ToLower(firstHysteria2Value(query, "udp_relay_mode", "udp-relay-mode")); value != "" {
		if value != "native" && value != "quic" {
			return nil, fmt.Errorf("unsupported tuic udp relay mode %q", value)
		}
		options["udp_relay_mode"] = value
	}
	if err := addTUICBooleans(options, query); err != nil {
		return nil, err
	}
	if heartbeat := firstHysteria2Value(query, "heartbeat"); heartbeat != "" {
		value, err := normalizeModernDuration(heartbeat, "tuic heartbeat")
		if err != nil {
			return nil, err
		}
		options["heartbeat"] = value
	}
	network, err := parseHysteria2Network(query)
	if err != nil {
		return nil, fmt.Errorf("invalid tuic network: %w", err)
	}
	if network != nil {
		options["network"] = network
	}
	return options, nil
}

func addTUICBooleans(options map[string]any, query url.Values) error {
	fields := []struct {
		name string
		keys []string
	}{
		{name: "udp_over_stream", keys: []string{"udp_over_stream", "udp-over-stream"}},
		{name: "zero_rtt_handshake", keys: []string{"zero_rtt_handshake", "zero-rtt-handshake", "0rtt"}},
	}
	for _, field := range fields {
		value, present, err := parseModernBool(query, field.keys...)
		if err != nil {
			return fmt.Errorf("%s: %w", field.name, err)
		}
		if present {
			options[field.name] = value
		}
	}
	if streamed, ok := options["udp_over_stream"].(bool); ok && streamed {
		if _, relayModeSet := options["udp_relay_mode"]; relayModeSet {
			return fmt.Errorf("udp_over_stream conflicts with udp_relay_mode")
		}
	}
	return nil
}

func buildAnyTLSLinkOptions(query url.Values) (map[string]any, error) {
	options := make(map[string]any)
	for _, field := range []struct {
		name string
		keys []string
	}{
		{name: "idle_session_check_interval", keys: []string{"idle_session_check_interval", "idle-session-check-interval"}},
		{name: "idle_session_timeout", keys: []string{"idle_session_timeout", "idle-session-timeout"}},
	} {
		raw := firstHysteria2Value(query, field.keys...)
		if raw == "" {
			continue
		}
		value, err := normalizeModernDuration(raw, "anytls "+field.name)
		if err != nil {
			return nil, err
		}
		options[field.name] = value
	}
	if raw := firstHysteria2Value(query, "min_idle_session", "min-idle-session"); raw != "" {
		value, err := parseModernNonNegativeInt(raw, "anytls min_idle_session")
		if err != nil {
			return nil, err
		}
		options["min_idle_session"] = value
	}
	return options, nil
}

func parseShadowTLSVersion(query url.Values) (int, error) {
	raw := firstHysteria2Value(query, "version")
	if raw == "" {
		return 3, nil
	}
	version, err := strconv.Atoi(raw)
	if err != nil || version < 1 || version > 3 {
		return 0, fmt.Errorf("invalid shadowtls version %q", raw)
	}
	return version, nil
}

func parseModernBool(query url.Values, keys ...string) (bool, bool, error) {
	raw := firstHysteria2Value(query, keys...)
	if raw == "" {
		return false, false, nil
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true, true, nil
	case "0", "false", "no", "off":
		return false, true, nil
	default:
		return false, true, fmt.Errorf("invalid boolean value %q", raw)
	}
}

func normalizeModernDuration(raw, field string) (string, error) {
	value := strings.TrimSpace(raw)
	if _, err := time.ParseDuration(value); err != nil {
		if _, integerErr := strconv.Atoi(value); integerErr == nil {
			value += "s"
		} else {
			return "", fmt.Errorf("invalid %s %q", field, raw)
		}
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration < 0 {
		return "", fmt.Errorf("invalid %s %q", field, raw)
	}
	return value, nil
}

func parseModernNonNegativeInt(raw, field string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid %s %q", field, raw)
	}
	return value, nil
}
