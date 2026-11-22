// Package singboxconfig compiles boxd's engine-neutral desired state into
// sing-box JSON. It deliberately does not import sing-box itself; the driver
// remains the only package that parses engine types.
package singboxconfig

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

const (
	directTag = "direct"
	rejectTag = "reject"

	dnsDomesticTag  = "dns-domestic"
	dnsForeignTag   = "dns-foreign"
	dnsDefaultTag   = "dns-default"
	dnsBootstrapTag = "dns-bootstrap"

	defaultURLTestURL      = "https://www.gstatic.com/generate_204"
	defaultURLTestInterval = 180
)

var domesticDomainSuffixes = []string{
	".cn",
	".中国",
	".公司",
	".网络",
}

// Compiler implements control.ConfigCompiler.
type Compiler struct{}

var _ control.ConfigCompiler = (*Compiler)(nil)

// New constructs a stateless compiler.
func New() *Compiler { return &Compiler{} }

// Compile renders the managed profile into a stable sing-box JSON document.
func (c *Compiler) Compile(ctx context.Context, state control.DesiredState) (kernel.ConfigDocument, error) {
	if err := ctx.Err(); err != nil {
		return kernel.ConfigDocument{}, err
	}
	compiled, err := compileState(state)
	if err != nil {
		return kernel.ConfigDocument{}, err
	}
	body, err := json.MarshalIndent(compiled, "", "  ")
	if err != nil {
		return kernel.ConfigDocument{}, kernel.WrapError(kernel.ErrCodeInternal, "config.compile", "marshal sing-box configuration", err)
	}
	digest := sha256.Sum256(body)
	return kernel.ConfigDocument{
		Format:   "json",
		Body:     body,
		Revision: "config:" + hex.EncodeToString(digest[:]),
	}, nil
}

func compileState(state control.DesiredState) (map[string]any, error) {
	index, err := buildIndex(state)
	if err != nil {
		return nil, err
	}
	outbounds, err := buildOutbounds(state, index)
	if err != nil {
		return nil, err
	}
	inbounds, err := buildInbounds(state)
	if err != nil {
		return nil, err
	}
	dnsOptions, _, err := buildDNS(state.DNS, refTag(state.Routes.DefaultEgress))
	if err != nil {
		return nil, err
	}
	routeOptions, err := buildRoute(state)
	if err != nil {
		return nil, err
	}
	routeOptions["default_domain_resolver"] = dnsBootstrapTag
	return map[string]any{
		"log": map[string]any{
			"level":     "info",
			"timestamp": true,
		},
		"dns":       dnsOptions,
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"route":     routeOptions,
	}, nil
}

type stateIndex struct {
	nodes  map[string]control.Node
	groups map[string]control.Group
	chains map[string]control.ProxyChain
}

func buildIndex(state control.DesiredState) (stateIndex, error) {
	index := stateIndex{
		nodes:  make(map[string]control.Node, len(state.Nodes)),
		groups: make(map[string]control.Group, len(state.Groups)),
		chains: make(map[string]control.ProxyChain, len(state.Chains)),
	}
	for _, node := range state.Nodes {
		index.nodes[node.ID] = node
	}
	for _, group := range state.Groups {
		index.groups[group.ID] = group
	}
	for _, chain := range state.Chains {
		index.chains[chain.ID] = chain
	}
	return index, nil
}

func buildInbounds(state control.DesiredState) ([]map[string]any, error) {
	result := make([]map[string]any, 0, len(state.Inbounds))
	for _, inbound := range state.Inbounds {
		if !inbound.Enabled {
			continue
		}
		tag := control.InboundTag(inbound.ID)
		switch inbound.Kind {
		case control.InboundKindMixed:
			item := map[string]any{
				"type":        "mixed",
				"tag":         tag,
				"listen":      inbound.Listen.Host,
				"listen_port": inbound.Listen.Port,
			}
			if inbound.Auth != nil {
				item["users"] = []map[string]any{{"username": inbound.Auth.Username, "password": inbound.Auth.Password}}
			}
			result = append(result, item)
		case control.InboundKindTun:
			item := map[string]any{
				"type":         "tun",
				"tag":          tag,
				"mtu":          inbound.Tun.MTU,
				"auto_route":   inbound.Tun.AutoRoute,
				"strict_route": inbound.Tun.StrictRoute,
			}
			if inbound.Tun.InterfaceName != "" {
				item["interface_name"] = inbound.Tun.InterfaceName
			}
			if len(inbound.Tun.Addresses) > 0 {
				item["address"] = inbound.Tun.Addresses
			}
			if inbound.Tun.Stack != "" {
				item["stack"] = inbound.Tun.Stack
			}
			result = append(result, item)
		default:
			return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", fmt.Sprintf("unsupported inbound kind %q", inbound.Kind))
		}
	}
	return result, nil
}

