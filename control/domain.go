package control

import "time"

// InboundKind identifies a local client entry point.
type InboundKind string

const (
	InboundKindMixed InboundKind = "mixed"
	InboundKindTun   InboundKind = "tun"
)

// EgressKind identifies a routeable proxy path.
type EgressKind string

const (
	EgressKindNode   EgressKind = "node"
	EgressKindGroup  EgressKind = "group"
	EgressKindChain  EgressKind = "chain"
	EgressKindDirect EgressKind = "direct"
	EgressKindReject EgressKind = "reject"
)

// GroupKind identifies the behavior of a proxy group.
type GroupKind string

const (
	GroupKindSelector GroupKind = "selector"
	GroupKindURLTest  GroupKind = "urltest"
)

// RouteMode is a one-click routing policy.
type RouteMode string

const (
	RouteModeRules  RouteMode = "rules"
	RouteModeGlobal RouteMode = "global"
	RouteModeDirect RouteMode = "direct"
)

// RouteAction decides what happens when a route rule matches.
type RouteAction string

const (
	RouteActionEgress RouteAction = "egress"
	RouteActionDirect RouteAction = "direct"
	RouteActionReject RouteAction = "reject"
)

// DNSProtocol is the transport used to reach a DNS server.
type DNSProtocol string

const (
	DNSProtocolUDP DNSProtocol = "udp"
	DNSProtocolDoH DNSProtocol = "doh"
	DNSProtocolDoT DNSProtocol = "dot"
)

// DNSDetour selects how DNS traffic leaves the local host.
type DNSDetour string

const (
	DNSDetourDirect       DNSDetour = "direct"
	DNSDetourRouteDefault DNSDetour = "route_default"
	DNSDetourEgress       DNSDetour = "egress"
)

// DNSProvider is a provider preset shown by clients.
type DNSProvider string

const (
	DNSProviderAliDNS     DNSProvider = "alidns"
	DNSProviderTencent    DNSProvider = "tencent"
	DNSProviderCloudflare DNSProvider = "cloudflare"
	DNSProviderGoogle     DNSProvider = "google"
	DNSProviderQuad9      DNSProvider = "quad9"
	DNSProviderCustom     DNSProvider = "custom"
)

// ListenAddress is the address and port exposed by a mixed inbound.
type ListenAddress struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

// InboundAuth configures optional username/password authentication.
type InboundAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// TunOptions contains the first-version TUN settings exposed by the product.
type TunOptions struct {
	InterfaceName string   `json:"interface_name,omitempty"`
	Addresses     []string `json:"addresses,omitempty"`
	MTU           int      `json:"mtu,omitempty"`
	Stack         string   `json:"stack,omitempty"`
	AutoRoute     bool     `json:"auto_route"`
	StrictRoute   bool     `json:"strict_route"`
}

// InboundProfile is a managed local listener.
type InboundProfile struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Kind    InboundKind   `json:"kind"`
	Enabled bool          `json:"enabled"`
	Listen  ListenAddress `json:"listen,omitempty"`
	UDP     bool          `json:"udp"`
	Auth    *InboundAuth  `json:"auth,omitempty"`
	Egress  *EgressRef    `json:"egress,omitempty"`
	Tun     TunOptions    `json:"tun,omitempty"`
}

// EgressRef references a stable egress identifier. Kind is required because
// nodes, groups, and chains have independent identifier namespaces.
type EgressRef struct {
	Kind EgressKind `json:"kind"`
	ID   string     `json:"id"`
}

// Subscription is a managed remote source of proxy nodes. Download metadata is
// part of the desired state so a refresh and its resulting node replacement
// commit atomically through the same profile transaction.
type Subscription struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	URL             string    `json:"url"`
	IntervalSeconds int       `json:"interval_seconds,omitempty"`
	Enabled         bool      `json:"enabled"`
	LastRefreshedAt time.Time `json:"last_refreshed_at,omitempty"`
	ETag            string    `json:"etag,omitempty"`
	LastModified    string    `json:"last_modified,omitempty"`
}

// Node is a single proxy endpoint. Options contains protocol-specific fields
// such as method, password, uuid, or transport configuration.
type Node struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Protocol       string         `json:"protocol"`
	Server         string         `json:"server"`
	Port           uint16         `json:"port"`
	Options        map[string]any `json:"options,omitempty"`
	SupportsUDP    *bool          `json:"supports_udp,omitempty"`
	SubscriptionID string         `json:"subscription_id,omitempty"`
}

// NodeSupportsUDP reports whether UDP is allowed by default.
func (n Node) NodeSupportsUDP() bool {
	return n.SupportsUDP == nil || *n.SupportsUDP
}

