package control

import (
	"context"
	"time"
)

// SchedulerStatus reports the subscription refresh worker state without
// exposing goroutines or transport details to host adapters.
type SchedulerStatus struct {
	Running         bool      `json:"running"`
	IntervalSeconds int       `json:"interval_seconds"`
	LastRunAt       time.Time `json:"last_run_at,omitempty"`
	NextRunAt       time.Time `json:"next_run_at,omitempty"`
	LastError       string    `json:"last_error,omitempty"`
}

// SchedulerAPI owns periodic subscription refreshes. Hosts only need to start
// and stop it; individual subscriptions remain manually refreshable through
// SubscriptionAPI.
type SchedulerAPI interface {
	Status(ctx context.Context) (SchedulerStatus, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	RunOnce(ctx context.Context) ([]SubscriptionRefreshResult, error)
}

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
	ImportLinks(ctx context.Context, links []string) (NodeImportResult, error)
	Save(ctx context.Context, node Node) (Node, error)
	Delete(ctx context.Context, id string) error
	Delay(ctx context.Context, id, url string) (NodeDelay, error)
	TestResults(ctx context.Context, nodeID string, limit int) ([]NodeTestResult, error)
	DecodeLink(ctx context.Context, link string) (Node, error)
	EncodeLink(ctx context.Context, node Node) (string, error)
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
