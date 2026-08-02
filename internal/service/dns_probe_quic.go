package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/miekg/dns"
	quic "github.com/sagernet/quic-go"
	"github.com/sagernet/quic-go/http3"
)

const dnsOverQUICALPN = "doq"

var newDNSProbeTLSConfig = func(serverName string) *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS13, ServerName: serverName}
}

// 可注入的 QUIC 拨号钩子，便于单测覆盖协议分支而不依赖真实网络。
var dialDNSProbeQUIC = func(ctx context.Context, addr string, tlsConf *tls.Config, conf *quic.Config) (dnsProbeQUICConn, error) {
	return quic.DialAddr(ctx, addr, tlsConf, conf)
}

// dnsProbeQUICConn 抽象 QUIC 连接所需子集。
type dnsProbeQUICConn interface {
	OpenStreamSync(ctx context.Context) (*quic.Stream, error)
	CloseWithError(code quic.ApplicationErrorCode, msg string) error
}

func exchangeDNSQUIC(ctx context.Context, msg *dns.Msg, addr, serverName string, timeout time.Duration) (*dns.Msg, error) {
	frame, err := packDNSQUICFrame(msg)
	if err != nil {
		return nil, err
	}
	tlsConfig, err := dnsProbeTLSConfig(serverName, dnsOverQUICALPN)
	if err != nil {
		return nil, err
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	connection, err := dialDNSProbeQUIC(probeCtx, addr, tlsConfig, &quic.Config{
		HandshakeIdleTimeout: timeout,
		MaxIdleTimeout:       timeout,
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = connection.CloseWithError(0, "dns probe complete") }()
	stream, err := connection.OpenStreamSync(probeCtx)
	if err != nil {
		return nil, err
	}
	defer stream.CancelRead(0)
	if _, err := io.Copy(stream, bytes.NewReader(frame)); err != nil {
		stream.CancelWrite(0)
		return nil, err
	}
	if err := stream.Close(); err != nil {
		return nil, err
	}
	return readDNSQUICResponse(stream)
}

func exchangeDNSHTTP3(ctx context.Context, msg *dns.Msg, server string, port int, path string, timeout time.Duration) (*dns.Msg, error) {
	endpoint, err := dnsHTTP3Endpoint(server, port, path)
	if err != nil {
		return nil, err
	}
	wire, err := packDNSHTTP3Message(msg)
	if err != nil {
		return nil, err
	}
	tlsConfig, err := dnsProbeTLSConfig(server, "")
	if err != nil {
		return nil, err
	}
	transport := &http3.Transport{TLSClientConfig: tlsConfig}
	defer func() { _ = transport.Close() }()
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(probeCtx, http.MethodPost, endpoint, bytes.NewReader(wire))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/dns-message")
	request.Header.Set("Accept", "application/dns-message")
	client := &http.Client{Timeout: timeout, Transport: transport, CheckRedirect: stopDNSProbeRedirect}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if response == nil || response.Body == nil {
		return nil, fmt.Errorf("doh3 response body is nil")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("doh3 status %d", response.StatusCode)
	}
	body, err := readDNSMessageBody(response.Body)
	if err != nil {
		return nil, err
	}
	return unpackDNSProbeMessage(body)
}

func dnsProbeTLSConfig(serverName, nextProto string) (*tls.Config, error) {
	tlsConfig := newDNSProbeTLSConfig(serverName)
	if tlsConfig == nil {
		return nil, fmt.Errorf("dns tls config is unavailable")
	}
	tlsConfig = tlsConfig.Clone()
	tlsConfig.MinVersion = tls.VersionTLS13
	if nextProto != "" {
		tlsConfig.NextProtos = []string{nextProto}
	}
	return tlsConfig, nil
}

func packDNSQUICFrame(msg *dns.Msg) ([]byte, error) {
	wire, err := packDNSProbeMessage(msg)
	if err != nil {
		return nil, err
	}
	if len(wire) > math.MaxUint16 {
		return nil, fmt.Errorf("doq request is too large")
	}
	frame := make([]byte, 2+len(wire))
	binary.BigEndian.PutUint16(frame, uint16(len(wire)))
	copy(frame[2:], wire)
	return frame, nil
}

func readDNSQUICResponse(reader io.Reader) (*dns.Msg, error) {
	var length [2]byte
	if _, err := io.ReadFull(reader, length[:]); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint16(length[:])
	if size == 0 {
		return nil, fmt.Errorf("empty doq response")
	}
	wire := make([]byte, int(size))
	if _, err := io.ReadFull(reader, wire); err != nil {
		return nil, err
	}
	response, err := unpackDNSProbeMessage(wire)
	if err != nil {
		return nil, err
	}
	if response.Id != 0 {
		return nil, fmt.Errorf("invalid doq response id %d", response.Id)
	}
	return response, nil
}

func packDNSHTTP3Message(msg *dns.Msg) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("dns message is required")
	}
	message := msg.Copy()
	message.Id = 0
	message.Compress = true
	return message.Pack()
}

func packDNSProbeMessage(msg *dns.Msg) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("dns message is required")
	}
	message := msg.Copy()
	message.Id = 0
	return message.Pack()
}

func unpackDNSProbeMessage(wire []byte) (*dns.Msg, error) {
	message := new(dns.Msg)
	if err := message.Unpack(wire); err != nil {
		return nil, err
	}
	return message, nil
}

func dnsHTTP3Endpoint(server string, port int, path string) (string, error) {
	server, err := normalizeDNSProbeServer(server)
	if err != nil {
		return "", err
	}
	port, err = normalizeDNSProbePort("h3", port)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if err := validateDNSProbePath(path); err != nil {
		return "", err
	}
	return (&url.URL{Scheme: "https", Host: joinHostPort(server, port), Path: path}).String(), nil
}

func stopDNSProbeRedirect(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}
