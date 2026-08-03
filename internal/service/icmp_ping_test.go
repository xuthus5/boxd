package service

import (
	"context"
	"errors"
	"net"
	"testing"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func TestICMPPingDirectEcho(t *testing.T) {
	originalEcho := ICMPEcho
	t.Cleanup(func() { ICMPEcho = originalEcho })

	ICMPEcho = func(_ context.Context, _ string) (float64, error) {
		return 7.5, nil
	}

	latency, err := ICMPPing(context.Background(), "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if latency != 7.5 {
		t.Fatalf("latency = %v", latency)
	}
}

func TestICMPPingPropagatesError(t *testing.T) {
	originalEcho := ICMPEcho
	t.Cleanup(func() { ICMPEcho = originalEcho })

	sentinel := errors.New("network unreachable")
	ICMPEcho = func(_ context.Context, _ string) (float64, error) {
		return 0, sentinel
	}

	_, err := ICMPPing(context.Background(), "1.1.1.1")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v", err)
	}
}

func TestProbeICMPEchoNoPrivilegeError(t *testing.T) {
	originalListen := listenICMP
	t.Cleanup(func() { listenICMP = originalListen })
	listenICMP = func(net.IP) (string, *icmp.PacketConn, error) {
		return "", nil, errors.New("operation not permitted")
	}
	_, err := probeICMPEcho(context.Background(), "127.0.0.1")
	if !errors.Is(err, errICMPNoPrivilege) {
		t.Fatalf("err = %v, want %v", err, errICMPNoPrivilege)
	}
}

func TestICMPPingInvalidTarget(t *testing.T) {
	originalEcho := ICMPEcho
	t.Cleanup(func() { ICMPEcho = originalEcho })
	ICMPEcho = func(_ context.Context, target string) (float64, error) {
		if err := validatePingTarget(target); err != nil {
			return 0, err
		}
		return 1, nil
	}
	if _, err := ICMPPing(context.Background(), "bad;host"); err == nil {
		t.Fatal("expected error for invalid target")
	}
	if _, err := ICMPPing(context.Background(), "ok.example.com"); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePingTarget(t *testing.T) {
	tests := map[string]bool{
		"1.1.1.1":     true,
		"example.com": true,
		"":            false,
		"bad;host":    false,
		"bad host":    false,
		"bad|host":    false,
		"bad`host":    false,
		"bad$host":    false,
		"bad\nhost":   false,
	}
	for target, want := range tests {
		err := validatePingTarget(target)
		if (err == nil) != want {
			t.Fatalf("validatePingTarget(%q) err=%v want=%v", target, err, want)
		}
	}
}

func TestICMPTypeEcho(t *testing.T) {
	if v4 := icmpTypeEcho(net.ParseIP("1.1.1.1")); v4 == ipv4.ICMPTypeEcho {
		// ok
	} else {
		t.Fatalf("ipv4 echo type = %v", v4)
	}
	if v6 := icmpTypeEcho(net.ParseIP("2001:db8::1")); v6 == ipv6.ICMPTypeEchoRequest {
		// ok
	} else {
		t.Fatalf("ipv6 echo type = %v", v6)
	}
}

func TestParseICMPReplyIPv4(t *testing.T) {
	ok, err := parseICMPReply("ip4", []byte{0, 0, 0, 0, 0, 0, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected echo reply match for ip4")
	}
	// 非 echo reply 类型（如 type=3 destination unreachable）
	ok, err = parseICMPReply("ip4", []byte{3, 0, 0, 0, 0, 0, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected non-echo reply to not match")
	}
	if _, err := parseICMPReply("bad", []byte{0, 0, 0, 0}); err == nil {
		t.Fatal("expected error for unsupported network")
	}
}

func TestParseICMPReplyIPv6(t *testing.T) {
	// IPv6 echo reply type=129
	ok, err := parseICMPReply("ip6", []byte{129, 0, 0, 0, 0, 0, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected echo reply match for ip6")
	}
}

func TestParseICMPProtocol(t *testing.T) {
	if got := parseICMPProtocol("ip4"); got != 1 {
		t.Fatalf("ip4 protocol = %d", got)
	}
	if got := parseICMPProtocol("ip6"); got != 58 {
		t.Fatalf("ip6 protocol = %d", got)
	}
}

func TestPeerMatches(t *testing.T) {
	ip := net.ParseIP("1.1.1.1")
	if !peerMatches(&net.IPAddr{IP: ip}, ip) {
		t.Fatal("matching peer should match")
	}
	if peerMatches(&net.IPAddr{IP: net.ParseIP("8.8.8.8")}, ip) {
		t.Fatal("different peer should not match")
	}
	// 非 IPAddr 类型默认视为匹配
	if !peerMatches(&net.TCPAddr{IP: ip, Port: 80}, ip) {
		t.Fatal("non-IPAddr should default to match")
	}
}
