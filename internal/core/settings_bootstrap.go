package core

import (
	"errors"
	"fmt"

	"go.etcd.io/bbolt"
)

// EnsureKernelAutostartDefault 仅为新建或恢复的默认配置设置安全自启。
// 事务内检查已有值，保留用户明确设置的 false 和其他既有选择。
func (m *SettingsManager) EnsureKernelAutostartDefault(created bool) (bool, error) {
	if m == nil || m.db == nil {
		return false, errors.New("settings database is unavailable")
	}
	enabled := false
	err := m.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(settingsBucket)
		if err != nil {
			return err
		}
		key := []byte("kernel_autostart")
		value := bucket.Get(key)
		if value == nil && created {
			if err := bucket.Put(key, []byte("true")); err != nil {
				return err
			}
			enabled = true
			return nil
		}
		enabled = string(value) == "true"
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("initialize kernel autostart: %w", err)
	}
	return enabled, nil
}
