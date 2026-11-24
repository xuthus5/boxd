// Package networkprobe implements host-level reachability checks for proxy
// endpoints. It intentionally does not depend on the proxy engine.
package networkprobe

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/xuthus5/boxd/control"
)

const defaultTimeout = 5 * time.Second

// Prober implements control.NodeProber with ICMP echo and TCP connect probes.
type Prober struct{}

var _ control.NodeProber = (*Prober)(nil)

// New constructs a stateless network prober.
func New() *Prober { return &Prober{} }

// ICMP measures an ICMP echo round trip to host.
func (p *Prober) ICMP(ctx context.Context, host string) (time.Duration, error) {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "" {
		return 0, errors.New("ping host is required")
	}
	if ip := net.ParseIP(host); ip != nil {
		return probeICMP(ctx, ip)
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return 0, fmt.Errorf("resolve %s: %w", host, err)
	}
	var lastErr error
	for _, address := range addresses {
		delay, probeErr := probeICMP(ctx, address.IP)
		if probeErr == nil {
			return delay, nil
		}
		lastErr = probeErr
	}
	if lastErr == nil {
		return 0, fmt.Errorf("resolve %s: no addresses", host)
	}
	return 0, lastErr
}

// TCP measures the time required to establish a TCP connection to host:port.
func (p *Prober) TCP(ctx context.Context, host string, port uint16) (time.Duration, error) {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "" || port == 0 {
		return 0, errors.New("tcp ping requires a host and port")
	}
	probeCtx, cancel := withTimeout(ctx)
	defer cancel()
	started := time.Now()
	connection, err := (&net.Dialer{}).DialContext(probeCtx, "tcp", net.JoinHostPort(host, strconv.Itoa(int(port))))
	if err != nil {
		return 0, err
	}
	defer connection.Close()
	return time.Since(started), nil
}

func probeICMP(ctx context.Context, ip net.IP) (time.Duration, error) {
	probeCtx, cancel := withTimeout(ctx)
	defer cancel()

	network := "udp6"
	listenAddress := "::"
	protocol := 58
	echoType := icmp.Type(ipv6.ICMPTypeEchoRequest)
	replyType := icmp.Type(ipv6.ICMPTypeEchoReply)
	destination := &net.UDPAddr{IP: ip}
	if ipv4Address := ip.To4(); ipv4Address != nil {
		network = "udp4"
		listenAddress = "0.0.0.0"
		protocol = 1
		echoType = ipv4.ICMPTypeEcho
		replyType = ipv4.ICMPTypeEchoReply
		destination = &net.UDPAddr{IP: ipv4Address}
	}

	connection, err := icmp.ListenPacket(network, listenAddress)
	if err != nil {
		return 0, fmt.Errorf("open ICMP socket: %w", err)
	}
	defer connection.Close()
	if deadline, ok := probeCtx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	}

	sequence := rand.IntN(1 << 15)
	message := icmp.Message{
		Type: echoType,
		Code: 0,
		Body: &icmp.Echo{
			ID:   0,
			Seq:  sequence,
			Data: []byte("boxd-probe"),
		},
	}
	payload, err := message.Marshal(nil)
	if err != nil {
		return 0, fmt.Errorf("build ICMP request: %w", err)
	}
	started := time.Now()
	if _, err := connection.WriteTo(payload, destination); err != nil {
		return 0, fmt.Errorf("send ICMP request: %w", err)
	}

	buffer := make([]byte, 1500)
	for {
		length, _, readErr := connection.ReadFrom(buffer)
		if readErr != nil {
			if probeCtx.Err() != nil {
				return 0, probeCtx.Err()
			}
			return 0, fmt.Errorf("read ICMP reply: %w", readErr)
		}
		reply, parseErr := icmp.ParseMessage(protocol, buffer[:length])
		if parseErr != nil || reply.Type != replyType {
			continue
		}
		echo, ok := reply.Body.(*icmp.Echo)
		if !ok || echo.Seq != sequence {
			continue
		}
		return time.Since(started), nil
	}
}

func withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < defaultTimeout {
		return context.WithDeadline(ctx, deadline)
	}
	return context.WithTimeout(ctx, defaultTimeout)
}
