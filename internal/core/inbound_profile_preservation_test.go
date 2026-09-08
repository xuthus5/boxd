package core

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestInboundProfilePreservesCustomMixedListener(t *testing.T) {
	custom := map[string]any{"type": "mixed", "tag": "mixed-in", "listen": "127.0.0.1", "listen_port": 8080}
	for _, mode := range []string{"proxy", "preserve"} {
		t.Run(mode, func(t *testing.T) {
			cfg := map[string]any{"inbounds": []any{custom}}
			result, err := BuildInboundProfile(cfg, InboundProfileOptions{Mode: mode}, NetworkCapabilities{Container: true})
			if err != nil || !reflect.DeepEqual(custom, result.Inbounds[0]) {
				t.Fatalf("custom port must not be rewritten: %#v, %v", result, err)
			}
		})
	}
}

func TestInboundProfileRejectsCustomizedManagedTUN(t *testing.T) {
	for _, tt := range []struct {
		name, field string
		value       any
	}{
		{name: "custom address", field: "address", value: []string{"192.0.2.1/24"}},
		{name: "custom mtu", field: "mtu", value: 1500},
		{name: "custom route policy", field: "auto_route", value: false},
		{name: "additional options", field: "route_exclude_address", value: []string{"192.0.2.0/24"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			custom := tunInboundFor(NetworkCapabilities{Platform: "linux"}, true)
			custom[tt.field] = tt.value
			cfg := map[string]any{"inbounds": []any{custom}}
			_, err := BuildInboundProfile(cfg, InboundProfileOptions{Mode: "proxy"}, NetworkCapabilities{})
			assertInboundProfileError(t, err, InboundProfileConflict)
			preserved, err := BuildInboundProfile(cfg, InboundProfileOptions{}, NetworkCapabilities{})
			if err != nil || !reflect.DeepEqual(custom, preserved.Inbounds[0]) {
				t.Fatalf("custom TUN settings must remain untouched: %#v, %v", preserved, err)
			}
		})
	}
}

func TestInboundProfileRecognizesDecodedLegacyTUN(t *testing.T) {
	const fixture = `{"inbounds":[{"type":"tun","tag":"tun-in","interface_name":"boxd0","address":["172.19.0.1/30","fdfe:dcba:9876::1/126"],"mtu":9000,"auto_route":true,"strict_route":true,"stack":"gvisor"}]}`
	var cfg map[string]any
	if err := json.Unmarshal([]byte(fixture), &cfg); err != nil {
		t.Fatal(err)
	}
	caps := NetworkCapabilities{Platform: "linux", TUNAvailable: true, IPv6Reason: NetworkIPv6Disabled}
	result, err := BuildInboundProfile(cfg, InboundProfileOptions{Mode: "tun"}, caps)
	if err != nil {
		t.Fatal(err)
	}
	actual := result.Inbounds[0].(map[string]any)
	assertProfileTUN(t, actual, 1)
	if actual["stack"] != "system" {
		t.Fatalf("platform stack must be adapted explicitly: %#v", actual)
	}
}

func TestInboundTemplateRecognitionRejectsInvalidJSONValues(t *testing.T) {
	valid := map[string]any{"type": "mixed"}
	invalid := map[string]any{"type": func() {}}
	if equalInboundJSON(invalid, valid) || equalInboundJSON(valid, invalid) {
		t.Fatal("invalid JSON values must never be considered managed templates")
	}
}
