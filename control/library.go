package control

import (
	"context"
	"time"
)

// SubscriptionRefreshResult reports the atomic outcome of one subscription
// download and node replacement.
type SubscriptionRefreshResult struct {
	Subscription Subscription `json:"subscription"`
	Added        int          `json:"added"`
	Updated      int          `json:"updated"`
	Removed      int          `json:"removed"`
	NotModified  bool         `json:"not_modified"`
}

// SubscriptionAPI manages remote node sources and their download metadata.
type SubscriptionAPI interface {
	List(ctx context.Context) ([]Subscription, error)
	Get(ctx context.Context, id string) (Subscription, error)
	Save(ctx context.Context, subscription Subscription) (Subscription, error)
	Delete(ctx context.Context, id string) error
	Refresh(ctx context.Context, id string) (SubscriptionRefreshResult, error)
	RefreshAll(ctx context.Context) ([]SubscriptionRefreshResult, error)
}

// NodeImportResult reports how many nodes were inserted or updated in one
// atomic import.
type NodeImportResult struct {
	Added   int    `json:"added"`
	Updated int    `json:"updated"`
	Nodes   []Node `json:"nodes"`
}

// NodeDelay is one measured outbound latency.
type NodeDelay struct {
	NodeID   string        `json:"node_id"`
	URL      string        `json:"url"`
	Latency  time.Duration `json:"latency"`
	TestedAt time.Time     `json:"tested_at"`
}

// NodeTestResult is persisted latency history for a node.
type NodeTestResult struct {
	NodeID   string        `json:"node_id"`
	Kind     string        `json:"kind"`
	URL      string        `json:"url,omitempty"`
	Latency  time.Duration `json:"latency"`
	Error    string        `json:"error,omitempty"`
	TestedAt time.Time     `json:"tested_at"`
}

// NodeAPI owns node import, editing, filtering, and latency measurement.
type NodeAPI interface {
	List(ctx context.Context, subscriptionID string) ([]Node, error)
	Get(ctx context.Context, id string) (Node, error)
	Import(ctx context.Context, nodes []Node) (NodeImportResult, error)
	Save(ctx context.Context, node Node) (Node, error)
	Delete(ctx context.Context, id string) error
	Delay(ctx context.Context, id, url string) (NodeDelay, error)
	TestResults(ctx context.Context, nodeID string, limit int) ([]NodeTestResult, error)
}

// PolicySnapshot is the complete routing and DNS intent consumed by clients.
type PolicySnapshot struct {
	Routes RouteProfile `json:"routes"`
	DNS    DNSProfile   `json:"dns"`
}

// PolicyAPI composes routing and DNS modules without duplicating their use
// cases. It is the stable entry point for policy-oriented clients.
type PolicyAPI interface {
	Get(ctx context.Context) (PolicySnapshot, error)
	Routes() RouteAPI
	DNS() DNSAPI
}

// RuleSetAPI manages rule-set definitions and their refresh metadata.
type RuleSetAPI interface {
	List(ctx context.Context) ([]RuleSet, error)
	Get(ctx context.Context, name string) (RuleSet, error)
	Save(ctx context.Context, definition RuleSetDefinition) (RuleSet, error)
	Delete(ctx context.Context, name string) error
	Refresh(ctx context.Context, name string) (RuleSet, error)
	RefreshAll(ctx context.Context) ([]RuleSet, error)
}

// BackupCreateOptions selects where a consistent database snapshot is stored.
type BackupCreateOptions struct {
	DestinationDir string `json:"destination_dir,omitempty"`
}

// BackupListOptions selects the directory whose snapshots are inspected.
type BackupListOptions struct {
	DestinationDir string `json:"destination_dir,omitempty"`
}

// BackupInfo describes a verified SQLite snapshot.
type BackupInfo struct {
	Path          string    `json:"path"`
	SizeBytes     int64     `json:"size_bytes"`
	CreatedAt     time.Time `json:"created_at"`
	SchemaVersion int       `json:"schema_version"`
	Revision      string    `json:"revision,omitempty"`
}

// BackupAPI creates and inspects consistent application snapshots.
type BackupAPI interface {
	Create(ctx context.Context, options BackupCreateOptions) (BackupInfo, error)
	Inspect(ctx context.Context, path string) (BackupInfo, error)
	List(ctx context.Context, options BackupListOptions) ([]BackupInfo, error)
}
