package main

type configVisualKind uint8

const (
	configVisualText configVisualKind = iota
	configVisualBool
	configVisualNumber
	configVisualSelect
	configVisualSecret
	configVisualStringList
	configVisualNumberList
)

type configVisualField struct {
	Path    string
	Label   string
	Kind    configVisualKind
	Options []string
	Help    string
}

func configVisualFields(section string) []configVisualField {
	switch section {
	case sectionLog:
		return logVisualFields()
	case sectionNTP:
		return ntpVisualFields()
	case sectionExperimental:
		return experimentalVisualFields()
	case sectionDNS:
		return dnsVisualFields()
	case sectionRoute:
		return routeVisualFields()
	case sectionCertificate:
		return certificateVisualFields()
	default:
		return nil
	}
}

func logVisualFields() []configVisualField {
	return []configVisualField{
		{Path: "disabled", Label: "Disable logging", Kind: configVisualBool},
		{Path: "level", Label: "Log level", Kind: configVisualSelect, Options: []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}},
		{Path: "output", Label: "Output file", Kind: configVisualText},
		{Path: "timestamp", Label: "Include timestamp", Kind: configVisualBool},
	}
}

func ntpVisualFields() []configVisualField {
	return []configVisualField{
		{Path: "enabled", Label: "Enable NTP", Kind: configVisualBool},
		{Path: "server", Label: "Server", Kind: configVisualText},
		{Path: "server_port", Label: "Server port", Kind: configVisualNumber},
		{Path: "interval", Label: "Update interval", Kind: configVisualText},
		{Path: "write_to_system", Label: "Write time to system", Kind: configVisualBool},
		{Path: "detour", Label: "Outbound detour", Kind: configVisualText},
		{Path: "bind_interface", Label: "Bind interface", Kind: configVisualText},
		{Path: "protect_path", Label: "Protect path", Kind: configVisualText},
		{Path: "connect_timeout", Label: "Connect timeout", Kind: configVisualText},
		{Path: "routing_mark", Label: "Routing mark", Kind: configVisualText},
	}
}

func experimentalVisualFields() []configVisualField {
	return []configVisualField{
		{Path: "cache_file.enabled", Label: "Enable cache file", Kind: configVisualBool},
		{Path: "cache_file.path", Label: "Cache path", Kind: configVisualText},
		{Path: "cache_file.cache_id", Label: "Cache ID", Kind: configVisualText},
		{Path: "cache_file.store_dns", Label: "Cache DNS", Kind: configVisualBool},
		{Path: "cache_file.store_fakeip", Label: "Cache FakeIP", Kind: configVisualBool},
		{Path: "clash_api.external_controller", Label: "Clash controller", Kind: configVisualText},
		{Path: "clash_api.external_ui", Label: "Clash external UI", Kind: configVisualText},
		{Path: "clash_api.secret", Label: "Clash secret", Kind: configVisualSecret},
		{Path: "clash_api.default_mode", Label: "Clash mode", Kind: configVisualSelect, Options: []string{"rule", "global", "direct"}},
		{Path: "v2ray_api.listen", Label: "V2Ray API listen", Kind: configVisualText},
		{Path: "v2ray_api.stats.enabled", Label: "Enable V2Ray stats", Kind: configVisualBool},
	}
}

func dnsVisualFields() []configVisualField {
	return []configVisualField{
		{Path: "final", Label: "Final DNS server", Kind: configVisualText},
		{Path: "strategy", Label: "Default strategy", Kind: configVisualSelect, Options: []string{"prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only"}},
		{Path: "disable_cache", Label: "Disable cache", Kind: configVisualBool},
		{Path: "disable_expire", Label: "Disable cache expiry", Kind: configVisualBool},
		{Path: "independent_cache", Label: "Independent cache", Kind: configVisualBool},
		{Path: "reverse_mapping", Label: "Reverse mapping", Kind: configVisualBool},
		{Path: "use_hosts", Label: "Use hosts", Kind: configVisualBool},
	}
}

func routeVisualFields() []configVisualField {
	return []configVisualField{
		{Path: "final", Label: "Final outbound", Kind: configVisualText},
		{Path: "auto_detect_interface", Label: "Auto-detect interface", Kind: configVisualBool},
		{Path: "find_process", Label: "Find process", Kind: configVisualBool},
		{Path: "override_android_vpn", Label: "Override Android VPN", Kind: configVisualBool},
	}
}

func certificateVisualFields() []configVisualField {
	return []configVisualField{
		{Path: "store", Label: "Certificate store", Kind: configVisualSelect, Options: []string{"system", "mozilla", "none"}},
		{Path: "certificate", Label: "Certificate PEM", Kind: configVisualText},
		{Path: "key", Label: "Private key PEM", Kind: configVisualSecret},
		{Path: "certificate_path", Label: "Certificate path", Kind: configVisualText},
		{Path: "key_path", Label: "Private key path", Kind: configVisualText},
	}
}
