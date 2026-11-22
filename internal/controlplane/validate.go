package controlplane

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"regexp"
	"strings"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

var stableIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type stateValidator struct {
	state control.DesiredState
	diags []kernel.Diagnostic
}

func normalizeState(state *control.DesiredState) {
	if state.Routes.Mode == "" {
		state.Routes.Mode = control.RouteModeRules
	}
	if state.Routes.DefaultEgress.Kind == "" {
		state.Routes.DefaultEgress = control.EgressRef{Kind: control.EgressKindDirect}
	}
	if state.DNS.Strategy == "" {
		state.DNS.Strategy = "prefer_ipv4"
	}

	for index := range state.Inbounds {
		inbound := &state.Inbounds[index]
		inbound.ID = strings.TrimSpace(inbound.ID)
		inbound.Name = strings.TrimSpace(inbound.Name)
		inbound.Listen.Host = strings.TrimSpace(inbound.Listen.Host)
		switch inbound.Kind {
		case control.InboundKindMixed:
			if inbound.Listen.Host == "" && inbound.Listen.Port == 0 {
				inbound.Listen = control.ListenAddress{Host: "::", Port: 1080}
			}
		case control.InboundKindTun:
			if len(inbound.Tun.Addresses) == 0 {
				inbound.Tun.Addresses = []string{"172.19.0.1/30", "fdfe:dcba:9876::1/126"}
			}
			if inbound.Tun.MTU == 0 {
				inbound.Tun.MTU = 9000
			}
			if inbound.Tun.Stack == "" {
				inbound.Tun.Stack = "mixed"
			}
		}
	}

	for index := range state.Nodes {
		node := &state.Nodes[index]
		node.ID = strings.TrimSpace(node.ID)
		node.Name = strings.TrimSpace(node.Name)
		node.Protocol = strings.ToLower(strings.TrimSpace(node.Protocol))
		node.Server = strings.TrimSpace(node.Server)
		node.SubscriptionID = strings.TrimSpace(node.SubscriptionID)
	}
	for index := range state.Subscriptions {
		subscription := &state.Subscriptions[index]
		subscription.ID = strings.TrimSpace(subscription.ID)
		subscription.Name = strings.TrimSpace(subscription.Name)
		subscription.URL = strings.TrimSpace(subscription.URL)
		subscription.ETag = strings.TrimSpace(subscription.ETag)
		subscription.LastModified = strings.TrimSpace(subscription.LastModified)
	}
	normalizeRuleSets(&state.RuleSets)
	ensureBuiltinRuleSets(&state.RuleSets)
	for index := range state.Groups {
		group := &state.Groups[index]
		group.ID = strings.TrimSpace(group.ID)
		group.Name = strings.TrimSpace(group.Name)
		if group.Kind == "" {
			group.Kind = control.GroupKindSelector
		}
		for memberIndex := range group.Members {
			group.Members[memberIndex] = strings.TrimSpace(group.Members[memberIndex])
		}
		group.Selected = strings.TrimSpace(group.Selected)
	}
	for index := range state.Chains {
		chain := &state.Chains[index]
		chain.ID = strings.TrimSpace(chain.ID)
		chain.Name = strings.TrimSpace(chain.Name)
		for hopIndex := range chain.Hops {
			chain.Hops[hopIndex].ID = strings.TrimSpace(chain.Hops[hopIndex].ID)
		}
	}
	for index := range state.Routes.Rules {
		rule := &state.Routes.Rules[index]
		rule.ID = strings.TrimSpace(rule.ID)
		rule.Name = strings.TrimSpace(rule.Name)
		rule.Egress.ID = strings.TrimSpace(rule.Egress.ID)
		normalizeMatch(&rule.Match)
	}
	state.Routes.DefaultEgress.ID = strings.TrimSpace(state.Routes.DefaultEgress.ID)
	normalizeDNSProfile(&state.DNS)
}

