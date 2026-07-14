package core

import (
	"net/url"
	"strings"
	"testing"
)

func TestParseProxyLinkVmess(t *testing.T) {
	link := "vmess://eyJhZGQiOiIxLjIuMy40IiwicG9ydCI6NDQzLCJpZCI6InV1aWQxMjMiLCJhaWQiOjAsIm5ldCI6InRjcCIsInRscyI6InRscyIsImhvc3QiOiJleGFtcGxlLmNvbSIsInBhdGgiOiIvIiwicHMiOiJ2bWVzcy10ZXN0In0="
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag != "vmess-test" {
		t.Errorf("expected 'vmess-test', got '%s'", result.Tag)
	}
	if result.Type != "vmess" {
		t.Errorf("expected 'vmess', got '%s'", result.Type)
	}
	if result.Server != "1.2.3.4" {
		t.Errorf("expected '1.2.3.4', got '%s'", result.Server)
	}
	if result.Port != 443 {
		t.Errorf("expected 443, got %d", result.Port)
	}
}

func TestParseProxyLinkSS(t *testing.T) {
	// ss://method:password@server:port
	link := "ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ=@192.168.1.1:8080#ss-test"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	if result.Type != "shadowsocks" {
		t.Errorf("expected 'shadowsocks', got '%s'", result.Type)
	}
	if result.Server != "192.168.1.1" {
		t.Errorf("expected '192.168.1.1', got '%s'", result.Server)
	}
	if result.Tag != "ss-test" {
		t.Errorf("expected 'ss-test', got '%s'", result.Tag)
	}
}

func TestParseProxyLinkTrojan(t *testing.T) {
	link := "trojan://password@example.com:443?security=tls#trojan-test"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag != "trojan-test" {
		t.Errorf("expected 'trojan-test', got '%s'", result.Tag)
	}
	if result.Type != "trojan" {
		t.Errorf("expected 'trojan', got '%s'", result.Type)
	}
	if result.Server != "example.com" {
		t.Errorf("expected 'example.com', got '%s'", result.Server)
	}
	if result.Port != 443 {
		t.Errorf("expected 443, got %d", result.Port)
	}
}

func TestParseProxyLinkVless(t *testing.T) {
	link := "vless://uuid@example.com:443?security=reality&sni=example.com&pbk=publickey&sid=shortid&type=tcp&flow=xtls-rprx-vision#vless-test"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag != "vless-test" {
		t.Errorf("expected 'vless-test', got '%s'", result.Tag)
	}
	if result.Type != "vless" {
		t.Errorf("expected 'vless', got '%s'", result.Type)
	}
}

func TestParseProxyLinkHysteria2(t *testing.T) {
	link := "hysteria2://password@example.com:443/?insecure=1#hy2-test"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag != "hy2-test" {
		t.Errorf("expected 'hy2-test', got '%s'", result.Tag)
	}
	if result.Type != "hysteria2" {
		t.Errorf("expected 'hysteria2', got '%s'", result.Type)
	}
	if result.Server != "example.com" {
		t.Errorf("expected 'example.com', got '%s'", result.Server)
	}
}

func TestParseProxyLinkUnsupported(t *testing.T) {
	_, err := ParseProxyLink("unknown://something")
	if err == nil {
		t.Fatal("expected error for unsupported scheme")
	}
}

