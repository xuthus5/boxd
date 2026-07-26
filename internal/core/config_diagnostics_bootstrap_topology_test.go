package core

import (
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

const explicitDNSOutboundBootstrapCycleConfig = `{
  "outbounds":[{
    "type":"socks","tag":"proxy","server":"proxy.example.com","server_port":1080,
    "domain_resolver":"remote"
  }],
  "dns":{"servers":[
    {"type":"udp","tag":"remote","server":"1.1.1.1","detour":"proxy"}
  ],"final":"remote"},
  "route":{"final":"proxy"}
}`

const defaultDNSOutboundBootstrapCycleConfig = `{
  "outbounds":[{
    "type":"socks","tag":"proxy","server":"proxy.example.com","server_port":1080
  }],
  "dns":{"servers":[
    {"type":"udp","tag":"remote","server":"1.1.1.1","detour":"proxy"}
  ],"final":"remote"},
  "route":{"final":"proxy","default_domain_resolver":"remote"}
}`

const detouredDNSOutboundBootstrapCycleConfig = `{
  "outbounds":[
    {
      "type":"socks","tag":"proxy","server":"proxy.example.com","server_port":1080,
      "detour":"underlay"
    },
    {"type":"direct","tag":"underlay","bind_interface":"lo"}
  ],
  "dns":{"servers":[
    {"type":"udp","tag":"remote","server":"1.1.1.1","detour":"proxy"}
  ],"final":"remote"},
  "route":{"final":"proxy","default_domain_resolver":"remote"}
}`

func TestAnalyzeConfigReportsDNSOutboundBootstrapCycles(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "explicit resolver", body: explicitDNSOutboundBootstrapCycleConfig},
		{name: "resolver object", body: bootstrapCycleConfig(`"domain_resolver":{"server":"remote"}`)},
		{name: "route default resolver", body: defaultDNSOutboundBootstrapCycleConfig},
		{name: "domain propagated to direct detour", body: detouredDNSOutboundBootstrapCycleConfig},
		{name: "domain propagated through selector", body: selectorDetourBootstrapCycleConfig()},
		{name: "domain propagated to wireguard endpoint", body: endpointDetourBootstrapCycleConfig("wireguard")},
		{name: "domain propagated to tailscale endpoint", body: endpointDetourBootstrapCycleConfig("tailscale")},
		{name: "single DNS fallback", body: bootstrapCycleConfig("")},
		{name: "selector dependency", body: selectorBootstrapCycleConfig()},
		{name: "wireguard endpoint", body: wireGuardBootstrapCycleConfig()},
		{name: "tailscale endpoint", body: tailscaleBootstrapCycleConfig()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := AnalyzeConfig([]byte(test.body))
			requireConfigDiagnostic(t, report.Issues, expectedConfigDiagnostic{
				code: "dns_dependency_cycle", severity: model.ConfigDiagnosticSeverityError,
				path: "dns.servers[0].detour",
			})
		})
	}
}

func TestAnalyzeConfigIgnoresNonCyclicDNSOutboundBootstrapEdges(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "IP server", body: nonCyclicBootstrapConfig("192.0.2.1", "proxy")},
		{name: "different DNS detour", body: nonCyclicBootstrapConfig("proxy.example.com", "direct")},
		{name: "IP DNS through direct", body: directDNSDetourConfig()},
		{name: "domain forwarded to proxy", body: proxyDetourWithoutLocalResolutionConfig()},
		{name: "ambiguous DNS fallback", body: ambiguousBootstrapConfig()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := AnalyzeConfig([]byte(test.body))
			requireNoConfigDiagnostic(t, report.Issues, "dns_dependency_cycle", "dns.servers[0].detour")
		})
	}
}

func TestBootstrapRemoteDomainDetection(t *testing.T) {
	tests := []struct {
		name     string
		entry    diagnosticEntry
		object   map[string]any
		expected bool
	}{
		{name: "server domain", entry: diagnosticEntry{typeName: "socks"}, object: map[string]any{"server": "proxy.example.com"}, expected: true},
		{name: "server IP", entry: diagnosticEntry{typeName: "socks"}, object: map[string]any{"server": "192.0.2.1"}},
		{name: "wireguard peer domain", entry: diagnosticEntry{typeName: "wireguard"}, object: map[string]any{"peers": []any{map[string]any{"address": "wg.example.com"}}}, expected: true},
		{name: "wireguard peer IP", entry: diagnosticEntry{typeName: "wireguard"}, object: map[string]any{"peers": []any{map[string]any{"address": "2001:db8::1"}}}},
		{name: "tailscale default", entry: diagnosticEntry{typeName: "tailscale"}, object: map[string]any{}, expected: true},
		{name: "tailscale domain", entry: diagnosticEntry{typeName: "tailscale"}, object: map[string]any{"control_url": "https://control.example.com"}, expected: true},
		{name: "tailscale IP", entry: diagnosticEntry{typeName: "tailscale"}, object: map[string]any{"control_url": "https://192.0.2.1"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := outboundHasDomainRemote(test.entry, test.object); actual != test.expected {
				t.Fatalf("outboundHasDomainRemote() = %v, want %v", actual, test.expected)
			}
		})
	}
}

func TestResolvesServerOnDetour(t *testing.T) {
	for _, test := range []struct {
		name     string
		typeName string
		expected bool
	}{
		{name: "naive", typeName: "naive", expected: true},
		{name: "tailscale", typeName: "TAILSCALE", expected: true},
		{name: "wireguard", typeName: "wireguard", expected: true},
		{name: "socks", typeName: "socks"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if actual := resolvesServerOnDetour(test.typeName); actual != test.expected {
				t.Fatalf("resolvesServerOnDetour(%q) = %v, want %v", test.typeName, actual, test.expected)
			}
		})
	}
}

