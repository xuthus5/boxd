package core

import (
	"go.etcd.io/bbolt"
)

var settingsBucket = []byte("settings")

type SettingsManager struct {
	db *bbolt.DB
}

func NewSettingsManager(db *bbolt.DB) *SettingsManager {
	_ = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(settingsBucket)
		return err
	})
	return &SettingsManager{db: db}
}

func (m *SettingsManager) Get(key string) string {
	var val string
	_ = m.db.View(func(tx *bbolt.Tx) error {
		data := tx.Bucket(settingsBucket).Get([]byte(key))
		if data != nil {
			val = string(data)
		}
		return nil
	})
	return val
}

func (m *SettingsManager) Set(key, value string) error {
	return m.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(settingsBucket).Put([]byte(key), []byte(value))
	})
}
