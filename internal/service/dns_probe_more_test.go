package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestNormalizeDNSProbeTarget(t *testing.T) {
	tests := []struct {
		name    string
		req     DNSProbeRequest
		wantErr bool
	}{
		{name: "udp default port", req: DNSProbeRequest{Type: "udp", Server: "1.1.1.1"}},
		{name: "address fallback", req: DNSProbeRequest{Address: "udp://dns.example:53"}},
		{name: "not probeable", req: DNSProbeRequest{Type: "hosts", Server: "1.1.1.1"}, wantErr: true},
		{name: "unsupported", req: DNSProbeRequest{Type: "weird", Server: "1.1.1.1"}, wantErr: true},
		{name: "legacy type", req: DNSProbeRequest{Type: "legacy", Server: "1.1.1.1"}},
		{name: "missing server", req: DNSProbeRequest{Type: "udp"}, wantErr: true},
		{name: "invalid server", req: DNSProbeRequest{Type: "udp", Server: "bad server"}, wantErr: true},
		{name: "invalid port", req: DNSProbeRequest{Type: "udp", Server: "1.1.1.1", ServerPort: 70000}, wantErr: true},
		{name: "bad path", req: DNSProbeRequest{Type: "udp", Server: "1.1.1.1", Path: "/bad?q=1"}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, _, _, err := normalizeDNSProbeTarget(test.req)
			if (err != nil) != test.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestNormalizeDNSProbeTargetAddressError(t *testing.T) {
	_, _, _, _, err := normalizeDNSProbeTarget(DNSProbeRequest{Address: "tls://:bad"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExchangeDNSDoHPost(t *testing.T) {
	original := newDNSHTTPClient
	t.Cleanup(func() { newDNSHTTPClient = original })
	response := new(dns.Msg)
	response.Rcode = dns.RcodeSuccess
	wire, err := response.Pack()
	if err != nil {
		t.Fatal(err)
	}
	newDNSHTTPClient = func(string, time.Duration) *http.Client {
		return &http.Client{Transport: probeRoundTripper(func(request *http.Request) (*http.Response, error) {
			if request.Method == http.MethodPost {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(wire)), Request: request}, nil
			}
			return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader("bad")), Request: request}, nil
		})}
	}
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	out, err := exchangeDNSDoH(context.Background(), message, "dns.example", 443, "/dns-query", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if out == nil {
		t.Fatal("nil response")
	}
}

func TestExchangeDNSDoHPostErrorStatus(t *testing.T) {
	original := newDNSHTTPClient
	t.Cleanup(func() { newDNSHTTPClient = original })
	newDNSHTTPClient = func(string, time.Duration) *http.Client {
		return &http.Client{Transport: probeRoundTripper(func(request *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(strings.NewReader("upstream")), Request: request}, nil
		})}
	}
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	if _, err := exchangeDNSDoH(context.Background(), message, "dns.example", 443, "/dns-query", time.Second); err == nil {
		t.Fatal("expected error")
	}
}

func TestExchangeDNSDoHPostNilBody(t *testing.T) {
	original := newDNSHTTPClient
	t.Cleanup(func() { newDNSHTTPClient = original })
	newDNSHTTPClient = func(string, time.Duration) *http.Client {
		return &http.Client{Transport: probeRoundTripper(func(request *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: nil, Request: request}, nil
		})}
	}
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	if _, err := exchangeDNSDoH(context.Background(), message, "dns.example", 443, "/dns-query", time.Second); err == nil {
		t.Fatal("expected nil body error")
	}
}

func TestExchangeDNSDoHReadBodyError(t *testing.T) {
	original := newDNSHTTPClient
	t.Cleanup(func() { newDNSHTTPClient = original })
	newDNSHTTPClient = func(string, time.Duration) *http.Client {
		return &http.Client{Transport: probeRoundTripper(func(request *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(errReader{}), Request: request}, nil
		})}
	}
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	if _, err := exchangeDNSDoH(context.Background(), message, "dns.example", 443, "/dns-query", time.Second); err == nil {
		t.Fatal("expected read error")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read boom") }

func TestClassifyProbeError(t *testing.T) {
	if got := classifyProbeError("connection refused", nil); got != ProbeErrorNetwork {
		t.Fatalf("got %q", got)
	}
	if got := classifyProbeError("", context.DeadlineExceeded); got != ProbeErrorTimeout {
		t.Fatalf("got %q", got)
	}
	if got := classifyProbeError("timeout happened", nil); got != ProbeErrorTimeout {
		t.Fatalf("got %q", got)
	}
}

func TestExchangeDNSUDPCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	if _, err := exchangeDNSUDP(ctx, message, "127.0.0.1:53", time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}
