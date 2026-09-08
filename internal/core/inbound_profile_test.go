package core

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestBuildInboundProfileModesAndAddressFamilies(t *testing.T) {
	for _, tt := range []struct {
		name, mode, ipv6, platform string
		available, container       bool
		count, addresses           int
	}{
		{name: "implicit preserve", count: 1},
		{name: "proxy with privileges", mode: "proxy", available: true, count: 1},
		{name: "container proxy", mode: "proxy", container: true, count: 1},
		{name: "dual stack linux", mode: "tun", platform: "linux", available: true, count: 2, addresses: 2},
		{name: "explicit IPv4", mode: "tun", ipv6: "off", available: true, count: 2, addresses: 1},
		{name: "disabled IPv6 auto", mode: "tun", ipv6: "auto", count: 2, addresses: 1},
		{name: "explicit IPv6", mode: "tun", ipv6: "on", available: true, count: 2, addresses: 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			caps := NetworkCapabilities{Platform: tt.platform, Container: tt.container, TUNAvailable: true,
				IPv6Available: tt.available, IPv6Reason: NetworkIPv6Disabled}
			result, err := BuildInboundProfile(nil, InboundProfileOptions{Mode: tt.mode, IPv6: tt.ipv6}, caps)
			if err != nil || len(result.Inbounds) != tt.count {
				t.Fatalf("want %d inbounds: result=%#v, error=%v", tt.count, result, err)
			}
			mixed := result.Inbounds[0].(map[string]any)
			if mixed["listen"] != defaultInboundListen(tt.container) {
				t.Fatalf("unexpected mixed listener: %#v", mixed)
			}
			if tt.mode == "tun" {
				assertProfileTUN(t, result.Inbounds[1].(map[string]any), tt.addresses)
			}
		})
	}
}

func assertProfileTUN(t *testing.T, inbound map[string]any, addresses int) {
	t.Helper()
	if inbound["type"] != "tun" || inbound["auto_route"] != true || inbound["strict_route"] != true {
		t.Fatalf("missing TUN routing safeguards: %#v", inbound)
	}
	if actual := len(inbound["address"].([]string)); actual != addresses {
		t.Fatalf("want %d address families, got %d", addresses, actual)
	}
}

func TestBuildInboundProfileReturnsActionableErrors(t *testing.T) {
	for _, tt := range []struct {
		name, mode, ipv6, reason, code string
		tun                            bool
	}{
		{name: "invalid mode", mode: "gateway", code: InboundProfileInvalid},
		{name: "invalid IPv6 setting", ipv6: "sometimes", code: InboundProfileInvalid},
		{name: "no TUN privilege", mode: "tun", code: InboundProfileTUN},
		{name: "required IPv6 disabled", mode: "tun", ipv6: "on", tun: true, reason: NetworkIPv6Disabled, code: InboundProfileIPv6},
		{name: "unknown IPv6 auto", mode: "tun", tun: true, reason: NetworkIPv6Unknown, code: InboundProfileIPv6Unknown},
		{name: "unreported IPv6 auto", mode: "tun", tun: true, code: InboundProfileIPv6Unknown},
	} {
		t.Run(tt.name, func(t *testing.T) {
			caps := NetworkCapabilities{TUNAvailable: tt.tun, IPv6Reason: tt.reason}
			_, err := BuildInboundProfile(nil, InboundProfileOptions{Mode: tt.mode, IPv6: tt.ipv6}, caps)
			assertInboundProfileError(t, err, tt.code)
		})
	}
}

func assertInboundProfileError(t *testing.T, err error, code string) {
	t.Helper()
	var profileErr *InboundProfileError
	if !errors.As(err, &profileErr) || profileErr.Code != code || profileErr.Error() == "" {
		t.Fatalf("want actionable %q error, got %v", code, err)
	}
}

func TestBuildInboundProfileUnknownIPv6RequiresExplicitChoice(t *testing.T) {
	caps := NetworkCapabilities{TUNAvailable: true, IPv6Reason: NetworkIPv6Unknown}
	for _, choice := range []string{"off", "on"} {
		t.Run(choice, func(t *testing.T) {
			result, err := BuildInboundProfile(nil, InboundProfileOptions{Mode: "tun", IPv6: choice}, caps)
			if err != nil {
				t.Fatal(err)
			}
			addresses := 1
			if choice == "on" {
				addresses = 2
			}
			assertProfileTUN(t, result.Inbounds[1].(map[string]any), addresses)
		})
	}
}

