package main

func configItemVisualFields(section, path string) []configVisualField {
	switch {
	case section == sectionRoute && path == "rules":
		return routeRuleItemFields()
	case section == sectionRoute && path == "rule_set":
		return routeRuleSetItemFields()
	case section == sectionDNS && path == "servers":
		return dnsServerItemFields()
	case section == sectionDNS && path == "rules":
		return dnsRuleItemFields()
	default:
		return nil
	}
}

func routeRuleItemFields() []configVisualField {
	return []configVisualField{
		{Path: "action", Label: "Action", Kind: configVisualSelect, Options: []string{"route", "direct", "reject", "hijack-dns", "sniff", "resolve"}},
		{Path: "outbound", Label: "Outbound", Kind: configVisualText, Help: "Target outbound for route actions"},
		{Path: "network", Label: "Network", Kind: configVisualStringList, Help: "tcp, udp"},
		{Path: "protocol", Label: "Protocol", Kind: configVisualStringList, Help: "bittorrent, http, tls, quic"},
		{Path: "inbound", Label: "Inbound tags", Kind: configVisualStringList},
		{Path: "domain", Label: "Domains", Kind: configVisualStringList},
		{Path: "domain_suffix", Label: "Domain suffixes", Kind: configVisualStringList},
		{Path: "domain_keyword", Label: "Domain keywords", Kind: configVisualStringList},
		{Path: "domain_regex", Label: "Domain regex", Kind: configVisualStringList},
		{Path: "ip_cidr", Label: "IP CIDRs", Kind: configVisualStringList},
		{Path: "source_ip_cidr", Label: "Source IP CIDRs", Kind: configVisualStringList},
		{Path: "port", Label: "Destination ports", Kind: configVisualNumberList, Help: "443, 8443-8445"},
		{Path: "source_port", Label: "Source ports", Kind: configVisualNumberList},
		{Path: "rule_set", Label: "Rule sets", Kind: configVisualStringList},
		{Path: "invert", Label: "Invert match", Kind: configVisualBool},
		{Path: "method", Label: "Reject method", Kind: configVisualSelect, Options: []string{"default", "drop", "reply"}},
		{Path: "server", Label: "Resolve server", Kind: configVisualText},
		{Path: "strategy", Label: "Resolve strategy", Kind: configVisualSelect, Options: []string{"prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only"}},
	}
}

func routeRuleSetItemFields() []configVisualField {
	return []configVisualField{
		{Path: "tag", Label: "Tag", Kind: configVisualText},
		{Path: "type", Label: "Type", Kind: configVisualSelect, Options: []string{"inline", "local", "remote"}},
		{Path: "format", Label: "Format", Kind: configVisualSelect, Options: []string{"source", "binary"}},
		{Path: "path", Label: "Local path", Kind: configVisualText},
		{Path: "url", Label: "Remote URL", Kind: configVisualText},
		{Path: "update_interval", Label: "Update interval", Kind: configVisualText},
		{Path: "download_detour", Label: "Download detour", Kind: configVisualText},
	}
}

func dnsServerItemFields() []configVisualField {
	return []configVisualField{
		{Path: "tag", Label: "Tag", Kind: configVisualText},
		{Path: "type", Label: "Type", Kind: configVisualSelect, Options: []string{"udp", "tcp", "tls", "https", "quic", "h3", "local", "hosts", "legacy"}},
		{Path: "server", Label: "Server", Kind: configVisualText},
		{Path: "server_port", Label: "Server port", Kind: configVisualNumber},
		{Path: "path", Label: "Path", Kind: configVisualText},
		{Path: "address", Label: "Address", Kind: configVisualText},
		{Path: "detour", Label: "Detour", Kind: configVisualText},
		{Path: "strategy", Label: "Strategy", Kind: configVisualSelect, Options: []string{"prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only"}},
	}
}

func dnsRuleItemFields() []configVisualField {
	return []configVisualField{
		{Path: "action", Label: "Action", Kind: configVisualSelect, Options: []string{"route", "reject", "predefined", "evaluate"}},
		{Path: "server", Label: "DNS server", Kind: configVisualText},
		{Path: "domain", Label: "Domains", Kind: configVisualStringList},
		{Path: "domain_suffix", Label: "Domain suffixes", Kind: configVisualStringList},
		{Path: "domain_keyword", Label: "Domain keywords", Kind: configVisualStringList},
		{Path: "domain_regex", Label: "Domain regex", Kind: configVisualStringList},
		{Path: "query_type", Label: "Query types", Kind: configVisualStringList, Help: "A, AAAA, HTTPS"},
		{Path: "network", Label: "Network", Kind: configVisualStringList},
		{Path: "ip_cidr", Label: "IP CIDRs", Kind: configVisualStringList},
		{Path: "source_ip_cidr", Label: "Source IP CIDRs", Kind: configVisualStringList},
		{Path: "port", Label: "Destination ports", Kind: configVisualNumberList},
		{Path: "source_port", Label: "Source ports", Kind: configVisualNumberList},
		{Path: "inbound", Label: "Inbound tags", Kind: configVisualStringList},
		{Path: "rule_set", Label: "Rule sets", Kind: configVisualStringList},
		{Path: "invert", Label: "Invert match", Kind: configVisualBool},
		{Path: "method", Label: "Reject method", Kind: configVisualSelect, Options: []string{"default", "drop", "reply"}},
		{Path: "rcode", Label: "Response code", Kind: configVisualText},
	}
}
