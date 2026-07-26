package model

import "time"

const (
	ConfigDiagnosticsHealthy = "healthy"
	ConfigDiagnosticsWarning = "warning"
	ConfigDiagnosticsError   = "error"

	ConfigDiagnosticSeverityWarning = "warning"
	ConfigDiagnosticSeverityError   = "error"
)

// ConfigDiagnostics describes the effective shape and likely problems of the
// currently persisted sing-box configuration.
type ConfigDiagnostics struct {
	Status    string                    `json:"status"`
	CheckedAt time.Time                 `json:"checked_at"`
	Summary   ConfigDiagnosticsSummary  `json:"summary"`
	Counts    ConfigDiagnosticsCounts   `json:"counts"`
	Features  ConfigDiagnosticsFeatures `json:"features"`
	Issues    []ConfigDiagnostic        `json:"issues"`
}

type ConfigDiagnosticsSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
}

type ConfigDiagnosticsCounts struct {
	Inbounds   int `json:"inbounds"`
	Outbounds  int `json:"outbounds"`
	Endpoints  int `json:"endpoints"`
	RouteRules int `json:"route_rules"`
	RuleSets   int `json:"rule_sets"`
	DNSServers int `json:"dns_servers"`
	DNSRules   int `json:"dns_rules"`
}

type ConfigDiagnosticsFeatures struct {
	TUN           bool `json:"tun"`
	ClashAPI      bool `json:"clash_api"`
	CacheFile     bool `json:"cache_file"`
	FakeIP        bool `json:"fakeip"`
	Selector      bool `json:"selector"`
	URLTest       bool `json:"urltest"`
	WireGuard     bool `json:"wireguard"`
	RemoteRuleSet bool `json:"remote_rule_set"`
}

type ConfigDiagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Path     string `json:"path,omitempty"`
	Value    string `json:"value,omitempty"`
	Detail   string `json:"detail,omitempty"`
}
