package core

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"
)

func TestValidateSubscriptionURL(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		wantBlocked bool
		wantErr     bool
	}{
		{name: "empty", value: "", wantErr: true},
		{name: "relative", value: "/subscriptions", wantErr: true},
		{name: "unsupported scheme", value: "ftp://example.com/feed", wantErr: true},
		{name: "missing host", value: "https:///feed", wantErr: true},
		{name: "whitespace", value: "https://example.com/a b", wantErr: true},
		{name: "invalid port", value: "https://example.com:bad/feed", wantErr: true},
		{name: "port out of range", value: "https://example.com:65536/feed", wantErr: true},
		{name: "loopback", value: "http://127.0.0.1:8080/feed", wantBlocked: true, wantErr: true},
		{name: "mapped loopback", value: "http://[::ffff:127.0.0.1]/feed", wantBlocked: true, wantErr: true},
		{name: "private IPv4", value: "https://192.168.1.20/feed", wantBlocked: true, wantErr: true},
		{name: "private IPv6", value: "https://[fd00::20]/feed", wantBlocked: true, wantErr: true},
		{name: "localhost name", value: "https://service.localhost/feed", wantBlocked: true, wantErr: true},
		{name: "shared address", value: "https://100.64.0.2/feed", wantBlocked: true, wantErr: true},
		{name: "public host", value: "https://example.com/feed?token=abc", wantErr: false},
		{name: "public IPv4", value: "https://8.8.8.8/feed", wantErr: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateSubscriptionURL(test.value)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, test.wantErr)
			}
			if test.wantBlocked && !errors.Is(err, errSubscriptionURLBlocked) {
				t.Fatalf("error = %v, want blocked URL error", err)
			}
		})
	}
}

func TestIsBlockedSubscriptionAddress(t *testing.T) {
	tests := []struct {
		value   string
		blocked bool
	}{
		{value: "0.0.0.0", blocked: true},
		{value: "0.0.0.1", blocked: true},
		{value: "127.0.0.1", blocked: true},
		{value: "10.0.0.1", blocked: true},
		{value: "169.254.169.254", blocked: true},
		{value: "198.18.0.1", blocked: true},
		{value: "2001:db8::1", blocked: true},
		{value: "64:ff9b::7f00:1", blocked: true},
		{value: "2002:7f00:1::", blocked: true},
		{value: "224.0.0.1", blocked: true},
		{value: "1.1.1.1", blocked: false},
		{value: "2606:4700:4700::1111", blocked: false},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			address, err := netip.ParseAddr(test.value)
			if err != nil {
				t.Fatal(err)
			}
			if got := isBlockedSubscriptionAddress(address); got != test.blocked {
				t.Fatalf("blocked = %v, want %v", got, test.blocked)
			}
		})
	}
	if !isBlockedSubscriptionAddress(netip.Addr{}) {
		t.Fatal("invalid address should be blocked")
	}
}

func TestSubscriptionDialControlRejectsResolvedPrivateAddresses(t *testing.T) {
	tests := []struct {
		name    string
		address string
		blocked bool
	}{
		{name: "loopback", address: "127.0.0.1:443", blocked: true},
		{name: "private", address: "10.0.0.1:443", blocked: true},
		{name: "ipv6 loopback", address: "[::1]:443", blocked: true},
		{name: "public", address: "1.1.1.1:443", blocked: false},
		{name: "malformed", address: "not-an-address", blocked: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := subscriptionDialControl(context.Background(), "tcp", test.address, nil)
			if (err != nil) != test.blocked {
				t.Fatalf("error = %v, blocked = %v", err, test.blocked)
			}
			if test.blocked && !errors.Is(err, errSubscriptionURLBlocked) {
				t.Fatalf("error = %v, want blocked URL error", err)
			}
		})
	}
	if err := subscriptionDialControl(context.Background(), "tcp", "[::1", nil); err == nil {
		t.Fatal("malformed host should be rejected")
	}
}

