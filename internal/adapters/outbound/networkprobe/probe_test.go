package networkprobe_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/adapters/outbound/networkprobe"
)

func TestTCPMeasuresReachableEndpoint(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer listener.Close()
	go func() {
		for {
			connection, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			_ = connection.Close()
		}
	}()
	port := listener.Addr().(*net.TCPAddr).Port
	if port <= 0 || port > 65535 {
		t.Fatalf("listener port = %d", port)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	delay, err := networkprobe.New().TCP(ctx, "127.0.0.1", uint16(port))
	if err != nil {
		t.Fatalf("TCP() error = %v", err)
	}
	if delay <= 0 {
		t.Fatalf("TCP() delay = %v", delay)
	}
}

func TestICMPMeasuresLoopback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	delay, err := networkprobe.New().ICMP(ctx, "127.0.0.1")
	if err != nil {
		t.Skipf("ICMP loopback is unavailable in this environment: %v", err)
	}
	if delay <= 0 {
		t.Fatalf("ICMP() delay = %v", delay)
	}
}
