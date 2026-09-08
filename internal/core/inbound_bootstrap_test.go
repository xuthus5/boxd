package core

import "testing"

func TestInitialInboundsRequireExplicitTUNSelection(t *testing.T) {
	for _, tt := range []struct {
		name      string
		tun       bool
		container bool
		listen    string
	}{
		{name: "local proxy without privilege", listen: "127.0.0.1"},
		{name: "privilege does not enable TUN", tun: true, listen: "127.0.0.1"},
		{name: "container listener", tun: true, container: true, listen: "0.0.0.0"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			inbounds := initialInbounds(NetworkCapabilities{Container: tt.container, TUNAvailable: tt.tun})
			if len(inbounds) != 1 {
				t.Fatalf("want one mixed inbound, got %d", len(inbounds))
			}
			mixed := inbounds[0].(map[string]any)
			if mixed["listen"] != tt.listen {
				t.Fatalf("want listen %q, got %#v", tt.listen, mixed)
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
			inbounds := initialInbounds(NetworkCapabilities{})
			if tt.tun {
				inbounds = append(inbounds, tunInboundTemplate())
			}
			cfg := map[string]any{"inbounds": inbounds}
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
