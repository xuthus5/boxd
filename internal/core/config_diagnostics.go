package core

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"

	"github.com/xuthus5/boxd/internal/model"
)

const diagnosticDetailLimit = 240

type diagnosticEntry struct {
	tag      string
	typeName string
	path     string
}

// AnalyzeConfigFile reads and analyzes one persisted sing-box configuration.
// It intentionally returns only structural metadata and non-secret references.
func AnalyzeConfigFile(path string) model.ConfigDiagnostics {
	body, err := os.ReadFile(path)
	if err != nil {
		code := "config_unreadable"
		if os.IsNotExist(err) {
			code = "config_missing"
		}
		report := newConfigDiagnostics()
		addDiagnostic(&report, code, model.ConfigDiagnosticSeverityError, "config", "", "")
		finishConfigDiagnostics(&report)
		return report
	}
	return AnalyzeConfig(body)
}

// AnalyzeConfig validates and inspects a sing-box JSON document without
// starting the kernel or touching the persisted configuration.
func AnalyzeConfig(body []byte) model.ConfigDiagnostics {
	report := newConfigDiagnostics()
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		addDiagnostic(&report, "invalid_json", model.ConfigDiagnosticSeverityError, "config", "", diagnosticDetail(err))
		finishConfigDiagnostics(&report)
		return report
	}
	cfg, ok := root.(map[string]any)
	if !ok || cfg == nil {
		addDiagnostic(&report, "invalid_root", model.ConfigDiagnosticSeverityError, "config", "", "")
		finishConfigDiagnostics(&report)
		return report
	}
	if err := validateSingBoxConfig(body); err != nil {
		addDiagnostic(&report, "invalid_singbox_config", model.ConfigDiagnosticSeverityError, "config", "", diagnosticDetail(err))
	}
	inspectTopology(&report, cfg)
	finishConfigDiagnostics(&report)
	return report
}

func newConfigDiagnostics() model.ConfigDiagnostics {
	return model.ConfigDiagnostics{
		Status:    model.ConfigDiagnosticsHealthy,
		CheckedAt: time.Now().UTC(),
		Issues:    make([]model.ConfigDiagnostic, 0),
	}
}

func validateSingBoxConfig(body []byte) error {
	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	defer cancel()
	var options option.Options
	return options.UnmarshalJSONContext(ctx, body)
}

func inspectTopology(report *model.ConfigDiagnostics, cfg map[string]any) {
	inbounds := diagnosticEntries(cfg, "inbounds")
	outbounds := diagnosticEntries(cfg, "outbounds")
	endpoints := diagnosticEntries(cfg, "endpoints")
	ruleSets := diagnosticEntriesFromRoute(cfg, "rule_set")
	dnsServers := diagnosticEntriesFromDNS(cfg)
	report.Counts.Inbounds = len(inbounds)
	report.Counts.Outbounds = len(outbounds)
	report.Counts.Endpoints = len(endpoints)
	report.Counts.RuleSets = len(ruleSets)
	report.Counts.RouteRules = countNestedArray(cfg, "route", "rules")
	report.Counts.DNSServers = len(dnsServers)
	report.Counts.DNSRules = countNestedArray(cfg, "dns", "rules")

	checkDuplicateTags(report, append(outbounds, endpoints...))
	checkDuplicateTags(report, inbounds)
	checkDuplicateTags(report, ruleSets)
	checkOutboundReferences(report, cfg, outbounds, endpoints, ruleSets)
	checkDNSReferences(report, cfg, dnsServers)
	checkInsecureTLS(report, cfg)
	if len(inbounds) == 0 {
		addDiagnostic(report, "no_inbounds", model.ConfigDiagnosticSeverityWarning, "inbounds", "", "")
	}
	if len(outbounds)+len(endpoints) == 0 {
		addDiagnostic(report, "no_outbounds", model.ConfigDiagnosticSeverityWarning, "outbounds", "", "")
	}
	setConfigFeatures(report, cfg, inbounds, outbounds, endpoints, ruleSets)
}

