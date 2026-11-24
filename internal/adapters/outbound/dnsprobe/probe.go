// Package dnsprobe performs active DNS endpoint checks from the control-plane
// host. It deliberately does not use the proxy engine or synthesize latency.
package dnsprobe

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/xuthus5/boxd/control"
)

const (
	defaultProbeTimeout = 5 * time.Second
	probeName           = "example.com."
)

// Prober implements control.DNSProber with real UDP, DoH, and DoT requests.
type Prober struct{}

var _ control.DNSProber = (*Prober)(nil)

func New() *Prober { return &Prober{} }

func (p *Prober) Probe(ctx context.Context, server control.DNSServerConfig) (time.Duration, error) {
	timeout := time.Duration(server.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultProbeTimeout
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	query, requestID, err := buildQuery()
	if err != nil {
		return 0, err
	}
	startedAt := time.Now()
	var response []byte
	switch server.Protocol {
	case control.DNSProtocolUDP:
		response, err = probeUDP(probeCtx, server.Address, query)
	case control.DNSProtocolDoH:
		response, err = probeDoH(probeCtx, server.Address, query)
	case control.DNSProtocolDoT:
		response, err = probeDoT(probeCtx, server.Address, query)
	default:
		return 0, fmt.Errorf("unsupported DNS protocol %q", server.Protocol)
	}
	if err != nil {
		return 0, err
	}
	if err := validateResponse(response, requestID); err != nil {
		return 0, err
	}
	return time.Since(startedAt), nil
}

func buildQuery() ([]byte, uint16, error) {
	var idBytes [2]byte
	if _, err := rand.Read(idBytes[:]); err != nil {
		return nil, 0, fmt.Errorf("generate DNS query ID: %w", err)
	}
	requestID := binary.BigEndian.Uint16(idBytes[:])
	message := dnsmessage.Message{
		Header: dnsmessage.Header{
			ID:               requestID,
			RecursionDesired: true,
		},
		Questions: []dnsmessage.Question{{
			Name:  dnsmessage.MustNewName(probeName),
			Type:  dnsmessage.TypeA,
			Class: dnsmessage.ClassINET,
		}},
	}
	body, err := message.Pack()
	if err != nil {
		return nil, 0, fmt.Errorf("pack DNS query: %w", err)
	}
	return body, requestID, nil
}

func validateResponse(body []byte, requestID uint16) error {
	var message dnsmessage.Message
	if err := message.Unpack(body); err != nil {
		return fmt.Errorf("parse DNS response: %w", err)
	}
	if !message.Response {
		return errors.New("DNS endpoint returned a query instead of a response")
	}
	if message.ID != requestID {
		return fmt.Errorf("DNS response ID %d does not match query ID %d", message.ID, requestID)
	}
	if message.RCode != dnsmessage.RCodeSuccess && message.RCode != dnsmessage.RCodeNameError {
		return fmt.Errorf("DNS endpoint returned %s", message.RCode.String())
	}
	return nil
}

func probeUDP(ctx context.Context, address string, query []byte) ([]byte, error) {
	host, port, err := parseHostPort(address, 53)
	if err != nil {
		return nil, err
	}
	connection, err := (&net.Dialer{}).DialContext(ctx, "udp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, fmt.Errorf("connect UDP DNS: %w", err)
	}
	defer connection.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	}
	if _, err := connection.Write(query); err != nil {
		return nil, fmt.Errorf("send UDP DNS query: %w", err)
	}
	response := make([]byte, 64<<10)
	length, err := connection.Read(response)
	if err != nil {
		return nil, fmt.Errorf("read UDP DNS response: %w", err)
	}
	return response[:length], nil
}

func probeDoH(ctx context.Context, address string, query []byte) ([]byte, error) {
	endpoint, err := parseDoHURL(address)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		Proxy:                 nil,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: endpoint.Hostname(),
		},
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(query))
	if err != nil {
		return nil, fmt.Errorf("build DoH request: %w", err)
	}
	request.Header.Set("Content-Type", "application/dns-message")
	request.Header.Set("Accept", "application/dns-message")
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send DoH query: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		return nil, fmt.Errorf("DoH endpoint returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		return nil, fmt.Errorf("read DoH response: %w", err)
	}
	return body, nil
}

func probeDoT(ctx context.Context, address string, query []byte) ([]byte, error) {
	host, port, err := parseHostPort(address, 853)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{}
	raw, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, fmt.Errorf("connect DoT endpoint: %w", err)
	}
	defer raw.Close()
	connection := tls.Client(raw, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host})
	if err := connection.HandshakeContext(ctx); err != nil {
		return nil, fmt.Errorf("establish DoT TLS session: %w", err)
	}
	defer connection.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	}
	frame := make([]byte, 2+len(query))
	binary.BigEndian.PutUint16(frame[:2], uint16(len(query)))
	copy(frame[2:], query)
	if _, err := connection.Write(frame); err != nil {
		return nil, fmt.Errorf("send DoT query: %w", err)
	}
	var lengthPrefix [2]byte
	if _, err := io.ReadFull(connection, lengthPrefix[:]); err != nil {
		return nil, fmt.Errorf("read DoT response length: %w", err)
	}
	length := int(binary.BigEndian.Uint16(lengthPrefix[:]))
	if length <= 0 || length > 64<<10 {
		return nil, fmt.Errorf("invalid DoT response length %d", length)
	}
	response := make([]byte, length)
	if _, err := io.ReadFull(connection, response); err != nil {
		return nil, fmt.Errorf("read DoT response: %w", err)
	}
	return response, nil
}

func parseDoHURL(address string) (*url.URL, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, errors.New("DoH address is empty")
	}
	if !strings.Contains(address, "://") {
		address = "https://" + address
	}
	parsed, err := url.Parse(address)
	if err != nil {
		return nil, fmt.Errorf("parse DoH address: %w", err)
	}
	if parsed.Hostname() == "" {
		return nil, errors.New("DoH address has no host")
	}
	if parsed.Scheme != "https" {
		return nil, fmt.Errorf("DoH requires https, got %q", parsed.Scheme)
	}
	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = "/dns-query"
	}
	return parsed, nil
}

func parseHostPort(raw string, defaultPort uint16) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", errors.New("DNS address is empty")
	}
	if index := strings.Index(raw, "://"); index >= 0 {
		raw = raw[index+3:]
	}
	raw = strings.TrimSuffix(raw, "/")
	if host, port, err := net.SplitHostPort(raw); err == nil {
		if host == "" || port == "" {
			return "", "", fmt.Errorf("invalid DNS address %q", raw)
		}
		return strings.Trim(host, "[]"), port, nil
	}
	host := strings.Trim(raw, "[]")
	if host == "" {
		return "", "", fmt.Errorf("invalid DNS address %q", raw)
	}
	return host, strconv.Itoa(int(defaultPort)), nil
}
