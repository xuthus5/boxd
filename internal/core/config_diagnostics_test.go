package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/model"
)

func TestAnalyzeConfigReportsHealthyTopologyAndFeatures(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "inbounds": [{"type":"mixed","tag":"mixed-in","listen":"127.0.0.1","listen_port":1080}],
  "outbounds": [
    {"type":"direct","tag":"direct"},
    {"type":"selector","tag":"proxy","outbounds":["direct"]},
    {"type":"urltest","tag":"auto","outbounds":["direct"]}
  ],
  "route": {
    "final":"proxy",
    "rules":[{"outbound":"proxy"}],
    "rule_set":[{"type":"inline","tag":"geo","rules":[]}]
  },
  "dns": {
    "servers":[{"type":"local","tag":"local"},{"type":"fakeip","tag":"fake","inet4_range":"198.18.0.0/15"}],
    "rules":[{"server":"local"}],
    "final":"local"
  },
  "experimental":{"cache_file":{"enabled":true},"clash_api":{"external_controller":"127.0.0.1:9090"}}
}`))

	if report.Status != model.ConfigDiagnosticsHealthy {
		t.Fatalf("status = %q, report = %+v", report.Status, report)
	}
	if report.Summary.Errors != 0 || report.Summary.Warnings != 0 {
		t.Fatalf("summary = %+v", report.Summary)
	}
	counts := report.Counts
	if counts.Inbounds != 1 || counts.Outbounds != 3 || counts.RouteRules != 1 ||
		counts.RuleSets != 1 || counts.DNSServers != 2 || counts.DNSRules != 1 {
		t.Fatalf("counts = %+v", report.Counts)
	}
	features := report.Features
	if !features.Selector || !features.URLTest || !features.FakeIP ||
		!features.CacheFile || !features.ClashAPI {
		t.Fatalf("features = %+v", report.Features)
	}
}

func TestAnalyzeConfigReportsSingBoxMigrationWarnings(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "inbounds":[{"type":"mixed","tag":"mixed","listen":"127.0.0.1","listen_port":1080}],
  "outbounds":[
    {"type":"direct","tag":"direct"},
    {"type":"socks","tag":"domain-node","server":"proxy.example.com","server_port":1080,"domain_strategy":"prefer_ipv4"}
  ],
  "route":{"rules":[{"outbound":"direct"}]},
  "dns":{
    "servers":[{"tag":"local","address":"local"},{"tag":"remote","address":"https://1.1.1.1/dns-query"}],
    "rules":[{"type":"logical","mode":"or","rules":[{"outbound":"proxy"}],"server":"local"}],
    "fakeip":{"enabled":true,"inet4_range":"198.18.0.0/15"}
  }
}`))

	if report.Status != model.ConfigDiagnosticsWarning {
		t.Fatalf("status = %q, report = %+v", report.Status, report)
	}
	if !hasDiagnostic(report.Issues, "legacy_dns_server", "local") {
		t.Fatalf("legacy DNS server issue missing: %+v", report.Issues)
	}
	if !hasDiagnostic(report.Issues, "legacy_dns_fakeip", "") {
		t.Fatalf("legacy FakeIP issue missing: %+v", report.Issues)
	}
	if !hasDiagnostic(report.Issues, "outbound_dns_rule_item", "proxy") {
		t.Fatalf("outbound DNS rule issue missing: %+v", report.Issues)
	}
	if !hasDiagnostic(report.Issues, "missing_domain_resolver", "domain-node") {
		t.Fatalf("missing resolver issue missing: %+v", report.Issues)
	}
	if !hasDiagnostic(report.Issues, "legacy_domain_strategy", "domain-node") {
		t.Fatalf("legacy domain strategy issue missing: %+v", report.Issues)
	}
}

