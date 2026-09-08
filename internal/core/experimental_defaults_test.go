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
	if _, exists := clash["external_controller"]; exists {
		t.Fatalf("internal mode control does not need a listener: %#v", clash)
	}
	if clash["default_mode"] != "rule" {
		t.Fatalf("mode = %#v", clash["default_mode"])
	}
	if len(result.Installed) == 0 {
		t.Fatal("expected installed subset")
	}
	if _, exists := result.Experimental["cache_file"]; exists {
		t.Fatal("must not emit an empty cache configuration without a data directory")
	}
}

func TestDefaultExperimentalInstallerEnablesSelectionCache(t *testing.T) {
	result, err := NewDefaultExperimentalInstaller().Install(map[string]any{}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cache, _ := result.Experimental["cache_file"].(map[string]any)
	if cache["enabled"] != true || cache["path"] == "" {
		t.Fatalf("selector choices need a private persistent cache: %#v", cache)
	}
	again, err := NewDefaultExperimentalInstaller().Install(map[string]any{"experimental": result.Experimental}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Installed) != 0 {
		t.Fatalf("repeated installation should preserve defaults: %#v", again.Installed)
	}
}

func TestDefaultExperimentalInstallerRequiresCacheDirectory(t *testing.T) {
	cfg := map[string]any{"experimental": map[string]any{"cache_file": map[string]any{"enabled": true}}}
	if _, err := NewDefaultExperimentalInstaller().Install(cfg, ""); err == nil {
		t.Fatal("an enabled cache must not silently use the working directory")
	}
}

func TestDefaultExperimentalInstallerRemovesEmptyCache(t *testing.T) {
	cfg := map[string]any{"experimental": map[string]any{"cache_file": map[string]any{}}}
	result, err := NewDefaultExperimentalInstaller().Install(cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := result.Experimental["cache_file"]; exists {
		t.Fatal("empty cache configuration must be omitted")
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
