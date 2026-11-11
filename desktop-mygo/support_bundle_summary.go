package main

import (
	"strings"
	"time"

	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"
)

func buildSupportBundle(collection supportBundleCollection, exportedAt time.Time) supportBundle {
	sources := collection.Sources
	return supportBundle{
		FormatVersion: supportBundleFormatVersion,
		Product:       "boxd",
		ExportedAt:    exportedAt.UTC().Format(time.RFC3339Nano),
		Redaction: supportBundleRedaction{
			Strategy: "allowlist", Omitted: append([]string(nil), supportBundleOmittedFields...),
		},
		Requests:  collection.Requests,
		Panel:     summarizeSupportPanel(sources),
		Service:   summarizeSupportService(sources.Service),
		Runtime:   summarizeSupportRuntime(sources.Memory),
		Config:    summarizeSupportConfig(sources),
		Resources: summarizeSupportResources(sources),
		Network:   summarizeSupportNetwork(sources.Network, collection.Requests["network"].Status == "ok"),
	}
}

func summarizeSupportPanel(sources supportBundleSources) supportBundlePanel {
	ready := (*bool)(nil)
	if sources.Readiness != nil {
		value := *sources.Readiness == "ready"
		ready = &value
	}
	return supportBundlePanel{
		Version:       supportText(sources.Version.Version),
		KernelVersion: supportText(sources.Version.KernelVersion),
		Ready:         ready,
		Preferences: supportBundlePreferences{
			Theme: supportToken(sources.Preferences.Theme), Language: supportToken(sources.Preferences.Language),
			MinimumLogLevel: supportToken(sources.Preferences.MinimumLogLevel),
		},
	}
}

func summarizeSupportService(status *model.ServiceStatus) supportBundleService {
	if status == nil {
		return supportBundleService{}
	}
	startedAt := ""
	if status.StartedAt != nil {
		startedAt = status.StartedAt.UTC().Format(time.RFC3339Nano)
	}
	lastError := supportText(status.LastError) != nil
	return supportBundleService{
		Available: true, Running: supportBool(status.Running), Uptime: supportText(status.Uptime),
		MemoryBytes: supportInt64(status.Memory), Version: supportText(status.Version),
		StartedAt: supportText(startedAt), LastErrorPresent: &lastError,
		LastErrorCode: supportToken(status.LastErrorCode),
	}
}

func summarizeSupportRuntime(memory *supportBundleMemory) supportBundleRuntime {
	if memory == nil {
		return supportBundleRuntime{}
	}
	return supportBundleRuntime{Available: true, Memory: &supportBundleMemoryStats{
		Alloc: supportUint64(memory.Alloc), Total: supportUint64(memory.Total), Sys: supportUint64(memory.Sys),
		NumGC: supportUint32(memory.NumGC), HeapInuse: supportUint64(memory.HeapInuse),
		StackInuse: supportUint64(memory.StackInuse), NumGoroutine: supportInt(memory.NumGoroutine),
	}}
}

func summarizeSupportConfig(sources supportBundleSources) supportBundleConfig {
	return supportBundleConfig{
		Diagnostics:  summarizeSupportDiagnostics(sources.Diagnostics),
		RuleSets:     summarizeSupportRuleSets(sources.RuleSets, sources.RuleSetAuto),
		ApplyHistory: summarizeSupportApplyHistory(sources.ApplyHistory),
	}
}

func summarizeSupportDiagnostics(diagnostics *model.ConfigDiagnostics) supportBundleDiagnostics {
	if diagnostics == nil {
		return supportBundleDiagnostics{IssueCodes: map[string]int{}, IssueSeverities: map[string]int{}}
	}
	codes := make([]string, 0, len(diagnostics.Issues))
	severities := make([]string, 0, len(diagnostics.Issues))
	for _, issue := range diagnostics.Issues {
		codes = append(codes, issue.Code)
		severities = append(severities, issue.Severity)
	}
	checkedAt := diagnostics.CheckedAt.UTC().Format(time.RFC3339Nano)
	return supportBundleDiagnostics{
		Available: true, Status: supportToken(diagnostics.Status), CheckedAt: supportText(checkedAt),
		Summary: &diagnostics.Summary, Counts: &diagnostics.Counts, Features: &diagnostics.Features,
		IssueCodes: supportCounts(codes), IssueSeverities: supportCounts(severities),
	}
}

