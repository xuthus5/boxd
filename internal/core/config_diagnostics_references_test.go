package core

import (
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

type expectedConfigDiagnostic struct {
	code     string
	severity string
	path     string
	value    string
}

const preflightReferenceParityConfig = `{
  "inbounds":[{"type":"mixed","tag":"mixed","listen":"127.0.0.1","listen_port":1080}],
  "outbounds":[
    {"type":"direct","tag":"direct"},
    {
      "type":"socks","tag":"node","server":"192.0.2.1","server_port":1080,
      "detour":"missing-detour","domain_resolver":{"server":"missing-outbound-dns"}
    },
    {"type":"selector","tag":"group","outbounds":["missing-member"],"default":"missing-default"},
    {"type":"urltest","tag":"empty","outbounds":[]}
  ],
  "endpoints":[{
    "type":"wireguard","tag":"edge","address":["10.0.0.1/32"],
    "private_key":"test","detour":"missing-endpoint-detour"
  }],
  "route":{
    "final":"missing-final",
    "default_domain_resolver":"missing-default-dns",
    "rule_set":[
      {
        "tag":"remote","type":"remote","format":"binary",
        "url":"https://example.com/a.srs","download_detour":"missing-rule-set-detour"
      },
      {"type":"inline","rules":[]}
    ],
    "rules":[{
      "type":"logical","mode":"or","rules":[{
        "outbound":"missing-route-outbound","action":"resolve",
        "server":"missing-route-dns","rule_set":["missing-route-set"]
      }]
    }],
    "geoip":{"path":"geoip.db","download_detour":"missing-geoip-detour"},
    "geosite":{"path":"geosite.db","download_detour":"missing-geosite-detour"}
  },
  "dns":{
    "servers":[
      {"type":"local","tag":"duplicate"},
      {"type":"fakeip","tag":"duplicate","inet4_range":"198.18.0.0/15"},
      {
        "type":"https","tag":"remote","server":"1.1.1.1",
        "detour":"missing-dns-detour","domain_resolver":{"server":"missing-dns-resolver"}
      },
      {"tag":"legacy","address":"https://1.1.1.1/dns-query","address_resolver":"missing-address-resolver"}
    ],
    "final":"missing-dns-final",
    "rules":[{
      "type":"logical","mode":"or",
      "rules":[{"server":"missing-nested-dns","rule_set":["missing-nested-set"]}],
      "server":"missing-rule-dns","rule_set":"missing-dns-set"
    }]
  },
  "ntp":{
    "enabled":true,"server":"time.apple.com","detour":"missing-ntp-detour",
    "domain_resolver":{"server":"missing-ntp-dns"}
  },
  "experimental":{"clash_api":{
    "external_controller":"127.0.0.1:9090",
    "external_ui_download_detour":"missing-ui-detour"
  }}
}`

var preflightReferenceParityDiagnostics = []expectedConfigDiagnostic{
	{code: "duplicate_tag", path: "dns.servers[1].tag", value: "duplicate"},
	{code: "missing_tag", path: "route.rule_set[1].tag"},
	{code: "empty_group", severity: model.ConfigDiagnosticSeverityError, path: "outbounds[3].outbounds", value: "urltest"},
	{code: "unknown_outbound_reference", path: "outbounds[1].detour", value: "missing-detour"},
	{code: "unknown_outbound_reference", path: "outbounds[2].outbounds[0]", value: "missing-member"},
	{code: "unknown_outbound_reference", path: "outbounds[2].default", value: "missing-default"},
	{code: "unknown_outbound_reference", path: "endpoints[0].detour", value: "missing-endpoint-detour"},
	{code: "unknown_outbound_reference", path: "route.rule_set[0].download_detour", value: "missing-rule-set-detour"},
	{code: "unknown_outbound_reference", path: "route.geoip.download_detour", value: "missing-geoip-detour"},
	{code: "unknown_outbound_reference", path: "route.geosite.download_detour", value: "missing-geosite-detour"},
	{code: "unknown_outbound_reference", path: "dns.servers[2].detour", value: "missing-dns-detour"},
	{code: "unknown_outbound_reference", path: "ntp.detour", value: "missing-ntp-detour"},
	{
		code:  "unknown_outbound_reference",
		path:  "experimental.clash_api.external_ui_download_detour",
		value: "missing-ui-detour",
	},
	{code: "unknown_dns_reference", path: "outbounds[1].domain_resolver.server", value: "missing-outbound-dns"},
	{code: "unknown_dns_reference", path: "route.default_domain_resolver", value: "missing-default-dns"},
	{code: "unknown_dns_reference", path: "route.rules[0].rules[0].server", value: "missing-route-dns"},
	{code: "unknown_dns_reference", path: "dns.servers[2].domain_resolver.server", value: "missing-dns-resolver"},
	{code: "unknown_dns_reference", path: "dns.servers[3].address_resolver", value: "missing-address-resolver"},
	{code: "unknown_dns_reference", path: "dns.rules[0].rules[0].server", value: "missing-nested-dns"},
	{code: "unknown_dns_reference", path: "ntp.domain_resolver.server", value: "missing-ntp-dns"},
	{code: "unknown_ruleset_reference", path: "dns.rules[0].rule_set", value: "missing-dns-set"},
	{code: "unknown_ruleset_reference", path: "dns.rules[0].rules[0].rule_set[0]", value: "missing-nested-set"},
}

func TestAnalyzeConfigReportsPreflightReferenceParity(t *testing.T) {
	report := AnalyzeConfig([]byte(preflightReferenceParityConfig))
	for _, item := range preflightReferenceParityDiagnostics {
		requireConfigDiagnostic(t, report.Issues, item)
	}
}

func TestAnalyzeConfigSkipsUnaddressableRuleSetDetours(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "outbounds":[{"type":"direct","tag":"direct"}],
  "route":{"rule_set":[
    {"type":"remote","download_detour":"missing-without-tag"},
    {"tag":"duplicate","type":"inline","rules":[]},
    {"tag":"duplicate","type":"remote","download_detour":"missing-duplicate"}
  ]}
}`))

	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
		code: "missing_tag", path: "route.rule_set[0].tag",
	})
	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
		code: "duplicate_tag", path: "route.rule_set[2].tag", value: "duplicate",
	})
	requireNoConfigDiagnostic(t, report.Issues, "unknown_outbound_reference", "route.rule_set[0].download_detour")
	requireNoConfigDiagnostic(t, report.Issues, "unknown_outbound_reference", "route.rule_set[2].download_detour")
}

func TestAnalyzeConfigMatchesPreflightTagAndGroupNormalization(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "outbounds":[
    {"type":"direct","tag":"duplicate"},
    {"type":"direct","tag":" duplicate "},
    {"type":"Selector","tag":"uppercase","outbounds":[]}
  ]
}`))

	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
		code: "duplicate_tag", path: "outbounds[1].tag", value: "duplicate",
	})
	requireNoConfigDiagnostic(t, report.Issues, "empty_group", "outbounds[2].outbounds")
}

