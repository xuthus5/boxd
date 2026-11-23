package nodelink

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/xuthus5/boxd/control"
)

func TestDecodeCommonShareLinks(t *testing.T) {
	ctx := context.Background()
	codec := New()
	ssUser := base64.RawURLEncoding.EncodeToString([]byte("aes-128-gcm:secret"))
	ssLegacy := base64.RawURLEncoding.EncodeToString([]byte("aes-128-gcm:secret@1.2.3.4:8388"))
	vmessBody, err := json.Marshal(map[string]any{
		"v": "2", "ps": "VMess Node", "add": "vmess.example.com", "port": "443",
		"id": "9b1c4d6e-0000-0000-0000-000000000001", "aid": "0", "scy": "auto",
		"net": "ws", "host": "cdn.example.com", "path": "/ws", "tls": "tls", "sni": "vmess.example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	vmessLink := "vmess://" + base64.RawStdEncoding.EncodeToString(vmessBody)
	tests := []struct {
		name     string
		link     string
		protocol string
		server   string
		port     uint16
	}{
		{"ss-sip002", "ss://" + ssUser + "@1.2.3.4:8388#SS%20Node", "shadowsocks", "1.2.3.4", 8388},
		{"ss-legacy", "ss://" + ssLegacy + "#Legacy", "shadowsocks", "1.2.3.4", 8388},
		{"vmess", vmessLink, "vmess", "vmess.example.com", 443},
		{"vless", "vless://9b1c4d6e-0000-0000-0000-000000000002@vless.example.com:443?security=tls&sni=vless.example.com&type=ws&path=%2Fws&host=cdn.example.com#VLESS", "vless", "vless.example.com", 443},
		{"trojan", "trojan://secret@trojan.example.com:443?sni=trojan.example.com#Trojan", "trojan", "trojan.example.com", 443},
		{"hysteria2", "hysteria2://secret@hy2.example.com:443?sni=hy2.example.com&obfs=salamander&obfs-password=obfs#HY2", "hysteria2", "hy2.example.com", 443},
		{"socks5", "socks5://user:pass@socks.example.com:1080#SOCKS", "socks", "socks.example.com", 1080},
		{"http", "https://user:pass@http.example.com:8080#HTTP", "http", "http.example.com", 8080},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			node, err := codec.Decode(ctx, test.link)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if node.Protocol != test.protocol || node.Server != test.server || node.Port != test.port {
				t.Fatalf("Decode() = %+v", node)
			}
			if node.Name == "" {
				t.Fatalf("Decode() name is empty: %+v", node)
			}
		})
	}
}

func TestCodecRoundTrip(t *testing.T) {
	ctx := context.Background()
	codec := New()
	nodes := []control.Node{
		{
			Name: "SS Node", Protocol: "shadowsocks", Server: "1.2.3.4", Port: 8388,
			Options: map[string]any{"method": "aes-128-gcm", "password": "secret"},
		},
		{
			Name: "VMess Node", Protocol: "vmess", Server: "vmess.example.com", Port: 443,
			Options: map[string]any{
				"uuid": "9b1c4d6e-0000-0000-0000-000000000001", "security": "auto",
				"tls":       map[string]any{"enabled": true, "server_name": "vmess.example.com"},
				"transport": map[string]any{"type": "ws", "path": "/ws", "headers": map[string]any{"Host": "cdn.example.com"}},
			},
		},
		{
			Name: "VLESS Node", Protocol: "vless", Server: "vless.example.com", Port: 443,
			Options: map[string]any{
				"uuid": "9b1c4d6e-0000-0000-0000-000000000002", "flow": "xtls-rprx-vision",
				"tls":       map[string]any{"enabled": true, "server_name": "vless.example.com", "utls": map[string]any{"enabled": true, "fingerprint": "chrome"}},
				"transport": map[string]any{"type": "grpc", "service_name": "grpc"},
			},
		},
		{
			Name: "Trojan Node", Protocol: "trojan", Server: "trojan.example.com", Port: 443,
			Options: map[string]any{"password": "secret", "tls": map[string]any{"enabled": true, "server_name": "trojan.example.com"}},
		},
		{
			Name: "HY2 Node", Protocol: "hysteria2", Server: "hy2.example.com", Port: 443,
			Options: map[string]any{"password": "secret", "tls": map[string]any{"enabled": true, "server_name": "hy2.example.com"}, "obfs": map[string]any{"type": "salamander", "password": "obfs"}},
		},
		{
			Name: "SOCKS Node", Protocol: "socks", Server: "socks.example.com", Port: 1080,
			Options: map[string]any{"username": "user", "password": "pass"},
		},
		{
			Name: "HTTP Node", Protocol: "http", Server: "http.example.com", Port: 8080,
			Options: map[string]any{"username": "user", "password": "pass", "tls": map[string]any{"enabled": true}},
		},
	}
	for _, node := range nodes {
		t.Run(node.Protocol, func(t *testing.T) {
			link, err := codec.Encode(ctx, node)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			decoded, err := codec.Decode(ctx, link)
			if err != nil {
				t.Fatalf("Decode(Encode()) error = %v, link = %s", err, link)
			}
			if decoded.Protocol != node.Protocol || decoded.Server != node.Server || decoded.Port != node.Port || decoded.Name != node.Name {
				t.Fatalf("round trip = %+v, want %+v", decoded, node)
			}
			for _, key := range []string{"method", "password", "uuid", "flow", "username"} {
				if optionString(node.Options, key) != optionString(decoded.Options, key) {
					t.Fatalf("round trip option %s = %q, want %q", key, optionString(decoded.Options, key), optionString(node.Options, key))
				}
			}
		})
	}
}

func TestKernelNodeCodecFeature(t *testing.T) {
	ctx := context.Background()
	codec := New()
	userInfo := base64.RawURLEncoding.EncodeToString([]byte("aes-128-gcm:secret"))
	spec, err := codec.DecodeNodeLink(ctx, "ss://"+userInfo+"@127.0.0.1:8388#Kernel%20Node")
	if err != nil {
		t.Fatalf("DecodeNodeLink() error = %v", err)
	}
	if spec.Protocol != "shadowsocks" || spec.Server != "127.0.0.1" || spec.Port != 8388 || spec.Metadata["options"] == "" {
		t.Fatalf("DecodeNodeLink() = %+v", spec)
	}
	link, err := codec.EncodeNodeLink(ctx, spec)
	if err != nil {
		t.Fatalf("EncodeNodeLink() error = %v", err)
	}
	decoded, err := codec.Decode(ctx, link)
	if err != nil || decoded.Protocol != "shadowsocks" || decoded.Name != "Kernel Node" {
		t.Fatalf("Decode(EncodeNodeLink()) = %+v, %v", decoded, err)
	}
}
