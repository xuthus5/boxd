package main

import (
	"os"
	"path/filepath"
	"testing"
)

const desktopTestConfigJSON = `{
  "log": {"level": "warn"},
  "inbounds": [{"type": "mixed", "tag": "mixed-in", "listen": "127.0.0.1", "listen_port": 1080}],
  "outbounds": [
    {"type": "direct", "tag": "direct"},
    {"type": "block", "tag": "block"},
    {"type": "selector", "tag": "proxy", "outbounds": ["block"]}
  ],
  "route": {"final": "proxy"}
}`

// 普通接口测试复用已安装状态，完整离线初始化由 TestInitRuntimeEmbedded 覆盖。
func newTestRuntimeWithService(t *testing.T) *desktopRuntime {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := desktopConfig{
		Mode:            "embedded",
		DataDir:         filepath.Join(dir, "data"),
		ConfigPath:      filepath.Join(dir, "config", "config.json"),
		Username:        "admin",
		Password:        "",
		RefreshInterval: 60,
	}
	if err := os.MkdirAll(filepath.Dir(cfg.ConfigPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.ConfigPath, []byte(desktopTestConfigJSON), 0600); err != nil {
		t.Fatal(err)
	}
	rt, err := initRuntime(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := rt.close(); err != nil {
			t.Errorf("close test runtime: %v", err)
		}
	})
	return rt
}