func normalizeRuleSets(ruleSets *[]control.RuleSetDefinition) {
	for index := range *ruleSets {
		ruleSet := &(*ruleSets)[index]
		ruleSet.Name = strings.TrimSpace(ruleSet.Name)
		ruleSet.URL = strings.TrimSpace(ruleSet.URL)
		if ruleSet.Format == "" {
			ruleSet.Format = control.RuleSetFormatBinary
		}
	}
}

func ensureBuiltinRuleSets(ruleSets *[]control.RuleSetDefinition) {
	existing := make(map[string]struct{}, len(*ruleSets))
	for _, ruleSet := range *ruleSets {
		existing[ruleSet.Name] = struct{}{}
	}
	for _, builtin := range control.DefaultRuleSetDefinitions() {
		if _, exists := existing[builtin.Name]; exists {
			continue
		}
		*ruleSets = append(*ruleSets, builtin)
	}
}

func normalizeMatch(match *control.RouteMatch) {
	normalizeStrings(match.InboundIDs)
	normalizeStrings(match.Domains)
	normalizeStrings(match.DomainSuffixes)
	normalizeStrings(match.DomainKeywords)
	normalizeStrings(match.IPCIDRs)
	normalizeStrings(match.SourceIPCIDRs)
	normalizeStrings(match.ProcessNames)
	normalizeStrings(match.ProcessPaths)
	normalizeStrings(match.RuleSets)
	normalizeStrings(match.Networks)
}

func normalizeStrings(values []string) {
	for index := range values {
		values[index] = strings.TrimSpace(values[index])
	}
}

func normalizeDNSProfile(profile *control.DNSProfile) {
	if profile.Strategy == "" {
		profile.Strategy = "prefer_ipv4"
	}
	normalizeDNSServer(&profile.Domestic)
	normalizeDNSServer(&profile.Foreign)
	normalizeDNSServer(&profile.Default)
}

func normalizeDNSServer(server *control.DNSServerConfig) {
	server.Address = strings.TrimSpace(server.Address)
	if server.Protocol == "" {
		switch {
		case strings.HasPrefix(strings.ToLower(server.Address), "https://"):
			server.Protocol = control.DNSProtocolDoH
		case strings.HasPrefix(strings.ToLower(server.Address), "tls://"):
			server.Protocol = control.DNSProtocolDoT
		default:
			server.Protocol = control.DNSProtocolUDP
		}
	}
	if server.Detour == "" {
		server.Detour = control.DNSDetourRouteDefault
	}
	if server.TimeoutSeconds <= 0 {
		server.TimeoutSeconds = 5
	}
}

func normalizeAndValidateState(state *control.DesiredState) error {
	normalizeState(state)
	validator := &stateValidator{state: *state}
	validator.validate()
	if len(validator.diags) > 0 {
		return kernel.NewValidationError(validator.diags)
	}
	return nil
}

func (v *stateValidator) add(path, code, message, hint string) {
	v.diags = append(v.diags, kernel.Diagnostic{
		Severity: "error",
		Code:     code,
		Path:     path,
		Message:  message,
		Hint:     hint,
	})
}

func (v *stateValidator) validate() {
	v.validateInbounds()
	v.validateSubscriptions()
	v.validateRuleSets()
	v.validateNodes()
	v.validateGroups()
	v.validateChains()
	v.validateRoutes()
	v.validateDNS()
	v.validateUDPPaths()
}

func (v *stateValidator) validateSubscriptions() {
	ids := make(map[string]int, len(v.state.Subscriptions))
	for index, subscription := range v.state.Subscriptions {
		path := fmt.Sprintf("subscriptions[%d]", index)
		if !validStableID(subscription.ID) {
			v.add(path+".id", "invalid_id", "subscription ID is required and may contain letters, digits, dot, underscore, and dash", "use a stable subscription identifier")
		} else if previous, exists := ids[subscription.ID]; exists {
			v.add(path+".id", "duplicate_id", fmt.Sprintf("subscription ID %q is also used by subscriptions[%d]", subscription.ID, previous), "use a unique subscription ID")
		} else {
			ids[subscription.ID] = index
		}
		if subscription.Name == "" {
			v.add(path+".name", "missing_name", "subscription name is required", "give the subscription a user-facing name")
		}
		if !validRemoteURL(subscription.URL) {
			v.add(path+".url", "invalid_url", fmt.Sprintf("subscription URL %q is not an HTTP or HTTPS URL", subscription.URL), "use an http:// or https:// subscription URL")
		}
		if subscription.IntervalSeconds < 0 {
			v.add(path+".interval_seconds", "invalid_interval", "subscription interval must not be negative", "use 0 to disable automatic refresh or a positive number of seconds")
		}
	}
}

