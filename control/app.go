// Package control defines the public control-plane application boundary.
package control

import (
	"context"
	"time"

	"github.com/xuthus5/boxd/kernel"
)

// Application is the single facade consumed by HTTP, Wails, MyGo, and CLI adapters.
type Application interface {
	Kernel() KernelAPI
	Config() ConfigAPI
	Setup() SetupAPI
	Inbounds() InboundAPI
	Egress() EgressAPI
	Subscriptions() SubscriptionAPI
	Nodes() NodeAPI
	Policies() PolicyAPI
	Routes() RouteAPI
	DNS() DNSAPI
	RuleSets() RuleSetAPI
	Settings() SettingsAPI
	Observability() ObservabilityAPI
	Backup() BackupAPI
	Security() SecurityAPI
}

// KernelAPI exposes lifecycle commands and a status query.
type KernelAPI interface {
	Status(ctx context.Context) (kernel.Status, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error
}

// ConfigAPI exposes opaque configuration operations.
type ConfigAPI interface {
	Read(ctx context.Context) (kernel.ConfigDocument, error)
	Validate(ctx context.Context, document kernel.ConfigDocument) (kernel.ValidationResult, error)
	Apply(ctx context.Context, document kernel.ConfigDocument, options kernel.ApplyOptions) (kernel.ApplyResult, error)
	Reconcile(ctx context.Context, options kernel.ApplyOptions) (kernel.ApplyResult, error)
	History(ctx context.Context, limit int) ([]kernel.ConfigRevision, error)
}

// SetupAPI creates idempotent first-run state and reports engine capabilities.
type SetupAPI interface {
	EnsureDefaults(ctx context.Context) (DesiredState, error)
	Capabilities(ctx context.Context) (kernel.EngineDescriptor, error)
}

// InboundAPI manages the first-version Mixed and TUN entries.
type InboundAPI interface {
	List(ctx context.Context) ([]InboundProfile, error)
	Get(ctx context.Context, id string) (InboundProfile, error)
	Save(ctx context.Context, profile InboundProfile) (InboundProfile, error)
	Delete(ctx context.Context, id string) error
}

// EgressAPI manages nodes, groups, chains, and runtime selector state.
type EgressAPI interface {
	Catalog(ctx context.Context) (EgressCatalog, error)
	SaveNode(ctx context.Context, node Node) (Node, error)
	DeleteNode(ctx context.Context, id string) error
	SaveGroup(ctx context.Context, group Group) (Group, error)
	DeleteGroup(ctx context.Context, id string) error
	SaveChain(ctx context.Context, chain ProxyChain) (ProxyChain, error)
	DeleteChain(ctx context.Context, id string) error
	SelectGroupMember(ctx context.Context, groupID, nodeID string) error
}

// RouteAPI owns one-click modes, presets, and ordered custom rules.
type RouteAPI interface {
	Get(ctx context.Context) (RouteProfile, error)
	SetMode(ctx context.Context, mode RouteMode) (RouteProfile, error)
	SetDefaultEgress(ctx context.Context, egress EgressRef) (RouteProfile, error)
	SetPresets(ctx context.Context, presets RoutePresets) (RouteProfile, error)
	SaveRule(ctx context.Context, rule RouteRule) (RouteRule, error)
	DeleteRule(ctx context.Context, id string) error
	ReplaceRules(ctx context.Context, rules []RouteRule) ([]RouteRule, error)
}

// DNSAPI owns the fixed three-slot DNS profile.
type DNSAPI interface {
	Get(ctx context.Context) (DNSProfile, error)
	Update(ctx context.Context, profile DNSProfile) (DNSProfile, error)
	Flush(ctx context.Context) error
}

// SettingsAPI stores transport-neutral application preferences.
type SettingsAPI interface {
	List(ctx context.Context) ([]Setting, error)
	Get(ctx context.Context, key string) (Setting, bool, error)
	Set(ctx context.Context, key, value string) (Setting, error)
}

// ObservabilityAPI exposes logs, traffic, connections, and live events.
type ObservabilityAPI interface {
	RecentLogs(limit int) []kernel.LogEntry
	Traffic(ctx context.Context) (kernel.TrafficSnapshot, error)
	Connections(ctx context.Context) ([]kernel.Connection, error)
	CloseConnection(ctx context.Context, id int64) error
	CloseConnections(ctx context.Context, filter kernel.ConnectionFilter) (int, error)
	Subscribe(ctx context.Context, types ...kernel.EventType) (<-chan kernel.Event, error)
}

// Setting is a single versioned preference value.
type Setting struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}