func buildOutbounds(state control.DesiredState, index stateIndex) ([]map[string]any, error) {
	result := []map[string]any{
		{"type": "direct", "tag": directTag},
		{"type": "block", "tag": rejectTag},
	}
	for _, node := range state.Nodes {
		outbound, err := nodeOutbound(node, control.NodeTag(node.ID), "")
		if err != nil {
			return nil, err
		}
		result = append(result, outbound)
	}
	for _, group := range state.Groups {
		outbound, err := groupOutbound(group, control.GroupTag(group.ID), nodeTags(group.Members))
		if err != nil {
			return nil, err
		}
		result = append(result, outbound)
	}
	for _, chain := range state.Chains {
		chainOutbounds, err := chainOutbounds(chain, index)
		if err != nil {
			return nil, err
		}
		result = append(result, chainOutbounds...)
	}
	return result, nil
}

func nodeOutbound(node control.Node, tag, detour string) (map[string]any, error) {
	if node.Protocol == "" || node.Server == "" || node.Port == 0 {
		return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", fmt.Sprintf("node %q is incomplete", node.ID))
	}
	result := make(map[string]any, len(node.Options)+5)
	for key, value := range node.Options {
		result[key] = value
	}
	result["type"] = node.Protocol
	result["tag"] = tag
	result["server"] = node.Server
	result["server_port"] = node.Port
	if detour != "" {
		result["detour"] = detour
	}
	return result, nil
}

func groupOutbound(group control.Group, tag string, members []string) (map[string]any, error) {
	selectedTag := ""
	if group.Selected != "" {
		selectedTag = control.NodeTag(group.Selected)
	}
	return groupOutboundWithTags(group, tag, members, selectedTag)
}

func chainOutbounds(chain control.ProxyChain, index stateIndex) ([]map[string]any, error) {
	result := make([]map[string]any, 0, len(chain.Hops)*2)
	previous := ""
	for hopIndex, hop := range chain.Hops {
		last := hopIndex == len(chain.Hops)-1
		hopTag := chainHopTag(chain.ID, hopIndex, last)
		switch hop.Kind {
		case control.EgressKindNode:
			node, exists := index.nodes[hop.ID]
			if !exists {
				return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", fmt.Sprintf("chain %q references missing node %q", chain.ID, hop.ID))
			}
			outbound, err := nodeOutbound(node, hopTag, previous)
			if err != nil {
				return nil, err
			}
			result = append(result, outbound)
		case control.EgressKindGroup:
			group, exists := index.groups[hop.ID]
			if !exists {
				return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", fmt.Sprintf("chain %q references missing group %q", chain.ID, hop.ID))
			}
			memberTags := make([]string, 0, len(group.Members))
			selectedTag := ""
			for _, memberID := range group.Members {
				node, found := index.nodes[memberID]
				if !found {
					return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", fmt.Sprintf("group %q references missing node %q", group.ID, memberID))
				}
				memberTag := hopTag + "-" + control.NodeTag(memberID)
				outbound, err := nodeOutbound(node, memberTag, previous)
				if err != nil {
					return nil, err
				}
				result = append(result, outbound)
				memberTags = append(memberTags, memberTag)
				if group.Selected == memberID {
					selectedTag = memberTag
				}
			}
			outbound, err := groupOutboundWithTags(group, hopTag, memberTags, selectedTag)
			if err != nil {
				return nil, err
			}
			result = append(result, outbound)
		default:
			return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", fmt.Sprintf("chain %q has invalid hop kind %q", chain.ID, hop.Kind))
		}
		previous = hopTag
	}
	return result, nil
}

