package core

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	M "github.com/sagernet/sing/common/metadata"
)

func TestSnellRuntimeTransportsVerifiedHTTPS(t *testing.T) {
	for _, versions := range [][2]int{{4, 5}, {6, 6}} {
		t.Run(strconv.Itoa(versions[0]), func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			t.Cleanup(server.Close)
			runtime := newSnellTestRuntime(t, versions)
			outbound, ok := runtime.box.Outbound().Outbound("snell-client")
			if !ok {
				t.Fatal("Snell outbound was not registered")
			}
			client := server.Client()
			transport := client.Transport.(*http.Transport).Clone()
			transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				return outbound.DialContext(ctx, network, M.ParseSocksaddr(address))
			}
			t.Cleanup(transport.CloseIdleConnections)
			client.Transport = transport
			client.Timeout = 5 * time.Second
			response, err := client.Get(server.URL)
			if err != nil {
				t.Fatalf("HTTPS through native Snell failed: %v", err)
			}
			_, readErr := io.Copy(io.Discard, response.Body)
			closeErr := response.Body.Close()
			if response.StatusCode != http.StatusNoContent || readErr != nil || closeErr != nil {
				t.Fatalf("Snell HTTPS response: status=%d read=%v close=%v", response.StatusCode, readErr, closeErr)
			}
		})
	}
}

func newSnellTestRuntime(t *testing.T, versions [2]int) realBox {
	t.Helper()
	port := snellTestPort(t)
	secret := rand.Text()
	cfg := map[string]any{
		"log": map[string]any{"disabled": true},
		"inbounds": []any{map[string]any{
			"type": "snell", "tag": "snell-server", "version": versions[1], "psk": secret,
			"listen": "127.0.0.1", "listen_port": port,
		}},
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "snell", "tag": "snell-client", "version": versions[0], "psk": secret,
				"server": "127.0.0.1", "server_port": port},
		},
		"route": map[string]any{"final": "direct"},
	}
	body, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx := include.Context(t.Context())
	var options option.Options
	if err := options.UnmarshalJSONContext(ctx, body); err != nil {
		t.Fatal(err)
	}
	runtime, err := newRealBox(box.Options{Context: ctx, Options: options})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := runtime.Close(); err != nil {
			t.Errorf("close Snell runtime: %v", err)
		}
	})
	if err := runtime.Start(); err != nil {
		t.Fatal(err)
	}
	return runtime
}

func snellTestPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}
