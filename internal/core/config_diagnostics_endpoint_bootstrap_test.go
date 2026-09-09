package core

import "testing"

func TestAnalyzeConfigDetectsVPNBootstrapCycles(t *testing.T) {
	for _, test := range []struct{ endpoint, dns string }{
		{endpoint: `{"type":"openconnect","tag":"vpn","server":"https://vpn.example.org","domain_resolver":"vpn-dns"}`, dns: "openconnect"},
		{endpoint: `{"type":"openvpn-client","tag":"vpn","servers":[{"server":"vpn.example.org","server_port":1194}],"domain_resolver":"vpn-dns"}`, dns: "openvpn"},
	} {
		report := AnalyzeConfig([]byte(`{"endpoints":[` + test.endpoint + `],"dns":{"servers":[{"type":"` + test.dns + `","tag":"vpn-dns","endpoint":"vpn"}]}}`))
		requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: "dns_dependency_cycle", path: "dns.servers[0].endpoint"})
	}
}

func TestVPNBootstrapRemoteClassification(t *testing.T) {
	for _, test := range []struct {
		server string
		want   bool
	}{
		{"", false}, {"https://192.0.2.1", false}, {"https://[2001:db8::1]", false},
		{"https://vpn.example.org", true}, {"vpn.example.org:443", true}, {"https://[invalid", false},
	} {
		if got := openConnectServerUsesDomain(test.server); got != test.want {
			t.Fatalf("server %q domain = %v, want %v", test.server, got, test.want)
		}
	}
	for _, test := range []struct {
		object map[string]any
		want   bool
	}{
		{map[string]any{"server": "vpn.example.org"}, true},
		{map[string]any{"server": "192.0.2.1"}, false},
		{map[string]any{"servers": []any{map[string]any{"server": "192.0.2.1"}}}, false},
		{map[string]any{"servers": []any{map[string]any{"server": "vpn.example.org"}}}, true},
	} {
		if got := openVPNServerUsesDomain(test.object); got != test.want {
			t.Fatalf("VPN remotes domain = %v, want %v", got, test.want)
		}
	}
}

func TestAnalyzeConfigRequiresVPNDomainResolversEvenWithDetours(t *testing.T) {
	for _, test := range []struct{ fields, path string }{
		{`"type":"openconnect","server":"https://vpn.example.org"`, "endpoints[0].server"},
		{`"type":"openvpn-client","servers":[{"server":"192.0.2.1"},{"server":"vpn.example.org"}]`, "endpoints[0].servers[1].server"},
	} {
		report := AnalyzeConfig([]byte(`{"endpoints":[{` + test.fields + `,"detour":"direct"}],"outbounds":[{"type":"direct","tag":"direct"}],"dns":{"servers":[{"type":"local","tag":"a"},{"type":"local","tag":"b"}]}}`))
		requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{code: "missing_domain_resolver", path: test.path, severity: "error"})
	}
	for _, fields := range []string{
		`"type":"openconnect","server":"https://192.0.2.1"`,
		`"type":"openvpn-client","servers":[{"server":"192.0.2.1"}]`,
	} {
		report := AnalyzeConfig([]byte(`{"endpoints":[{` + fields + `}],"dns":{"servers":[{"type":"local","tag":"a"},{"type":"local","tag":"b"}]}}`))
		if hasDiagnostic(report.Issues, "missing_domain_resolver", "") {
			t.Fatalf("IP VPN endpoint should not require resolution: %+v", report.Issues)
		}
	}
}
