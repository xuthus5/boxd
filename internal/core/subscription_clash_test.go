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