func TestAnalyzeConfigAcceptsModernReferenceForms(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "outbounds":[
    {"type":"direct","tag":"direct"},
    {
      "type":"socks","tag":"node","server":"192.0.2.1","server_port":1080,
      "detour":"direct","domain_resolver":{"server":"remote"}
    },
    {"type":"selector","tag":"group","outbounds":["direct"],"default":"direct"}
  ],
  "route":{
    "final":"group",
    "default_domain_resolver":"remote",
    "rule_set":[{
      "tag":"geo","type":"remote","format":"binary",
      "url":"https://example.com/a.srs","download_detour":"direct"
    }],
    "rules":[{
      "type":"logical","mode":"or",
      "rules":[{"outbound":"direct","action":"resolve","server":"remote","rule_set":["geo"]}]
    }]
  },
  "dns":{
    "servers":[
      {"type":"local","tag":"local"},
      {"type":"https","tag":"remote","server":"1.1.1.1","detour":"direct","domain_resolver":"local"}
    ],
    "final":"remote",
    "rules":[{"type":"logical","mode":"or","rules":[{"server":"local","rule_set":["geo"]}],"server":"remote"}]
  },
  "ntp":{"enabled":true,"server":"time.apple.com","detour":"direct","domain_resolver":{"server":"remote"}},
  "experimental":{"clash_api":{"external_controller":"127.0.0.1:9090","external_ui_download_detour":"direct"}}
}`))

	for _, issue := range report.Issues {
		switch issue.Code {
		case "duplicate_tag", "missing_tag", "empty_group", "unknown_outbound_reference",
			"unknown_dns_reference", "unknown_ruleset_reference", "invalid_group_default",
			"outbound_dependency_cycle", "dns_dependency_cycle", "invalid_dns_default",
			"multiple_fakeip_dns_servers":
			t.Fatalf("unexpected reference diagnostic = %#v", issue)
		}
	}
}

func TestAnalyzeConfigIgnoresDisabledNTPAndIncompleteEntries(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "route":{"rule_set":[null]},
  "dns":{"servers":[null]},
  "ntp":{"enabled":false,"detour":"missing-outbound","domain_resolver":"missing-dns"}
}`))
	if hasDiagnostic(report.Issues, "unknown_outbound_reference", "missing-outbound") ||
		hasDiagnostic(report.Issues, "unknown_dns_reference", "missing-dns") ||
		hasDiagnostic(report.Issues, "missing_tag", "") {
		t.Fatalf("ignored reference diagnostics = %+v", report.Issues)
	}
}

func requireConfigDiagnostic(
	t *testing.T,
	issues []model.ConfigDiagnostic,
	expected expectedConfigDiagnostic,
) {
	t.Helper()
	for _, issue := range issues {
		valueMatches := expected.value == "" || issue.Value == expected.value
		severityMatches := expected.severity == "" || issue.Severity == expected.severity
		if issue.Code == expected.code && issue.Path == expected.path && valueMatches && severityMatches {
			return
		}
	}
	t.Fatalf(
		"diagnostic %s at %s with value %q missing: %+v",
		expected.code,
		expected.path,
		expected.value,
		issues,
	)
}

func requireNoConfigDiagnostic(t *testing.T, issues []model.ConfigDiagnostic, code, path string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code && issue.Path == path {
			t.Fatalf("unexpected diagnostic %s at %s: %+v", code, path, issues)
		}
	}
}
