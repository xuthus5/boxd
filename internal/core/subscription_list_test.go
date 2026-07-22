package core

import (
	"encoding/json"
	"testing"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
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

func TestSubscriptionListSortsByLastUpdatedDescending(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()
	manager := NewSubscriptionManager(db, t.TempDir())

	older, err := manager.Create(SubscriptionParams{Name: "older", URL: "https://example.com/a", IntervalMin: 60})
	if err != nil {
		t.Fatal(err)
	}
	newer, err := manager.Create(SubscriptionParams{Name: "newer", URL: "https://example.com/b", IntervalMin: 60})
	if err != nil {
		t.Fatal(err)
	}

	older.LastUpdated = older.LastUpdated.Add(-2 * time.Hour)
	newer.LastUpdated = newer.LastUpdated.Add(-time.Minute)
	if err := db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(subBucket)
		for _, sub := range []*model.Subscription{older, newer} {
			body, err := json.Marshal(sub)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(sub.ID), body); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("seed last_updated: %v", err)
	}

	subs, err := manager.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 2 {
		t.Fatalf("len = %d", len(subs))
	}
	if subs[0].ID != newer.ID || subs[1].ID != older.ID {
		t.Fatalf("order = %q then %q, want newer first", subs[0].Name, subs[1].Name)
	}
}
