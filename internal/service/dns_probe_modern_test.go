package service

import (
	"strings"
	"testing"
)

func TestDNSProbeRejectsRemovedDNSFormats(t *testing.T) {
	for _, request := range []DNSProbeRequest{
		{Type: "legacy", Server: "1.1.1.1"},
		{Address: "https://dns.example/dns-query"},
		{Type: "https", Server: "dns.example", Address: "https://dns.example/dns-query"},
		{Server: "1.1.1.1"},
	} {
		result := probeDNSServer(t.Context(), request)
		if result.Success || !strings.Contains(result.Error, "unsupported legacy DNS format") {
			t.Fatalf("legacy request must fail before network access: %#v", result)
		}
	}
}

func TestDNSProbeModernHTTP3AndContextDependentTypes(t *testing.T) {
	protocol, server, port, _, err := normalizeDNSProbeTarget(DNSProbeRequest{Type: "http3", Server: "[::1]"})
	if err != nil || protocol != "h3" || server != "::1" || port != 443 {
		t.Fatalf("modern HTTP3 target: %s %s %d %v", protocol, server, port, err)
	}
	for _, kind := range []string{"resolved", "mdns", "openvpn", "openconnect"} {
		_, _, _, _, err := normalizeDNSProbeTarget(DNSProbeRequest{Type: kind})
		if err == nil || !strings.Contains(err.Error(), "not probeable") {
			t.Fatalf("context-dependent DNS %s: %v", kind, err)
		}
	}
}
