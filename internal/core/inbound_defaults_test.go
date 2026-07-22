package core

import "testing"

func TestDefaultInboundsInstallerInstall(t *testing.T) {
	installer := NewDefaultInboundsInstaller()
	result, err := installer.Install(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Inbounds) != 2 {
		t.Fatalf("inbounds = %#v", result.Inbounds)
	}
	if len(result.Installed) != 2 {
		t.Fatalf("installed = %#v", result.Installed)
	}
	byTag := map[string]map[string]any{}
	for _, item := range result.Inbounds {
		ob := item.(map[string]any)
		byTag[ob["tag"].(string)] = ob
	}
	if byTag["mixed-in"]["type"] != "mixed" || byTag["mixed-in"]["listen_port"] != 1080 {
		t.Fatalf("mixed-in = %#v", byTag["mixed-in"])
	}
	if byTag["tun-in"]["type"] != "tun" || byTag["tun-in"]["auto_route"] != true {
		t.Fatalf("tun-in = %#v", byTag["tun-in"])
	}
}

func TestDefaultInboundsInstallerPreservesExisting(t *testing.T) {
	installer := NewDefaultInboundsInstaller()
	cfg := map[string]any{
		"inbounds": []any{
			map[string]any{"tag": "mixed-in", "type": "mixed", "listen": "127.0.0.1", "listen_port": 2080},
			map[string]any{"type": "http"}, // no tag passthrough
		},
	}
	result, err := installer.Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	byTag := map[string]map[string]any{}
	passthrough := 0
	for _, item := range result.Inbounds {
		ob, ok := item.(map[string]any)
		if !ok {
			continue
		}
		tag, _ := ob["tag"].(string)
		if tag == "" {
			passthrough++
			continue
		}
		byTag[tag] = ob
	}
	if byTag["mixed-in"]["listen_port"] != 2080 {
		t.Fatalf("preserved mixed port = %#v", byTag["mixed-in"]["listen_port"])
	}
	if _, ok := byTag["tun-in"]; !ok {
		t.Fatal("expected tun-in to be added")
	}
	if passthrough != 1 {
		t.Fatalf("passthrough = %d", passthrough)
	}
}
