package core

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sagernet/sing-box/common/srs"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"

	"github.com/xuthus5/boxd/internal/model"
)

func TestAnalyzeConfigUsesLocalRuleSetMetadata(t *testing.T) {
	queryPath := writeDiagnosticTestFile(t, "queries.json", []byte(`{"version":5,"rules":[{"query_type":"A"}]}`))
	ipData := diagnosticBinaryRuleSet(t, `{"version":5,"rules":[{"ip_cidr":"192.0.2.0/24"}]}`)
	ipPath := writeDiagnosticTestFile(t, "ip.srs", ipData)
	for _, ruleset := range []map[string]any{
		{"type": "local", "tag": "query", "path": queryPath},
	} {
		cfg := map[string]any{
			"dns": map[string]any{"servers": []any{map[string]any{"type": "local", "tag": "local"}}, "rules": []any{
				map[string]any{"strategy": "ipv4_only", "server": "local"}, map[string]any{"rule_set": "query", "server": "local"},
				map[string]any{"rule_set": "ip", "server": "local"},
			}},
			"route": map[string]any{"rule_set": []any{ruleset, map[string]any{"type": "local", "tag": "ip", "path": ipPath}}},
		}
		report := AnalyzeConfig(diagnosticTestJSON(t, cfg))
		requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: "dns_legacy_mode_conflict", path: "dns.rules[0].strategy"})
		requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: "dns_response_requires_match", path: "dns.rules[2].rule_set"})
	}
}

func TestAnalyzeConfigDoesNotTreatRemoteSeedAsCurrentMetadata(t *testing.T) {
	queryPath := writeDiagnosticTestFile(t, "queries.json", []byte(`{"version":5,"rules":[{"query_type":"A"}]}`))
	cfg := map[string]any{
		"dns": map[string]any{"rules": []any{map[string]any{"strategy": "ipv4_only", "server": "local"}, map[string]any{"rule_set": "query", "server": "local"}}},
		"route": map[string]any{"rule_set": []any{map[string]any{
			"type": "remote", "tag": "query", "url": "https://example.org/query.json", "initial_path": queryPath,
		}}},
	}
	report := AnalyzeConfig(diagnosticTestJSON(t, cfg))
	requireNoConfigDiagnostic(t, report.Issues, "dns_legacy_mode_conflict", "dns.rules[0].strategy")
}

func TestAnalyzeConfigFileReportsMissingUnreadableAndInvalidRuleSets(t *testing.T) {
	for _, test := range []struct{ path, code string }{
		{filepath.Join(t.TempDir(), "missing.json"), "ruleset_file_missing"},
		{t.TempDir(), "ruleset_file_unreadable"},
		{writeDiagnosticTestFile(t, "broken.json", []byte(`{"version":500,"private":"no-content-in-report"}`)), "ruleset_file_invalid"},
	} {
		cfg := map[string]any{"route": map[string]any{"rule_set": []any{map[string]any{
			"type": "local", "tag": "local", "format": "source", "path": test.path,
		}}}}
		path := writeDiagnosticTestFile(t, "config.json", diagnosticTestJSON(t, cfg))
		report := AnalyzeConfigFile(path)
		requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: test.code, path: "route.rule_set[0].path", severity: model.ConfigDiagnosticSeverityError})
		if bytes.Contains(diagnosticTestJSON(t, report), []byte("no-content-in-report")) {
			t.Fatal("rule-set content leaked into diagnostics")
		}
	}
}

func TestAnalyzeConfigFileExpandsMultipleRuleSetTags(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "present.json"), []byte(`{"version":5,"rules":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := map[string]any{"route": map[string]any{"rule_set": []any{map[string]any{
		"type": "local", "tag": []any{"present", "missing"}, "path": filepath.Join(dir, "{tag}.json"),
	}}}}
	path := writeDiagnosticTestFile(t, "config.json", diagnosticTestJSON(t, cfg))
	report := AnalyzeConfigFile(path)
	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: "ruleset_file_missing", path: "route.rule_set[0].path", value: "missing"})
	if countDiagnostics(report.Issues, "ruleset_file_missing") != 1 {
		t.Fatalf("existing rule-set marked missing: %+v", report.Issues)
	}
}

func TestDiagnosticRuleSetFileParsingAndShapes(t *testing.T) {
	for _, content := range [][]byte{nil, []byte(`{"version":0}`), []byte(`{"version":500}`)} {
		if _, err := parseDiagnosticRuleSet(content, "source"); err == nil {
			t.Fatal("invalid source ruleset accepted")
		}
		if _, err := parseDiagnosticRuleSet(content, "binary"); err == nil {
			t.Fatal("invalid binary ruleset accepted")
		}
	}
	content := []byte(`{"version":5,"rules":[{"type":"logical","mode":"or","rules":[{"query_type":"A"},{"domain":"example.org"},{"ip_cidr":"192.0.2.0/24"}]}]}`)
	parsed, err := parseDiagnosticRuleSet(content, "source")
	if err != nil {
		t.Fatal(err)
	}
	if shape := diagnosticShapeForRules(parsed.Rules); !shape.ip || !shape.other || !shape.query {
		t.Fatalf("metadata shape = %+v", shape)
	}
	report := newConfigDiagnostics()
	checkPersistedRuleSetFiles(&report, []byte(`{`))
	if len(report.Issues) != 0 {
		t.Fatalf("syntax errors belong to JSON diagnostics: %+v", report.Issues)
	}
}

func writeDiagnosticTestFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func diagnosticTestJSON(t *testing.T, value any) []byte {
	t.Helper()
	content, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func diagnosticBinaryRuleSet(t *testing.T, source string) []byte {
	t.Helper()
	var rules option.PlainRuleSetCompat
	if err := json.Unmarshal([]byte(source), &rules); err != nil {
		t.Fatal(err)
	}
	var content bytes.Buffer
	if err := srs.Write(&content, rules.Options, C.RuleSetVersionCurrent); err != nil {
		t.Fatal(err)
	}
	return content.Bytes()
}