func groupOutboundWithTags(group control.Group, tag string, members []string, selectedTag string) (map[string]any, error) {
	if len(members) == 0 {
		return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", fmt.Sprintf("group %q has no members", group.ID))
	}
	result := map[string]any{
		"type":      string(group.Kind),
		"tag":       tag,
		"outbounds": members,
	}
	switch group.Kind {
	case control.GroupKindSelector:
		if selectedTag == "" {
			selectedTag = members[0]
		}
		result["default"] = selectedTag
	case control.GroupKindURLTest:
		url := group.URL
		if url == "" {
			url = defaultURLTestURL
		}
		interval := group.IntervalSeconds
		if interval <= 0 {
			interval = defaultURLTestInterval
		}
		result["url"] = url
		result["interval"] = strconv.Itoa(interval) + "s"
	default:
		return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", fmt.Sprintf("group %q has unsupported kind %q", group.ID, group.Kind))
	}
	return result, nil
}

func chainHopTag(chainID string, index int, last bool) string {
	if last {
		return control.ChainTag(chainID)
	}
	return control.ChainTag(chainID) + "-hop-" + strconv.Itoa(index+1)
}

func buildDNS(profile control.DNSProfile, routeDefaultTag string) (map[string]any, string, error) {
	servers := []map[string]any{{"type": "local", "tag": dnsBootstrapTag}}
	if profile.Domestic.Enabled {
		server, err := dnsServer("domestic", dnsDomesticTag, profile.Domestic, routeDefaultTag)
		if err != nil {
			return nil, "", err
		}
		servers = append(servers, server)
	}
	if profile.Foreign.Enabled {
		server, err := dnsServer("foreign", dnsForeignTag, profile.Foreign, routeDefaultTag)
		if err != nil {
			return nil, "", err
		}
		servers = append(servers, server)
	}
	if profile.Default.Enabled && !profile.Default.FollowForeign {
		server, err := dnsServer("default", dnsDefaultTag, profile.Default, routeDefaultTag)
		if err != nil {
			return nil, "", err
		}
		servers = append(servers, server)
	}
	if len(servers) == 0 {
		return nil, "", kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", "no DNS server is enabled")
	}

	finalTag := ""
	if profile.Default.Enabled && !profile.Default.FollowForeign {
		finalTag = dnsDefaultTag
	} else if profile.Foreign.Enabled {
		finalTag = dnsForeignTag
	} else if profile.Domestic.Enabled {
		finalTag = dnsDomesticTag
	}
	result := map[string]any{
		"servers":       servers,
		"final":         finalTag,
		"disable_cache": !profile.Cache,
	}
	if profile.Domestic.Enabled {
		result["rules"] = []map[string]any{{
			"domain_suffix": domesticDomainSuffixes,
			"server":        dnsDomesticTag,
		}}
	}
	switch profile.Strategy {
	case "prefer_ipv4", "prefer_ipv6":
		result["strategy"] = profile.Strategy
	case "auto", "":
	default:
		return nil, "", kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", fmt.Sprintf("unsupported DNS strategy %q", profile.Strategy))
	}
	return result, finalTag, nil
}

func dnsServer(slot, tag string, server control.DNSServerConfig, routeDefaultTag string) (map[string]any, error) {
	if server.Address == "" {
		return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", fmt.Sprintf("%s DNS address is empty", slot))
	}
	endpoint, err := parseDNSEndpoint(server.Address, server.Protocol)
	if err != nil {
		return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", fmt.Sprintf("parse %s DNS address: %v", slot, err))
	}
	result := map[string]any{
		"tag":    tag,
		"type":   endpoint.transportType,
		"server": endpoint.host,
	}
	if endpoint.port != 0 {
		result["server_port"] = endpoint.port
	}
	if endpoint.path != "" {
		result["path"] = endpoint.path
	}
	if net.ParseIP(endpoint.host) == nil {
		result["domain_resolver"] = dnsBootstrapTag
	}
	switch server.Detour {
	case control.DNSDetourDirect:
	case control.DNSDetourRouteDefault:
		if routeDefaultTag != "" && routeDefaultTag != directTag && routeDefaultTag != rejectTag {
			result["detour"] = routeDefaultTag
		}
	case control.DNSDetourEgress:
		if server.Egress == nil {
			return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", fmt.Sprintf("%s DNS egress is empty", slot))
		}
		egressTag := refTag(*server.Egress)
		if egressTag == directTag || egressTag == rejectTag || egressTag == "" {
			return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", fmt.Sprintf("%s DNS egress must be a node, group, or chain", slot))
		}
		result["detour"] = egressTag
	default:
		return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", fmt.Sprintf("unsupported %s DNS detour %q", slot, server.Detour))
	}
	return result, nil
}

