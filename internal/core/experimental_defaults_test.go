package core

import "testing"

func TestDefaultExperimentalInstallerInstall(t *testing.T) {
	installer := NewDefaultExperimentalInstaller()
	result, err := installer.Install(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	clash, ok := result.Experimental["clash_api"].(map[string]any)
	if !ok {
		t.Fatalf("clash_api missing: %#v", result.Experimental)
	}
	if clash["external_controller"] != "127.0.0.1:9090" {
		t.Fatalf("controller = %#v", clash["external_controller"])
	}
	if clash["default_mode"] != "rule" {
		t.Fatalf("mode = %#v", clash["default_mode"])
	}
	if len(result.Installed) == 0 {
		t.Fatal("expected installed subset")
	}
}

func TestDefaultExperimentalInstallerPreservesExisting(t *testing.T) {
	installer := NewDefaultExperimentalInstaller()
	cfg := map[string]any{
		"experimental": map[string]any{
			"clash_api": map[string]any{
				"external_controller": "127.0.0.1:9999",
				"secret":              "keep-me",
			},
		},
	}
	result, err := installer.Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	clash := result.Experimental["clash_api"].(map[string]any)
	if clash["external_controller"] != "127.0.0.1:9999" {
		t.Fatalf("controller overwritten: %#v", clash["external_controller"])
	}
	if clash["secret"] != "keep-me" {
		t.Fatalf("secret lost: %#v", clash["secret"])
	}
	if clash["default_mode"] != "rule" {
		t.Fatalf("default_mode not filled: %#v", clash["default_mode"])
	}
}
