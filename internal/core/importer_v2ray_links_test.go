package core

import (
	"net/url"
	"reflect"
	"testing"
)

func TestParseTrojanLinkBuildsSingBoxTLSAndTransport(t *testing.T) {
	result, err := ParseProxyLink("trojan://secret@example.com:443?security=tls&sni=cdn.example&allowInsecure=1&alpn=h2%2Chttp%2F1.1&type=ws&host=ws.example&path=%2Ftrojan#trojan-ws")
	if err != nil {
		t.Fatal(err)
	}
	config, ok := result.Config.(map[string]any)
	if !ok {
		t.Fatalf("config type = %T", result.Config)
	}
	if _, exists := config["network"]; exists {
		t.Fatal("sing-box config must not contain a network field for transport links")
	}
	transport, ok := config["transport"].(map[string]any)
	if !ok || transport["type"] != "ws" || transport["path"] != "/trojan" {
		t.Fatalf("transport = %#v", config["transport"])
	}
	if headers, ok := transport["headers"].(map[string]any); !ok || headers["Host"] != "ws.example" {
		t.Fatalf("transport headers = %#v", transport["headers"])
	}
	tls, ok := config["tls"].(map[string]any)
	if !ok {
		t.Fatalf("tls = %#v", config["tls"])
	}
	wantTLS := map[string]any{
		"enabled":     true,
		"server_name": "cdn.example",
		"insecure":    true,
		"alpn":        []string{"h2", "http/1.1"},
	}
	if !reflect.DeepEqual(tls, wantTLS) {
		t.Fatalf("tls = %#v, want %#v", tls, wantTLS)
	}
}

func TestParseVlessLinkBuildsTLSWithoutSNIAndGRPCTransport(t *testing.T) {
	result, err := ParseProxyLink("vless://uuid@example.com:443?security=tls&allowInsecure=1&type=grpc&serviceName=proxy#vless-grpc")
	if err != nil {
		t.Fatal(err)
	}
	config, ok := result.Config.(map[string]any)
	if !ok {
		t.Fatalf("config type = %T", result.Config)
	}
	if _, exists := config["network"]; exists {
		t.Fatal("sing-box config must not contain a Clash network field")
	}
	transport, ok := config["transport"].(map[string]any)
	if !ok || transport["type"] != "grpc" || transport["service_name"] != "proxy" {
		t.Fatalf("transport = %#v", config["transport"])
	}
	tls, ok := config["tls"].(map[string]any)
	if !ok || tls["enabled"] != true || tls["insecure"] != true {
		t.Fatalf("tls = %#v", config["tls"])
	}
}

func TestParseVlessLinkBuildsRealityTLSAndWebsocketTransport(t *testing.T) {
	result, err := ParseProxyLink("vless://uuid@example.com:443?security=reality&sni=example.com&fp=chrome&pbk=public-key&sid=short-id&type=ws&host=cdn.example&path=%2Fws#vless-reality")
	if err != nil {
		t.Fatal(err)
	}
	config := result.Config.(map[string]any)
	transport := config["transport"].(map[string]any)
	if transport["type"] != "ws" || transport["path"] != "/ws" {
		t.Fatalf("transport = %#v", transport)
	}
	tls := config["tls"].(map[string]any)
	if tls["server_name"] != "example.com" {
		t.Fatalf("tls = %#v", tls)
	}
	if utls := tls["utls"].(map[string]any); utls["fingerprint"] != "chrome" || utls["enabled"] != true {
		t.Fatalf("utls = %#v", tls["utls"])
	}
	reality := tls["reality"].(map[string]any)
	if reality["public_key"] != "public-key" || reality["short_id"] != "short-id" || reality["enabled"] != true {
		t.Fatalf("reality = %#v", reality)
	}
}

func TestParseV2RayLinkRejectsUnsupportedTransport(t *testing.T) {
	for _, scheme := range []string{"trojan", "vless"} {
		t.Run(scheme, func(t *testing.T) {
			link := scheme + "://credential@example.com:443?type=kcp#unsupported"
			_, err := ParseProxyLink(link)
			if err == nil {
				t.Fatal("expected unsupported transport error")
			}
		})
	}
}

func TestParseV2RayLinksRejectMissingCredentials(t *testing.T) {
	for _, link := range []string{
		"trojan://@example.com:443",
		"vless://@example.com:443",
	} {
		if _, err := ParseProxyLink(link); err == nil {
			t.Errorf("link %q unexpectedly succeeded", link)
		}
	}
}

func TestBuildLinkTLSConfigHonorsDisabledAndAliases(t *testing.T) {
	if got := buildLinkTLSConfig(url.Values{}, false); got != nil {
		t.Fatalf("empty TLS config = %#v, want nil", got)
	}
	if got := buildLinkTLSConfig(url.Values{"security": {"none"}, "sni": {"ignored.example"}}, false); got != nil {
		t.Fatalf("disabled TLS config = %#v, want nil", got)
	}
	got := buildLinkTLSConfig(url.Values{
		"tls":            {"1"},
		"servername":     {"server.example"},
		"allow_insecure": {"yes"},
	}, false)
	if got["enabled"] != true || got["server_name"] != "server.example" || got["insecure"] != true {
		t.Fatalf("TLS config = %#v", got)
	}
}
