package main

import "github.com/xuthus5/boxd/internal/model"

var supportBundleOmittedFields = []string{
	"passwords and JWT secrets",
	"authentication tokens",
	"subscription URLs and names",
	"node tags, server addresses, and raw node settings",
	"raw sing-box configuration",
	"log messages and connection targets",
	"filesystem paths",
	"rule-set URLs, paths, and ETags",
}

type supportBundle struct {
	FormatVersion int                                  `json:"format_version"`
	Product       string                               `json:"product"`
	ExportedAt    string                               `json:"exported_at"`
	Redaction     supportBundleRedaction               `json:"redaction"`
	Requests      map[string]supportBundleRequestState `json:"requests"`
	Panel         supportBundlePanel                   `json:"panel"`
	Service       supportBundleService                 `json:"service"`
	Runtime       supportBundleRuntime                 `json:"runtime"`
	Config        supportBundleConfig                  `json:"config"`
	Resources     supportBundleResources               `json:"resources"`
	Network       supportBundleNetwork                 `json:"network"`
}

type supportBundleRedaction struct {
	Strategy string   `json:"strategy"`
	Omitted  []string `json:"omitted"`
}

type supportBundlePanel struct {
	Version       *string                  `json:"version"`
	KernelVersion *string                  `json:"kernel_version"`
	Ready         *bool                    `json:"ready"`
	Preferences   supportBundlePreferences `json:"preferences"`
}

type supportBundlePreferences struct {
	Theme           *string `json:"theme"`
	Language        *string `json:"language"`
	MinimumLogLevel *string `json:"minimum_log_level"`
}

type supportBundleService struct {
	Available        bool    `json:"available"`
	Running          *bool   `json:"running"`
	Uptime           *string `json:"uptime"`
	MemoryBytes      *int64  `json:"memory_bytes"`
	Version          *string `json:"version"`
	StartedAt        *string `json:"started_at"`
	LastErrorPresent *bool   `json:"last_error_present"`
	LastErrorCode    *string `json:"last_error_code"`
}

type supportBundleRuntime struct {
	Available bool                      `json:"available"`
	Memory    *supportBundleMemoryStats `json:"memory"`
}

type supportBundleMemoryStats struct {
	Alloc        *uint64 `json:"alloc"`
	Total        *uint64 `json:"total"`
	Sys          *uint64 `json:"sys"`
	NumGC        *uint32 `json:"num_gc"`
	HeapInuse    *uint64 `json:"heap_inuse"`
	StackInuse   *uint64 `json:"stack_inuse"`
	NumGoroutine *int    `json:"num_goroutine"`
}

type supportBundleConfig struct {
	Diagnostics  supportBundleDiagnostics  `json:"diagnostics"`
	RuleSets     supportBundleRuleSets     `json:"rule_sets"`
	ApplyHistory supportBundleApplyHistory `json:"apply_history"`
}

type supportBundleDiagnostics struct {
	Available       bool                             `json:"available"`
	Status          *string                          `json:"status"`
	CheckedAt       *string                          `json:"checked_at"`
	Summary         *model.ConfigDiagnosticsSummary  `json:"summary"`
	Counts          *model.ConfigDiagnosticsCounts   `json:"counts"`
	Features        *model.ConfigDiagnosticsFeatures `json:"features"`
	IssueCodes      map[string]int                   `json:"issue_codes"`
	IssueSeverities map[string]int                   `json:"issue_severities"`
}

type supportBundleRuleSets struct {
	Available       bool                    `json:"available"`
	Total           int                     `json:"total"`
	Builtin         int                     `json:"builtin"`
	Updatable       int                     `json:"updatable"`
	WithLastUpdated int                     `json:"with_last_updated"`
	Types           map[string]int          `json:"types"`
	Formats         map[string]int          `json:"formats"`
	AutoUpdate      supportBundleAutoUpdate `json:"auto_update"`
}

type supportBundleAutoUpdate struct {
	Available bool    `json:"available"`
	Enabled   *bool   `json:"enabled"`
	Interval  *string `json:"interval"`
}

type supportBundleApplyHistory struct {
	Available       bool           `json:"available"`
	Count           int            `json:"count"`
	FailureCount    int            `json:"failure_count"`
	Statuses        map[string]int `json:"statuses"`
	ErrorCodes      map[string]int `json:"error_codes"`
	LatestAppliedAt *string        `json:"latest_applied_at"`
}

type supportBundleResources struct {
	Subscriptions supportBundleSubscriptions `json:"subscriptions"`
	Nodes         supportBundleNodes         `json:"nodes"`
}

type supportBundleSubscriptions struct {
	Available   bool           `json:"available"`
	Total       int            `json:"total"`
	Failed      int            `json:"failed"`
	WithTraffic int            `json:"with_traffic"`
	ErrorCodes  map[string]int `json:"error_codes"`
}

type supportBundleNodes struct {
	Available      bool           `json:"available"`
	Total          int            `json:"total"`
	Types          map[string]int `json:"types"`
	Sources        map[string]int `json:"sources"`
	ProbeSamples   int            `json:"probe_samples"`
	ProbeSuccesses int            `json:"probe_successes"`
	ProbeFailures  int            `json:"probe_failures"`
	AverageLatency *float64       `json:"average_latency_ms"`
}

type supportBundleNetwork struct {
	Available   bool `json:"available"`
	Interfaces  int  `json:"interfaces"`
	IPAddresses int  `json:"ip_addresses"`
	IPv4        int  `json:"ipv4_addresses"`
	IPv6        int  `json:"ipv6_addresses"`
}