func summarizeSupportRuleSets(items []model.RuleSetStatusItem, auto *model.RuleSetAutoUpdate) supportBundleRuleSets {
	types := make([]string, 0, len(items))
	formats := make([]string, 0, len(items))
	withLastUpdated := 0
	for _, item := range items {
		types = append(types, item.Type)
		formats = append(formats, item.Format)
		if item.LastUpdated != nil {
			withLastUpdated++
		}
	}
	result := supportBundleRuleSets{
		Available: items != nil, Total: len(items), WithLastUpdated: withLastUpdated,
		Types: supportCounts(types), Formats: supportCounts(formats),
	}
	for _, item := range items {
		if item.Builtin {
			result.Builtin++
		}
		if item.Updatable {
			result.Updatable++
		}
	}
	if auto != nil {
		result.AutoUpdate = supportBundleAutoUpdate{
			Available: true, Enabled: supportBool(auto.Enabled), Interval: supportText(auto.Interval),
		}
	}
	return result
}

func summarizeSupportApplyHistory(events []model.ConfigApplyEvent) supportBundleApplyHistory {
	statuses := make([]string, 0, len(events))
	errorCodes := make([]string, 0, len(events))
	failures := 0
	latest := ""
	for _, event := range events {
		statuses = append(statuses, event.Status)
		errorCodes = append(errorCodes, event.ErrorCode)
		if event.Status == model.ConfigApplyStatusRolledBack || event.Status == model.ConfigApplyStatusValidateFailed {
			failures++
		}
		applied := event.AppliedAt.UTC().Format(time.RFC3339Nano)
		if applied > latest {
			latest = applied
		}
	}
	return supportBundleApplyHistory{
		Available: events != nil, Count: len(events), FailureCount: failures,
		Statuses: supportCounts(statuses), ErrorCodes: supportCounts(errorCodes),
		LatestAppliedAt: supportText(latest),
	}
}

func summarizeSupportResources(sources supportBundleSources) supportBundleResources {
	errorCodes := make([]string, 0, len(sources.Subscriptions))
	failed, withTraffic := 0, 0
	for _, item := range sources.Subscriptions {
		errorCodes = append(errorCodes, item.ErrorCode)
		if strings.TrimSpace(item.Error) != "" {
			failed++
		}
		if item.Traffic != nil {
			withTraffic++
		}
	}
	nodeTypes := make([]string, 0, len(sources.Nodes))
	nodeSources := make([]string, 0, len(sources.Nodes))
	for _, node := range sources.Nodes {
		nodeTypes = append(nodeTypes, node.Type)
		nodeSources = append(nodeSources, node.Source)
	}
	probes := summarizeSupportProbes(sources.NodeHistory)
	return supportBundleResources{
		Subscriptions: supportBundleSubscriptions{
			Available: sources.Subscriptions != nil, Total: len(sources.Subscriptions),
			Failed: failed, WithTraffic: withTraffic, ErrorCodes: supportCounts(errorCodes),
		},
		Nodes: supportBundleNodes{
			Available: sources.Nodes != nil, Total: len(sources.Nodes),
			Types: supportCounts(nodeTypes), Sources: supportCounts(nodeSources),
			ProbeSamples: probes.samples, ProbeSuccesses: probes.successes,
			ProbeFailures: probes.failures, AverageLatency: probes.average,
		},
	}
}

type supportProbeSummary struct {
	samples, successes, failures int
	average                      *float64
}

func summarizeSupportProbes(history map[string]map[string][]model.LatencyPoint) supportProbeSummary {
	summary := supportProbeSummary{}
	var total float64
	for _, byType := range history {
		for _, points := range byType {
			for _, point := range points {
				summary.samples++
				if point.Success {
					summary.successes++
				} else {
					summary.failures++
				}
				total += point.LatencyMs
			}
		}
	}
	if summary.samples > 0 {
		average := total / float64(summary.samples)
		summary.average = &average
	}
	return summary
}

func summarizeSupportNetwork(interfaces []service.InterfaceInfo, available bool) supportBundleNetwork {
	result := supportBundleNetwork{Available: available, Interfaces: len(interfaces)}
	for _, iface := range interfaces {
		for _, address := range iface.IPs {
			result.IPAddresses++
			if strings.Contains(address, ":") {
				result.IPv6++
			} else {
				result.IPv4++
			}
		}
	}
	return result
}

func supportCounts(values []string) map[string]int {
	counts := make(map[string]int)
	for _, value := range values {
		if token := supportBundleToken(value); token != "" {
			counts[token]++
		}
	}
	return counts
}

func supportText(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if len(trimmed) > 160 {
		trimmed = trimmed[:160]
	}
	return &trimmed
}

func supportToken(value string) *string {
	token := supportBundleToken(value)
	if token == "" {
		return nil
	}
	return &token
}

func supportBool(value bool) *bool       { return &value }
func supportInt(value int) *int          { return &value }
func supportUint64(value uint64) *uint64 { return &value }
func supportUint32(value uint32) *uint32 { return &value }
func supportInt64(value int64) *int64    { return &value }