// Group is a selector or URL-test group. Members are node IDs.
type Group struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Kind            GroupKind `json:"kind"`
	Members         []string  `json:"members"`
	Selected        string    `json:"selected,omitempty"`
	URL             string    `json:"url,omitempty"`
	IntervalSeconds int       `json:"interval_seconds,omitempty"`
}

// ProxyHop is one step in a proxy chain.
type ProxyHop struct {
	Kind EgressKind `json:"kind"`
	ID   string     `json:"id"`
}

// ProxyChain is ordered from the local host toward the target.
type ProxyChain struct {
	ID   string     `json:"id"`
	Name string     `json:"name"`
	Hops []ProxyHop `json:"hops"`
}

// EgressCatalog is a consistent snapshot of all egress definitions.
type EgressCatalog struct {
	Nodes  []Node       `json:"nodes"`
	Groups []Group      `json:"groups"`
	Chains []ProxyChain `json:"chains"`
}

// PortRange is an inclusive TCP/UDP port range.
type PortRange struct {
	From uint16 `json:"from"`
	To   uint16 `json:"to"`
}

// RouteMatch describes common first-version match conditions.
type RouteMatch struct {
	InboundIDs      []string    `json:"inbound_ids,omitempty"`
	Domains         []string    `json:"domains,omitempty"`
	DomainSuffixes  []string    `json:"domain_suffixes,omitempty"`
	DomainKeywords  []string    `json:"domain_keywords,omitempty"`
	IPCIDRs         []string    `json:"ip_cidrs,omitempty"`
	SourceIPCIDRs   []string    `json:"source_ip_cidrs,omitempty"`
	ProcessNames    []string    `json:"process_names,omitempty"`
	ProcessPaths    []string    `json:"process_paths,omitempty"`
	RuleSets        []string    `json:"rule_sets,omitempty"`
	Networks        []string    `json:"networks,omitempty"`
	Ports           []uint16    `json:"ports,omitempty"`
	PortRanges      []PortRange `json:"port_ranges,omitempty"`
	PrivateIP       bool        `json:"private_ip,omitempty"`
	SourcePrivateIP bool        `json:"source_private_ip,omitempty"`
}

// RouteRule binds a match to an action. Slice order is evaluation order.
type RouteRule struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Disabled bool        `json:"disabled,omitempty"`
	Match    RouteMatch  `json:"match"`
	Action   RouteAction `json:"action"`
	Egress   EgressRef   `json:"egress,omitempty"`
}

// RoutePresets enables built-in policies without exposing raw rule JSON.
type RoutePresets struct {
	BypassLAN   bool `json:"bypass_lan"`
	ChinaDirect bool `json:"china_direct"`
	AdBlock     bool `json:"ad_block"`
}

// RouteProfile is the complete routing intent.
type RouteProfile struct {
	Mode          RouteMode    `json:"mode"`
	DefaultEgress EgressRef    `json:"default_egress"`
	Presets       RoutePresets `json:"presets"`
	Rules         []RouteRule  `json:"rules"`
}

// RuleSetFormat is the on-disk format understood by the engine.
type RuleSetFormat string

const (
	RuleSetFormatBinary RuleSetFormat = "binary"
	RuleSetFormatSource RuleSetFormat = "source"
)

// RuleSetDefinition is the install-time identity of one rule set. It is part
// of the desired state because route rules reference it by stable name.
type RuleSetDefinition struct {
	Name    string        `json:"name"`
	URL     string        `json:"url"`
	Format  RuleSetFormat `json:"format"`
	Enabled bool          `json:"enabled"`
	Builtin bool          `json:"builtin,omitempty"`
}

// RuleSetStatus is refresh metadata that does not affect engine configuration.
type RuleSetStatus struct {
	Name         string    `json:"name"`
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"last_modified,omitempty"`
	SHA256       string    `json:"sha256,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
	Error        string    `json:"error,omitempty"`
}

// RuleSet combines its managed definition with the last refresh status.
type RuleSet struct {
	Name    string        `json:"name"`
	URL     string        `json:"url"`
	Format  RuleSetFormat `json:"format"`
	Enabled bool          `json:"enabled"`
	Builtin bool          `json:"builtin,omitempty"`
	Status  RuleSetStatus `json:"status"`
}

// DNSServerConfig is one of the three fixed DNS slots.
type DNSServerConfig struct {
	Enabled        bool        `json:"enabled"`
	Provider       DNSProvider `json:"provider"`
	Protocol       DNSProtocol `json:"protocol"`
	Address        string      `json:"address"`
	Detour         DNSDetour   `json:"detour"`
	Egress         *EgressRef  `json:"egress,omitempty"`
	TimeoutSeconds int         `json:"timeout_seconds"`
	FollowForeign  bool        `json:"follow_foreign,omitempty"`
}

