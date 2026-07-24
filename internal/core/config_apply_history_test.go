package core

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

func setupConfigApplyHistoryDB(t *testing.T) (*bbolt.DB, func()) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "boxd.db")
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	return db, func() { _ = db.Close() }
}

func TestConfigApplyHistoryAppendAndList(t *testing.T) {
	db, cleanup := setupConfigApplyHistoryDB(t)
	defer cleanup()
	manager := NewConfigApplyHistoryManager(db)

	first := NewConfigApplyEvent("update", model.StatusOK, []byte(`{"a":1}`), nil)
	first.AppliedAt = time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	if err := manager.Append(first); err != nil {
		t.Fatal(err)
	}
	second := NewConfigApplyEvent("raw", model.StatusRolledBack, []byte(`{"a":2}`), errors.New("restart failed"))
	second.AppliedAt = time.Date(2026, 7, 23, 11, 0, 0, 0, time.UTC)
	if err := manager.Append(second); err != nil {
		t.Fatal(err)
	}

	events, err := manager.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("len = %d, want 2", len(events))
	}
	if events[0].Source != "raw" || events[0].Status != "rolled_back" {
		t.Fatalf("newest = %+v", events[0])
	}
	if events[0].Error != "restart failed" {
		t.Fatalf("error = %q", events[0].Error)
	}
	if events[0].ErrorCode != KernelErrorRestartFailed {
		t.Fatalf("error_code = %q", events[0].ErrorCode)
	}
	if events[1].Source != "update" || events[1].Status != "applied" {
		t.Fatalf("older = %+v", events[1])
	}
	if events[0].Hash == events[1].Hash {
		t.Fatal("expected different hashes")
	}
}

func TestConfigApplyHistoryTrimsToLimit(t *testing.T) {
	db, cleanup := setupConfigApplyHistoryDB(t)
	defer cleanup()
	manager := NewConfigApplyHistoryManager(db)
	manager.limit = 3

	for i := 0; i < 5; i++ {
		event := NewConfigApplyEvent("update", model.StatusOK, []byte{byte(i)}, nil)
		event.ID = fmt.Sprintf("e-%d", i)
		if err := manager.Append(event); err != nil {
			t.Fatal(err)
		}
	}
	events, err := manager.List(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("len = %d, want 3", len(events))
	}
}

func TestConfigApplyHistoryListEmptyAndNil(t *testing.T) {
	events, err := (*ConfigApplyHistoryManager)(nil).List(5)
	if err != nil || len(events) != 0 {
		t.Fatalf("nil manager = %v %v", events, err)
	}
	db, cleanup := setupConfigApplyHistoryDB(t)
	defer cleanup()
	manager := NewConfigApplyHistoryManager(db)
	events, err = manager.List(5)
	if err != nil || len(events) != 0 {
		t.Fatalf("empty = %v %v", events, err)
	}
	if ConfigBodyHash(nil) == "" {
		t.Fatal("expected hash for empty body")
	}
	event := NewConfigApplyEvent(" ", " ", nil, nil)
	if event.Source != "unknown" || event.Status != "applied" {
		t.Fatalf("defaults = %+v", event)
	}
}

func TestConfigApplyHistoryIgnoresCorruptPayload(t *testing.T) {
	db, cleanup := setupConfigApplyHistoryDB(t)
	defer cleanup()
	if err := db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(configApplyHistoryBucket)
		if err != nil {
			return err
		}
		return bucket.Put(configApplyHistoryKey, []byte("not-json"))
	}); err != nil {
		t.Fatal(err)
	}
	manager := NewConfigApplyHistoryManager(db)
	events, err := manager.List(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("corrupt list = %+v", events)
	}
	if err := manager.Append(NewConfigApplyEvent("dns_defaults", model.StatusOK, []byte(`{}`), nil)); err != nil {
		t.Fatal(err)
	}
	events, err = manager.List(5)
	if err != nil || len(events) != 1 {
		t.Fatalf("after append = %v %v", events, err)
	}
}

func TestConfigApplyHistoryValidateStatuses(t *testing.T) {
	db, cleanup := setupConfigApplyHistoryDB(t)
	defer cleanup()
	manager := NewConfigApplyHistoryManager(db)

	ok := NewConfigApplyEvent("validate", "validated", []byte(`{"ok":true}`), nil)
	if err := manager.Append(ok); err != nil {
		t.Fatal(err)
	}
	failed := NewConfigApplyEvent("validate", "validate_failed", []byte(`{"bad":true}`), errors.New("inbounds[0].type: required"))
	if err := manager.Append(failed); err != nil {
		t.Fatal(err)
	}
	events, err := manager.List(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("len = %d", len(events))
	}
	if events[0].Status != "validate_failed" || events[0].Source != "validate" {
		t.Fatalf("failed event = %+v", events[0])
	}
	if events[0].Error == "" || events[0].ErrorCode == "" {
		t.Fatalf("expected error fields, got %+v", events[0])
	}
	if events[1].Status != "validated" {
		t.Fatalf("ok event = %+v", events[1])
	}
}
