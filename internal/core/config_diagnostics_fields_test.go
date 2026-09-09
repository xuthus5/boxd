package core

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

func TestAnalyzeConfigReportsRemovedFieldsAsErrors(t *testing.T) {
	tests := []struct{ body, path string }{
		{`{"inbounds":[{"type":"tun","inet4_address":["172.19.0.1/30"]}]}`, "inbounds[0].inet4_address"},
		{`{"inbounds":[{"type":"tun","gso":true}]}`, "inbounds[0].gso"},
		{`{"inbounds":[{"type":"mixed","sniff":true}]}`, "inbounds[0].sniff"},
		{`{"inbounds":[{"type":"mixed","proxy_protocol":true}]}`, "inbounds[0].proxy_protocol"},
		{`{"outbounds":[{"type":"direct","override_address":"192.0.2.1"}]}`, "outbounds[0].override_address"},
		{`{"outbounds":[{"type":"wireguard"}]}`, "outbounds[0].type"},
		{`{"outbounds":[{"type":"dns"}]}`, "outbounds[0].type"},
		{`{"outbounds":[{"type":"shadowsocksr"}]}`, "outbounds[0].type"},
		{`{"outbounds":[{"type":"trojan","tls":{"enabled":true,"ech":{"pq_signature_schemes_enabled":true}}}]}`, "outbounds[0].tls.ech.pq_signature_schemes_enabled"},
		{`{"route":{"rules":[{"geoip":"cn","action":"reject"}]}}`, "route.rules[0].geoip"},
		{`{"route":{"geosite":{"path":"geosite.db"}}}`, "route.geosite"},
		{`{"dns":{"rules":[{"rule_set_ipcidr_match_source":true,"action":"reject"}]}}`, "dns.rules[0].rule_set_ipcidr_match_source"},
	}
	for _, test := range tests {
		report := AnalyzeConfig([]byte(test.body))
		requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: "removed_config_field", path: test.path, severity: model.ConfigDiagnosticSeverityError})
	}
}

func TestAnalyzeConfigReportsRemovedDNSFormatsIncludingDisabledFakeIP(t *testing.T) {
	for _, value := range []string{`{"enabled":true}`, `{"enabled":false}`, `null`} {
		report := AnalyzeConfig([]byte(`{"dns":{"fakeip":` + value + `}}`))
		requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: "legacy_dns_fakeip", path: "dns.fakeip", severity: model.ConfigDiagnosticSeverityError})
	}
	report := AnalyzeConfig([]byte(`{"dns":{"servers":[{"tag":"old","address":"local"}]}}`))
	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: "legacy_dns_server", path: "dns.servers[0]", severity: model.ConfigDiagnosticSeverityError})
}

func TestAnalyzeConfigReportsSupportedDeprecationsAsWarnings(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "dns":{"independent_cache":false},
  "experimental":{"cache_file":{"enabled":true,"store_rdrc":true}},
  "inbounds":[{"type":"trojan","tls":{"enabled":true,"acme":{"domain":["example.org"]}}}],
  "outbounds":[{"type":"direct","tag":"direct"}],
  "route":{"rule_set":[{"type":"remote","tag":"r","url":"https://example.org/r.srs","download_detour":"direct"}]}
}`))
	for _, path := range []string{"dns.independent_cache", "experimental.cache_file.store_rdrc", "inbounds[0].tls.acme", "route.rule_set[0].download_detour"} {
		requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: "deprecated_config_field", path: path, severity: model.ConfigDiagnosticSeverityWarning})
	}
}

func TestAnalyzeConfigReportsNestedActionFieldsEvenWhenZero(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "route":{"rules":[{"type":"logical","mode":"and","rules":[{"action":"","outbound":""}],"action":"reject"}]},
  "dns":{"rules":[{"type":"logical","mode":"and","rules":[{"server":"","timeout":"0s"}],"action":"reject"}]}
}`))
	for _, path := range []string{"route.rules[0].rules[0].action", "route.rules[0].rules[0].outbound", "dns.rules[0].rules[0].server", "dns.rules[0].rules[0].timeout"} {
		requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: "nested_rule_action_unsupported", path: path, severity: model.ConfigDiagnosticSeverityError})
	}
}

func TestDiagnosticStructuralWalkIgnoresCredentialsAndHeaders(t *testing.T) {
	cfg := map[string]any{
		"http_clients": []any{map[string]any{"tag": "client", "headers": map[string]any{"http_client": "private-value", "tls": map[string]any{"acme": "private-value"}}}},
	}
	report := newConfigDiagnostics()
	checkSingBox114Diagnostics(&report, cfg)
	checkDeprecatedConfigFields(&report, cfg)
	if len(report.Issues) != 0 {
		t.Fatalf("arbitrary payload keys are not configuration: %+v", report.Issues)
	}
}

func TestDiagnosticHelpersHandleMalformedObjects(t *testing.T) {
	if got := diagnosticObjects([]any{nil, false}, "x"); len(got) != 0 {
		t.Fatalf("objects = %#v", got)
	}
	for _, value := range []any{nil, false, "", []any{}, map[string]any{}, float64(0)} {
		if diagnosticFieldActive(value) {
			t.Fatalf("inactive value = %#v", value)
		}
	}
	if !diagnosticFieldActive(json.Number("1")) {
		t.Fatal("other present value should be active")
	}
	keys := make(map[string]struct{})
	collectDiagnosticJSONKeys(keys, reflect.TypeFor[*string]())
	if len(keys) != 0 {
		t.Fatalf("unexpected primitive keys = %#v", keys)
	}
}
