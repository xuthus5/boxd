package core

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
)

func TestDetectContainerUsesExplicitFlagOrRuntimeMarker(t *testing.T) {
	for _, tt := range []struct {
		name, value, marker string
		want                bool
	}{
		{name: "host"},
		{name: "true", value: "true", want: true},
		{name: "one", value: "1", want: true},
		{name: "normalized true", value: " TRUE ", want: true},
		{name: "invalid flag", value: "yes"},
		{name: "Podman marker", marker: "/run/.containerenv", want: true},
		{name: "Docker marker", marker: "/.dockerenv", want: true},
		{name: "marker takes precedence", value: "false", marker: "/.dockerenv", want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stat := func(path string) (os.FileInfo, error) {
				if path == tt.marker {
					return nil, nil
				}
				return nil, os.ErrNotExist
			}
			if got := detectContainer(func(string) string { return tt.value }, stat); got != tt.want {
				t.Fatalf("want container=%v, got %v", tt.want, got)
			}
		})
	}
}

func TestDetectNetworkCapabilitiesReportsCompleteContract(t *testing.T) {
	t.Setenv("BOXD_CONTAINER", "1")
	caps := DetectNetworkCapabilities()
	if !caps.Container || caps.Platform == "" || caps.DefaultListen != "0.0.0.0" {
		t.Fatalf("unexpected capabilities: %+v", caps)
	}
	if !caps.TUNAvailable && caps.TUNReason == "" || !caps.IPv6Available && caps.IPv6Reason == "" {
		t.Fatalf("unavailable capabilities require reasons: %+v", caps)
	}
	body, err := json.Marshal(caps)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"platform", "container", "tun_available", "tun_reason", "ipv6_available", "ipv6_reason", "default_listen"} {
		if _, ok := fields[field]; !ok {
			t.Fatalf("capability contract is missing %s", field)
		}
	}
}

func TestWindowsIPv6PolicyDistinguishesPreferenceAndDisabling(t *testing.T) {
	for _, tt := range []struct {
		name     string
		disabled uint64
		err      error
		want     bool
		reason   string
	}{
		{name: "default registry value", want: true},
		{name: "prefer IPv4 is not disabled", disabled: 0x20, want: true},
		{name: "native interfaces disabled", disabled: 0x10, reason: NetworkIPv6Disabled},
		{name: "all interfaces disabled", disabled: 0xff, reason: NetworkIPv6Disabled},
		{name: "tunnel policy uncertain", disabled: 0x01, reason: NetworkIPv6Unknown},
		{name: "registry unreadable", err: os.ErrPermission, reason: NetworkIPv6Unknown},
	} {
		t.Run(tt.name, func(t *testing.T) {
			available, reason := windowsIPv6Policy(tt.disabled, tt.err)
			if available != tt.want || reason != tt.reason {
				t.Fatalf("want %v/%s, got %v/%s", tt.want, tt.reason, available, reason)
			}
		})
	}
}

func TestPrivilegedTUNCapability(t *testing.T) {
	for _, tt := range []struct {
		name     string
		elevated bool
		err      error
		reason   string
	}{
		{name: "elevated", elevated: true},
		{name: "not elevated", reason: NetworkTUNPermission},
		{name: "token unavailable", err: errors.New("token query failed"), reason: NetworkTUNUnknown},
	} {
		t.Run(tt.name, func(t *testing.T) {
			available, reason := privilegedTUNCapability(tt.elevated, tt.err)
			if available != (tt.reason == "") || reason != tt.reason {
				t.Fatalf("unexpected capability: %v/%s", available, reason)
			}
		})
	}
}
