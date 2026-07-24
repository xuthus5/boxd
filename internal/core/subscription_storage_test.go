package core

import (
	"errors"
	"testing"

	"go.etcd.io/bbolt"
)

func TestDecodeStoredSubscription(t *testing.T) {
	if _, err := decodeStoredSubscription(nil, "missing"); err == nil {
		t.Fatal("nil bucket should fail")
	}
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)
	manager := NewSubscriptionManager(db, t.TempDir())
	if err := db.View(func(tx *bbolt.Tx) error {
		got, decodeErr := decodeStoredSubscription(tx.Bucket(subBucket), "missing")
		if decodeErr != nil || got != nil {
			return errors.Join(decodeErr, errors.New("missing subscription should return nil"))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(subBucket).Put([]byte("broken"), []byte("{"))
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.View(func(tx *bbolt.Tx) error {
		_, decodeErr := decodeStoredSubscription(tx.Bucket(subBucket), "broken")
		return decodeErr
	}); err == nil {
		t.Fatal("invalid subscription should fail to decode")
	}
	if manager.Get("missing") != nil {
		t.Fatal("missing subscription should not be returned")
	}
}