func diagnosticEntries(cfg map[string]any, key string) []diagnosticEntry {
	items, _ := cfg[key].([]any)
	entries := make([]diagnosticEntry, 0, len(items))
	for index, item := range items {
		object, ok := item.(map[string]any)
		if !ok || object == nil {
			continue
		}
		tag := stringValue(object["tag"])
		if tag == "" {
			tag = strconv.Itoa(index)
		}
		entries = append(entries, diagnosticEntry{
			tag:      tag,
			typeName: stringValue(object["type"]),
			path:     key + "[" + strconv.Itoa(index) + "]",
		})
	}
	return entries
}

func diagnosticEntriesFromRoute(cfg map[string]any, key string) []diagnosticEntry {
	section := objectValue(cfg["route"])
	if section == nil {
		return nil
	}
	items, _ := section[key].([]any)
	entries := make([]diagnosticEntry, 0, len(items))
	for index, item := range items {
		object, ok := item.(map[string]any)
		if !ok || object == nil {
			continue
		}
		entries = append(entries, diagnosticEntry{
			tag:      stringValue(object["tag"]),
			typeName: stringValue(object["type"]),
			path:     "route." + key + "[" + strconv.Itoa(index) + "]",
		})
	}
	return entries
}

func diagnosticEntriesFromDNS(cfg map[string]any) []diagnosticEntry {
	section := objectValue(cfg["dns"])
	if section == nil {
		return nil
	}
	items, _ := section["servers"].([]any)
	entries := make([]diagnosticEntry, 0, len(items))
	for index, item := range items {
		object, ok := item.(map[string]any)
		if !ok || object == nil {
			continue
		}
		tag := stringValue(object["tag"])
		if tag == "" {
			tag = strconv.Itoa(index)
		}
		entries = append(entries, diagnosticEntry{
			tag:  tag,
			path: "dns.servers[" + strconv.Itoa(index) + "]",
		})
	}
	return entries
}

func countNestedArray(cfg map[string]any, sectionKey, arrayKey string) int {
	section := objectValue(cfg[sectionKey])
	if section == nil {
		return 0
	}
	items, _ := section[arrayKey].([]any)
	return len(items)
}

func finishConfigDiagnostics(report *model.ConfigDiagnostics) {
	report.Summary = model.ConfigDiagnosticsSummary{}
	for _, issue := range report.Issues {
		switch issue.Severity {
		case model.ConfigDiagnosticSeverityError:
			report.Summary.Errors++
		case model.ConfigDiagnosticSeverityWarning:
			report.Summary.Warnings++
		}
	}
	switch {
	case report.Summary.Errors > 0:
		report.Status = model.ConfigDiagnosticsError
	case report.Summary.Warnings > 0:
		report.Status = model.ConfigDiagnosticsWarning
	default:
		report.Status = model.ConfigDiagnosticsHealthy
	}
}

func addDiagnostic(report *model.ConfigDiagnostics, code, severity, path, value, detail string) {
	report.Issues = append(report.Issues, model.ConfigDiagnostic{
		Code: code, Severity: severity, Path: path, Value: value, Detail: detail,
	})
}

func diagnosticDetail(err error) string {
	if err == nil {
		return ""
	}
	detail := strings.Join(strings.Fields(err.Error()), " ")
	if len(detail) > diagnosticDetailLimit {
		return detail[:diagnosticDetailLimit] + "…"
	}
	return detail
}

func objectValue(value any) map[string]any {
	object, _ := value.(map[string]any)
	return object
}

func objectAtPath(cfg map[string]any, path string) map[string]any {
	parts := strings.SplitN(path, "[", 2)
	if len(parts) != 2 {
		return nil
	}
	section := strings.TrimSuffix(parts[0], "]")
	indexText := strings.TrimSuffix(parts[1], "]")
	index, err := strconv.Atoi(indexText)
	if err != nil {
		return nil
	}
	items, _ := cfg[section].([]any)
	if index < 0 || index >= len(items) {
		return nil
	}
	return objectValue(items[index])
}

func stringValues(value any) []string {
	if single, ok := value.(string); ok {
		return []string{single}
	}
	items, _ := value.([]any)
	values := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok {
			values = append(values, value)
		}
	}
	return values
}
