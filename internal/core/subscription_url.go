package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"
)

const subscriptionHTTPTimeout = 30 * time.Second

var (
	errSubscriptionURLInvalid = errors.New("invalid subscription URL")
	errSubscriptionURLBlocked = errors.New("subscription URL targets a private or local address")
)

var blockedSubscriptionPrefixes = [...]netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/96"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/32"),
	netip.MustParsePrefix("2001:2::/48"),
	netip.MustParsePrefix("2001:10::/28"),
	netip.MustParsePrefix("2001:20::/28"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("fec0::/10"),
}

func ValidateHTTPURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("%w: URL is required", errSubscriptionURLInvalid)
	}
	if strings.TrimSpace(rawURL) != rawURL || strings.IndexFunc(rawURL, unicode.IsSpace) >= 0 {
		return fmt.Errorf("%w: URL must not contain whitespace", errSubscriptionURLInvalid)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: %v", errSubscriptionURLInvalid, err)
	}
	if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
		return fmt.Errorf("%w: URL must use http or https", errSubscriptionURLInvalid)
	}
	if parsed.Opaque != "" || parsed.Host == "" || parsed.Hostname() == "" {
		return fmt.Errorf("%w: URL must be absolute", errSubscriptionURLInvalid)
	}
	if port := parsed.Port(); port != "" {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return fmt.Errorf("%w: port is out of range", errSubscriptionURLInvalid)
		}
	}
	return nil
}

func ValidateSubscriptionURL(rawURL string) error {
	if err := ValidateHTTPURL(rawURL); err != nil {
		return err
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if isBlockedSubscriptionHostname(parsed.Hostname()) {
		return errSubscriptionURLBlocked
	}
	if address, err := netip.ParseAddr(strings.TrimSuffix(parsed.Hostname(), ".")); err == nil && isBlockedSubscriptionAddress(address) {
		return errSubscriptionURLBlocked
	}
	return nil
}

func ValidatePublicHTTPURL(rawURL string) error {
	return ValidateSubscriptionURL(rawURL)
}

func isBlockedSubscriptionHostname(hostname string) bool {
	normalized := strings.TrimSuffix(strings.ToLower(hostname), ".")
	return normalized == "localhost" || strings.HasSuffix(normalized, ".localhost")
}

func isBlockedSubscriptionAddress(address netip.Addr) bool {
	if !address.IsValid() {
		return true
	}
	address = address.Unmap()
	if address.IsLoopback() || address.IsPrivate() || address.IsLinkLocalUnicast() || address.IsUnspecified() || address.IsMulticast() {
		return true
	}
	for _, prefix := range blockedSubscriptionPrefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func subscriptionDialControl(_ context.Context, _ string, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%w: cannot inspect dial address", errSubscriptionURLBlocked)
	}
	addressValue, err := netip.ParseAddr(strings.Trim(host, "[]"))
	if err != nil || isBlockedSubscriptionAddress(addressValue) {
		return fmt.Errorf("%w: dial address is not public", errSubscriptionURLBlocked)
	}
	return nil
}

func newSubscriptionHTTPClient() *http.Client {
	return newPublicHTTPClient(subscriptionHTTPTimeout)
}

func newPublicHTTPClient(timeout time.Duration) *http.Client {
	return newPublicHTTPClientWithTransport(timeout, newPublicHTTPTransport())
}

func newPublicHTTPTransport() http.RoundTripper {
	dialer := &net.Dialer{ControlContext: subscriptionDialControl}
	return &http.Transport{
		DialContext:         dialer.DialContext,
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     90 * time.Second,
	}
}

func newSubscriptionHTTPClientWithTransport(transport http.RoundTripper) *http.Client {
	return newPublicHTTPClientWithTransport(subscriptionHTTPTimeout, transport)
}

func newPublicHTTPClientWithTransport(timeout time.Duration, transport http.RoundTripper) *http.Client {
	if timeout <= 0 {
		timeout = subscriptionHTTPTimeout
	}
	if transport == nil {
		transport = newPublicHTTPTransport()
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			return ValidatePublicHTTPURL(req.URL.String())
		},
	}
}
