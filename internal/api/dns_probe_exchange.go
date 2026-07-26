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
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func exchangeDNSUDP(ctx context.Context, msg *dns.Msg, addr string, timeout time.Duration) (*dns.Msg, error) {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client := &dns.Client{Net: "udp", Timeout: timeout}
	resp, _, err := client.ExchangeContext(probeCtx, msg, addr)
	return resp, err
}

func exchangeDNSTCP(ctx context.Context, msg *dns.Msg, addr string, timeout time.Duration) (*dns.Msg, error) {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client := &dns.Client{Net: "tcp", Timeout: timeout}
	resp, _, err := client.ExchangeContext(probeCtx, msg, addr)
	return resp, err
}

func exchangeDNSTLS(ctx context.Context, msg *dns.Msg, addr, serverName string, timeout time.Duration) (*dns.Msg, error) {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client := &dns.Client{
		Net:     "tcp-tls",
		Timeout: timeout,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: serverName,
		},
	}
	resp, _, err := client.ExchangeContext(probeCtx, msg, addr)
	return resp, err
}

func exchangeDNSDoH(ctx context.Context, msg *dns.Msg, server string, port int, path string, timeout time.Duration) (*dns.Msg, error) {
	server, err := normalizeDNSProbeServer(server)
	if err != nil {
		return nil, err
	}
	port, err = normalizeDNSProbePort("https", port)
	if err != nil {
		return nil, err
	}
	wire, err := msg.Pack()
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if err := validateDNSProbePath(path); err != nil {
		return nil, err
	}
	endpointURL := &url.URL{Scheme: "https", Host: joinHostPort(server, port), Path: path}
	endpoint := endpointURL.String()
	q := url.Values{}
	q.Set("dns", base64.RawURLEncoding.EncodeToString(wire))
	reqURL := endpoint + "?" + q.Encode()

	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(probeCtx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/dns-message")

	client := newDNSHTTPClient(server, timeout)
	if client == nil {
		return nil, fmt.Errorf("doh http client is unavailable")
	}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		if httpResp != nil && httpResp.Body != nil {
			_ = httpResp.Body.Close()
		}
		if probeCtx.Err() != nil {
			return nil, probeCtx.Err()
		}
		return exchangeDNSDoHPost(probeCtx, client, endpoint, wire)
	}
	if httpResp == nil || httpResp.Body == nil {
		return nil, fmt.Errorf("doh response body is nil")
	}
	if httpResp.StatusCode != http.StatusOK {
		_ = httpResp.Body.Close()
		if probeCtx.Err() != nil {
			return nil, probeCtx.Err()
		}
		return exchangeDNSDoHPost(probeCtx, client, endpoint, wire)
	}
	defer func() { _ = httpResp.Body.Close() }()
	body, err := readDNSMessageBody(httpResp.Body)
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
	if httpResp == nil || httpResp.Body == nil {
		return nil, fmt.Errorf("doh response body is nil")
	}
	defer func() { _ = httpResp.Body.Close() }()
	if httpResp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(httpResp.Body, 1024))
		if readErr != nil {
			return nil, fmt.Errorf("doh status %d", httpResp.StatusCode)
		}
		return nil, fmt.Errorf("doh status %d: %s", httpResp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := readDNSMessageBody(httpResp.Body)
	if err != nil {
		return nil, err
	}
	out := new(dns.Msg)
	if err := out.Unpack(body); err != nil {
		return nil, err
	}
	return out, nil
}

const maxDNSMessageBytes = 64 * 1024

func readDNSMessageBody(reader io.Reader) ([]byte, error) {
	if reader == nil {
		return nil, fmt.Errorf("doh response body is nil")
	}
	body, err := io.ReadAll(io.LimitReader(reader, maxDNSMessageBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxDNSMessageBytes {
		return nil, fmt.Errorf("doh response is too large")
	}
	return body, nil
}