func TestBootstrapDependencyGraphReachability(t *testing.T) {
	graph := bootstrapDependencyGraph{}
	graph.add("", "ignored")
	graph.add("ignored", "")
	graph.add("a", "b")
	graph.add("b", "a")
	graph.add("b", "target")

	if !graph.reaches("a", "target") {
		t.Fatal("reaches(a, target) = false, want true")
	}
	if graph.reaches("a", "missing") {
		t.Fatal("reaches(a, missing) = true, want false")
	}
}

func bootstrapCycleConfig(resolverField string) string {
	separator := ""
	if resolverField != "" {
		separator = ","
	}
	return `{
  "outbounds":[{
    "type":"socks","tag":"proxy","server":"proxy.example.com","server_port":1080` + separator + resolverField + `
  }],
  "dns":{"servers":[
    {"type":"udp","tag":"remote","server":"1.1.1.1","detour":"proxy"}
  ],"final":"remote"},
  "route":{"final":"proxy"}
}`
}

func selectorBootstrapCycleConfig() string {
	return `{
  "outbounds":[
    {"type":"selector","tag":"group","outbounds":["proxy"]},
    {"type":"socks","tag":"proxy","server":"proxy.example.com","server_port":1080,"domain_resolver":"remote"}
  ],
  "dns":{"servers":[
    {"type":"udp","tag":"remote","server":"1.1.1.1","detour":"group"}
  ],"final":"remote"},
  "route":{"final":"group"}
}`
}

func selectorDetourBootstrapCycleConfig() string {
	return `{
  "outbounds":[
    {"type":"socks","tag":"proxy","server":"proxy.example.com","server_port":1080,"detour":"group"},
    {"type":"selector","tag":"group","outbounds":["underlay"]},
    {"type":"direct","tag":"underlay","bind_interface":"lo"}
  ],
  "dns":{"servers":[{"type":"udp","tag":"remote","server":"1.1.1.1","detour":"proxy"}],"final":"remote"},
  "route":{"final":"proxy","default_domain_resolver":"remote"}
}`
}

func endpointDetourBootstrapCycleConfig(typeName string) string {
	endpoint := `{"type":"tailscale","tag":"edge","control_url":"https://192.0.2.1"}`
	if typeName == "wireguard" {
		endpoint = `{"type":"wireguard","tag":"edge","peers":[{"address":"192.0.2.2","port":51820}]}`
	}
	return `{
  "outbounds":[
    {"type":"socks","tag":"proxy","server":"proxy.example.com","server_port":1080,"detour":"edge"}
  ],
  "endpoints":[` + endpoint + `],
  "dns":{"servers":[{"type":"udp","tag":"remote","server":"1.1.1.1","detour":"proxy"}],"final":"remote"},
  "route":{"final":"proxy"}
}`
}

func wireGuardBootstrapCycleConfig() string {
	return `{
  "endpoints":[{
    "type":"wireguard","tag":"edge","peers":[{"address":"wg.example.com","port":51820}],
    "domain_resolver":"remote"
  }],
  "dns":{"servers":[{"type":"udp","tag":"remote","server":"1.1.1.1","detour":"edge"}],"final":"remote"},
  "route":{"final":"edge"}
}`
}

func tailscaleBootstrapCycleConfig() string {
	return `{
  "endpoints":[{"type":"tailscale","tag":"tail"}],
  "dns":{"servers":[{"type":"udp","tag":"remote","server":"1.1.1.1","detour":"tail"}],"final":"remote"},
  "route":{"final":"tail","default_domain_resolver":"remote"}
}`
}

func nonCyclicBootstrapConfig(server, dnsDetour string) string {
	return `{
  "outbounds":[
    {"type":"socks","tag":"proxy","server":"` + server + `","server_port":1080,"domain_resolver":"remote"},
    {"type":"direct","tag":"direct"}
  ],
  "dns":{"servers":[
    {"type":"udp","tag":"remote","server":"1.1.1.1","detour":"` + dnsDetour + `"}
  ],"final":"remote"},
  "route":{"final":"proxy"}
}`
}

func ambiguousBootstrapConfig() string {
	return `{
  "outbounds":[{
    "type":"socks","tag":"proxy","server":"proxy.example.com","server_port":1080
  }],
  "dns":{"servers":[
    {"type":"udp","tag":"remote","server":"1.1.1.1","detour":"proxy"},
    {"type":"udp","tag":"backup","server":"1.0.0.1"}
  ],"final":"remote"},
  "route":{"final":"proxy"}
}`
}

func directDNSDetourConfig() string {
	return `{
  "outbounds":[
    {"type":"socks","tag":"proxy","server":"192.0.2.2","server_port":1080},
    {"type":"direct","tag":"direct","bind_interface":"lo"}
  ],
  "dns":{"servers":[{"type":"udp","tag":"remote","server":"1.1.1.1","detour":"direct"}],"final":"remote"},
  "route":{"final":"proxy","default_domain_resolver":"remote"}
}`
}

func proxyDetourWithoutLocalResolutionConfig() string {
	return `{
  "outbounds":[
    {"type":"socks","tag":"proxy","server":"proxy.example.com","server_port":1080,"detour":"underlay"},
    {"type":"socks","tag":"underlay","server":"192.0.2.2","server_port":1080}
  ],
  "dns":{"servers":[{"type":"udp","tag":"remote","server":"1.1.1.1","detour":"proxy"}],"final":"remote"},
  "route":{"final":"proxy","default_domain_resolver":"remote"}
}`
}
