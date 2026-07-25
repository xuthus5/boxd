package core

import (
	"fmt"
	"net/url"

	"github.com/xuthus5/boxd/internal/model"
)

func parseHysteria(u *url.URL) (*model.ImportResult, error) {
	server, port, err := parseModernLinkServer(u, "hysteria", 443)
	if err != nil {
		return nil, err
	}
	query := u.Query()
	config := map[string]any{
		"type":        "hysteria",
		"server":      server,
		"server_port": port,
	}
	auth, err := parseHysteriaAuth(u, query)
	if err != nil {
		return nil, err
	}
	for key, value := range auth {
		config[key] = value
	}
	options, err := buildHysteriaOptions(query)
	if err != nil {
		return nil, fmt.Errorf("invalid hysteria options: %w", err)
	}
	for key, value := range options {
		config[key] = value
	}
	tlsConfig, err := buildHysteria2TLS(query, server)
	if err != nil {
		return nil, fmt.Errorf("invalid hysteria TLS: %w", err)
	}
	config["tls"] = tlsConfig
	tag := nodeName(u, fmt.Sprintf("hysteria-%s-%d", server, port))
	return &model.ImportResult{Tag: tag, Type: "hysteria", Server: server, Port: port, Config: config}, nil
}

func parseHysteriaAuth(u *url.URL, query url.Values) (map[string]any, error) {
	userInfo, err := userInfoValue(u)
	if err != nil {
		return nil, fmt.Errorf("invalid hysteria credentials: %w", err)
	}
	auth := firstNonEmpty(userInfo, query.Get("auth_str"), query.Get("auth"), query.Get("password"))
	if auth == "" {
		return nil, nil
	}
	return map[string]any{"auth_str": auth}, nil
}

func buildHysteriaOptions(query url.Values) (map[string]any, error) {
	options := make(map[string]any)
	if err := addHysteriaPortOptions(options, query); err != nil {
		return nil, err
	}
	if err := addHysteriaBandwidthOptions(options, query); err != nil {
		return nil, err
	}
	if err := addHysteriaWindowOptions(options, query); err != nil {
		return nil, err
	}
	if obfs := firstHysteria2Value(query, "obfs", "obfs-password"); obfs != "" {
		options["obfs"] = obfs
	}
	network, err := parseHysteria2Network(query)
	if err != nil {
		return nil, fmt.Errorf("invalid network: %w", err)
	}
	if network != nil {
		options["network"] = network
	}
	return options, nil
}

func addHysteriaPortOptions(options map[string]any, query url.Values) error {
	ports, err := parseHysteria2Ports(query)
	if err != nil {
		return err
	}
	if len(ports) > 0 {
		options["server_ports"] = ports
	}
	if interval := firstHysteria2Value(query, "hop_interval", "hop-interval", "hopInterval"); interval != "" {
		value, err := normalizeModernDuration(interval, "hysteria hop interval")
		if err != nil {
			return err
		}
		options["hop_interval"] = value
	}
	return nil
}

func addHysteriaBandwidthOptions(options map[string]any, query url.Values) error {
	for _, field := range []struct {
		name string
		keys []string
	}{
		{name: "up_mbps", keys: []string{"upmbps", "up_mbps"}},
		{name: "down_mbps", keys: []string{"downmbps", "down_mbps"}},
	} {
		value, present, err := parseHysteria2Mbps(query, field.keys...)
		if err != nil {
			return fmt.Errorf("%s: %w", field.name, err)
		}
		if present {
			options[field.name] = value
		}
	}
	for _, field := range []string{"up", "down"} {
		if value := firstHysteria2Value(query, field); value != "" {
			options[field] = value
		}
	}
	return nil
}

func addHysteriaWindowOptions(options map[string]any, query url.Values) error {
	for _, field := range []struct {
		name string
		keys []string
	}{
		{name: "recv_window_conn", keys: []string{"recv_window_conn", "recv-window-conn"}},
		{name: "recv_window", keys: []string{"recv_window", "recv-window"}},
	} {
		raw := firstHysteria2Value(query, field.keys...)
		if raw == "" {
			continue
		}
		value, err := parseModernNonNegativeInt(raw, "hysteria "+field.name)
		if err != nil {
			return err
		}
		options[field.name] = value
	}
	value, present, err := parseModernBool(query, "disable_mtu_discovery", "disable-mtu-discovery")
	if err != nil {
		return fmt.Errorf("disable_mtu_discovery: %w", err)
	}
	if present {
		options["disable_mtu_discovery"] = value
	}
	return nil
}