func TestAnalyzeConfigAvoidsMigrationFalsePositives(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "default resolver",
			body: `{
  "inbounds":[{"type":"mixed","tag":"mixed","listen":"127.0.0.1","listen_port":1080}],
  "outbounds":[{"type":"socks","tag":"domain-node","server":"proxy.example.com","server_port":1080}],
  "route":{"default_domain_resolver":"local"},
  "dns":{"servers":[{"type":"local","tag":"local"},{"type":"udp","tag":"remote","server":"1.1.1.1"}]}
}`,
		},
		{
			name: "IP server",
			body: `{
  "inbounds":[{"type":"mixed","tag":"mixed","listen":"127.0.0.1","listen_port":1080}],
  "outbounds":[{"type":"socks","tag":"ip-node","server":"192.0.2.1","server_port":1080}],
  "dns":{"servers":[{"type":"local","tag":"local"},{"type":"udp","tag":"remote","server":"1.1.1.1"}]}
}`,
		},
		{
			name: "single DNS server",
			body: `{
  "inbounds":[{"type":"mixed","tag":"mixed","listen":"127.0.0.1","listen_port":1080}],
  "outbounds":[{"type":"socks","tag":"domain-node","server":"proxy.example.com","server_port":1080}],
  "dns":{"servers":[{"type":"local","tag":"local"}]}
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := AnalyzeConfig([]byte(tt.body))
			if hasMigrationDiagnostic(report.Issues) {
				t.Fatalf("unexpected migration warning: %+v", report.Issues)
			}
		})
	}
}

func TestAnalyzeConfigReportsActionableIssues(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "outbounds": [
    {"type":"direct","tag":"dup"},
    {"type":"selector","tag":"dup","outbounds":["missing"]}
  ],
  "route":{"final":"missing","rules":[{"outbound":"unknown"},{"rule_set":["absent"]}]},
  "dns":{"servers":[{"address":"local","tag":"dns"}],"final":"absent","rules":[{"server":"unknown"}]},
  "inbounds":[{"type":"mixed","tag":"in","listen":"0.0.0.0","listen_port":1080,"tls":{"enabled":true,"insecure":true}}]
}`))

	if report.Status != model.ConfigDiagnosticsError {
		t.Fatalf("status = %q, report = %+v", report.Status, report)
	}
	if report.Summary.Errors < 4 || report.Summary.Warnings < 1 {
		t.Fatalf("summary = %+v, issues = %+v", report.Summary, report.Issues)
	}
	if !hasDiagnostic(report.Issues, "duplicate_tag", "dup") {
		t.Fatalf("duplicate tag issue missing: %+v", report.Issues)
	}
	if !hasDiagnostic(report.Issues, "unknown_outbound_reference", "missing") {
		t.Fatalf("outbound issue missing: %+v", report.Issues)
	}
	if !hasDiagnostic(report.Issues, "unknown_dns_reference", "absent") {
		t.Fatalf("dns issue missing: %+v", report.Issues)
	}
	if !hasDiagnostic(report.Issues, "tls_insecure", "") {
		t.Fatalf("tls issue missing: %+v", report.Issues)
	}
}

func TestAnalyzeConfigReportsParseAndSemanticFailures(t *testing.T) {
	tests := []struct {
		name string
		body string
		code string
	}{
		{name: "invalid json", body: `{`, code: "invalid_json"},
		{name: "invalid root", body: `[]`, code: "invalid_root"},
		{name: "invalid sing-box config", body: `{"outbounds":[{"type":"not-real","tag":"bad"}]}`, code: "invalid_singbox_config"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := AnalyzeConfig([]byte(tt.body))
			if report.Status != model.ConfigDiagnosticsError {
				t.Fatalf("status = %q, report = %+v", report.Status, report)
			}
			if !hasDiagnostic(report.Issues, tt.code, "") {
				t.Fatalf("issue %q missing: %+v", tt.code, report.Issues)
			}
		})
	}
}

