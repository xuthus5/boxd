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
	Scheduler() SchedulerAPI
	Nodes() NodeAPI
	Policies() PolicyAPI
	Routes() RouteAPI
	DNS() DNSAPI
	RuleSets() RuleSetAPI
	Settings() SettingsAPI
	Observability() ObservabilityAPI
	ActivityLogs() ActivityLogAPI
	Audit() AuditAPI
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
	Diagnose(ctx context.Context, document kernel.ConfigDocument) ([]kernel.Diagnostic, error)
	Apply(ctx context.Context, document kernel.ConfigDocument, options kernel.ApplyOptions) (kernel.ApplyResult, error)
	Reconcile(ctx context.Context, options kernel.ApplyOptions) (kernel.ApplyResult, error)
	History(ctx context.Context, limit int) ([]kernel.ConfigRevision, error)
}

// SetupAPI creates idempotent first-run state and reports engine capabilities.
type SetupAPI interface {
	EnsureDefaults(ctx context.Context) (DesiredState, error)
	Capabilities(ctx context.Context) (kernel.EngineDescriptor, error)
}

// InboundPatch is a partial inbound update. Nil fields are unchanged.
type InboundPatch struct {
	Name    *string        `json:"name,omitempty"`
	Enabled *bool          `json:"enabled,omitempty"`
	Listen  *ListenAddress `json:"listen,omitempty"`
	UDP     *bool          `json:"udp,omitempty"`
}

// InboundAPI manages the first-version Mixed and TUN entries.
type InboundAPI interface {
	List(ctx context.Context) ([]InboundProfile, error)
	Get(ctx context.Context, id string) (InboundProfile, error)
	Save(ctx context.Context, profile InboundProfile) (InboundProfile, error)
	Patch(ctx context.Context, id string, patch InboundPatch) (InboundProfile, error)
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

// RouteRulePatch is a partial route-rule update. Nil fields are unchanged.
type RouteRulePatch struct {
	Name     *string      `json:"name,omitempty"`
	Disabled *bool        `json:"disabled,omitempty"`
	Match    *RouteMatch  `json:"match,omitempty"`
	Action   *RouteAction `json:"action,omitempty"`
	Egress   *EgressRef   `json:"egress,omitempty"`
}

// RouteAPI owns one-click modes, presets, and ordered custom rules.
type RouteAPI interface {
	Get(ctx context.Context) (RouteProfile, error)
	SetMode(ctx context.Context, mode RouteMode) (RouteProfile, error)
	SetDefaultEgress(ctx context.Context, egress EgressRef) (RouteProfile, error)
	SetPresets(ctx context.Context, presets RoutePresets) (RouteProfile, error)
	SaveRule(ctx context.Context, rule RouteRule) (RouteRule, error)
	PatchRule(ctx context.Context, id string, patch RouteRulePatch) (RouteRule, error)
	DeleteRule(ctx context.Context, id string) error
	ReplaceRules(ctx context.Context, rules []RouteRule) ([]RouteRule, error)
}

// DNSProfilePatch is a partial DNS profile update. Nil fields are unchanged.
type DNSProfilePatch struct {
	Strategy *string          `json:"strategy,omitempty"`
	Cache    *bool            `json:"cache,omitempty"`
	Domestic *DNSServerConfig `json:"domestic,omitempty"`
	Foreign  *DNSServerConfig `json:"foreign,omitempty"`
	Default  *DNSServerConfig `json:"default,omitempty"`
}

// DNSAPI owns the fixed three-slot DNS profile.
type DNSAPI interface {
	Get(ctx context.Context) (DNSProfile, error)
	Update(ctx context.Context, profile DNSProfile) (DNSProfile, error)
	Patch(ctx context.Context, patch DNSProfilePatch) (DNSProfile, error)
	Flush(ctx context.Context) error
	Test(ctx context.Context) ([]DNSProbeResult, error)
	ProbeResults(ctx context.Context) ([]DNSProbeResult, error)
}

// SettingsAPI stores transport-neutral application preferences.
type SettingsAPI interface {
	List(ctx context.Context) ([]Setting, error)
	Get(ctx context.Context, key string) (Setting, bool, error)
	Set(ctx context.Context, key, value string) (Setting, error)
}

// ObservabilityAPI exposes engine logs, sampled traffic, connections, and live
// events. Sampling is owned by the control plane so every host observes the
// same counters instead of recomputing rates from different request cadences.
type ObservabilityAPI interface {
	Snapshot(ctx context.Context) (ObservationSnapshot, error)
	StartSampling(ctx context.Context) error
	StopSampling()
	RecentLogs(limit int) []kernel.LogEntry
	Traffic(ctx context.Context) (kernel.TrafficSnapshot, error)
	Connections(ctx context.Context) ([]kernel.Connection, error)
	ClosedConnections(ctx context.Context) ([]kernel.Connection, error)
	CloseConnection(ctx context.Context, id int64) error
	CloseConnections(ctx context.Context, filter kernel.ConnectionFilter) (int, error)
	Subscribe(ctx context.Context, types ...kernel.EventType) (<-chan kernel.Event, error)
}

// ObservedTraffic is the sampled cumulative and current rate for one entity.
type ObservedTraffic struct {
	UploadTotal   int64 `json:"upload_total"`
	DownloadTotal int64 `json:"download_total"`
	UploadRate    int64 `json:"upload_rate"`
	DownloadRate  int64 `json:"download_rate"`
}

// Total returns the sampled cumulative bytes.
func (t ObservedTraffic) Total() int64 { return t.UploadTotal + t.DownloadTotal }

// TotalRate returns the sampled bytes per second.
func (t ObservedTraffic) TotalRate() int64 { return t.UploadRate + t.DownloadRate }

// ObservationSnapshot is a transport-neutral view of live per-entity traffic.
// Available is false when the engine is stopped or observation is unsupported.
type ObservationSnapshot struct {
	Available bool                       `json:"available"`
	SampledAt time.Time                  `json:"sampled_at"`
	Nodes     map[string]ObservedTraffic `json:"nodes"`
	Inbounds  map[string]ObservedTraffic `json:"inbounds"`
	RuleHits  map[string]uint64          `json:"rule_hits"`
}

// ActivityLogEntry is a bounded, structured application log record suitable
// for HTTP SSE, a native desktop event stream, or a CLI tail.
type ActivityLogEntry struct {
	ID      uint64         `json:"id"`
	Time    time.Time      `json:"time"`
	Level   string         `json:"level"`
	Source  string         `json:"source"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

// ActivityLogAPI owns the shared application activity log. Host adapters must
// consume this service instead of maintaining private log hubs.
type ActivityLogAPI interface {
	Recent(limit int) []ActivityLogEntry
	Subscribe(ctx context.Context) (<-chan ActivityLogEntry, error)
	Record(ctx context.Context, entry ActivityLogEntry) ActivityLogEntry
}

// Setting is a single versioned preference value.
type Setting struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}