type dnsEndpoint struct {
	transportType string
	host          string
	port          uint16
	path          string
}

func parseDNSEndpoint(address string, protocol control.DNSProtocol) (dnsEndpoint, error) {
	switch protocol {
	case control.DNSProtocolDoH:
		parsed, err := parseURLWithDefaultScheme(address, "https")
		if err != nil {
			return dnsEndpoint{}, err
		}
		endpoint := dnsEndpoint{transportType: "https", host: parsed.Hostname(), path: parsed.EscapedPath()}
		if port := parsed.Port(); port != "" {
			value, err := strconv.ParseUint(port, 10, 16)
			if err != nil {
				return dnsEndpoint{}, err
			}
			endpoint.port = uint16(value)
		}
		return endpoint, nil
	case control.DNSProtocolDoT:
		parsed, err := parseURLWithDefaultScheme(address, "tls")
		if err != nil {
			return dnsEndpoint{}, err
		}
		endpoint := dnsEndpoint{transportType: "tls", host: parsed.Hostname()}
		if port := parsed.Port(); port != "" {
			value, err := strconv.ParseUint(port, 10, 16)
			if err != nil {
				return dnsEndpoint{}, err
			}
			endpoint.port = uint16(value)
		} else {
			endpoint.port = 853
		}
		return endpoint, nil
	case control.DNSProtocolUDP:
		parsed, err := parseURLWithDefaultScheme(address, "udp")
		if err != nil {
			return dnsEndpoint{}, err
		}
		endpoint := dnsEndpoint{transportType: "udp", host: parsed.Hostname()}
		if port := parsed.Port(); port != "" {
			value, err := strconv.ParseUint(port, 10, 16)
			if err != nil {
				return dnsEndpoint{}, err
			}
			endpoint.port = uint16(value)
		} else {
			endpoint.port = 53
		}
		return endpoint, nil
	default:
		return dnsEndpoint{}, fmt.Errorf("unsupported protocol %q", protocol)
	}
}

func parseURLWithDefaultScheme(address, scheme string) (*url.URL, error) {
	if !strings.Contains(address, "://") {
		address = scheme + "://" + address
	}
	parsed, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	if parsed.Hostname() == "" {
		return nil, fmt.Errorf("hostname is missing")
	}
	return parsed, nil
}

func buildRoute(state control.DesiredState) (map[string]any, error) {
	finalTag := refTag(state.Routes.DefaultEgress)
	if state.Routes.Mode == control.RouteModeDirect {
		finalTag = directTag
	}
	result := map[string]any{
		"final":                 finalTag,
		"auto_detect_interface": true,
	}
	rules := []map[string]any{{"action": "sniff"}}
	if state.Routes.Mode != control.RouteModeRules {
		result["rules"] = rules
		return result, nil
	}

	rules = append(make([]map[string]any, 0, len(state.Routes.Rules)+8), rules...)
	for _, inbound := range state.Inbounds {
		if !inbound.Enabled || inbound.Egress == nil {
			continue
		}
		rules = append(rules, map[string]any{
			"inbound":  []string{control.InboundTag(inbound.ID)},
			"action":   "route",
			"outbound": refTag(*inbound.Egress),
		})
	}
	if state.Routes.Presets.BypassLAN {
		rules = append(rules, map[string]any{
			"ip_is_private": true,
			"action":        "route",
			"outbound":      directTag,
		})
	}
	if state.Routes.Presets.ChinaDirect {
		rules = append(rules, map[string]any{
			"rule_set": []string{"geosite-cn", "geoip-cn"},
			"action":   "route",
			"outbound": directTag,
		})
	}
	if state.Routes.Presets.AdBlock {
		rules = append(rules, map[string]any{
			"rule_set": []string{"geosite-ads"},
			"action":   "reject",
		})
	}

	findProcess := false
	for _, rule := range state.Routes.Rules {
		if rule.Disabled {
			continue
		}
		rendered, err := routeRule(rule)
		if err != nil {
			return nil, err
		}
		if len(rule.Match.ProcessNames) > 0 || len(rule.Match.ProcessPaths) > 0 {
			findProcess = true
		}
		rules = append(rules, rendered)
	}
	if len(rules) > 0 {
		result["rules"] = rules
	}
	if findProcess {
		result["find_process"] = true
	}

	ruleSets, err := buildRuleSets(state)
	if err != nil {
		return nil, err
	}
	if len(ruleSets) > 0 {
		result["rule_set"] = ruleSets
	}
	return result, nil
}

