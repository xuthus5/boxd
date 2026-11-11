package runtime

import (
	"errors"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := loadConfig(func(string) string { return "" }, func() (string, error) {
		return "/home/tester", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != modeEmbedded {
		t.Errorf("mode = %q", cfg.Mode)
	}
	if cfg.DataDir != "/home/tester/.boxd" {
		t.Errorf("data dir = %q", cfg.DataDir)
	}
	if cfg.ConfigPath != "/home/tester/.boxd/config.json" {
		t.Errorf("config path = %q", cfg.ConfigPath)
	}
	if cfg.Username != "admin" {
		t.Errorf("username = %q", cfg.Username)
	}
	if cfg.RefreshInterval != 60 {
		t.Errorf("refresh interval = %d", cfg.RefreshInterval)
	}
}

func TestLoadConfigEnvironment(t *testing.T) {
	values := map[string]string{
		"BOXD_DESKTOP_MODE":          "remote",
		"BOXD_REMOTE_URL":            "https://boxd.example",
		"BOXD_DATA_DIR":              "/tmp/boxd-data",
		"BOXD_CONFIG":                "/tmp/boxd-data/config.json",
		"BOXD_USERNAME":              "operator",
		"BOXD_PASSWORD":              "secret",
		"BOXD_SKIP_KERNEL_AUTOSTART": "1",
	}
	cfg, err := loadConfig(func(key string) string { return values[key] }, func() (string, error) {
		return "", errors.New("no home")
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != modeRemote || cfg.RemoteURL != "https://boxd.example" {
		t.Errorf("remote config = %+v", cfg)
	}
	if cfg.DataDir != "/tmp/boxd-data" || cfg.ConfigPath != "/tmp/boxd-data/config.json" {
		t.Errorf("paths = %q, %q", cfg.DataDir, cfg.ConfigPath)
	}
	if cfg.Username != "operator" || cfg.Password != "secret" {
		t.Errorf("credentials = %q, %q", cfg.Username, cfg.Password)
	}
	if !cfg.SkipAutostart {
		t.Error("skip autostart was not parsed")
	}
}

func TestOpenRejectsRemoteMode(t *testing.T) {
	if _, err := Open(Config{Mode: modeRemote}); err == nil {
		t.Fatal("expected remote mode error")
	}
}
