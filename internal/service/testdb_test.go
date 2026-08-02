package service

import (
	"path/filepath"
	"strings"
	"testing"

	"go.etcd.io/bbolt"
)

func newTestDB(t *testing.T) *bbolt.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func containsSubstring(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
