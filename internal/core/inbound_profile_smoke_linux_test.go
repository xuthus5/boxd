//go:build integration && linux

package core

import (
	"encoding/json"
	"os"
	"testing"
)

func selectTUNSmokeProfile(t *testing.T, cfg map[string]any, path string) {
	t.Helper()
	profile, err := BuildInboundProfile(cfg, InboundProfileOptions{Mode: "tun"}, DetectNetworkCapabilities())
	if err != nil {
		t.Fatal(err)
	}
	cfg["inbounds"] = profile.Inbounds
	ConfigureDefaultTUNRouting(cfg)
	body, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
}
