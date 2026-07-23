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

func TestClashProxyTransportAndExtraTypes(t *testing.T) {
	cases := []struct {
		name  string
		in    map[string]any
		typ   string
		check func(t *testing.T, raw map[string]any)
	}{
		{
			name: "vless-ws-reality",
			in: map[string]any{
				"name": "vless-ws", "type": "vless", "server": "1.1.1.1", "port": 443,
				"uuid": "u", "network": "ws", "tls": true, "servername": "example.com",
				"ws-opts":      map[string]any{"path": "/ws", "headers": map[string]any{"Host": "example.com"}},
				"reality-opts": map[string]any{"public-key": "pk", "short-id": "sid"},
			},
			typ: "vless",
			check: func(t *testing.T, raw map[string]any) {
				transport := raw["transport"].(map[string]any)
				if transport["type"] != "ws" || transport["path"] != "/ws" {
					t.Fatalf("transport = %#v", transport)
				}
				tls := raw["tls"].(map[string]any)
				if tls["reality"] == nil {
					t.Fatalf("tls = %#v", tls)
				}
			},
		},
		{
			name: "socks5",
			in: map[string]any{
				"name": "socks-1", "type": "socks5", "server": "2.2.2.2", "port": 1080,
				"username": "u", "password": "p",
			},
			typ: "socks",
			check: func(t *testing.T, raw map[string]any) {
				if raw["username"] != "u" || raw["version"] != "5" {
					t.Fatalf("raw = %#v", raw)
				}
			},
		},
		{
			name: "http",
			in: map[string]any{
				"name": "http-1", "type": "http", "server": "3.3.3.3", "port": 8080,
				"username": "u", "password": "p", "tls": true, "sni": "http.example",
			},
			typ: "http",
			check: func(t *testing.T, raw map[string]any) {
				if raw["tls"] == nil {
					t.Fatalf("tls missing: %#v", raw)
				}
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			out, err := clashProxyToOutbound(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if out.Type != tt.typ {
				t.Fatalf("type = %s want %s", out.Type, tt.typ)
			}
			raw, ok := out.Raw.(map[string]any)
			if !ok {
				t.Fatalf("raw type = %T", out.Raw)
			}
			tt.check(t, raw)
		})
	}
}

func TestClashHelpersExtra(t *testing.T) {
	if got := asStringSlice([]string{"a", "b"}); len(got) != 2 || got[0] != "a" {
		t.Fatalf("string slice = %#v", got)
	}
	if got := asStringSlice([]any{"x", 1, ""}); len(got) != 2 || got[1] != "1" {
		t.Fatalf("any slice = %#v", got)
	}
	if got := asStringSlice("solo"); len(got) != 1 || got[0] != "solo" {
		t.Fatalf("string = %#v", got)
	}
	if got := asStringSlice(""); got != nil {
		t.Fatalf("empty string = %#v", got)
	}
	if got := asStringSlice(12); got != nil {
		t.Fatalf("unsupported = %#v", got)
	}
}

func TestClashShadowsocksPluginAndTransports(t *testing.T) {
	ss, err := clashProxyToOutbound(map[string]any{
		"name": "ss-plugin", "type": "ss", "server": "1.1.1.1", "port": 8388,
		"cipher": "aes-128-gcm", "password": "p",
		"plugin": "v2ray-plugin", "plugin-opts": "mode=websocket",
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := ss.Raw.(map[string]any)
	if raw["plugin"] != "v2ray-plugin" || raw["plugin_opts"] != "mode=websocket" {
		t.Fatalf("plugin raw = %#v", raw)
	}

	ssMap, err := clashProxyToOutbound(map[string]any{
		"name": "ss-plugin-map", "type": "shadowsocks", "server": "1.1.1.1", "port": 8388,
		"cipher": "aes-128-gcm", "password": "p",
		"plugin": "obfs", "plugin-opts": map[string]any{"mode": "http"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if ssMap.Raw.(map[string]any)["plugin_opts"].(map[string]any)["mode"] != "http" {
		t.Fatalf("plugin map = %#v", ssMap.Raw)
	}

	grpcOut, err := clashProxyToOutbound(map[string]any{
		"name": "vmess-grpc", "type": "vmess", "server": "1.1.1.1", "port": 443,
		"uuid": "u", "network": "grpc", "grpc-service-name": "GunService",
	})
	if err != nil {
		t.Fatal(err)
	}
	transport := grpcOut.Raw.(map[string]any)["transport"].(map[string]any)
	if transport["type"] != "grpc" || transport["service_name"] != "GunService" {
		t.Fatalf("grpc transport = %#v", transport)
	}

	h2, err := clashProxyToOutbound(map[string]any{
		"name": "vmess-h2", "type": "vmess", "server": "1.1.1.1", "port": 443,
		"uuid": "u", "network": "h2", "path": "/h2", "host": "h2.example",
	})
	if err != nil {
		t.Fatal(err)
	}
	h2t := h2.Raw.(map[string]any)["transport"].(map[string]any)
	if h2t["path"] != "/h2" {
		t.Fatalf("h2 transport = %#v", h2t)
	}
}

func TestClashHysteria2AndTUICExtras(t *testing.T) {
	hy, err := clashProxyToOutbound(map[string]any{
		"name": "hy2", "type": "hy2", "server": "1.1.1.1", "port": 443,
		"password": "p", "up": 100, "down": 200, "obfs": "salamander", "obfs-password": "op",
		"sni": "hy.example", "skip-cert-verify": true, "alpn": []any{"h3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := hy.Raw.(map[string]any)
	if raw["up_mbps"] != 100 || raw["down_mbps"] != 200 {
		t.Fatalf("bandwidth = %#v", raw)
	}
	if raw["obfs"] == nil {
		t.Fatalf("obfs missing %#v", raw)
	}
	tls := raw["tls"].(map[string]any)
	if tls["insecure"] != true || tls["server_name"] != "hy.example" {
		t.Fatalf("tls = %#v", tls)
	}

	tuic, err := clashProxyToOutbound(map[string]any{
		"name": "tuic2", "type": "tuic", "server": "2.2.2.2", "port": 443,
		"uuid": "u", "password": "p", "congestion-controller": "bbr", "sni": "tuic.example",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tuic.Raw.(map[string]any)["congestion_control"] != "bbr" {
		t.Fatalf("tuic = %#v", tuic.Raw)
	}
}
