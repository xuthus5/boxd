package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// writeInitialConfig 使用同目录硬链接提交，避免覆盖并发创建的用户配置。
func writeInitialConfig(path string, body []byte) (bool, error) {
	tempPath, err := prepareInitialConfig(path, body)
	if err != nil {
		return false, err
	}
	defer func() { _ = os.Remove(tempPath) }()
	if err := os.Link(tempPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			_, existingErr := configFileExists(path)
			return false, existingErr
		}
		return false, fmt.Errorf("publish default config: %w", err)
	}
	return true, nil
}

func prepareInitialConfig(path string, body []byte) (string, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}
	file, err := os.CreateTemp(dir, ".boxd-config-*")
	if err != nil {
		return "", fmt.Errorf("create config temporary file: %w", err)
	}
	tempPath := file.Name()
	if err := writeAndCloseInitialConfig(file, body); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("write default config: %w", err)
	}
	return tempPath, nil
}

func writeAndCloseInitialConfig(file *os.File, body []byte) error {
	if _, err := file.Write(body); err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Sync(); err != nil {
		return errors.Join(err, file.Close())
	}
	return file.Close()
}