func routeRule(rule control.RouteRule) (map[string]any, error) {
	rendered := make(map[string]any, 20)
	match := rule.Match
	if len(match.InboundIDs) > 0 {
		tags := make([]string, 0, len(match.InboundIDs))
		for _, id := range match.InboundIDs {
			tags = append(tags, control.InboundTag(id))
		}
		rendered["inbound"] = tags
	}
	if len(match.Domains) > 0 {
		rendered["domain"] = match.Domains
	}
	if len(match.DomainSuffixes) > 0 {
		rendered["domain_suffix"] = match.DomainSuffixes
	}
	if len(match.DomainKeywords) > 0 {
		rendered["domain_keyword"] = match.DomainKeywords
	}
	if len(match.IPCIDRs) > 0 {
		rendered["ip_cidr"] = match.IPCIDRs
	}
	if len(match.SourceIPCIDRs) > 0 {
		rendered["source_ip_cidr"] = match.SourceIPCIDRs
	}
	if len(match.ProcessNames) > 0 {
		rendered["process_name"] = match.ProcessNames
	}
	if len(match.ProcessPaths) > 0 {
		rendered["process_path"] = match.ProcessPaths
	}
	if len(match.RuleSets) > 0 {
		rendered["rule_set"] = match.RuleSets
	}
	if len(match.Networks) > 0 {
		networks := make([]string, 0, len(match.Networks))
		for _, network := range match.Networks {
			networks = append(networks, strings.ToLower(network))
		}
		rendered["network"] = networks
	}
	if len(match.Ports) > 0 {
		rendered["port"] = match.Ports
	}
	if len(match.PortRanges) > 0 {
		ranges := make([]string, 0, len(match.PortRanges))
		for _, item := range match.PortRanges {
			ranges = append(ranges, fmt.Sprintf("%d-%d", item.From, item.To))
		}
		rendered["port_range"] = ranges
	}
	if match.PrivateIP {
		rendered["ip_is_private"] = true
	}
	if match.SourcePrivateIP {
		rendered["source_ip_is_private"] = true
	}

	switch rule.Action {
	case control.RouteActionDirect:
		rendered["action"] = "route"
		rendered["outbound"] = directTag
	case control.RouteActionReject:
		rendered["action"] = "reject"
	case control.RouteActionEgress:
		rendered["action"] = "route"
		rendered["outbound"] = refTag(rule.Egress)
	default:
		return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", fmt.Sprintf("route rule %q has unsupported action %q", rule.ID, rule.Action))
	}
	return rendered, nil
}

func buildRuleSets(state control.DesiredState) ([]map[string]any, error) {
	definitions := make(map[string]control.RuleSetDefinition, len(state.RuleSets))
	for _, ruleSet := range state.RuleSets {
		definitions[ruleSet.Name] = ruleSet
	}
	needed := make(map[string]struct{})
	if state.Routes.Presets.ChinaDirect {
		needed["geosite-cn"] = struct{}{}
		needed["geoip-cn"] = struct{}{}
	}
	if state.Routes.Presets.AdBlock {
		needed["geosite-ads"] = struct{}{}
	}
	for _, rule := range state.Routes.Rules {
		if rule.Disabled {
			continue
		}
		for _, tag := range rule.Match.RuleSets {
			needed[tag] = struct{}{}
		}
	}
	tags := make([]string, 0, len(needed))
	for tag := range needed {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	result := make([]map[string]any, 0, len(tags))
	for _, tag := range tags {
		definition, exists := definitions[tag]
		if !exists || !definition.Enabled {
			return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "config.compile", fmt.Sprintf("unknown rule set %q", tag))
		}
		result = append(result, map[string]any{
			"type":   "remote",
			"tag":    definition.Name,
			"format": string(definition.Format),
			"url":    definition.URL,
		})
	}
	return result, nil
}

func refTag(ref control.EgressRef) string {
	switch ref.Kind {
	case control.EgressKindDirect:
		return directTag
	case control.EgressKindReject:
		return rejectTag
	case control.EgressKindNode:
		return control.NodeTag(ref.ID)
	case control.EgressKindGroup:
		return control.GroupTag(ref.ID)
	case control.EgressKindChain:
		return control.ChainTag(ref.ID)
	default:
		return ""
	}
}

func nodeTags(ids []string) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		result = append(result, control.NodeTag(id))
	}
	return result
}
