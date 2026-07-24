package api

import (
	"strings"
	"testing"
)

func TestNormalizeDNSProbeServerValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{name: "empty", raw: ""},
		{name: "leading whitespace", raw: " dns.example"},
		{name: "path", raw: "dns.example/path"},
		{name: "bad bracket", raw: "[::1"},
		{name: "ipv4", raw: "1.1.1.1", want: "1.1.1.1", ok: true},
		{name: "bracketed ipv6", raw: "[::1]", want: "::1", ok: true},
		{name: "hostname", raw: "dns.example", want: "dns.example", ok: true},
		{name: "trailing dot", raw: "dns.example.", want: "dns.example.", ok: true},
		{name: "empty label", raw: "dns..example"},
		{name: "leading hyphen", raw: "-dns.example"},
		{name: "trailing hyphen", raw: "dns-.example"},
		{name: "invalid character", raw: "dns_example"},
		{name: "too long label", raw: strings.Repeat("a", 64) + ".example"},
		{name: "too long host", raw: strings.Repeat("a.", 127) + "example"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeDNSProbeServer(test.raw)
			if test.ok {
				if err != nil || got != test.want {
					t.Fatalf("got %q err=%v, want %q", got, err, test.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error, got %q", got)
			}
		})
	}
}

func TestDNSProbePortPathAndDomainValidation(t *testing.T) {
	t.Parallel()
	if port, err := normalizeDNSProbePort("udp", 0); err != nil || port != 53 {
		t.Fatalf("default port = %d err=%v", port, err)
	}
	for _, port := range []int{-1, 65536} {
		if _, err := normalizeDNSProbePort("udp", port); err == nil {
			t.Fatalf("port %d should fail", port)
		}
	}
	if port, err := parseDNSProbePort("853"); err != nil || port != 853 {
		t.Fatalf("parsed port = %d err=%v", port, err)
	}
	for _, raw := range []string{"bad", "0", "65536"} {
		if _, err := parseDNSProbePort(raw); err == nil {
			t.Fatalf("port %q should fail", raw)
		}
	}
	for _, path := range []string{"", "/dns-query", "dns-query"} {
		if err := validateDNSProbePath(path); err != nil {
			t.Fatalf("path %q error = %v", path, err)
		}
	}
	for _, path := range []string{"/dns-query?x=1", "/dns-query#fragment", strings.Repeat("x", maxDNSProbePathLength+1), "/dns\x00query"} {
		if err := validateDNSProbePath(path); err == nil {
			t.Fatalf("path %q should fail", path)
		}
	}
	for _, domain := range []string{"", "example.com"} {
		if err := validateDNSProbeDomain(domain); err != nil {
			t.Fatalf("domain %q error = %v", domain, err)
		}
	}
	for _, domain := range []string{"example/com", "example.com?x", "example com", "example\x00.com", strings.Repeat("a", maxDNSProbeDomainLength+1)} {
		if err := validateDNSProbeDomain(domain); err == nil {
			t.Fatalf("domain %q should fail", domain)
		}
	}
}