func (v *stateValidator) validateRuleSets() {
	ids := make(map[string]int, len(v.state.RuleSets))
	available := make(map[string]control.RuleSetDefinition, len(v.state.RuleSets))
	for index, ruleSet := range v.state.RuleSets {
		path := fmt.Sprintf("rule_sets[%d]", index)
		if !validStableID(ruleSet.Name) {
			v.add(path+".name", "invalid_id", "rule set name is required and may contain letters, digits, dot, underscore, and dash", "use a stable rule set name")
		} else if previous, exists := ids[ruleSet.Name]; exists {
			v.add(path+".name", "duplicate_id", fmt.Sprintf("rule set %q is also used by rule_sets[%d]", ruleSet.Name, previous), "use a unique rule set name")
		} else {
			ids[ruleSet.Name] = index
			available[ruleSet.Name] = ruleSet
		}
		if !validRemoteURL(ruleSet.URL) {
			v.add(path+".url", "invalid_url", fmt.Sprintf("rule set URL %q is not an HTTP or HTTPS URL", ruleSet.URL), "use an http:// or https:// rule set URL")
		}
		switch ruleSet.Format {
		case control.RuleSetFormatBinary, control.RuleSetFormatSource:
		default:
			v.add(path+".format", "invalid_format", fmt.Sprintf("unsupported rule set format %q", ruleSet.Format), "use binary or source")
		}
	}
	if v.state.Routes.Presets.ChinaDirect {
		for _, name := range []string{"geosite-cn", "geoip-cn"} {
			if !enabledRuleSet(available, name) {
				v.add("routes.presets.china_direct", "missing_rule_set", fmt.Sprintf("China-direct preset requires enabled rule set %q", name), "enable the built-in rule set or disable the preset")
			}
		}
	}
	if v.state.Routes.Presets.AdBlock && !enabledRuleSet(available, "geosite-ads") {
		v.add("routes.presets.ad_block", "missing_rule_set", "ad-block preset requires enabled rule set \"geosite-ads\"", "enable the built-in rule set or disable the preset")
	}
}

func enabledRuleSet(available map[string]control.RuleSetDefinition, name string) bool {
	ruleSet, exists := available[name]
	return exists && ruleSet.Enabled
}

func validRemoteURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func (v *stateValidator) validateInbounds() {
	ids := make(map[string]int, len(v.state.Inbounds))
	addresses := make(map[string]int, len(v.state.Inbounds))
	enabledTun := 0
	for index, inbound := range v.state.Inbounds {
		path := fmt.Sprintf("inbounds[%d]", index)
		if !validStableID(inbound.ID) {
			v.add(path+".id", "invalid_id", "inbound ID is required and may contain letters, digits, dot, underscore, and dash", "use a stable identifier such as mixed or tun")
		} else if previous, exists := ids[inbound.ID]; exists {
			v.add(path+".id", "duplicate_id", fmt.Sprintf("inbound ID %q is also used by inbounds[%d]", inbound.ID, previous), "use a unique inbound ID")
		} else {
			ids[inbound.ID] = index
		}
		if inbound.Name == "" {
			v.add(path+".name", "missing_name", "inbound name is required", "give the inbound a user-facing name")
		}
		switch inbound.Kind {
		case control.InboundKindMixed:
			if net.ParseIP(inbound.Listen.Host) == nil {
				v.add(path+".listen.host", "invalid_listen_host", fmt.Sprintf("listen host %q is not a valid IP address", inbound.Listen.Host), "use :: for dual-stack or 127.0.0.1 for local-only access")
			}
			if inbound.Listen.Port == 0 {
				v.add(path+".listen.port", "invalid_listen_port", "listen port must be between 1 and 65535", "use 1080 unless another port is required")
			}
			if inbound.Auth != nil && (strings.TrimSpace(inbound.Auth.Username) == "" || inbound.Auth.Password == "") {
				v.add(path+".auth", "invalid_auth", "both username and password are required when authentication is enabled", "remove auth or provide both credentials")
			}
			if inbound.Enabled {
				address := net.JoinHostPort(inbound.Listen.Host, fmt.Sprint(inbound.Listen.Port))
				if previous, exists := addresses[address]; exists {
					v.add(path+".listen", "duplicate_listen", fmt.Sprintf("listen address %s is also used by inbounds[%d]", address, previous), "choose a different listen address or port")
				} else {
					addresses[address] = index
				}
			}
		case control.InboundKindTun:
			if inbound.Enabled {
				enabledTun++
			}
			for addressIndex, address := range inbound.Tun.Addresses {
				if _, err := netip.ParsePrefix(address); err != nil {
					v.add(fmt.Sprintf("%s.tun.addresses[%d]", path, addressIndex), "invalid_tun_address", fmt.Sprintf("TUN address %q is not a CIDR prefix", address), "use an address such as 172.19.0.1/30")
				}
			}
			if inbound.Tun.MTU < 0 || inbound.Tun.MTU > 65535 {
				v.add(path+".tun.mtu", "invalid_mtu", "TUN MTU must be between 0 and 65535", "use the default MTU unless the platform requires otherwise")
			}
			switch inbound.Tun.Stack {
			case "", "mixed", "system", "gvisor":
			default:
				v.add(path+".tun.stack", "invalid_tun_stack", fmt.Sprintf("unsupported TUN stack %q", inbound.Tun.Stack), "use mixed, system, or gvisor")
			}
		default:
			v.add(path+".kind", "invalid_inbound_kind", fmt.Sprintf("unsupported inbound kind %q", inbound.Kind), "first version supports mixed and tun")
		}
		if inbound.Egress != nil {
			v.validateEgressRef(path+".egress", *inbound.Egress, false, true)
		}
	}
	if enabledTun > 1 {
		v.add("inbounds", "duplicate_tun", "more than one TUN inbound is enabled", "enable at most one TUN inbound")
	}
}

func (v *stateValidator) validateNodes() {
	ids := make(map[string]int, len(v.state.Nodes))
	for index, node := range v.state.Nodes {
		path := fmt.Sprintf("nodes[%d]", index)
		if !validStableID(node.ID) {
			v.add(path+".id", "invalid_id", "node ID is required and may contain letters, digits, dot, underscore, and dash", "use a stable node identifier")
		} else if previous, exists := ids[node.ID]; exists {
			v.add(path+".id", "duplicate_id", fmt.Sprintf("node ID %q is also used by nodes[%d]", node.ID, previous), "use a unique node ID")
		} else {
			ids[node.ID] = index
		}
		if node.Name == "" {
			v.add(path+".name", "missing_name", "node name is required", "give the node a user-facing name")
		}
		if node.Protocol == "" {
			v.add(path+".protocol", "missing_protocol", "node protocol is required", "use the sing-box outbound type such as shadowsocks, trojan, hysteria2, or vless")
		}
		if node.Server == "" {
			v.add(path+".server", "missing_server", "node server is required", "provide the proxy server hostname or IP address")
		}
		if node.Port == 0 {
			v.add(path+".port", "invalid_port", "node port must be between 1 and 65535", "provide the proxy server port")
		}
		if node.SubscriptionID != "" && !v.subscriptionExists(node.SubscriptionID) {
			v.add(path+".subscription_id", "missing_subscription", fmt.Sprintf("subscription %q does not exist", node.SubscriptionID), "remove the subscription reference or create the subscription first")
		}
	}
	for index, node := range v.state.Nodes {
		for key := range node.Options {
			switch key {
			case "type", "tag", "server", "server_port", "detour":
				v.add(fmt.Sprintf("nodes[%d].options.%s", index, key), "reserved_option", fmt.Sprintf("option %q is managed by boxd", key), "remove the reserved key from node options")
			}
		}
	}
}

