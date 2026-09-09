package core

import (
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

type dnsResponseDiagnosticCase struct {
	name    string
	rules   string
	code    string
	path    string
	warning bool
}

var dnsResponseDiagnosticCases = []dnsResponseDiagnosticCase{
	{name: "missing server", rules: `{"action":"evaluate"}`, code: "dns_missing_server", path: "dns.rules[0].server"},
	{name: "anonymous response without evaluation", rules: `{"match_response":true,"action":"respond"}`, code: "dns_evaluate_required", path: "dns.rules[0].match_response"},
	{name: "respond without evaluation", rules: `{"action":"respond"}`, code: "dns_evaluate_required", path: "dns.rules[0].action"},
	{name: "undefined evaluate tag", rules: `{"match_response":"later","action":"respond"}`, code: "unknown_dns_evaluate_reference", path: "dns.rules[0].match_response"},
	{name: "duplicate tag", rules: `{"action":"evaluate","server":"local","tag":"a"},{"action":"evaluate","server":"local","tag":"a"}`, code: "duplicate_dns_evaluate_tag", path: "dns.rules[1].tag"},
	{name: "unreferenced tag", rules: `{"action":"evaluate","server":"local","tag":"a"}`, code: "unused_dns_evaluate_tag", path: "dns.rules[0].tag", warning: true},
	{name: "overwritten response", rules: `{"action":"evaluate","server":"local"},{"action":"evaluate","server":"remote"}`, code: "dns_evaluate_overwritten", path: "dns.rules[0].action", warning: true},
	{name: "speculative without race", rules: `{"action":"route","server":"local","speculative":true}`, code: "dns_speculative_without_race", path: "dns.rules[0].speculative", warning: true},
	{name: "race missing response", rules: `{"action":"route","server":"local","race":true}`, code: "dns_race_requires_response", path: "dns.rules[0].race"},
	{name: "race not final", rules: `{"action":"evaluate","server":"local","race":true}`, code: "dns_race_invalid_action", path: "dns.rules[0].race"},
	{name: "race speculative conflict", rules: `{"action":"route","server":"local","race":true,"speculative":true}`, code: "dns_race_speculative_conflict", path: "dns.rules[0].speculative"},
	{name: "evaluate fakeip", rules: `{"action":"evaluate","server":"fake"}`, code: "dns_evaluate_fakeip", path: "dns.rules[0].server"},
	{name: "logical respond tag", rules: `{"action":"evaluate","server":"local","tag":"a"},{"type":"logical","mode":"and","rules":[{"match_response":"a"}],"action":"respond"}`, code: "dns_logical_respond_tag", path: "dns.rules[1].action"},
	{name: "client subnet conflict", rules: `{"server":"local","client_subnet":"192.0.2.0/24","remove_client_subnet":true}`, code: "dns_client_subnet_conflict", path: "dns.rules[0].remove_client_subnet"},
	{name: "response requires match", rules: `{"response_rcode":"NOERROR","server":"local"}`, code: "dns_response_requires_match", path: "dns.rules[0].response_rcode"},
	{name: "numeric rcode requires match", rules: `{"response_rcode":0,"server":"local"}`, code: "dns_response_requires_match", path: "dns.rules[0].response_rcode"},
	{name: "strategy query conflict", rules: `{"strategy":"ipv4_only","server":"local"},{"query_type":"AAAA","action":"reject"}`, code: "dns_legacy_mode_conflict", path: "dns.rules[0].strategy"},
	{name: "legacy accept empty query conflict", rules: `{"rule_set_ip_cidr_accept_empty":true,"server":"local"},{"ip_version":4,"action":"reject"}`, code: "dns_legacy_mode_conflict", path: "dns.rules[0].rule_set_ip_cidr_accept_empty"},
	{name: "address filter query conflict", rules: `{"ip_is_private":true,"server":"local"},{"query_type":"AAAA","action":"reject"}`, code: "dns_response_requires_match", path: "dns.rules[0].ip_is_private"},
	{name: "legacy strategy supported", rules: `{"strategy":"prefer_ipv4","server":"local"}`, code: "deprecated_config_field", path: "dns.rules[0].strategy", warning: true},
	{name: "legacy address filter supported", rules: `{"ip_cidr":"192.0.2.0/24","server":"local"}`, code: "deprecated_config_field", path: "dns.rules[0].ip_cidr", warning: true},
	{name: "legacy accept empty supported", rules: `{"rule_set_ip_cidr_accept_empty":true,"server":"local"}`, code: "deprecated_config_field", path: "dns.rules[0].rule_set_ip_cidr_accept_empty", warning: true},
}

func TestAnalyzeConfigDNSResponseDiagnostics(t *testing.T) {
	for _, test := range dnsResponseDiagnosticCases {
		t.Run(test.name, func(t *testing.T) {
			report := AnalyzeConfig([]byte(dnsResponseTestConfig(test.rules)))
			severity := model.ConfigDiagnosticSeverityError
			if test.warning {
				severity = model.ConfigDiagnosticSeverityWarning
			}
			requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: test.code, path: test.path, severity: severity})
		})
	}
}

func dnsResponseTestConfig(rules string) string {
	return `{
  "inbounds":[{"type":"mixed","tag":"mixed","listen":"127.0.0.1","listen_port":0}],
  "outbounds":[{"type":"direct","tag":"direct"}],
  "dns":{"servers":[
    {"type":"local","tag":"local"},
    {"type":"udp","tag":"remote","server":"192.0.2.1"},
    {"type":"fakeip","tag":"fake","inet4_range":"198.18.0.0/15"}
  ],"final":"local","rules":[` + rules + `]}
}`
}

func TestAnalyzeConfigAcceptsModernResponseFlows(t *testing.T) {
	for _, rules := range []string{
		`{"action":"evaluate","server":"local"},{"match_response":true,"ip_is_private":true,"action":"respond"}`,
		`{"action":"evaluate","server":"local","tag":"a"},{"match_response":"a","response_rcode":"NOERROR","action":"respond","race":true},{"server":"remote","speculative":true}`,
		`{"action":"evaluate","server":"local"},{"action":"respond"},{"action":"evaluate","server":"remote"},{"type":"logical","mode":"and","rules":[{"match_response":true,"response_rcode":0},{"query_type":"A"}],"action":"respond"}`,
		`{"action":"evaluate","server":"local","tag":"a"},{"type":"logical","mode":"or","rules":[{"type":"logical","mode":"and","rules":[{"match_response":"a"}]}],"action":"reject","race":true}`,
		`{"query_type":"A","strategy":"as_is","server":"local"}`,
	} {
		report := AnalyzeConfig([]byte(dnsResponseTestConfig(rules)))
		if report.Status != model.ConfigDiagnosticsHealthy {
			t.Fatalf("modern DNS flow should be healthy: %+v", report.Issues)
		}
	}
}

func TestAnalyzeConfigTaggedEvaluateDoesNotDefineAnonymousResponse(t *testing.T) {
	report := AnalyzeConfig([]byte(dnsResponseTestConfig(`{"action":"evaluate","server":"local","tag":"a"},{"match_response":true,"action":"respond"}`)))
	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
		code: "dns_evaluate_required", path: "dns.rules[1].match_response", severity: model.ConfigDiagnosticSeverityError,
	})
}