func TestAnalyzeConfigWarnsWhenTrafficEntryIsAbsent(t *testing.T) {
	report := AnalyzeConfig([]byte(`{"outbounds":[{"type":"direct","tag":"direct"}]}`))
	if report.Status != model.ConfigDiagnosticsWarning {
		t.Fatalf("status = %q, report = %+v", report.Status, report)
	}
	if !hasDiagnostic(report.Issues, "no_inbounds", "") {
		t.Fatalf("missing no-inbounds warning: %+v", report.Issues)
	}
}

func TestAnalyzeConfigUsesRuntimeDefaultTags(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "inbounds":[{"type":"mixed","listen":"127.0.0.1","listen_port":1080}],
  "outbounds":[{"type":"direct"}],
  "route":{"final":"0"}
}`))
	if report.Status != model.ConfigDiagnosticsHealthy {
		t.Fatalf("default-tag report = %+v", report)
	}

	report = AnalyzeConfig([]byte(`{
  "inbounds":[{"type":"mixed","tag":"mixed","listen":"127.0.0.1","listen_port":1080}],
  "outbounds":[{"type":"direct","tag":"proxy"}],
  "route":{"final":"direct"}
}`))
	if !hasDiagnostic(report.Issues, "unknown_outbound_reference", "direct") {
		t.Fatalf("missing common-tag issue: %+v", report.Issues)
	}
}

func TestAnalyzeConfigReportsNestedRuleSetReferences(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "inbounds":[{"type":"mixed","tag":"mixed","listen":"127.0.0.1","listen_port":1080}],
  "outbounds":[{"type":"direct","tag":"direct"}],
  "route":{"rules":[{"type":"logical","mode":"or","rules":[{"rule_set":"nested-missing"}],"outbound":"direct"}]}
}`))
	if !hasDiagnostic(report.Issues, "unknown_ruleset_reference", "nested-missing") {
		t.Fatalf("nested rule-set issue missing: %+v", report.Issues)
	}
}

func TestAnalyzeConfigReportsEveryInsecureTLS(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "inbounds":[
    {"type":"mixed","tag":"one","listen":"127.0.0.1","listen_port":1080,"tls":{"enabled":true,"insecure":true}},
    {"type":"mixed","tag":"two","listen":"127.0.0.1","listen_port":1081,"tls":{"enabled":true,"insecure":true}}
  ],
  "outbounds":[{"type":"direct","tag":"direct"}]
}`))
	if countDiagnostics(report.Issues, "tls_insecure") != 2 {
		t.Fatalf("TLS issues = %+v", report.Issues)
	}
}

func TestAnalyzeConfigFileReportsReadErrorsAndTimestamp(t *testing.T) {
	report := AnalyzeConfigFile(filepath.Join(t.TempDir(), "missing.json"))
	if report.Status != model.ConfigDiagnosticsError || !hasDiagnostic(report.Issues, "config_missing", "") {
		t.Fatalf("missing-file report = %+v", report)
	}
	if report.CheckedAt.IsZero() || report.CheckedAt.Location() != time.UTC {
		t.Fatalf("checked_at = %v", report.CheckedAt)
	}

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"outbounds":[{"type":"direct","tag":"direct"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	report = AnalyzeConfigFile(path)
	if report.Counts.Outbounds != 1 {
		t.Fatalf("file report = %+v", report)
	}
}

func hasDiagnostic(issues []model.ConfigDiagnostic, code, value string) bool {
	for _, issue := range issues {
		if issue.Code == code && (value == "" || issue.Value == value || strings.Contains(issue.Detail, value)) {
			return true
		}
	}
	return false
}

func countDiagnostics(issues []model.ConfigDiagnostic, code string) int {
	count := 0
	for _, issue := range issues {
		if issue.Code == code {
			count++
		}
	}
	return count
}

func hasMigrationDiagnostic(issues []model.ConfigDiagnostic) bool {
	for _, issue := range issues {
		switch issue.Code {
		case "legacy_dns_server",
			"legacy_dns_fakeip",
			"outbound_dns_rule_item",
			"missing_domain_resolver",
			"legacy_domain_strategy":
			return true
		}
	}
	return false
}
