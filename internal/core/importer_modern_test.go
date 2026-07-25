package core

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

const testTUICUUID = "2DD61D93-75D8-4DA4-AC0E-6AECE7EAC365"

func TestParseModernHelpersHandleMissingValues(t *testing.T) {
	if uuid, password, err := parseTUICCredentials(nil); err == nil || uuid != "" || password != "" {
		t.Fatalf("nil TUIC credentials = %q/%q, %v", uuid, password, err)
	}
	if uuid, password, err := parseTUICCredentials(&url.URL{}); err != nil || uuid != "" || password != "" {
		t.Fatalf("missing TUIC credentials = %q/%q, %v", uuid, password, err)
	}
	if server, port, err := parseModernLinkServer(nil, "tuic", 443); err == nil || server != "" || port != 0 {
		t.Fatalf("nil server = %q:%d, %v", server, port, err)
	}
	if server, port, err := parseModernLinkServer(&url.URL{}, "tuic", 443); err == nil || server != "" || port != 0 {
		t.Fatalf("missing server = %q:%d, %v", server, port, err)
	}
	for _, raw := range []string{"-1s", "fast"} {
		if _, err := normalizeModernDuration(raw, "test duration"); err == nil {
			t.Errorf("duration %q unexpectedly succeeded", raw)
		}
	}
}

func TestParseTUICLinkMapsSingBoxOptions(t *testing.T) {
	link := "tuic://" + testTUICUUID + ":secret%20pass@example.com:443?congestion_control=bbr&udp_relay_mode=quic&zero_rtt_handshake=1&heartbeat=10&network=tcp%2Cudp&sni=cdn.example&insecure=true&alpn=h3%2Chq&pinSHA256=" + url.QueryEscape(testSHA256PinBase64) + "#tuic-full"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	config := result.Config.(map[string]any)
	if config["uuid"] != testTUICUUID || config["password"] != "secret pass" {
		t.Fatalf("credentials = %#v", config)
	}
	if config["congestion_control"] != "bbr" || config["udp_relay_mode"] != "quic" || config["zero_rtt_handshake"] != true {
		t.Fatalf("protocol options = %#v", config)
	}
	if config["heartbeat"] != "10s" {
		t.Fatalf("heartbeat = %#v", config["heartbeat"])
	}
	if networks, ok := config["network"].([]string); !ok || len(networks) != 2 || networks[0] != "tcp" || networks[1] != "udp" {
		t.Fatalf("network = %#v", config["network"])
	}
	tls, ok := config["tls"].(map[string]any)
	if !ok || tls["server_name"] != "cdn.example" || tls["insecure"] != true {
		t.Fatalf("tls = %#v", config["tls"])
	}
	if alpn, ok := tls["alpn"].([]string); !ok || len(alpn) != 2 || alpn[0] != "h3" || alpn[1] != "hq" {
		t.Fatalf("alpn = %#v", tls["alpn"])
	}
}

func TestParseTUICLinkSupportsQueryCredentials(t *testing.T) {
	result, err := ParseProxyLink("tuic://example.com?uuid=" + testTUICUUID + "&password=query-pass")
	if err != nil {
		t.Fatal(err)
	}
	config := result.Config.(map[string]any)
	if config["uuid"] != testTUICUUID || config["password"] != "query-pass" || result.Port != 443 {
		t.Fatalf("result = %#v", result)
	}
	if tls := config["tls"].(map[string]any); tls["server_name"] != "example.com" {
		t.Fatalf("default TLS SNI = %#v", tls)
	}
}

func TestParseTUICLinkRejectsInvalidOptions(t *testing.T) {
	base := "tuic://" + testTUICUUID + ":pass@example.com:443?"
	links := []string{
		base + "congestion_control=reno",
		base + "udp_relay_mode=bad",
		base + "udp_over_stream=1&udp_relay_mode=quic",
		base + "heartbeat=fast",
		base + "network=icmp",
		base + "insecure=maybe",
	}
	for _, link := range links {
		_, err := ParseProxyLink(link)
		if err == nil || !strings.Contains(err.Error(), "tuic") {
			t.Errorf("link %q error = %v", link, err)
		}
	}
	if _, err := ParseProxyLink("tuic://example.com#missing-uuid"); err == nil {
		t.Fatal("expected missing UUID error")
	}
}

