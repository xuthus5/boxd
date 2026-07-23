package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/miekg/dns"
)

var newDNSHTTPClient = func(server string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			TLSHandshakeTimeout: timeout,
			TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12, ServerName: server},
		},
	}
}

func exchangeDNSUDP(msg *dns.Msg, addr string, timeout time.Duration) (*dns.Msg, error) {
	client := &dns.Client{Net: "udp", Timeout: timeout}
	resp, _, err := client.Exchange(msg, addr)
	return resp, err
}

func exchangeDNSTCP(msg *dns.Msg, addr string, timeout time.Duration) (*dns.Msg, error) {
	client := &dns.Client{Net: "tcp", Timeout: timeout}
	resp, _, err := client.Exchange(msg, addr)
	return resp, err
}

func exchangeDNSTLS(msg *dns.Msg, addr, serverName string, timeout time.Duration) (*dns.Msg, error) {
	client := &dns.Client{
		Net:     "tcp-tls",
		Timeout: timeout,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: serverName,
		},
	}
	resp, _, err := client.Exchange(msg, addr)
	return resp, err
}

func exchangeDNSDoH(msg *dns.Msg, server string, port int, path string, timeout time.Duration) (*dns.Msg, error) {
	wire, err := msg.Pack()
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	endpoint := fmt.Sprintf("https://%s%s", joinHostPort(server, port), path)
	q := url.Values{}
	q.Set("dns", base64.RawURLEncoding.EncodeToString(wire))
	reqURL := endpoint + "?" + q.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/dns-message")

	client := newDNSHTTPClient(server, timeout)
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return exchangeDNSDoHPost(ctx, client, endpoint, wire)
	}
	defer func() { _ = httpResp.Body.Close() }()
	if httpResp.StatusCode != http.StatusOK {
		return exchangeDNSDoHPost(ctx, client, endpoint, wire)
	}
	body, err := io.ReadAll(io.LimitReader(httpResp.Body, 64*1024))
	if err != nil {
		return nil, err
	}
	out := new(dns.Msg)
	if err := out.Unpack(body); err != nil {
		return nil, err
	}
	return out, nil
}

func exchangeDNSDoHPost(ctx context.Context, client *http.Client, endpoint string, wire []byte) (*dns.Msg, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(wire))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/dns-message")
	httpReq.Header.Set("Accept", "application/dns-message")
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = httpResp.Body.Close() }()
	if httpResp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(httpResp.Body, 1024))
		if readErr != nil {
			return nil, fmt.Errorf("doh status %d", httpResp.StatusCode)
		}
		return nil, fmt.Errorf("doh status %d: %s", httpResp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(io.LimitReader(httpResp.Body, 64*1024))
	if err != nil {
		return nil, err
	}
	out := new(dns.Msg)
	if err := out.Unpack(body); err != nil {
		return nil, err
	}
	return out, nil
}
