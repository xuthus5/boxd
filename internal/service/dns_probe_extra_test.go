package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestParseLegacyDNSAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		proto   string
		wantErr bool
	}{
		{name: "empty", address: "", wantErr: true},
		{name: "https scheme", address: "https://dns.example/dns-query", proto: "udp"},
		{name: "h3 scheme", address: "h3://dns.example/dns-query", proto: "udp"},
		{name: "tls scheme", address: "tls://dns.example:853", proto: "udp"},
		{name: "quic scheme", address: "quic://dns.example:853", proto: "udp"},
		{name: "tcp scheme", address: "tcp://dns.example:53", proto: "udp"},
		{name: "udp scheme", address: "udp://dns.example:53", proto: "udp"},
		{name: "hostport", address: "1.1.1.1:53", proto: "udp"},
		{name: "plain host", address: "1.1.1.1", proto: "udp"},
		{name: "invalid port", address: "1.1.1.1:99999", wantErr: true},
		{name: "https bad host", address: "https:///dns-query", wantErr: true},
		{name: "https userinfo", address: "https://user@dns.example/", wantErr: true},
		{name: "invalid colon", address: "a:b:c", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, _, _, err := parseLegacyDNSAddress(test.address, test.proto, 0, "/dns-query")
			if (err != nil) != test.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestSplitSchemeHost(t *testing.T) {
	tests := []struct {
		name     string
		hostport string
		wantErr  bool
	}{
		{name: "empty", hostport: "", wantErr: true},
		{name: "hostport", hostport: "dns.example:853"},
		{name: "bracketed", hostport: "[2001:db8::1]"},
		{name: "invalid colon", hostport: "a:b:c", wantErr: true},
		{name: "plain", hostport: "dns.example"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, _, _, err := splitSchemeHost("tls", test.hostport, 853)
			if (err != nil) != test.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestDefaultDNSPort(t *testing.T) {
	tests := map[string]int{
		"tls": 853, "quic": 853, "https": 443, "h3": 443, "udp": 53, "other": 53,
	}
	for proto, want := range tests {
		if got := defaultDNSPort(proto); got != want {
			t.Fatalf("%s = %d, want %d", proto, got, want)
		}
	}
}

func TestParseDNSProbePort(t *testing.T) {
	if _, err := parseDNSProbePort("abc"); err == nil {
		t.Fatal("expected error for non-numeric")
	}
	if _, err := parseDNSProbePort("70000"); err == nil {
		t.Fatal("expected error for out-of-range")
	}
	if _, err := parseDNSProbePort("0"); err == nil {
		t.Fatal("expected error for zero")
	}
	if got, err := parseDNSProbePort("443"); err != nil || got != 443 {
		t.Fatalf("got %d err %v", got, err)
	}
}

func TestClassifyProbeErrorMessage(t *testing.T) {
	tests := map[string]string{
		"service not available":     ProbeErrorUnavailable,
		"unsupported type":          ProbeErrorUnsupported,
		"invalid server":            ProbeErrorInvalidInput,
		"no response":               ProbeErrorNoResponse,
		"empty dns response":        ProbeErrorEmpty,
		"dns rcode servfail":        ProbeErrorDNSRcode,
		"context deadline exceeded": ProbeErrorTimeout,
		"connection refused":        ProbeErrorNetwork,
		"totally unknown failure":   ProbeErrorUnknown,
		"":                          ProbeErrorUnknown,
	}
	for msg, want := range tests {
		if got := classifyProbeErrorMessage(msg); got != want {
			t.Fatalf("%q = %q, want %q", msg, got, want)
		}
	}
}

func TestClassifyProbeErrorValue(t *testing.T) {
	if got := classifyProbeErrorValue(nil); got != "" {
		t.Fatalf("nil = %q", got)
	}
	if got := classifyProbeErrorValue(errors.New("boom")); got != "" {
		t.Fatalf("plain = %q", got)
	}
}

func TestFailedTestResult(t *testing.T) {
	result := failedTestResult("", errors.New("network down"))
	if result.ErrorCode != ProbeErrorNetwork {
		t.Fatalf("code = %q", result.ErrorCode)
	}
	result = failedTestResult("", nil)
	if result.Error != "probe failed" {
		t.Fatalf("message = %q", result.Error)
	}
}

func TestReadDNSQUICResponse(t *testing.T) {
	message := new(dns.Msg)
	message.Id = 0
	message.Rcode = dns.RcodeSuccess
	wire, err := packDNSProbeMessage(message)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 2+len(wire))
	binary.BigEndian.PutUint16(buf, uint16(len(wire)))
	copy(buf[2:], wire)

	out, err := readDNSQUICResponse(bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	if out.Id != 0 {
		t.Fatalf("id = %d", out.Id)
	}

	if _, err := readDNSQUICResponse(bytes.NewReader([]byte{0})); err == nil {
		t.Fatal("expected short read error")
	}
	empty := make([]byte, 2)
	if _, err := readDNSQUICResponse(bytes.NewReader(empty)); err == nil {
		t.Fatal("expected empty doq error")
	}
	bad := make([]byte, 4)
	binary.BigEndian.PutUint16(bad, 2)
	if _, err := readDNSQUICResponse(bytes.NewReader(bad)); err == nil {
		t.Fatal("expected unpack error")
	}
	wireWithID := packTestMessageWithID(t, 5)
	buf2 := make([]byte, 2+len(wireWithID))
	binary.BigEndian.PutUint16(buf2, uint16(len(wireWithID)))
	copy(buf2[2:], wireWithID)
	if _, err := readDNSQUICResponse(bytes.NewReader(buf2)); err == nil {
		t.Fatal("expected invalid id error")
	}
}

func packTestMessageWithID(t *testing.T, id int) []byte {
	t.Helper()
	message := new(dns.Msg)
	message.Id = uint16(id)
	message.Rcode = dns.RcodeSuccess
	wire, err := message.Pack()
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func TestNormalizeDNSProbeServer(t *testing.T) {
	tests := []struct {
		server  string
		wantErr bool
	}{
		{server: "", wantErr: true},
		{server: "with space", wantErr: true},
		{server: "bad/url", wantErr: true},
		{server: "1.1.1.1"},
		{server: "dns.example.com"},
		{server: "bad..com", wantErr: true},
		{server: "-bad.example", wantErr: true},
		{server: strings.Repeat("a", 254) + ".com", wantErr: true},
		{server: "2001:db8::1"},
		{server: "a:invalid", wantErr: true},
	}
	for _, test := range tests {
		_, err := normalizeDNSProbeServer(test.server)
		if (err != nil) != test.wantErr {
			t.Fatalf("%q err = %v, wantErr %v", test.server, err, test.wantErr)
		}
	}
}

func TestValidateDNSProbePath(t *testing.T) {
	if err := validateDNSProbePath(""); err != nil {
		t.Fatal(err)
	}
	if err := validateDNSProbePath("/dns-query"); err != nil {
		t.Fatal(err)
	}
	if err := validateDNSProbePath(strings.Repeat("a", maxDNSProbePathLength+1)); err == nil {
		t.Fatal("expected too long error")
	}
	if err := validateDNSProbePath("/bad?query=1"); err == nil {
		t.Fatal("expected query error")
	}
	if err := validateDNSProbePath("/bad\x00"); err == nil {
		t.Fatal("expected control char error")
	}
}

func TestExchangeDNSDoHErrors(t *testing.T) {
	original := newDNSHTTPClient
	t.Cleanup(func() { newDNSHTTPClient = original })
	newDNSHTTPClient = func(string, time.Duration) *http.Client {
		return &http.Client{Transport: probeRoundTripper(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("boom")
		})}
	}
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	if _, err := exchangeDNSDoH(context.Background(), message, "dns.example", 443, "/dns-query", time.Second); err == nil {
		t.Fatal("expected error")
	}
}

func TestExchangeDNSDoHNilClient(t *testing.T) {
	original := newDNSHTTPClient
	t.Cleanup(func() { newDNSHTTPClient = original })
	newDNSHTTPClient = func(string, time.Duration) *http.Client {
		return nil
	}
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	if _, err := exchangeDNSDoH(context.Background(), message, "dns.example", 443, "/dns-query", time.Second); err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestExchangeDNSHTTP3Errors(t *testing.T) {
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	if _, err := exchangeDNSHTTP3(context.Background(), message, "bad server", 443, "/dns-query", time.Second); err == nil {
		t.Fatal("expected error for invalid server")
	}
	if _, err := exchangeDNSHTTP3(context.Background(), message, "dns.example", 70000, "/dns-query", time.Second); err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestStopDNSProbeRedirect(t *testing.T) {
	if err := stopDNSProbeRedirect(nil, nil); err == nil {
		t.Fatal("expected redirect stop error")
	}
}

func TestExchangeDNSQUICPackErrors(t *testing.T) {
	if _, err := packDNSQUICFrame(nil); err == nil {
		t.Fatal("expected error for nil message")
	}
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	frame, err := packDNSQUICFrame(message)
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) < 2 {
		t.Fatalf("frame too short: %d", len(frame))
	}
}

func TestDNSProbeServiceProbeUnsupported(t *testing.T) {
	svc := &ServiceSet{dnsProbe: &DNSProbe{}}
	result, err := svc.DNSProbe().Probe(context.Background(), DNSProbeRequest{
		Type: "local", Server: "1.1.1.1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("expected failure for local type")
	}
	if result.ErrorCode != ProbeErrorUnsupported {
		t.Fatalf("code = %q", result.ErrorCode)
	}
}
