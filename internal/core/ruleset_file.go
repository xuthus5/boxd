package core

import (
	"os"
	"path/filepath"
	"time"
)

func atomicWriteFile0600(path string, data []byte) error {
	return atomicWriteRuleSetFile(path, data, time.Time{})
}

func atomicWriteRuleSetFile(path string, data []byte, modified time.Time) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".ruleset-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}
	if !modified.IsZero() {
		if err := os.Chtimes(tmpName, modified, modified); err != nil {
			_ = tmp.Close()
			return err
		}
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}