func (v *stateValidator) validateGroups() {
	ids := make(map[string]int, len(v.state.Groups))
	for index, group := range v.state.Groups {
		path := fmt.Sprintf("groups[%d]", index)
		if !validStableID(group.ID) {
			v.add(path+".id", "invalid_id", "group ID is required and may contain letters, digits, dot, underscore, and dash", "use a stable group identifier")
		} else if previous, exists := ids[group.ID]; exists {
			v.add(path+".id", "duplicate_id", fmt.Sprintf("group ID %q is also used by groups[%d]", group.ID, previous), "use a unique group ID")
		} else {
			ids[group.ID] = index
		}
		if group.Name == "" {
			v.add(path+".name", "missing_name", "group name is required", "give the group a user-facing name")
		}
		switch group.Kind {
		case control.GroupKindSelector, control.GroupKindURLTest:
		default:
			v.add(path+".kind", "invalid_group_kind", fmt.Sprintf("unsupported group kind %q", group.Kind), "use selector or urltest")
		}
		if len(group.Members) == 0 {
			v.add(path+".members", "empty_group", "group must contain at least one node", "add one or more nodes to the group")
		}
		members := make(map[string]struct{}, len(group.Members))
		for memberIndex, memberID := range group.Members {
			if !v.nodeExists(memberID) {
				v.add(fmt.Sprintf("%s.members[%d]", path, memberIndex), "missing_node", fmt.Sprintf("node %q does not exist", memberID), "remove the missing node or create it first")
			}
			if _, exists := members[memberID]; exists {
				v.add(fmt.Sprintf("%s.members[%d]", path, memberIndex), "duplicate_member", fmt.Sprintf("node %q is listed more than once", memberID), "remove the duplicate member")
			}
			members[memberID] = struct{}{}
		}
		if group.Selected != "" {
			if _, exists := members[group.Selected]; !exists {
				v.add(path+".selected", "invalid_selected_member", fmt.Sprintf("selected node %q is not a group member", group.Selected), "select one of the group members")
			}
		}
	}
}

func (v *stateValidator) validateChains() {
	ids := make(map[string]int, len(v.state.Chains))
	for index, chain := range v.state.Chains {
		path := fmt.Sprintf("chains[%d]", index)
		if !validStableID(chain.ID) {
			v.add(path+".id", "invalid_id", "chain ID is required and may contain letters, digits, dot, underscore, and dash", "use a stable chain identifier")
		} else if previous, exists := ids[chain.ID]; exists {
			v.add(path+".id", "duplicate_id", fmt.Sprintf("chain ID %q is also used by chains[%d]", chain.ID, previous), "use a unique chain ID")
		} else {
			ids[chain.ID] = index
		}
		if chain.Name == "" {
			v.add(path+".name", "missing_name", "chain name is required", "give the chain a user-facing name")
		}
		if len(chain.Hops) == 0 {
			v.add(path+".hops", "empty_chain", "proxy chain must contain at least one hop", "add a node or group to the chain")
		}
		if len(chain.Hops) > 4 {
			v.add(path+".hops", "chain_too_long", "proxy chain may contain at most 4 hops", "remove hops until the chain has at most 4")
		}
		seen := make(map[string]struct{}, len(chain.Hops))
		for hopIndex, hop := range chain.Hops {
			hopPath := fmt.Sprintf("%s.hops[%d]", path, hopIndex)
			switch hop.Kind {
			case control.EgressKindNode:
				if !v.nodeExists(hop.ID) {
					v.add(hopPath, "missing_node", fmt.Sprintf("node %q does not exist", hop.ID), "create the node or choose another hop")
				}
			case control.EgressKindGroup:
				if !v.groupExists(hop.ID) {
					v.add(hopPath, "missing_group", fmt.Sprintf("group %q does not exist", hop.ID), "create the group or choose another hop")
				}
			default:
				v.add(hopPath+".kind", "invalid_chain_hop", fmt.Sprintf("chain hop kind %q is not allowed", hop.Kind), "chain hops must reference a node or a group")
			}
			key := string(hop.Kind) + "\x00" + hop.ID
			if _, exists := seen[key]; exists {
				v.add(hopPath, "duplicate_hop", fmt.Sprintf("chain hop %s/%s is repeated", hop.Kind, hop.ID), "remove duplicate hops to avoid loops")
			}
			seen[key] = struct{}{}
		}
	}
}

