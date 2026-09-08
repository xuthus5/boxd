package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// BackupSetupSource 保存模块化重建前的原始字节，损坏 JSON 也可完整恢复。
func BackupSetupSource(path, dataDir, expectedHash string) error {
	body, err := ReadSetupSource(path)
	if err != nil {
		return err
	}
	if ConfigContentHash(body) != expectedHash {
		return ErrSetupStale
	}
	if len(body) == 0 {
		return nil
	}
	dir := filepath.Join(dataDir, "config-backups")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config backup directory: %w", err)
	}
	target := filepath.Join(dir, "before-setup-"+expectedHash+".json")
	created, err := writeInitialConfig(target, body)
	if err != nil {
		return fmt.Errorf("backup original configuration: %w", err)
	}
	if created {
		return nil
	}
	existing, err := os.ReadFile(target)
	if err != nil || ConfigContentHash(existing) != expectedHash {
		return fmt.Errorf("existing config backup could not be verified: %s", target)
	}
	return nil
}
