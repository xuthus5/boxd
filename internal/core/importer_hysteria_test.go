package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

func TestParseHysteriaLinkMapsSingBoxOptions(t *testing.T) {
	link := "hysteria://example.com:443?auth=secret&sni=cdn.example&insecure=1&upmbps=50&downmbps=100&obfs=obfs-pass&mport=443-8443&hop_interval=30&recv_window_conn=1000&recv_window=2000&disable_mtu_discovery=true&network=tcp%2Cudp#hy1-full"
	result, err := ParseProxyLink(link)
	if err != nil {
		t.Fatal(err)
	}
	config := result.Config.(map[string]any)
	if config["auth_str"] != "secret" || config["up_mbps"] != 50 || config["down_mbps"] != 100 {
		t.Fatalf("config = %#v", config)
	}
	if config["obfs"] != "obfs-pass" || config["hop_interval"] != "30s" {
		t.Fatalf("protocol options = %#v", config)
	}
	if ports, ok := config["server_ports"].([]string); !ok || len(ports) != 1 || ports[0] != "443:8443" {
		t.Fatalf("server ports = %#v", config["server_ports"])
	}
	if networks, ok := config["network"].([]string); !ok || len(networks) != 2 {
		t.Fatalf("network = %#v", config["network"])
	}
	tls := config["tls"].(map[string]any)
	if tls["server_name"] != "cdn.example" || tls["insecure"] != true {
		t.Fatalf("tls = %#v", tls)
	}
}

func TestParseHysteriaLinkDefaultsAndRejectsInvalidOptions(t *testing.T) {
	result, err := ParseProxyLink("hysteria://secret@example.com#hy1-default")
	if err != nil {
		t.Fatal(err)
	}
	if result.Port != 443 || result.Config.(map[string]any)["auth_str"] != "secret" {
		t.Fatalf("default result = %#v", result)
	}
	links := []string{
		"hysteria://example.com?upmbps=fast",
		"hysteria://example.com?recv_window=-1",
		"hysteria://example.com?disable_mtu_discovery=maybe",
		"hysteria://example.com?network=icmp",
		"hysteria://example.com?mport=0",
	}
	for _, link := range links {
		_, err := ParseProxyLink(link)
		if err == nil || !strings.Contains(err.Error(), "hysteria") {
			t.Errorf("link %q error = %v", link, err)
		}
	}
}

func TestParseHysteriaLinkProducesSingBoxConfig(t *testing.T) {
	result, err := ParseProxyLink("hysteria://example.com?auth_str=secret&upmbps=10&downmbps=20")
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
