package core

import (
	"context"
	"strings"
	"testing"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"

	"github.com/xuthus5/boxd/internal/model"
)

func TestAnalyzeConfigReportsDNSDependencyCycles(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "dns":{"servers":[
    {"type":"https","tag":"modern-a","server":"1.1.1.1","domain_resolver":"modern-b"},
    {"type":"https","tag":"modern-b","server":"1.0.0.1","domain_resolver":{"server":"modern-a"}},
    {"tag":"legacy-a","address":"tls://8.8.8.8","address_resolver":"legacy-b"},
    {"tag":"legacy-b","address":"tls://8.8.4.4","address_resolver":"legacy-a"}
  ],"final":"modern-a"}
}`))

	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
		code: "dns_dependency_cycle", severity: model.ConfigDiagnosticSeverityError,
		path: "dns.servers[1].domain_resolver.server", value: "modern-a",
	})
	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
		code: "dns_dependency_cycle", severity: model.ConfigDiagnosticSeverityError,
		path: "dns.servers[3].address_resolver", value: "legacy-a",
	})
}

func TestAnalyzeConfigReportsInvalidDNSDefaults(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		path  string
		value string
	}{
		{
			name: "explicit modern fakeip",
			body: `{"dns":{"servers":[
        {"type":"local","tag":"local"},
        {"type":"fakeip","tag":"fake","inet4_range":"198.18.0.0/15"}
      ],"final":"fake"}}`,
			path:  "dns.final",
			value: "fake",
		},
		{
			name: "implicit legacy fakeip",
			body: `{"dns":{"fakeip":{"enabled":true,"inet4_range":"198.18.0.0/15"},"servers":[
        {"tag":"legacy-fake","address":"fakeip"},
        {"type":"local","tag":"local"}
      ]}}`,
			path:  "dns.servers[0].address",
			value: "legacy-fake",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := AnalyzeConfig([]byte(test.body))
			requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
				code: "invalid_dns_default", severity: model.ConfigDiagnosticSeverityError,
				path: test.path, value: test.value,
			})
		})
	}
}

func TestAnalyzeConfigReportsEveryExtraFakeIPDNSServer(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "dns":{"servers":[
    {"type":"local","tag":"local"},
    {"type":"fakeip","tag":"fake-modern","inet4_range":"198.18.0.0/15"},
    {"tag":"fake-legacy","address":"fakeip"},
    {"type":"fakeip","tag":"fake-extra","inet4_range":"198.19.0.0/16"}
  ],"fakeip":{"enabled":true,"inet4_range":"198.18.0.0/15"},"final":"local"}
}`))

	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
		code: "multiple_fakeip_dns_servers", severity: model.ConfigDiagnosticSeverityError,
		path: "dns.servers[2].address", value: "fake-legacy",
	})
	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
		code: "multiple_fakeip_dns_servers", severity: model.ConfigDiagnosticSeverityError,
		path: "dns.servers[3].type", value: "fake-extra",
	})
	if countDiagnostics(report.Issues, "multiple_fakeip_dns_servers") != 2 {
		t.Fatalf("multiple FakeIP diagnostics = %+v", report.Issues)
	}
}

func TestAnalyzeConfigIgnoresValidOrUnknownDNSTopologyEdges(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "dns":{"servers":[
    {"type":"local","tag":"local"},
    {"type":"https","tag":"remote","server":"1.1.1.1","domain_resolver":"local"},
    {"type":"fakeip","tag":"fake","inet4_range":"198.18.0.0/15"},
    {"type":"https","tag":"missing-edge","server":"8.8.8.8","domain_resolver":"absent"}
  ],"final":"remote"}
}`))

	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
		code: "unknown_dns_reference", severity: model.ConfigDiagnosticSeverityError,
		path: "dns.servers[3].domain_resolver", value: "absent",
	})
	for _, code := range []string{
		"dns_dependency_cycle",
		"invalid_dns_default",
		"multiple_fakeip_dns_servers",
	} {
		if countDiagnostics(report.Issues, code) != 0 {
			t.Fatalf("unexpected %s diagnostic: %+v", code, report.Issues)
		}
	}
}

func TestDNSTopologyDiagnosticsMatchSingBoxRuntimeErrors(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		message string
	}{
		{
			name: "dependency cycle",
			body: `{"dns":{"servers":[
        {"type":"https","tag":"a","server":"1.1.1.1","domain_resolver":"b"},
        {"type":"https","tag":"b","server":"1.0.0.1","domain_resolver":"a"}
      ],"final":"a"}}`,
			message: "circular server dependency",
		},
		{
			name: "default fakeip",
			body: `{"dns":{"servers":[{
        "type":"fakeip","tag":"fake","inet4_range":"198.18.0.0/15"
      }]}}`,
			message: "default server cannot be fakeip",
		},
		{
			name: "multiple fakeip",
			body: `{"dns":{"servers":[
        {"type":"local","tag":"local"},
        {"type":"fakeip","tag":"fake-a","inet4_range":"198.18.0.0/16"},
        {"type":"fakeip","tag":"fake-b","inet4_range":"198.19.0.0/16"}
      ],"final":"local"}}`,
			message: "multiple fakeip server are not supported",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := singBoxDNSRuntimeError(test.body)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("runtime error = %v, want message %q", err, test.message)
			}
		})
	}
}

func singBoxDNSRuntimeError(body string) error {
	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	defer cancel()
	var options option.Options
	if err := options.UnmarshalJSONContext(ctx, []byte(body)); err != nil {
		return err
	}
	instance, err := box.New(box.Options{Context: ctx, Options: options})
	if err != nil {
		return err
	}
	if err := instance.Start(); err != nil {
		return err
	}
	return instance.Close()
}
