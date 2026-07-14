package config

import "testing"

func TestParseDefaults(t *testing.T) {
	cfg := Parse()
	if cfg.Listen != "[::]:9091" {
		t.Errorf("expected default listen [::]:9091, got %s", cfg.Listen)
	}
	if cfg.Username != "admin" {
		t.Errorf("expected default username admin, got %s", cfg.Username)
	}
	if cfg.RefreshInterval != 60 {
		t.Errorf("expected default refresh interval 60, got %d", cfg.RefreshInterval)
	}
	if cfg.CORSAllowedOrigins != nil {
		t.Errorf("expected nil cors origins, got %#v", cfg.CORSAllowedOrigins)
	}
}

func TestParseFromEnv(t *testing.T) {
	t.Setenv("BOXUI_LISTEN", "0.0.0.0:8080")
	t.Setenv("BOXUI_USERNAME", "testuser")
	t.Setenv("BOXUI_PASSWORD", "testpass")
	t.Setenv("BOXUI_DATA_DIR", "/tmp/boxui-test")
	t.Setenv("BOXUI_REFRESH_INTERVAL", "30")
	t.Setenv("BOXUI_CORS_ALLOWED_ORIGINS", "https://a.example, https://b.example ")

	cfg := Parse()
	if cfg.Listen != "0.0.0.0:8080" {
		t.Errorf("expected 0.0.0.0:8080, got %s", cfg.Listen)
	}
	if cfg.Username != "testuser" {
		t.Errorf("expected testuser, got %s", cfg.Username)
	}
	if cfg.Password != "testpass" {
		t.Errorf("expected testpass, got %s", cfg.Password)
	}
	if cfg.DataDir != "/tmp/boxui-test" {
		t.Errorf("expected /tmp/boxui-test, got %s", cfg.DataDir)
	}
	if cfg.RefreshInterval != 30 {
		t.Errorf("expected 30, got %d", cfg.RefreshInterval)
	}
	if len(cfg.CORSAllowedOrigins) != 2 || cfg.CORSAllowedOrigins[0] != "https://a.example" || cfg.CORSAllowedOrigins[1] != "https://b.example" {
		t.Errorf("unexpected cors origins %#v", cfg.CORSAllowedOrigins)
	}
}

func TestGetEnv(t *testing.T) {
	t.Setenv("TEST_EXISTS", "value")

	if v := getEnv("TEST_EXISTS", "default"); v != "value" {
		t.Errorf("expected value, got %s", v)
	}
	if v := getEnv("TEST_MISSING", "default"); v != "default" {
		t.Errorf("expected default, got %s", v)
	}
}

func TestGetEnvInt(t *testing.T) {
	t.Setenv("TEST_INT", "42")

	if v := getEnvInt("TEST_INT", 0); v != 42 {
		t.Errorf("expected 42, got %d", v)
	}
	if v := getEnvInt("TEST_MISSING", 10); v != 10 {
		t.Errorf("expected 10, got %d", v)
	}
	if v := getEnvInt("TEST_INVALID", 5); v != 5 {
		t.Errorf("expected 5 on invalid, got %d", v)
	}
}

func TestGetEnvList(t *testing.T) {
	t.Setenv("TEST_LIST", "a, b ,,c ")

	if v := getEnvList("TEST_LIST"); len(v) != 3 || v[0] != "a" || v[1] != "b" || v[2] != "c" {
		t.Errorf("unexpected list %#v", v)
	}
	if v := getEnvList("TEST_LIST_MISSING"); v != nil {
		t.Errorf("expected nil for missing list, got %#v", v)
	}
}

func TestParseWithPortEnv(t *testing.T) {
	t.Setenv("BOXUI_PORT", "7777")
	cfg := Parse()
	if cfg.Listen != "[::]:7777" {
		t.Errorf("expected [::]:7777, got %s", cfg.Listen)
	}
}

func TestParseListenOverridesPort(t *testing.T) {
	t.Setenv("BOXUI_LISTEN", "0.0.0.0:8888")
	t.Setenv("BOXUI_PORT", "7777")
	cfg := Parse()
	if cfg.Listen != "0.0.0.0:8888" {
		t.Errorf("expected 0.0.0.0:8888 (listen overrides port), got %s", cfg.Listen)
	}
}

func TestResolveListen(t *testing.T) {
	t.Setenv("BOXUI_LISTEN", "")
	t.Setenv("BOXUI_PORT", "")
	if v := resolveListen(); v != "[::]:9091" {
		t.Errorf("expected [::]:9091, got %s", v)
	}

	t.Setenv("BOXUI_PORT", "12345")
	if v := resolveListen(); v != "[::]:12345" {
		t.Errorf("expected [::]:12345, got %s", v)
	}

	t.Setenv("BOXUI_LISTEN", "127.0.0.1:80")
	if v := resolveListen(); v != "127.0.0.1:80" {
		t.Errorf("expected 127.0.0.1:80, got %s", v)
	}
}