func TestBuildInboundProfilePreservesCustomAndUnrelatedValues(t *testing.T) {
	inbounds := []any{
		map[string]any{"tag": "mixed-in", "type": "mixed", "listen": "192.0.2.10", "listen_port": 8080},
		map[string]any{"tag": "tun-in", "type": "tun", "interface_name": "user0"},
		map[string]any{"tag": "http-in", "type": "http", "listen_port": 8081},
		map[string]any{"type": "socks"}, nil, "invalid passthrough",
	}
	cfg := map[string]any{"inbounds": inbounds}
	before, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	result, err := BuildInboundProfile(cfg, InboundProfileOptions{Mode: "preserve"}, NetworkCapabilities{Container: true})
	if err != nil || !reflect.DeepEqual(inbounds, result.Inbounds) {
		t.Fatalf("custom configuration was not preserved: result=%#v, error=%v", result, err)
	}
	after, err := json.Marshal(cfg)
	if err != nil || string(before) != string(after) {
		t.Fatalf("profile builder mutated its input: %v", err)
	}
}

func TestBuildInboundProfileAdaptsOnlyManagedDefaults(t *testing.T) {
	managed := tunInboundFor(NetworkCapabilities{Platform: "linux"}, true)
	cfg := map[string]any{"inbounds": []any{managed}}
	before, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	caps := NetworkCapabilities{Platform: "linux", TUNAvailable: true, IPv6Reason: NetworkIPv6Disabled}
	result, err := BuildInboundProfile(cfg, InboundProfileOptions{Mode: "tun"}, caps)
	if err != nil {
		t.Fatal(err)
	}
	assertProfileTUN(t, result.Inbounds[0].(map[string]any), 1)
	after, err := json.Marshal(cfg)
	if err != nil || string(before) != string(after) {
		t.Fatalf("existing default was modified in place: %v", err)
	}
	repeated, err := BuildInboundProfile(map[string]any{"inbounds": result.Inbounds}, InboundProfileOptions{Mode: "tun"}, caps)
	if err != nil || !reflect.DeepEqual(result, repeated) {
		t.Fatalf("repeated profile selection changed defaults: result=%#v, error=%v", repeated, err)
	}
}

func TestBuildInboundProfileProxyRemovesManagedTUN(t *testing.T) {
	cfg := map[string]any{"inbounds": []any{
		mixedInboundFor(NetworkCapabilities{}), tunInboundFor(NetworkCapabilities{Platform: "linux"}, true),
	}}
	result, err := BuildInboundProfile(cfg, InboundProfileOptions{Mode: "proxy"}, NetworkCapabilities{Container: true})
	if err != nil || len(result.Inbounds) != 1 {
		t.Fatalf("managed TUN was not removed: result=%#v, error=%v", result, err)
	}
	if result.Inbounds[0].(map[string]any)["listen"] != "0.0.0.0" {
		t.Fatal("managed mixed listener was not adapted for the container")
	}
}

func TestBuildInboundProfileRejectsCustomTUNConflicts(t *testing.T) {
	for _, tt := range []struct {
		name string
		item any
	}{
		{name: "custom interface", item: map[string]any{"tag": "tun-in", "type": "tun", "interface_name": "mine0"}},
		{name: "another TUN", item: map[string]any{"tag": "another", "type": "tun"}},
		{name: "untagged TUN", item: map[string]any{"type": "tun"}},
		{name: "reserved TUN tag", item: map[string]any{"tag": "tun-in", "type": "http"}},
		{name: "reserved mixed tag", item: map[string]any{"tag": "mixed-in", "type": "socks"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := map[string]any{"inbounds": []any{tt.item}}
			for _, mode := range []string{"proxy", "tun"} {
				_, err := BuildInboundProfile(cfg, InboundProfileOptions{Mode: mode, IPv6: "off"}, NetworkCapabilities{TUNAvailable: true})
				assertInboundProfileError(t, err, InboundProfileConflict)
			}
		})
	}
}

func TestBuildInboundProfileRejectsAmbiguousCollections(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value any
	}{
		{name: "not an array", value: "inbounds"},
		{name: "duplicate managed tag", value: []any{map[string]any{"tag": "mixed-in"}, map[string]any{"tag": "mixed-in"}}},
		{name: "duplicate custom tag", value: []any{map[string]any{"tag": "mine"}, map[string]any{"tag": "mine"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildInboundProfile(map[string]any{"inbounds": tt.value}, InboundProfileOptions{}, NetworkCapabilities{})
			assertInboundProfileError(t, err, InboundProfileConflict)
		})
	}
}
