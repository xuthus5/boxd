package core

import (
	"net/url"
	"testing"
)

func TestDecodeUserInfoErrors(t *testing.T) {
	_, err := decodeUserInfo("%zz")
	if err == nil {
		t.Fatal("expected invalid user info error")
	}
}

func TestUserInfoValueHandlesMissingUser(t *testing.T) {
	for _, input := range []*url.URL{nil, {}} {
		value, err := userInfoValue(input)
		if err != nil {
			t.Fatal(err)
		}
		if value != "" {
			t.Fatalf("value = %q, want empty", value)
		}
	}
}

func TestParseProxyLinkDecodesUserInfoCredentials(t *testing.T) {
	tests := []struct {
		name   string
		link   string
		kind   string
		field  string
		wanted string
	}{
		{name: "trojan", link: "trojan://pass%20word@example.com:443#trojan", kind: "trojan", field: "password", wanted: "pass word"},
		{name: "vless", link: "vless://uuid%20part@example.com:443#vless", kind: "vless", field: "uuid", wanted: "uuid part"},
		{name: "hysteria2", link: "hysteria2://pass%20word@example.com:443#hysteria2", kind: "hysteria2", field: "password", wanted: "pass word"},
		{name: "wireguard", link: "wireguard://private%20key@example.com:51820#wireguard", kind: "wireguard", field: "private_key", wanted: "private key"},
		{name: "anytls", link: "anytls://pass%3Aword@example.com:443#anytls", kind: "anytls", field: "password", wanted: "pass:word"},
		{name: "anytls-plus", link: "anytls://pass+word@example.com:443#anytls-plus", kind: "anytls", field: "password", wanted: "pass+word"},
		{name: "shadowtls", link: "shadowtls://pass%20word@example.com:443#shadowtls", kind: "shadowtls", field: "password", wanted: "pass word"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseProxyLink(tt.link)
			if err != nil {
				t.Fatal(err)
			}
			if result.Type != tt.kind {
				t.Fatalf("type = %q, want %q", result.Type, tt.kind)
			}
			config, ok := result.Config.(map[string]any)
			if !ok {
				t.Fatalf("config type = %T, want map[string]any", result.Config)
			}
			if got := config[tt.field]; got != tt.wanted {
				t.Errorf("%s = %v, want %q", tt.field, got, tt.wanted)
			}
		})
	}
}
