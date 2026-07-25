package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

func makeSSRLink(fragment string) string {
	params := url.Values{
		"obfsparam":  {base64.RawURLEncoding.EncodeToString([]byte("obfs.example"))},
		"protoparam": {base64.RawURLEncoding.EncodeToString([]byte("protocol.example"))},
		"remarks":    {base64.RawURLEncoding.EncodeToString([]byte("SSR full"))},
	}
	password := base64.RawURLEncoding.EncodeToString([]byte("secret"))
	plain := strings.Join([]string{"example.com", "443", "origin", "aes-256-cfb", "tls1.2_ticket_auth", password}, ":")
	payload := base64.RawURLEncoding.EncodeToString([]byte(plain + "/?" + params.Encode()))
	return "ssr://" + payload + fragment
}

func TestParseSSRLinkMapsSingBoxOptions(t *testing.T) {
	result, err := ParseProxyLink(makeSSRLink("#ssr-tag"))
	if err != nil {
		t.Fatal(err)
	}
	config := result.Config.(map[string]any)
	if result.Tag != "ssr-tag" || result.Server != "example.com" || result.Port != 443 {
		t.Fatalf("result = %#v", result)
	}
	if config["method"] != "aes-256-cfb" || config["password"] != "secret" {
		t.Fatalf("credentials = %#v", config)
	}
	if config["protocol"] != "origin" || config["protocol_param"] != "protocol.example" || config["obfs"] != "tls1.2_ticket_auth" || config["obfs_param"] != "obfs.example" {
		t.Fatalf("SSR options = %#v", config)
	}
}

func TestParseSSRLinkUsesRemarksAndSupportsLegacyAuthority(t *testing.T) {
	result, err := ParseProxyLink(makeSSRLink(""))
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag != "SSR full" {
		t.Fatalf("tag = %q", result.Tag)
	}
	legacy, err := ParseProxyLink("ssr://example.com:1234#legacy")
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Server != "example.com" || legacy.Port != 1234 || legacy.Tag != "legacy" {
		t.Fatalf("legacy result = %#v", legacy)
	}
}

func TestParseSSRLinkRejectsInvalidPayload(t *testing.T) {
	invalid := []string{
		"ssr://" + base64.RawURLEncoding.EncodeToString([]byte("example.com:bad:origin:method:obfs:pass")),
		"ssr://" + base64.RawURLEncoding.EncodeToString([]byte("example.com:443:origin:method:obfs")),
		"ssr://" + base64.RawURLEncoding.EncodeToString([]byte("example.com:443:origin::obfs:pass")),
		"ssr://" + base64.RawURLEncoding.EncodeToString([]byte("example.com:443:origin:method:obfs:pass/?bad=%ZZ")),
	}
	for _, link := range invalid {
		_, err := ParseProxyLink(link)
		if err == nil || !strings.Contains(err.Error(), "ssr") {
			t.Errorf("link %q error = %v", link, err)
		}
	}
	if _, err := ParseProxyLink("ssr://not a server"); err == nil {
		t.Fatal("expected invalid SSR authority error")
	}
}

func TestSSRHelpersHandleFallbackValues(t *testing.T) {
	if got := decodeSSRValue("plain%20value"); got != "plain value" {
		t.Fatalf("decoded fallback = %q", got)
	}
	if got := decodeSSRValue("bad%zz"); got != "bad%zz" {
		t.Fatalf("invalid fallback = %q", got)
	}
	if _, _, err := extractSSRPayload("not-a-link"); err == nil {
		t.Fatal("expected SSR payload error")
	}
	if _, _, err := extractSSRPayload("ssr://payload#bad%zz"); err == nil {
		t.Fatal("expected SSR tag error")
	}
	if _, err := parseSSRAuthority("example.com:bad", ""); err == nil {
		t.Fatal("expected SSR port error")
	}
}

func TestParseSSRLinkProducesSingBoxConfig(t *testing.T) {
	result, err := ParseProxyLink(makeSSRLink(""))
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{"outbounds": []any{result.Config}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	defer cancel()
	var config option.Options
	if err := config.UnmarshalJSONContext(ctx, body); err != nil {
		t.Fatalf("generated config rejected by sing-box: %v", err)
	}
}