// DNSProfile intentionally exposes only domestic, foreign, and fallback slots.
type DNSProfile struct {
	Domestic DNSServerConfig `json:"domestic"`
	Foreign  DNSServerConfig `json:"foreign"`
	Default  DNSServerConfig `json:"default"`
	Strategy string          `json:"strategy"`
	Cache    bool            `json:"cache"`
}

// DesiredState is the managed control-plane source of truth.
type DesiredState struct {
	Revision      string              `json:"revision"`
	UpdatedAt     time.Time           `json:"updated_at"`
	Inbounds      []InboundProfile    `json:"inbounds"`
	Subscriptions []Subscription      `json:"subscriptions,omitempty"`
	Nodes         []Node              `json:"nodes"`
	Groups        []Group             `json:"groups"`
	Chains        []ProxyChain        `json:"chains"`
	Routes        RouteProfile        `json:"routes"`
	DNS           DNSProfile          `json:"dns"`
	RuleSets      []RuleSetDefinition `json:"rule_sets,omitempty"`
}

// NodeTag returns the stable engine-facing tag for a node.
func NodeTag(id string) string { return "node-" + id }

// GroupTag returns the stable engine-facing tag for a group.
func GroupTag(id string) string { return "group-" + id }

// ChainTag returns the stable engine-facing tag for a chain.
func ChainTag(id string) string { return "chain-" + id }

// InboundTag returns the stable engine-facing tag for an inbound.
func InboundTag(id string) string { return "inbound-" + id }

// DefaultInboundProfiles returns the first-run Mixed and TUN entries.
func DefaultInboundProfiles() []InboundProfile {
	return []InboundProfile{
		{
			ID:      "mixed",
			Name:    "Mixed",
			Kind:    InboundKindMixed,
			Enabled: true,
			Listen:  ListenAddress{Host: "::", Port: 1080},
			UDP:     true,
		},
		{
			ID:      "tun",
			Name:    "TUN",
			Kind:    InboundKindTun,
			Enabled: false,
			Tun: TunOptions{
				Addresses: []string{"172.19.0.1/30", "fdfe:dcba:9876::1/126"},
				MTU:       9000,
				Stack:     "mixed",
				AutoRoute: true,
			},
		},
	}
}

// DefaultRouteProfile keeps unmatched traffic direct until the user selects a
// proxy egress. Common LAN and China-direct presets are on by default.
func DefaultRouteProfile() RouteProfile {
	return RouteProfile{
		Mode:          RouteModeRules,
		DefaultEgress: EgressRef{Kind: EgressKindDirect},
		Presets: RoutePresets{
			BypassLAN:   true,
			ChinaDirect: true,
			AdBlock:     false,
		},
	}
}

// DefaultRuleSetDefinitions returns the built-in rule sets used by routing
// presets. They are ordinary definitions so custom rule sets use the same
// validation, refresh, and compilation path.
func DefaultRuleSetDefinitions() []RuleSetDefinition {
	return []RuleSetDefinition{
		{
			Name:    "geosite-cn",
			URL:     "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs",
			Format:  RuleSetFormatBinary,
			Enabled: true,
			Builtin: true,
		},
		{
			Name:    "geoip-cn",
			URL:     "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs",
			Format:  RuleSetFormatBinary,
			Enabled: true,
			Builtin: true,
		},
		{
			Name:    "geosite-ads",
			URL:     "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs",
			Format:  RuleSetFormatBinary,
			Enabled: true,
			Builtin: true,
		},
	}
}

// DefaultDNSProfile implements the agreed domestic/foreign/default slots.
func DefaultDNSProfile() DNSProfile {
	return DNSProfile{
		Domestic: DNSServerConfig{
			Enabled:        true,
			Provider:       DNSProviderAliDNS,
			Protocol:       DNSProtocolDoH,
			Address:        "https://dns.alidns.com/dns-query",
			Detour:         DNSDetourDirect,
			TimeoutSeconds: 5,
		},
		Foreign: DNSServerConfig{
			Enabled:        true,
			Provider:       DNSProviderCloudflare,
			Protocol:       DNSProtocolDoH,
			Address:        "https://1.1.1.1/dns-query",
			Detour:         DNSDetourRouteDefault,
			TimeoutSeconds: 5,
		},
		Default: DNSServerConfig{
			Enabled:        true,
			Protocol:       DNSProtocolDoH,
			Detour:         DNSDetourRouteDefault,
			TimeoutSeconds: 5,
			FollowForeign:  true,
		},
		Strategy: "prefer_ipv4",
		Cache:    true,
	}
}

// DefaultDesiredState is the idempotent first-run profile.
func DefaultDesiredState() DesiredState {
	return DesiredState{
		Inbounds: DefaultInboundProfiles(),
		Routes:   DefaultRouteProfile(),
		DNS:      DefaultDNSProfile(),
		RuleSets: DefaultRuleSetDefinitions(),
	}
}