func (v *stateValidator) validateRoutes() {
	switch v.state.Routes.Mode {
	case control.RouteModeRules, control.RouteModeGlobal, control.RouteModeDirect:
	default:
		v.add("routes.mode", "invalid_route_mode", fmt.Sprintf("unsupported route mode %q", v.state.Routes.Mode), "use rules, global, or direct")
	}
	v.validateEgressRef("routes.default_egress", v.state.Routes.DefaultEgress, true, false)
	if v.state.Routes.Mode == control.RouteModeGlobal && v.state.Routes.DefaultEgress.Kind == control.EgressKindDirect {
		v.add("routes.default_egress", "missing_proxy_egress", "global mode requires a proxy egress", "select a node, group, or chain before enabling global mode")
	}

	ids := make(map[string]int, len(v.state.Routes.Rules))
	for index, rule := range v.state.Routes.Rules {
		path := fmt.Sprintf("routes.rules[%d]", index)
		if !validStableID(rule.ID) {
			v.add(path+".id", "invalid_id", "route rule ID is required and may contain letters, digits, dot, underscore, and dash", "use a stable route rule identifier")
		} else if previous, exists := ids[rule.ID]; exists {
			v.add(path+".id", "duplicate_id", fmt.Sprintf("route rule ID %q is also used by routes.rules[%d]", rule.ID, previous), "use a unique route rule ID")
		} else {
			ids[rule.ID] = index
		}
		if rule.Name == "" {
			v.add(path+".name", "missing_name", "route rule name is required", "give the rule a user-facing name")
		}
		if rule.Disabled {
			continue
		}
		switch rule.Action {
		case control.RouteActionEgress:
			v.validateEgressRef(path+".egress", rule.Egress, false, true)
		case control.RouteActionDirect, control.RouteActionReject:
		default:
			v.add(path+".action", "invalid_route_action", fmt.Sprintf("unsupported route action %q", rule.Action), "use egress, direct, or reject")
		}
		if !matchHasCondition(rule.Match) {
			v.add(path+".match", "empty_route_match", "route rule must contain at least one match condition", "add a domain, IP, port, process, inbound, or rule set condition")
		}
		v.validateMatch(path+".match", rule.Match)
	}
}

func (v *stateValidator) validateMatch(path string, match control.RouteMatch) {
	for index, cidr := range match.IPCIDRs {
		if _, err := netip.ParsePrefix(cidr); err != nil {
			v.add(fmt.Sprintf("%s.ip_cidrs[%d]", path, index), "invalid_ip_cidr", fmt.Sprintf("IP CIDR %q is invalid", cidr), "use a value such as 10.0.0.0/8")
		}
	}
	for index, cidr := range match.SourceIPCIDRs {
		if _, err := netip.ParsePrefix(cidr); err != nil {
			v.add(fmt.Sprintf("%s.source_ip_cidrs[%d]", path, index), "invalid_source_ip_cidr", fmt.Sprintf("source IP CIDR %q is invalid", cidr), "use a value such as 192.168.0.0/16")
		}
	}
	for index, inboundID := range match.InboundIDs {
		if !v.inboundExists(inboundID) {
			v.add(fmt.Sprintf("%s.inbound_ids[%d]", path, index), "missing_inbound", fmt.Sprintf("inbound %q does not exist", inboundID), "choose an existing inbound")
		}
	}
	for index, tag := range match.RuleSets {
		if !enabledRuleSet(v.ruleSetIndex(), tag) {
			v.add(fmt.Sprintf("%s.rule_sets[%d]", path, index), "unknown_rule_set", fmt.Sprintf("rule set %q is not installed and enabled", tag), "install and enable the rule set before referencing it")
		}
	}
	for index, network := range match.Networks {
		switch strings.ToLower(network) {
		case "tcp", "udp":
		default:
			v.add(fmt.Sprintf("%s.networks[%d]", path, index), "invalid_network", fmt.Sprintf("network %q is invalid", network), "use tcp or udp")
		}
	}
	for index, portRange := range match.PortRanges {
		if portRange.From == 0 || portRange.To == 0 || portRange.From > portRange.To {
			v.add(fmt.Sprintf("%s.port_ranges[%d]", path, index), "invalid_port_range", "port range must satisfy 1 <= from <= to <= 65535", "correct the port range")
		}
	}
}

