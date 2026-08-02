package service

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xuthus5/boxd/internal/core"
)

func restartFailureMessage(err error) string {
	detail := ""
	if err != nil {
		detail = strings.TrimSpace(err.Error())
	}
	if detail == "" {
		return "restart failed after config save"
	}
	return "restart failed after config save: " + detail
}

func atomicWriteFile(path string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()

	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	if err := tempFile.Chmod(0600); err != nil {
		return err
	}
	if _, err := tempFile.Write(body); err != nil {
		return err
	}
	if err := tempFile.Sync(); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, path)
}

func rollbackConfigFile(path string, previous []byte, previousExists bool) error {
	if previousExists {
		return atomicWriteFile(path, previous)
	}
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func applyDNSDefaults(cfg map[string]any, result *core.DNSDefaultsResult) {
	cfg["dns"] = result.DNS
	route, _ := cfg["route"].(map[string]any)
	if route == nil {
		route = map[string]any{}
	}
	route["default_domain_resolver"] = result.DefaultDomainResolver
	cfg["route"] = route
}

func mergeRuleSets(existing []any, installed []map[string]any) []any {
	indexByTag := make(map[string]int, len(existing))
	merged := make([]any, 0, len(existing)+len(installed))
	for _, item := range existing {
		merged = append(merged, item)
		if m, ok := item.(map[string]any); ok {
			if tag, _ := m["tag"].(string); tag != "" {
				indexByTag[tag] = len(merged) - 1
			}
		}
	}
	for _, item := range installed {
		tag, _ := item["tag"].(string)
		if idx, ok := indexByTag[tag]; ok {
			merged[idx] = item
			continue
		}
		merged = append(merged, item)
	}
	return merged
}
