package core

import (
	"testing"

	"go.etcd.io/bbolt"
)

func setupSettingsDB(t *testing.T) (*bbolt.DB, func()) {
	t.Helper()
	path := t.TempDir() + "/test.db"
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	return db, func() {
		_ = db.Close()
	}
}

func TestNewSettingsManager(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()

	sm := NewSettingsManager(db)
	if sm == nil {
		t.Fatal("expected non-nil SettingsManager")
	}
}

func TestSettingsGetSet(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()

	sm := NewSettingsManager(db)

	val := sm.Get("non_existent")
	if val != "" {
		t.Errorf("expected empty for missing key, got '%s'", val)
	}

	if err := sm.Set("url_test", "http://example.com/"); err != nil {
		t.Fatal(err)
	}

	val = sm.Get("url_test")
	if val != "http://example.com/" {
		t.Errorf("expected 'http://example.com/', got '%s'", val)
	}
}

func TestSettingsUpdateExisting(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()

	sm := NewSettingsManager(db)

	if err := sm.Set("key1", "value1"); err != nil {
		t.Fatal(err)
	}
	if err := sm.Set("key1", "value2"); err != nil {
		t.Fatal(err)
	}

	val := sm.Get("key1")
	if val != "value2" {
		t.Errorf("expected 'value2', got '%s'", val)
	}
}

func TestSettingsMultipleKeys(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()

	sm := NewSettingsManager(db)

	if err := sm.Set("key1", "val1"); err != nil {
		t.Fatal(err)
	}
	if err := sm.Set("key2", "val2"); err != nil {
		t.Fatal(err)
	}

	if sm.Get("key1") != "val1" {
		t.Error("key1 mismatch")
	}
	if sm.Get("key2") != "val2" {
		t.Error("key2 mismatch")
	}
}