func (v *stateValidator) validateDNS() {
	switch v.state.DNS.Strategy {
	case "auto", "prefer_ipv4", "prefer_ipv6":
	default:
		v.add("dns.strategy", "invalid_dns_strategy", fmt.Sprintf("unsupported DNS strategy %q", v.state.DNS.Strategy), "use auto, prefer_ipv4, or prefer_ipv6")
	}
	v.validateDNSServer("dns.domestic", v.state.DNS.Domestic, false)
	v.validateDNSServer("dns.foreign", v.state.DNS.Foreign, false)
	v.validateDNSServer("dns.default", v.state.DNS.Default, true)
	if v.state.DNS.Default.Enabled && v.state.DNS.Default.FollowForeign && !v.state.DNS.Foreign.Enabled {
		v.add("dns.default", "missing_foreign_dns", "default DNS follows the foreign slot, but the foreign slot is disabled", "enable the foreign DNS or configure a default DNS server")
	}
	if !v.state.DNS.Domestic.Enabled && !v.state.DNS.Foreign.Enabled && !(v.state.DNS.Default.Enabled && !v.state.DNS.Default.FollowForeign) {
		v.add("dns", "no_dns_server", "at least one DNS slot must be enabled", "enable domestic, foreign, or default DNS")
	}
}

func (v *stateValidator) validateDNSServer(path string, server control.DNSServerConfig, allowFollow bool) {
	if !server.Enabled {
		return
	}
	if server.FollowForeign {
		if !allowFollow {
			v.add(path+".follow_foreign", "invalid_dns_inheritance", "only the default DNS slot can follow another slot", "disable follow_foreign for domestic and foreign DNS")
		}
		return
	}
	if server.Address == "" {
		v.add(path+".address", "missing_dns_address", "DNS server address is required", "provide a server hostname, IP address, or DoH URL")
	}
	switch server.Protocol {
	case control.DNSProtocolUDP, control.DNSProtocolDoH, control.DNSProtocolDoT:
	default:
		v.add(path+".protocol", "invalid_dns_protocol", fmt.Sprintf("unsupported DNS protocol %q", server.Protocol), "use udp, doh, or dot")
	}
	switch server.Detour {
	case control.DNSDetourDirect, control.DNSDetourRouteDefault:
	case control.DNSDetourEgress:
		if server.Egress == nil {
			v.add(path+".egress", "missing_dns_egress", "DNS egress is required when detour is egress", "select a node, group, or chain")
		} else {
			v.validateEgressRef(path+".egress", *server.Egress, false, true)
		}
	default:
		v.add(path+".detour", "invalid_dns_detour", fmt.Sprintf("unsupported DNS detour %q", server.Detour), "use direct, route_default, or egress")
	}
}

func (v *stateValidator) validateUDPPaths() {
	udpInbound := false
	for _, inbound := range v.state.Inbounds {
		if inbound.Enabled && inbound.Kind == control.InboundKindMixed && inbound.UDP {
			udpInbound = true
			break
		}
	}
	if !udpInbound {
		return
	}
	for chainIndex, chain := range v.state.Chains {
		for hopIndex, hop := range chain.Hops {
			if hop.Kind != control.EgressKindNode {
				continue
			}
			node, exists := v.findNode(hop.ID)
			if exists && !node.NodeSupportsUDP() {
				v.add(fmt.Sprintf("chains[%d].hops[%d]", chainIndex, hopIndex), "udp_unsupported", fmt.Sprintf("node %q disables UDP but a Mixed inbound has UDP enabled", hop.ID), "enable UDP on the node or turn off UDP for the Mixed inbound")
			}
		}
	}
}

