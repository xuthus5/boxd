package core

import (
	"testing"

	"go.etcd.io/bbolt"
)

func TestSubscriptionListReturnsDecodeError(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()
	manager := NewSubscriptionManager(db, t.TempDir())
	if err := db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(subBucket).Put([]byte("broken"), []byte("{"))
	}); err != nil {
		t.Fatalf("saving invalid subscription: %v", err)
	}

	if _, err := manager.List(); err == nil {
		t.Fatal("expected subscription decode error")
	}
}

func TestSubscriptionListReturnsDatabaseError(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()
	manager := NewSubscriptionManager(db, t.TempDir())
	if err := db.Close(); err != nil {
		t.Fatalf("closing database: %v", err)
	}

	if _, err := manager.List(); err == nil {
		t.Fatal("expected subscription database error")
	}
}
