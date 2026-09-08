package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var initialRecoveryMu sync.Mutex

type initialRecoveryIO struct {
	prepare func(string, []byte) (string, error)
	claim   func(string) (string, error)
	match   func(string, initialConfigSnapshot) (bool, error)
	secure  func(string) error
	publish func(string, string) error
}

type initialRecovery struct {
	path     string
	snapshot initialConfigSnapshot
	io       initialRecoveryIO
}

func defaultInitialRecoveryIO() initialRecoveryIO {
	return initialRecoveryIO{
		prepare: prepareInitialConfig, claim: claimInitialConfig, match: initialSnapshotMatches,
		secure: secureInitialBackup, publish: os.Link,
	}
}

// 恢复先保留原文件，再使用排他硬链接发布，避免覆盖并发写入的新配置。
func recoverInitialConfig(path string, body []byte, snapshot initialConfigSnapshot) (bool, error) {
	initialRecoveryMu.Lock()
	defer initialRecoveryMu.Unlock()
	recovery := initialRecovery{path: path, snapshot: snapshot, io: defaultInitialRecoveryIO()}
	return recovery.apply(body)
}

func (r initialRecovery) apply(body []byte) (bool, error) {
	if matches, err := r.io.match(r.path, r.snapshot); !matches || err != nil {
		return false, err
	}
	staged, err := r.io.prepare(r.path, body)
	if err != nil {
		return false, err
	}
	defer func() { _ = os.Remove(staged) }()
	backup, err := r.io.claim(r.path)
	if err != nil {
		return false, err
	}
	if matches, err := r.io.match(backup, r.snapshot); !matches || err != nil {
		return restoreInitialClaim(r.path, backup, err)
	}
	if err := r.io.secure(backup); err != nil {
		return restoreInitialClaim(r.path, backup, err)
	}
	if err := r.io.publish(staged, r.path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return false, nil
		}
		return restoreInitialClaim(r.path, backup, err)
	}
	return true, nil
}

func claimInitialConfig(path string) (string, error) {
	file, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".boxd-backup-*")
	if err != nil {
		return "", fmt.Errorf("prepare config backup: %w", err)
	}
	backup := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(backup)
		return "", err
	}
	if err := os.Rename(path, backup); err != nil {
		_ = os.Remove(backup)
		return "", fmt.Errorf("preserve previous config: %w", err)
	}
	return backup, nil
}

func secureInitialBackup(path string) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	if err := file.Chmod(0600); err != nil {
		return errors.Join(err, file.Close())
	}
	return errors.Join(file.Sync(), file.Close())
}

func restoreInitialClaim(path, backup string, cause error) (bool, error) {
	if err := os.Link(backup, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return false, cause
		}
		return false, fmt.Errorf("restore config preserved at %s: %w", backup, errors.Join(cause, err))
	}
	return false, errors.Join(cause, os.Remove(backup))
}
