package core

import (
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

func TestAnalyzeConfigAcceptsMultiTagRuleSetReferences(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "inbounds":[{"type":"mixed","tag":"in","listen":"127.0.0.1"}],
  "outbounds":[{"type":"direct","tag":"direct"}],
  "route":{"rule_set":[{"type":"local","tag":["cn","proxy"],"format":"binary","path":"/tmp/{tag}.srs"}],"rules":[{"rule_set":["cn","proxy"],"outbound":"direct"}]}
}`))
	if report.Status != model.ConfigDiagnosticsHealthy || report.Counts.RuleSets != 2 {
		t.Fatalf("multi-tag diagnostics = %+v", report)
	}
}

func TestAnalyzeConfigMultiTagRuleSetErrors(t *testing.T) {
	for _, test := range []struct{ fields, path, code string }{
		{fields: `"type":"inline","tag":["a","b"],"rules":[]`, path: "route.rule_set[0].tag", code: "invalid_ruleset_tags"},
		{fields: `"type":"local","tag":["a","b"],"path":"/tmp/file.srs"`, path: "route.rule_set[0].path", code: "invalid_ruleset_tags"},
		{fields: `"type":"remote","tag":["a","b"],"url":"https://example.org/a.srs"`, path: "route.rule_set[0].url", code: "invalid_ruleset_tags"},
		{fields: `"type":"remote","tag":["a","b"],"url":"https://example.org/{tag}.srs","initial_path":"/tmp/file.srs"`, path: "route.rule_set[0].initial_path", code: "invalid_ruleset_tags"},
		{fields: `"type":"local","tag":["a","a"],"path":"/tmp/{tag}.srs"`, path: "route.rule_set[0].tag[1]", code: "duplicate_tag"},
		{fields: `"type":"local","tag":["a",""],"path":"/tmp/{tag}.srs"`, path: "route.rule_set[0].tag", code: "missing_tag"},
	} {
		report := AnalyzeConfig([]byte(`{"route":{"rule_set":[{` + test.fields + `}]}}`))
		requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: test.code, path: test.path, severity: model.ConfigDiagnosticSeverityError})
	}
}

func TestAnalyzeConfigPreferredByChecksCapabilities(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "outbounds":[{"type":"direct","tag":"direct"},{"type":"bridge","tag":"bridge"}],
  "route":{"rules":[{"preferred_by":["direct","missing","bridge"],"action":"reject"}]},
  "dns":{"servers":[{"type":"local","tag":"local"},{"type":"udp","tag":"remote","server":"192.0.2.1"}],"rules":[{"preferred_by":["remote","local"],"action":"reject"}]}
}`))
	for _, expected := range []expectedConfigDiagnostic{
		{code: "invalid_preferred_by_reference", path: "route.rules[0].preferred_by", value: "direct"},
		{code: "invalid_preferred_by_reference", path: "route.rules[0].preferred_by", value: "missing"},
		{code: "invalid_preferred_by_reference", path: "dns.rules[0].preferred_by", value: "remote"},
	} {
		requireConfigDiagnostic(t, report.Issues, expected)
	}
	if countDiagnostics(report.Issues, "invalid_preferred_by_reference") != 3 {
		t.Fatalf("valid preferred_by types misdiagnosed: %+v", report.Issues)
	}
}

func TestAnalyzeConfigDNSRulesUseInlineRuleSetQueryMetadata(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "dns":{"servers":[{"type":"local","tag":"local"}],"rules":[{"strategy":"ipv4_only","server":"local"},{"rule_set":"queries","server":"local"}]},
  "route":{"rule_set":[{"type":"inline","tag":"queries","rules":[{"type":"logical","mode":"or","rules":[{"query_type":"A"}]}]}]}
}`))
	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: "dns_legacy_mode_conflict", path: "dns.rules[0].strategy"})
}

func TestAnalyzeConfigDNSRulesSeparatePureAndMixedIPRuleSets(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "dns":{"servers":[{"type":"local","tag":"local"}],"rules":[{"query_type":"A","server":"local"},{"rule_set":"ip","server":"local"},{"rule_set":"mixed","server":"local"}]},
  "route":{"rule_set":[{"type":"inline","tag":"ip","rules":[{"ip_cidr":"192.0.2.0/24"}]},{"type":"inline","tag":"mixed","rules":[{"ip_cidr":"192.0.2.0/24"},{"domain_suffix":"example.org"}]}]}
}`))
	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: "dns_response_requires_match", path: "dns.rules[1].rule_set"})
	requireNoConfigDiagnostic(t, report.Issues, "dns_response_requires_match", "dns.rules[2].rule_set")
}
