package core

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func vmessLink(t *testing.T, payload map[string]any) string {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return "vmess://" + base64.StdEncoding.EncodeToString(data)
}

func TestParseProxyLinkVmessUsesSingBoxFields(t *testing.T) {
	result, err := ParseProxyLink(vmessLink(t, map[string]any{
		"add": "example.com", "port": 443, "id": "uuid", "aid": 0,
		"scy": "auto", "net": "ws", "tls": "tls", "host": "cdn.example.com", "path": "/ws",
	}))
	if err != nil {
		t.Fatal(err)
	}
	config, ok := result.Config.(map[string]any)
	if !ok {
		t.Fatalf("config type = %T", result.Config)
	}
	if _, exists := config["network"]; exists {
		t.Fatal("sing-box config must not contain Clash network field")
	}
	if _, exists := config["ws-opts"]; exists {
		t.Fatal("sing-box config must not contain Clash ws-opts field")
	}
	if config["security"] != "auto" {
		t.Errorf("security = %v, want auto", config["security"])
	}
	transport, ok := config["transport"].(map[string]any)
	if !ok {
		t.Fatalf("transport = %#v", config["transport"])
	}
	if transport["type"] != "ws" || transport["path"] != "/ws" {
		t.Errorf("transport = %#v", transport)
	}
	tls, ok := config["tls"].(map[string]any)
	if !ok || tls["enabled"] != true || tls["server_name"] != "cdn.example.com" {
		t.Errorf("tls = %#v", config["tls"])
	}
}

func TestParseProxyLinkVmessAcceptsStringEncodedValues(t *testing.T) {
	result, err := ParseProxyLink(vmessLink(t, map[string]any{
		"add": "example.com", "port": "443", "id": "uuid", "aid": "0",
		"net": "ws", "tls": "tls", "host": "cdn.example.com", "path": "/ws",
		"allowInsecure": "1", "fp": "chrome",
	}))
	if err != nil {
		t.Fatal(err)
	}
	config := result.Config.(map[string]any)
	if result.Port != 443 || config["alter_id"] != 0 {
		t.Fatalf("result = %#v", result)
	}
	tls := config["tls"].(map[string]any)
	if tls["insecure"] != true {
		t.Fatalf("tls = %#v", tls)
	}
	if utls := tls["utls"].(map[string]any); utls["fingerprint"] != "chrome" {
		t.Fatalf("utls = %#v", utls)
	}
}

func TestParseProxyLinkVmessRejectsUnsupportedTransport(t *testing.T) {
	_, err := ParseProxyLink(vmessLink(t, map[string]any{
		"add": "example.com", "port": 443, "id": "uuid", "net": "kcp",
	}))
	if err == nil {
		t.Fatal("expected unsupported transport error")
	}
}

func TestParseProxyLinkVmessRejectsInvalidRequiredValues(t *testing.T) {
	for _, payload := range []map[string]any{
		{"add": "example.com", "port": "bad", "id": "uuid"},
		{"add": "example.com", "port": 70000, "id": "uuid"},
		{"add": "example.com", "port": 443, "id": "uuid", "aid": -1},
		{"add": "example.com", "port": 443, "id": "uuid", "allowInsecure": "maybe"},
		{"add": "", "port": 443, "id": "uuid"},
		{"add": "example.com", "port": 443, "id": ""},
	} {
		if _, err := ParseProxyLink(vmessLink(t, payload)); err == nil {
			t.Errorf("payload %#v unexpectedly succeeded", payload)
		}
	}
}

func TestBuildV2RayTransport(t *testing.T) {
	tests := []struct {
		name        string
		network     string
		path        string
		host        string
		serviceName string
		wantType    string
		wantNil     bool
	}{
		{name: "empty", wantNil: true},
		{name: "tcp", network: "tcp", wantNil: true},
		{name: "websocket", network: "websocket", path: "/ws", host: "cdn.example", wantType: "ws"},
		{name: "http", network: "http", path: "/h2", host: "h2.example", wantType: "http"},
		{name: "grpc explicit", network: "grpc", path: "/fallback", serviceName: "service", wantType: "grpc"},
		{name: "grpc path", network: "grpc", path: "service", wantType: "grpc"},
		{name: "quic", network: "quic", wantType: "quic"},
		{name: "httpupgrade", network: "httpupgrade", path: "/upgrade", host: "upgrade.example", wantType: "httpupgrade"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildV2RayTransport(tt.network, tt.path, tt.host, tt.serviceName)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantNil {
				if got != nil {
					t.Fatalf("transport = %#v, want nil", got)
				}
				return
			}
			if got["type"] != tt.wantType {
				t.Fatalf("transport = %#v", got)
			}
		})
	}

	if _, err := buildV2RayTransport("kcp", "", "", ""); err == nil {
		t.Fatal("expected unsupported transport error")
	}
}

func TestBuildTLSConfig(t *testing.T) {
	if got := buildTLSConfig(false, "", false, ""); got != nil {
		t.Fatalf("disabled TLS = %#v, want nil", got)
	}
	got := buildTLSConfig(true, "sni.example", true, "h2, http/1.1")
	if got["enabled"] != true || got["server_name"] != "sni.example" || got["insecure"] != true {
		t.Fatalf("TLS = %#v", got)
	}
	protocols, ok := got["alpn"].([]string)
	if !ok || len(protocols) != 2 || protocols[0] != "h2" || protocols[1] != "http/1.1" {
		t.Fatalf("ALPN = %#v", got["alpn"])
	}
}