func TestParseAnyTLSLinkMapsSingBoxOptions(t *testing.T) {
	result, err := ParseProxyLink("anytls://secret@example.com:443?sni=cdn.example&insecure=1&alpn=h2%2Ch3&idle_session_check_interval=10&idle_session_timeout=20s&min_idle_session=3#anytls-full")
	if err != nil {
		t.Fatal(err)
	}
	config := result.Config.(map[string]any)
	if config["idle_session_check_interval"] != "10s" || config["idle_session_timeout"] != "20s" || config["min_idle_session"] != 3 {
		t.Fatalf("session options = %#v", config)
	}
	tls := config["tls"].(map[string]any)
	if tls["server_name"] != "cdn.example" || tls["insecure"] != true {
		t.Fatalf("tls = %#v", tls)
	}
	if alpn, ok := tls["alpn"].([]string); !ok || len(alpn) != 2 {
		t.Fatalf("alpn = %#v", tls["alpn"])
	}
}

func TestParseAnyTLSLinkRejectsInvalidOptions(t *testing.T) {
	links := []string{
		"anytls://pass@example.com?idle_session_timeout=fast",
		"anytls://pass@example.com?min_idle_session=-1",
		"anytls://pass@example.com?insecure=maybe",
		"anytls://pass@example.com?security=none",
	}
	for _, link := range links {
		if _, err := ParseProxyLink(link); err == nil || !strings.Contains(err.Error(), "anytls") {
			t.Errorf("link %q error = %v", link, err)
		}
	}
}

func TestParseShadowTLSLinkMapsSingBoxOptions(t *testing.T) {
	result, err := ParseProxyLink("shadowtls://secret@example.com:443?version=2&sni=cdn.example&insecure=1&alpn=h2%2Ch1&pinSHA256=" + url.QueryEscape(testSHA256PinBase64) + "#shadow-full")
	if err != nil {
		t.Fatal(err)
	}
	config := result.Config.(map[string]any)
	if config["version"] != 2 || config["password"] != "secret" {
		t.Fatalf("config = %#v", config)
	}
	tls := config["tls"].(map[string]any)
	if tls["server_name"] != "cdn.example" || tls["insecure"] != true {
		t.Fatalf("tls = %#v", tls)
	}
	if hashes, ok := tls["certificate_public_key_sha256"].([]string); !ok || len(hashes) != 1 || hashes[0] != testSHA256PinBase64 {
		t.Fatalf("pins = %#v", tls["certificate_public_key_sha256"])
	}
}

func TestParseShadowTLSLinkRejectsInvalidVersion(t *testing.T) {
	for _, version := range []string{"0", "4", "bad"} {
		_, err := ParseProxyLink("shadowtls://secret@example.com?version=" + version)
		if err == nil || !strings.Contains(err.Error(), "shadowtls") {
			t.Errorf("version %q error = %v", version, err)
		}
	}
	if version, err := parseShadowTLSVersion(nil); err != nil || version != 3 {
		t.Fatalf("default version = %d, err = %v", version, err)
	}
}

func TestModernProtocolLinksProduceSingBoxConfig(t *testing.T) {
	links := []string{
		"tuic://" + testTUICUUID + ":pass@example.com?network=tcp%2Cudp&heartbeat=10s",
		"anytls://pass@example.com?idle_session_check_interval=10s&min_idle_session=1",
		"shadowtls://pass@example.com?version=3",
	}
	for _, link := range links {
		result, err := ParseProxyLink(link)
		if err != nil {
			t.Fatal(err)
		}
		body, err := json.Marshal(map[string]any{"outbounds": []any{result.Config}})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(include.Context(context.Background()))
		var config option.Options
		err = config.UnmarshalJSONContext(ctx, body)
		cancel()
		if err != nil {
			t.Fatalf("%s generated invalid sing-box config: %v", result.Type, err)
		}
	}
}