func TestParseProxyLinkInvalid(t *testing.T) {
	_, err := ParseProxyLink("not-a-valid-url-at-all")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestParseProxyLinkSSR(t *testing.T) {
	link := "ssr://example.com:1234"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	if result.Type != "shadowsocksr" {
		t.Errorf("expected 'shadowsocksr', got '%s'", result.Type)
	}
}

func TestNodeName(t *testing.T) {
	tests := []struct {
		url      string
		fallback string
		expected string
	}{
		{"https://example.com#myname", "fallback", "myname"},
		{"https://example.com#my%20name", "fallback", "my name"},
		{"https://example.com", "fallback", "fallback"},
	}
	for _, tt := range tests {
		u, _ := url.Parse(tt.url)
		result := nodeName(u, tt.fallback)
		if result != tt.expected {
			t.Errorf("nodeName(%q) = %q, want %q", tt.url, result, tt.expected)
		}
	}
}

// ---- 新增：WireGuard / TUIC / AnyTLS / ShadowTLS 链接导入测试 ----

func TestParseProxyLinkWireGuard(t *testing.T) {
	link := "wireguard://cGx3dGVzdC1wcml2YXRlLWtleQ==@10.0.0.1:51820?public_key=peer pubkey&address=10.0.0.2/32&mtu=1280#wg-test"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag != "wg-test" {
		t.Errorf("tag = %q, want wg-test", result.Tag)
	}
	if result.Type != "wireguard" {
		t.Errorf("type = %q, want wireguard", result.Type)
	}
	if result.Server != "10.0.0.1" {
		t.Errorf("server = %q, want 10.0.0.1", result.Server)
	}
	if result.Port != 51820 {
		t.Errorf("port = %d, want 51820", result.Port)
	}
}

func TestParseProxyLinkTUIC(t *testing.T) {
	link := "tuic://uuid-here:password%20here@example.com:443?congestion_control=bbr&udp_relay_mode=quic&sni=example.com&alpn=h3#tuic-test"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag != "tuic-test" {
		t.Errorf("tag = %q, want tuic-test", result.Tag)
	}
	if result.Type != "tuic" {
		t.Errorf("type = %q, want tuic", result.Type)
	}
	if result.Server != "example.com" {
		t.Errorf("server = %q, want example.com", result.Server)
	}
	if result.Port != 443 {
		t.Errorf("port = %d, want 443", result.Port)
	}
}

func TestParseProxyLinkAnyTLS(t *testing.T) {
	link := "anytls://password%20here@example.com:443?sni=example.com&insecure=0#anytls-test"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag != "anytls-test" {
		t.Errorf("tag = %q, want anytls-test", result.Tag)
	}
	if result.Type != "anytls" {
		t.Errorf("type = %q, want anytls", result.Type)
	}
	if result.Server != "example.com" {
		t.Errorf("server = %q, want example.com", result.Server)
	}
	if result.Port != 443 {
		t.Errorf("port = %d, want 443", result.Port)
	}
}

func TestParseProxyLinkShadowTLS(t *testing.T) {
	link := "shadowtls://password%20here@example.com:443?version=3&sni=example.com#stls-test"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag != "stls-test" {
		t.Errorf("tag = %q, want stls-test", result.Tag)
	}
	if result.Type != "shadowtls" {
		t.Errorf("type = %q, want shadowtls", result.Type)
	}
	if result.Server != "example.com" {
		t.Errorf("server = %q, want example.com", result.Server)
	}
	if result.Port != 443 {
		t.Errorf("port = %d, want 443", result.Port)
	}
}

func TestParseProxyLinkWireGuardMissingKey(t *testing.T) {
	_, err := ParseProxyLink("wireguard://@10.0.0.1:51820#wg")
	if err == nil || !strings.Contains(err.Error(), "missing private key") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseProxyLinkTUICMissingUUID(t *testing.T) {
	_, err := ParseProxyLink("tuic://:password@example.com:443#tuic")
	if err == nil || !strings.Contains(err.Error(), "missing uuid") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseProxyLinkAnyTLSMissingPassword(t *testing.T) {
	_, err := ParseProxyLink("anytls://@example.com:443#anytls")
	if err == nil || !strings.Contains(err.Error(), "missing password") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseProxyLinkShadowTLSMissingPassword(t *testing.T) {
	_, err := ParseProxyLink("shadowtls://@example.com:443#stls")
	if err == nil || !strings.Contains(err.Error(), "missing password") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseProxyLinkWireGuardDefaults(t *testing.T) {
	// 缺 port 和 address 时应用默认值
	link := "wireguard://dGVzdC1rZXk=@10.0.0.1?public_key=peerkey#wg-defaults"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	if result.Port != 51820 {
		t.Errorf("port = %d, want default 51820", result.Port)
	}
}

func TestParseProxyLinkTUICDefaults(t *testing.T) {
	// 缺 port 时应用默认 443
	link := "tuic://uuid:password@example.com#tuic-defaults"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	if result.Port != 443 {
		t.Errorf("port = %d, want default 443", result.Port)
	}
	if sni, ok := result.Config.(map[string]any)["tls"].(map[string]any)["server_name"].(string); !ok || sni != "example.com" {
		t.Errorf("expected sni default to server, got %v", result.Config)
	}
}

func TestParseProxyLinkShadowTLSDefaultVersion(t *testing.T) {
	link := "shadowtls://password@example.com:443#stls"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	if ver, ok := result.Config.(map[string]any)["version"].(int); !ok || ver != 3 {
		t.Errorf("expected default version 3, got %v", result.Config)
	}
}
