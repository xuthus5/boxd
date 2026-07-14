package core

import (
	"encoding/base64"
	"testing"
)

func TestParseVmessRawURLEncoding(t *testing.T) {
	// 使用 RawURLEncoding 编码的 vmess 链接（不含 padding）
	jsonPayload := `{"add":"1.2.3.4","port":443,"id":"uuid123","aid":0,"net":"ws","tls":"tls","host":"example.com","path":"/ws","ps":"raw-url-test"}`
	encoded := base64.RawURLEncoding.EncodeToString([]byte(jsonPayload))
	link := "vmess://" + encoded

	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag != "raw-url-test" {
		t.Errorf("tag = %q, want 'raw-url-test'", result.Tag)
	}
	if result.Server != "1.2.3.4" {
		t.Errorf("server = %q, want '1.2.3.4'", result.Server)
	}
}

func TestParseVmessInvalidBase64(t *testing.T) {
	_, err := ParseProxyLink("vmess://!!!invalid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestParseVmessInvalidJSON(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("not json"))
	_, err := ParseProxyLink("vmess://" + encoded)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseVmessEmptyPS(t *testing.T) {
	jsonPayload := `{"add":"5.6.7.8","port":8080,"id":"uuid","aid":1,"net":"tcp","tls":"","host":"","path":"","ps":""}`
	encoded := base64.StdEncoding.EncodeToString([]byte(jsonPayload))
	result, err := ParseProxyLink("vmess://" + encoded)
	if err != nil {
		t.Fatal(err)
	}
	expected := "vmess-5.6.7.8-8080"
	if result.Tag != expected {
		t.Errorf("tag = %q, want %q", result.Tag, expected)
	}
	// TLS should be false
	cfg, ok := result.Config.(map[string]any)
	if !ok {
		t.Fatal("config is not a map")
	}
	tlsVal, ok := cfg["tls"]
	if !ok {
		t.Fatal("tls field missing")
	}
	if tlsVal != false {
		t.Errorf("tls = %v, want false", tlsVal)
	}
}

func TestParseSSMissingCredentials(t *testing.T) {
	_, err := ParseProxyLink("ss://1.2.3.4:443#test")
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}
}

func TestParseSSWithPortInHost(t *testing.T) {
	// ss://method:password@server:443#tag - port in host part
	result, err := ParseProxyLink("ss://aes-256-gcm:secretpass@server.example.com:8388#my-ss-node")
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag != "my-ss-node" {
		t.Errorf("tag = %q, want 'my-ss-node'", result.Tag)
	}
	if result.Port != 8388 {
		t.Errorf("port = %d, want 8388", result.Port)
	}
}

func TestParseSSNoPort(t *testing.T) {
	result, err := ParseProxyLink("ss://aes-256-gcm:pass@server.example.com#no-port")
	if err != nil {
		t.Fatal(err)
	}
	if result.Port != 0 {
		t.Errorf("port = %d, want 0", result.Port)
	}
}

func TestParseProxyLinkUnsupportedScheme(t *testing.T) {
	_, err := ParseProxyLink("unknown://example.com")
	if err == nil {
		t.Fatal("expected error for unsupported scheme")
	}
}

func TestParseProxyLinkInvalidURL(t *testing.T) {
	_, err := ParseProxyLink("://invalid")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestNodeNameWithFragment(t *testing.T) {
	// 测试 URL fragment 解码
	result, err := ParseProxyLink("trojan://pass@1.2.3.4:443#my%20node")
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag != "my node" {
		t.Errorf("tag = %q, want 'my node'", result.Tag)
	}
}

func TestNodeNameWithRawFragment(t *testing.T) {
	// fragment 无法解码时使用原始值
	result, err := ParseProxyLink("trojan://pass@1.2.3.4:443#plain-name")
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag != "plain-name" {
		t.Errorf("tag = %q, want 'plain-name'", result.Tag)
	}
}

func TestNodeNameFallback(t *testing.T) {
	// 无 fragment 时使用 fallback
	result, err := ParseProxyLink("trojan://pass@1.2.3.4:443")
	if err != nil {
		t.Fatal(err)
	}
	expected := "trojan-1.2.3.4-443"
	if result.Tag != expected {
		t.Errorf("tag = %q, want %q", result.Tag, expected)
	}
}

func TestParseWireGuardMissingPrivateKey(t *testing.T) {
	_, err := ParseProxyLink("wireguard://@1.2.3.4:51820#test")
	if err == nil {
		t.Fatal("expected error for missing private key")
	}
}

func TestParseTUICMissingUUID(t *testing.T) {
	_, err := ParseProxyLink("tuic://:pass@1.2.3.4:443#test")
	if err == nil {
		t.Fatal("expected error for missing uuid")
	}
}

func TestParseAnyTLSMissingPassword(t *testing.T) {
	_, err := ParseProxyLink("anytls://@1.2.3.4:443#test")
	if err == nil {
		t.Fatal("expected error for missing password")
	}
}

func TestParseShadowTLSMissingPassword(t *testing.T) {
	_, err := ParseProxyLink("shadowtls://@1.2.3.4:443#test")
	if err == nil {
		t.Fatal("expected error for missing password")
	}
}

func TestParseHysteria2Link(t *testing.T) {
	result, err := ParseProxyLink("hysteria2://pass@1.2.3.4:443#hy2-test")
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag != "hy2-test" {
		t.Errorf("tag = %q, want 'hy2-test'", result.Tag)
	}
	if result.Type != "hysteria2" {
		t.Errorf("type = %q, want 'hysteria2'", result.Type)
	}
}

func TestParseHysteria2Alias(t *testing.T) {
	result, err := ParseProxyLink("hy2://pass@1.2.3.4:443#hy2-alias")
	if err != nil {
		t.Fatal(err)
	}
	if result.Type != "hysteria2" {
		t.Errorf("type = %q, want 'hysteria2'", result.Type)
	}
}

func TestParseWireGuardDefaultPort(t *testing.T) {
	result, err := ParseProxyLink("wireguard://privkey@1.2.3.4#wg-test")
	if err != nil {
		t.Fatal(err)
	}
	if result.Port != 51820 {
		t.Errorf("port = %d, want 51820 (default)", result.Port)
	}
}

func TestParseTUICDefaultPort(t *testing.T) {
	result, err := ParseProxyLink("tuic://uuid:pass@1.2.3.4#tuic-test")
	if err != nil {
		t.Fatal(err)
	}
	if result.Port != 443 {
		t.Errorf("port = %d, want 443 (default)", result.Port)
	}
}

func TestParseAnyTLSDefaultPort(t *testing.T) {
	result, err := ParseProxyLink("anytls://pass@1.2.3.4#anytls-test")
	if err != nil {
		t.Fatal(err)
	}
	if result.Port != 443 {
		t.Errorf("port = %d, want 443 (default)", result.Port)
	}
}

func TestParseShadowTLSDefaultVersion(t *testing.T) {
	result, err := ParseProxyLink("shadowtls://pass@1.2.3.4:443#stls-test")
	if err != nil {
		t.Fatal(err)
	}
	cfg, ok := result.Config.(map[string]any)
	if !ok {
		t.Fatal("config is not a map")
	}
	version, ok := cfg["version"]
	if !ok {
		t.Fatal("version field missing")
	}
	if version != 3 {
		t.Errorf("version = %v, want 3 (default)", version)
	}
}

func TestParseWireGuardWithMTU(t *testing.T) {
	link := "wireguard://privkey@1.2.3.4:51820?public_key=pubkey&address=10.0.0.2/32&mtu=1400#wg-mtu"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	cfgMap, ok := result.Config.(map[string]any)
	if !ok {
		t.Fatal("config is not a map")
	}
	mtu, ok := cfgMap["mtu"]
	if !ok {
		t.Fatal("mtu field missing")
	}
	if mtu != 1400 {
		t.Errorf("mtu = %v, want 1400", mtu)
	}
}
