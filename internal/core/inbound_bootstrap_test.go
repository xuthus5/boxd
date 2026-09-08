package core

import "testing"

func TestInitialInboundsRespectTUNCapability(t *testing.T) {
	for _, tt := range []struct {
		name  string
		tun   bool
		count int
	}{
		{name: "local proxy without privilege", count: 1},
		{name: "TUN with privilege", tun: true, count: 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			inbounds := initialInbounds(tt.tun)
			if len(inbounds) != tt.count {
				t.Fatalf("want %d inbounds, got %d", tt.count, len(inbounds))
			}
			mixed := inbounds[0].(map[string]any)
			if mixed["listen"] != "127.0.0.1" {
				t.Fatal("first-run proxy must only listen on loopback")
			}
			if tt.tun {
				tun := inbounds[1].(map[string]any)
				if tun["auto_route"] != true || tun["strict_route"] != true {
					t.Fatalf("TUN must capture DNS with strict routing: %#v", tun)
				}
			}
		})
	}
}

func TestConfigureDefaultTUNRouting(t *testing.T) {
	for _, tt := range []struct {
		name   string
		tun    bool
		route  map[string]any
		detect any
	}{
		{name: "mixed needs no interface override"},
		{name: "TUN creates route", tun: true, detect: true},
		{name: "TUN completes route", tun: true, route: map[string]any{"final": "proxy"}, detect: true},
		{name: "preserve interface", tun: true, route: map[string]any{"default_interface": "eth1"}},
		{name: "preserve mark", tun: true, route: map[string]any{"default_mark": 128}},
		{name: "preserve detection choice", tun: true, route: map[string]any{"auto_detect_interface": false}, detect: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := map[string]any{"inbounds": initialInbounds(tt.tun)}
			if tt.route != nil {
				cfg["route"] = tt.route
			}
			ConfigureDefaultTUNRouting(cfg)
			route, _ := cfg["route"].(map[string]any)
			if route["auto_detect_interface"] != tt.detect {
				t.Fatalf("interface detection: want %#v, got %#v", tt.detect, route)
			}
		})
	}
}