func (v *stateValidator) validateEgressRef(path string, ref control.EgressRef, specialAllowed, requireID bool) {
	switch ref.Kind {
	case control.EgressKindDirect, control.EgressKindReject:
		if !specialAllowed {
			v.add(path+".kind", "invalid_egress_kind", fmt.Sprintf("egress kind %q is not allowed here", ref.Kind), "select a node, group, or chain")
		}
		if requireID && ref.ID == "" {
			v.add(path+".id", "missing_egress_id", "egress ID is required", "select a concrete egress")
		}
	case control.EgressKindNode:
		if ref.ID == "" {
			v.add(path+".id", "missing_egress_id", "node ID is required", "select a node")
		} else if !v.nodeExists(ref.ID) {
			v.add(path, "missing_node", fmt.Sprintf("node %q does not exist", ref.ID), "create the node or choose another egress")
		}
	case control.EgressKindGroup:
		if ref.ID == "" {
			v.add(path+".id", "missing_egress_id", "group ID is required", "select a group")
		} else if !v.groupExists(ref.ID) {
			v.add(path, "missing_group", fmt.Sprintf("group %q does not exist", ref.ID), "create the group or choose another egress")
		}
	case control.EgressKindChain:
		if ref.ID == "" {
			v.add(path+".id", "missing_egress_id", "chain ID is required", "select a chain")
		} else if !v.chainExists(ref.ID) {
			v.add(path, "missing_chain", fmt.Sprintf("chain %q does not exist", ref.ID), "create the chain or choose another egress")
		}
	default:
		v.add(path+".kind", "invalid_egress_kind", fmt.Sprintf("unsupported egress kind %q", ref.Kind), "use node, group, chain, direct, or reject")
	}
}

func (v *stateValidator) nodeExists(id string) bool {
	_, exists := v.findNode(id)
	return exists
}

func (v *stateValidator) findNode(id string) (control.Node, bool) {
	for _, node := range v.state.Nodes {
		if node.ID == id {
			return node, true
		}
	}
	return control.Node{}, false
}

func (v *stateValidator) groupExists(id string) bool {
	for _, group := range v.state.Groups {
		if group.ID == id {
			return true
		}
	}
	return false
}

func (v *stateValidator) chainExists(id string) bool {
	for _, chain := range v.state.Chains {
		if chain.ID == id {
			return true
		}
	}
	return false
}

func (v *stateValidator) inboundExists(id string) bool {
	for _, inbound := range v.state.Inbounds {
		if inbound.ID == id {
			return true
		}
	}
	return false
}

func (v *stateValidator) subscriptionExists(id string) bool {
	for _, subscription := range v.state.Subscriptions {
		if subscription.ID == id {
			return true
		}
	}
	return false
}

func (v *stateValidator) ruleSetIndex() map[string]control.RuleSetDefinition {
	result := make(map[string]control.RuleSetDefinition, len(v.state.RuleSets))
	for _, ruleSet := range v.state.RuleSets {
		result[ruleSet.Name] = ruleSet
	}
	return result
}

func matchHasCondition(match control.RouteMatch) bool {
	return len(match.InboundIDs) > 0 || len(match.Domains) > 0 || len(match.DomainSuffixes) > 0 ||
		len(match.DomainKeywords) > 0 || len(match.IPCIDRs) > 0 || len(match.SourceIPCIDRs) > 0 ||
		len(match.ProcessNames) > 0 || len(match.ProcessPaths) > 0 || len(match.RuleSets) > 0 ||
		len(match.Networks) > 0 || len(match.Ports) > 0 || len(match.PortRanges) > 0 ||
		match.PrivateIP || match.SourcePrivateIP
}

func validStableID(id string) bool {
	return stableIDPattern.MatchString(id)
}