func TestSubscriptionHTTPClientValidatesRedirects(t *testing.T) {
	requestCount := 0
	client := newSubscriptionHTTPClientWithTransport(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		if requestCount == 1 {
			return &http.Response{
				StatusCode: http.StatusFound,
				Header:     http.Header{"Location": []string{"http://127.0.0.1/private"}},
				Body:       io.NopCloser(strings.NewReader("redirect")),
				Request:    req,
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("ok")),
			Request:    req,
		}, nil
	}))
	request, err := http.NewRequest(http.MethodGet, "https://example.com/feed", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(request)
	if !errors.Is(err, errSubscriptionURLBlocked) {
		t.Fatalf("error = %v, want blocked redirect", err)
	}
	if requestCount != 1 {
		t.Fatalf("request count = %d, want 1", requestCount)
	}
}

func TestSubscriptionHTTPClientFollowsPublicRedirect(t *testing.T) {
	requestCount := 0
	client := newSubscriptionHTTPClientWithTransport(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		if requestCount == 1 {
			return &http.Response{
				StatusCode: http.StatusFound,
				Header:     http.Header{"Location": []string{"https://example.com/next"}},
				Body:       io.NopCloser(strings.NewReader("redirect")),
				Request:    req,
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("ok")),
			Request:    req,
		}, nil
	}))
	request, err := http.NewRequest(http.MethodGet, "https://example.com/feed", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if requestCount != 2 || response.StatusCode != http.StatusOK {
		t.Fatalf("count = %d status = %d", requestCount, response.StatusCode)
	}
}

func TestPublicHTTPClientUsesSafeFallbackTransport(t *testing.T) {
	client := newPublicHTTPClientWithTransport(0, nil)
	if client == nil || client.Transport == nil {
		t.Fatalf("client = %#v", client)
	}
	if client.Timeout != subscriptionHTTPTimeout {
		t.Fatalf("timeout = %v, want %v", client.Timeout, subscriptionHTTPTimeout)
	}
}

func TestProxyAddressAllowed(t *testing.T) {
	t.Setenv("https_proxy", "http://127.0.0.1:7890")
	t.Setenv("no_proxy", "")
	if !proxyAddressAllowed("127.0.0.1:7890") {
		t.Fatal("proxy port should be allowed")
	}
	if proxyAddressAllowed("1.2.3.4:443") {
		t.Fatal("non-proxy address should be blocked")
	}
	if proxyAddressAllowed("127.0.0.1:7891") {
		t.Fatal("non-proxy port should be blocked")
	}
}

func TestProxyAddressAllowedIPv6AndSchemePort(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://[::1]:8080")
	if !proxyAddressAllowed("[::1]:8080") {
		t.Fatal("ipv6 proxy should be allowed")
	}
	t.Setenv("HTTPS_PROXY", "http://proxy.example.com")
	if !proxyAddressAllowed("proxy.example.com:80") {
		t.Fatal("scheme-default port proxy should be allowed")
	}
}

func TestProxyAddressAllowedIgnoredWithoutEnv(t *testing.T) {
	for _, key := range []string{"https_proxy", "http_proxy", "all_proxy", "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY"} {
		t.Setenv(key, "")
	}
	if proxyAddressAllowed("127.0.0.1:7890") {
		t.Fatal("proxy address must be rejected when no proxy env is set")
	}
}

func TestPublicHTTPTransportHonorsProxyFromEnvironment(t *testing.T) {
	transport, ok := newPublicHTTPTransport().(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T", newPublicHTTPTransport())
	}
	t.Setenv("https_proxy", "http://127.0.0.1:7890")
	req, err := http.NewRequest(http.MethodGet, "https://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	proxyURL, proxyErr := transport.Proxy(req)
	if proxyErr != nil {
		t.Fatal(proxyErr)
	}
	if proxyURL == nil || proxyURL.Host != "127.0.0.1:7890" {
		t.Fatalf("proxy = %v, want 127.0.0.1:7890", proxyURL)
	}
	if transport.DialContext == nil {
		t.Fatal("DialContext must be set")
	}
}

func TestPublicDialControlAllowsProxyAndBlocksPrivate(t *testing.T) {
	t.Setenv("https_proxy", "http://127.0.0.1:7890")
	ctx := context.Background()
	if err := publicDialControl(ctx, "tcp", "127.0.0.1:7890", nil); err != nil {
		t.Fatalf("proxy endpoint should dial: %v", err)
	}
	if err := publicDialControl(ctx, "tcp", "192.168.1.10:80", nil); err == nil {
		t.Fatal("private target must be rejected")
	}
}
