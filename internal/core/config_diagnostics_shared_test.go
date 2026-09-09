package core

import (
	"encoding/json"
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

const diagnosticSharedReferencesConfig = `{
  "inbounds":[{"type":"trojan","tag":"in","detour":"missing-in","tls":{"enabled":true,"certificate_provider":"missing-provider"}}],
  "outbounds":[{"type":"direct","tag":"direct"}],
  "http_clients":[{"tag":"client","detour":"missing-out","domain_resolver":"missing-dns"},{"tag":"client"},{}],
  "certificate_providers":[{"type":"tailscale","tag":"cert","endpoint":"missing-endpoint"}],
  "services":[{"type":"api","tag":"api","tls":{"enabled":true,"certificate_provider":{"type":"tailscale","endpoint":"direct"}}}],
  "route":{"default_http_client":"absent","rule_set":[
    {"tag":"r","type":"remote","url":"https://example.org/r.srs","http_client":"missing-client"}
  ]},
  "dns":{"servers":[{"type":"local","tag":"local"},{"type":"openvpn","tag":"vpn-dns","endpoint":"direct"}]}
}`

func TestAnalyzeConfigReportsNewSharedReferences(t *testing.T) {
	report := AnalyzeConfig([]byte(diagnosticSharedReferencesConfig))
	for _, expected := range []expectedConfigDiagnostic{
		{code: "unknown_inbound_reference", path: "inbounds[0].detour"},
		{code: "unknown_certificate_provider_reference", path: "inbounds[0].tls.certificate_provider"},
		{code: "unknown_outbound_reference", path: "http_clients[0].detour"},
		{code: "unknown_dns_reference", path: "http_clients[0].domain_resolver"},
		{code: "duplicate_tag", path: "http_clients[1].tag"},
		{code: "missing_required_field", path: "http_clients[2].tag"},
		{code: "unknown_http_client_reference", path: "route.default_http_client"},
		{code: "unknown_http_client_reference", path: "route.rule_set[0].http_client"},
		{code: "invalid_endpoint_reference", path: "certificate_providers[0].endpoint"},
		{code: "invalid_endpoint_reference", path: "services[0].tls.certificate_provider.endpoint"},
		{code: "invalid_endpoint_reference", path: "dns.servers[1].endpoint"},
	} {
		expected.severity = model.ConfigDiagnosticSeverityError
		requireConfigDiagnostic(t, report.Issues, expected)
	}
}

func TestAnalyzeConfigAcceptsSharedAndInlineProviders(t *testing.T) {
	report := analyzeConfigForBuild([]byte(`{
  "inbounds":[{"type":"trojan","tag":"in","listen":"127.0.0.1","tls":{"enabled":true,"certificate_provider":"cert"}}],
  "outbounds":[{"type":"direct","tag":"direct"}],
  "http_clients":[{"tag":"client","detour":"direct","domain_resolver":"local"}],
  "certificate_providers":[{"type":"acme","tag":"cert","domain":["example.org"],"http_client":"client"}],
  "services":[{"type":"api","tag":"api","tls":{"enabled":true,"certificate_provider":{"type":"acme","domain":["example.org"],"http_client":{"detour":"direct"}}}}],
  "dns":{"servers":[{"type":"local","tag":"local"}]},
  "route":{"default_http_client":"client","rule_set":[{"type":"remote","tag":"r","url":"https://example.org/r.srs","http_client":"client"}]},
  "network_namespaces":[{"type":"default","tag":"existing","path":"/proc/self/ns/net"}]
}`), testKernelBuild("linux", "with_acme", false))
	if report.Status != model.ConfigDiagnosticsHealthy {
		t.Fatalf("valid shared configuration = %+v", report.Issues)
	}
}

func TestAnalyzeConfigCertificateSourceConflicts(t *testing.T) {
	for _, extra := range []string{`"acme":{"domain":["example.org"]}`, `"reality":{"enabled":true}`} {
		report := AnalyzeConfig([]byte(`{"inbounds":[{"type":"trojan","tls":{"enabled":true,"certificate_provider":{"type":"acme"},` + extra + `}}]}`))
		requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: "certificate_provider_conflict", path: "inbounds[0].tls.certificate_provider"})
	}
}

func TestAnalyzeConfigRemoteHTTPClientSelection(t *testing.T) {
	for _, test := range []struct{ extra, fields, code string }{
		{fields: ``, code: "implicit_http_client"},
		{fields: `,"http_client":{}`},
		{fields: `,"http_client":{"detour":"direct"},"download_detour":"direct"`, code: "http_client_detour_conflict"},
		{fields: `,"http_client":{"detour":"direct"}`},
		{extra: `"http_clients":[{"tag":"default","detour":"direct"}],`},
	} {
		body := `{` + test.extra + `"outbounds":[{"type":"direct","tag":"direct"}],"route":{"rule_set":[{"type":"remote","tag":"r","url":"https://example.org/r.srs"` + test.fields + `}]}}`
		report := AnalyzeConfig([]byte(body))
		if test.code != "" {
			requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: test.code, path: "route.rule_set[0].http_client"})
		} else {
			requireNoConfigDiagnostic(t, report.Issues, "implicit_http_client", "route.rule_set[0].http_client")
		}
	}
}

func TestAnalyzeConfigNetworkNamespaceValidation(t *testing.T) {
	var cfg map[string]any
	if err := json.Unmarshal([]byte(`{"network_namespaces":[{"type":"default","tag":"a"},{"type":"unshare","tag":"a"},{"type":"unshare"}]}`), &cfg); err != nil {
		t.Fatal(err)
	}
	report := newConfigDiagnostics()
	checkNetworkNamespacesForPlatform(&report, cfg, "linux")
	for _, expected := range []expectedConfigDiagnostic{
		{code: "missing_required_field", path: "network_namespaces[0].path"},
		{code: "duplicate_tag", path: "network_namespaces[1].tag"},
		{code: "missing_required_field", path: "network_namespaces[2].tag"},
	} {
		requireConfigDiagnostic(t, report.Issues, expected)
	}
	report = newConfigDiagnostics()
	checkNetworkNamespacesForPlatform(&report, cfg, "darwin")
	if got := countDiagnostics(report.Issues, "network_namespace_unsupported"); got != 3 {
		t.Fatalf("platform errors = %d, want 3", got)
	}
}

func TestAnalyzeConfigChecksDNSEndpointKindsAndUniqueness(t *testing.T) {
	report := AnalyzeConfig([]byte(`{
  "endpoints":[{"type":"openconnect","tag":"oc","server":"https://192.0.2.1"},{"type":"tailscale","tag":"ts"}],
  "dns":{"servers":[{"type":"openconnect","tag":"one","endpoint":"oc"},{"type":"openconnect","tag":"two","endpoint":"oc"},{"type":"openvpn","tag":"missing"},{"type":"tailscale","tag":"ts-dns","endpoint":"ts"}]}
}`))
	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: "duplicate_endpoint_dns", path: "dns.servers[1].endpoint"})
	requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: "missing_required_field", path: "dns.servers[2].endpoint"})
	if hasDiagnostic(report.Issues, "no_inbounds", "") || hasDiagnostic(report.Issues, "invalid_endpoint_reference", "") {
		t.Fatalf("valid endpoint references or traffic entries misdiagnosed: %+v", report.Issues)
	}
}
