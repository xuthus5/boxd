package core

import "testing"

func TestDefaultExperimentalInstallerInstall(t *testing.T) {
	installer := NewDefaultExperimentalInstaller()
	result, err := installer.Install(map[string]any{}, "")
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
	result, err := installer.Install(cfg, "")
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

func TestDefaultExperimentalInstallerCacheFileWithPath(t *testing.T) {
	installer := NewDefaultExperimentalInstaller()
	cfg := map[string]any{
		"experimental": map[string]any{
			"cache_file": map[string]any{
				"enabled": true,
			},
		},
	}
	result, err := installer.Install(cfg, "/var/lib/boxd")
	if err != nil {
		t.Fatal(err)
	}
	cacheFile, ok := result.Experimental["cache_file"].(map[string]any)
	if !ok {
		t.Fatalf("cache_file missing: %#v", result.Experimental)
	}
	if cacheFile["enabled"] != true {
		t.Fatalf("cache_file.enabled = %#v", cacheFile["enabled"])
	}
	if cacheFile["path"] != "/var/lib/boxd/cache.db" {
		t.Fatalf("cache_file.path = %#v, want /var/lib/boxd/cache.db", cacheFile["path"])
	}
	installedCache, ok := result.Installed["cache_file"].(map[string]any)
	if !ok {
		t.Fatal("cache_file not in installed")
	}
	if installedCache["path"] != "/var/lib/boxd/cache.db" {
		t.Fatalf("installed cache_file.path = %#v", installedCache["path"])
	}
}

func TestDefaultExperimentalInstallerCacheFileWithExistingPath(t *testing.T) {
	installer := NewDefaultExperimentalInstaller()
	cfg := map[string]any{
		"experimental": map[string]any{
			"cache_file": map[string]any{
				"enabled": true,
				"path":    "/custom/path/cache.db",
			},
		},
	}
	result, err := installer.Install(cfg, "/var/lib/boxd")
	if err != nil {
		t.Fatal(err)
	}
	cacheFile := result.Experimental["cache_file"].(map[string]any)
	if cacheFile["path"] != "/custom/path/cache.db" {
		t.Fatalf("cache_file.path overwritten: %#v", cacheFile["path"])
	}
	if _, ok := result.Installed["cache_file"]; ok {
		t.Fatal("cache_file should not be in installed when path already exists")
	}
}

func TestDefaultExperimentalInstallerCacheFileDisabled(t *testing.T) {
	installer := NewDefaultExperimentalInstaller()
	cfg := map[string]any{
		"experimental": map[string]any{
			"cache_file": map[string]any{
				"enabled": false,
			},
		},
	}
	result, err := installer.Install(cfg, "/var/lib/boxd")
	if err != nil {
		t.Fatal(err)
	}
	cacheFile := result.Experimental["cache_file"].(map[string]any)
	if _, ok := cacheFile["path"]; ok {
		t.Fatal("cache_file.path should not be set when disabled")
	}
}
