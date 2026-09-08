package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

type initialConfigSnapshot struct {
	body []byte
	info os.FileInfo
}

func inspectBootstrapConfig(path string) (*initialConfigSnapshot, bool, error) {
	exists, err := configFileExists(path)
	if err != nil || !exists {
		return nil, !exists, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, false, fmt.Errorf("inspect existing config: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, false, nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, false, fmt.Errorf("read existing config: %w", err)
	}
	if !bootstrapConfigNeedsRecovery(body) {
		return nil, false, nil
	}
	return &initialConfigSnapshot{body: body, info: info}, true, nil
}

func bootstrapConfigNeedsRecovery(body []byte) bool {
	if len(bytes.TrimSpace(body)) == 0 {
		return true
	}
	var cfg map[string]any
	if err := json.Unmarshal(body, &cfg); err != nil {
		return false
	}
	if len(cfg) == 0 {
		return true
	}
	for _, listen := range []string{"::", "127.0.0.1", "0.0.0.0"} {
		if equalInboundJSON(cfg, legacyBootstrapConfig(listen)) {
			return true
		}
	}
	return false
}

func legacyBootstrapConfig(listen string) map[string]any {
	return map[string]any{
		"log":      map[string]any{"level": "info", "timestamp": true},
		"inbounds": []any{mixedInboundFor(NetworkCapabilities{DefaultListen: listen})},
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "block", "tag": "block"},
		},
		"route": map[string]any{"final": "direct"},
	}
}

func initialSnapshotMatches(path string, snapshot initialConfigSnapshot) (bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() || !os.SameFile(info, snapshot.info) || !info.ModTime().Equal(snapshot.info.ModTime()) {
		return false, nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	return bytes.Equal(body, snapshot.body), nil
}
