package core

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseHysteria2OfficialShareLinkIPv6(t *testing.T) {
	result, err := ParseProxyLink("hy2://pass@[2001:db8::1]:443,8443-8444/?hop-interval=15#ipv6")
	if err != nil {
		t.Fatal(err)
	}
	if result.Server != "2001:db8::1" || result.Port != 443 {
		t.Fatalf("server = %s:%d", result.Server, result.Port)
	}
	config := result.Config.(map[string]any)
	if !reflect.DeepEqual(config["server_ports"], []string{"443:443", "8443:8444"}) {
		t.Fatalf("server ports = %#v", config["server_ports"])
	}
	if config["hop_interval"] != "15s" {
		t.Fatalf("hop interval = %#v", config["hop_interval"])
	}
}

func TestParseHysteria2OfficialShareLinkRejectsConflicts(t *testing.T) {
	_, err := ParseProxyLink("hysteria2://pass@example.com:443-8443/?mport=1000-2000")
	if err == nil || !strings.Contains(err.Error(), "conflicting port hopping") {
		t.Fatalf("error = %v", err)
	}
}

func TestHysteria2AuthorityValidation(t *testing.T) {
	for _, raw := range []string{"[2001:db8::1", "[2001:db8::1]bad", "2001:db8::1:443-8443"} {
		if _, _, _, err := splitHysteria2Authority(raw); err == nil {
			t.Errorf("authority %q unexpectedly succeeded", raw)
		}
	}
	if _, _, err := normalizeHysteria2PortUnion("-1,443"); err == nil {
		t.Fatal("negative port union unexpectedly succeeded")
	}
}
