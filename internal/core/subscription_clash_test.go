package core

import "testing"

func TestParseClashSubscription(t *testing.T) {
	body := []byte(`
proxies:
  - name: "ss-node"
    type: ss
    server: 1.2.3.4
    port: 8388
    cipher: aes-256-gcm
    password: secret
  - name: "trojan-node"
    type: trojan
    server: example.com
    port: 443
    password: pass
    sni: example.com
  - name: "skip-me"
    type: ssr
    server: 9.9.9.9
    port: 1
`)
	outbounds := parseSubscriptionContent(body)
	if len(outbounds) != 2 {
		t.Fatalf("outbounds = %#v", outbounds)
	}
	if outbounds[0].Tag != "ss-node" || outbounds[0].Type != "shadowsocks" {
		t.Fatalf("first = %#v", outbounds[0])
	}
	if outbounds[1].Tag != "trojan-node" || outbounds[1].Type != "trojan" {
		t.Fatalf("second = %#v", outbounds[1])
	}
}

func TestParseClashSubscriptionInvalid(t *testing.T) {
	if got := parseClashSubscription([]byte("not: yaml: [")); got != nil {
		t.Fatalf("got %#v", got)
	}
	if got := parseClashSubscription([]byte("proxies: []")); got != nil {
		t.Fatalf("empty proxies = %#v", got)
	}
}

func TestClashProxyToOutboundProtocols(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]any
		typ  string
	}{
		{
			name: "vmess",
			in: map[string]any{
				"name": "vmess-1", "type": "vmess", "server": "1.1.1.1", "port": 443,
				"uuid": "u", "alterId": 0, "cipher": "auto", "network": "ws", "tls": true, "servername": "sni.example",
			},
			typ: "vmess",
		},
		{
			name: "vless-reality",
			in: map[string]any{
				"name": "vless-1", "type": "vless", "server": "2.2.2.2", "port": "443",
				"uuid": "u", "flow": "xtls-rprx-vision", "network": "tcp", "tls": "tls", "sni": "sni",
				"reality-opts": map[string]any{"public-key": "pk", "short-id": "sid"},
			},
			typ: "vless",
		},
		{
			name: "hysteria2",
			in: map[string]any{
				"name": "hy2-1", "type": "hysteria2", "server": "3.3.3.3", "port": 443,
				"password": "p", "sni": "hy2.example",
			},
			typ: "hysteria2",
		},
		{
			name: "tuic",
			in: map[string]any{
				"name": "tuic-1", "type": "tuic", "server": "4.4.4.4", "port": 443,
				"uuid": "u", "password": "p", "sni": "tuic.example",
			},
			typ: "tuic",
		},
		{
			name: "ss-alias",
			in: map[string]any{
				"type": "shadowsocks", "server": "5.5.5.5", "port": 8388,
				"method": "aes-128-gcm", "password": "x",
			},
			typ: "shadowsocks",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := clashProxyToOutbound(tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if out.Type != tc.typ || out.Server == "" || out.Port <= 0 {
				t.Fatalf("out = %#v", out)
			}
			if out.Tag == "" {
				t.Fatal("empty tag")
			}
		})
	}
}

func TestClashProxyToOutboundRejects(t *testing.T) {
	if _, err := clashProxyToOutbound(map[string]any{"type": "ss"}); err == nil {
		t.Fatal("expected incomplete error")
	}
	if _, err := clashProxyToOutbound(map[string]any{
		"name": "x", "type": "ssr", "server": "1.1.1.1", "port": 1,
	}); err == nil {
		t.Fatal("expected unsupported type")
	}
}

func TestClashHelpers(t *testing.T) {
	if asString("a") != "a" || asString(1) != "1" || asString(int64(2)) != "2" || asString(3.0) != "3" {
		t.Fatal("asString numbers")
	}
	if asString(true) != "true" || asString(false) != "false" || asString(struct{}{}) != "" {
		t.Fatal("asString misc")
	}
	if asInt("12") != 12 || asInt(12) != 12 || asInt(int64(13)) != 13 || asInt(14.0) != 14 || asInt(struct{}{}) != 0 {
		t.Fatal("asInt")
	}
	if !asBool(true) || !asBool("yes") || !asBool(1) || !asBool(1.0) || asBool("no") || asBool(0) {
		t.Fatal("asBool")
	}
	m := map[string]any{"a": "", "b": "ok"}
	if firstString(m, "a", "b") != "ok" || firstString(m, "x") != "" {
		t.Fatal("firstString")
	}
}
