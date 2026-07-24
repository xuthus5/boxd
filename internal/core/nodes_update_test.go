package core

import (
	"errors"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

func TestNodeManagerRenamePreservesProbeState(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()

	nm := NewNodeManager(db)
	if err := nm.Add(model.Outbound{Tag: "old", Type: "vless", Server: "old.example", Port: 443}); err != nil {
		t.Fatal(err)
	}
	if err := nm.SaveTestResult("old_tcp", model.TestResult{Tag: "old", TestType: "tcp", Success: true, LatencyMs: 12}); err != nil {
		t.Fatal(err)
	}
	if err := nm.AppendTestHistory("old", "tcp", model.LatencyPoint{Success: true, LatencyMs: 12}); err != nil {
		t.Fatal(err)
	}

	err := nm.Update("old", model.Outbound{Tag: "new", Type: "vless", Server: "new.example", Port: 8443})
	if err != nil {
		t.Fatalf("rename error = %v", err)
	}
	if nm.Get("old") != nil || nm.Get("new") == nil {
		t.Fatal("rename did not move node")
	}
	if _, ok := nm.GetAllTestResults()["new_tcp"]; !ok {
		t.Fatal("latest test result was not moved")
	}
	if _, ok := nm.GetAllTestResults()["old_tcp"]; ok {
		t.Fatal("old latest test result was not removed")
	}
	if moved := nm.GetAllTestResults()["new_tcp"]["tcp"]; moved.Tag != "new" {
		t.Fatalf("moved result tag = %q, want new", moved.Tag)
	}
	if len(nm.GetTestHistory("new")["tcp"]) != 2 || len(nm.GetTestHistory("old")) != 0 {
		t.Fatal("probe history was not moved")
	}
}

func TestNodeManagerUpdateSameTagPreservesState(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()

	nm := NewNodeManager(db)
	if err := nm.Add(model.Outbound{Tag: "same", Type: "vless"}); err != nil {
		t.Fatal(err)
	}
	if err := nm.SaveTestResult("same_tcp", model.TestResult{Tag: "same", TestType: "tcp", Success: true}); err != nil {
		t.Fatal(err)
	}
	if err := nm.Update("same", model.Outbound{Tag: "same", Type: "trojan"}); err != nil {
		t.Fatal(err)
	}
	if got := nm.Get("same"); got == nil || got.Type != "trojan" {
		t.Fatalf("updated node = %+v", got)
	}
	if got := len(nm.GetTestHistory("same")["tcp"]); got != 1 {
		t.Fatalf("history length = %d, want 1", got)
	}
}

func TestNodeManagerUpdateMissingNodeAndMarshalError(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()

	nm := NewNodeManager(db)
	if err := nm.Update("missing", model.Outbound{Tag: "new", Type: "vless"}); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("missing update error = %v", err)
	}
	if err := nm.Add(model.Outbound{Tag: "old", Type: "vless"}); err != nil {
		t.Fatal(err)
	}
	err := nm.Update("old", model.Outbound{Tag: "new", Type: "vless", Raw: func() {}})
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestNodeManagerRenameKeepsUnrelatedAndCorruptResults(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()

	nm := NewNodeManager(db)
	if err := nm.Add(model.Outbound{Tag: "old", Type: "vless"}); err != nil {
		t.Fatal(err)
	}
	if err := db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(resultBucket)
		if err := b.Put([]byte("old_bad"), []byte("{")); err != nil {
			return err
		}
		return b.Put([]byte("oldish_tcp"), []byte(`{"results":{"tcp":{"tag":"oldish","test_type":"tcp"}}}`))
	}); err != nil {
		t.Fatal(err)
	}
	if err := nm.Update("old", model.Outbound{Tag: "new", Type: "vless"}); err != nil {
		t.Fatal(err)
	}
	var movedCorrupt, oldCorrupt, unrelatedResult bool
	if err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(resultBucket)
		movedCorrupt = b.Get([]byte("new_bad")) != nil
		oldCorrupt = b.Get([]byte("old_bad")) != nil
		unrelatedResult = b.Get([]byte("oldish_tcp")) != nil
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !movedCorrupt || oldCorrupt {
		t.Fatal("corrupt result was not moved")
	}
	if !unrelatedResult {
		t.Fatal("unrelated result was moved")
	}
}

func TestNodeManagerOptionalBucketsMayBeMissing(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()

	nm := NewNodeManager(db)
	if err := nm.Add(model.Outbound{Tag: "old", Type: "vless"}); err != nil {
		t.Fatal(err)
	}
	if err := db.Update(func(tx *bbolt.Tx) error {
		if err := tx.DeleteBucket(resultBucket); err != nil {
			return err
		}
		return tx.DeleteBucket(historyBucket)
	}); err != nil {
		t.Fatal(err)
	}
	if err := nm.Update("old", model.Outbound{Tag: "new", Type: "vless"}); err != nil {
		t.Fatal(err)
	}
	if err := nm.Delete("new"); err != nil {
		t.Fatal(err)
	}
}

func TestNodeManagerRenameRejectsTagConflict(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()

	nm := NewNodeManager(db)
	if err := nm.Add(model.Outbound{Tag: "old", Type: "vless", Server: "old.example"}); err != nil {
		t.Fatal(err)
	}
	if err := nm.Add(model.Outbound{Tag: "taken", Type: "trojan", Server: "taken.example"}); err != nil {
		t.Fatal(err)
	}

	err := nm.Update("old", model.Outbound{Tag: "taken", Type: "vless", Server: "new.example"})
	if !errors.Is(err, ErrNodeTagConflict) {
		t.Fatalf("error = %v, want ErrNodeTagConflict", err)
	}
	if nm.Get("old") == nil || nm.Get("taken").Type != "trojan" {
		t.Fatal("tag conflict changed existing nodes")
	}
}

func TestNodeManagerDeleteRemovesProbeResults(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()

	nm := NewNodeManager(db)
	if err := nm.Add(model.Outbound{Tag: "remove-me", Type: "vless"}); err != nil {
		t.Fatal(err)
	}
	if err := nm.SaveTestResult("remove-me_tcp", model.TestResult{Tag: "remove-me", TestType: "tcp", Success: true}); err != nil {
		t.Fatal(err)
	}
	if err := nm.SaveTestResult("remove-me_http", model.TestResult{Tag: "remove-me", TestType: "http", Success: true}); err != nil {
		t.Fatal(err)
	}
	if err := nm.SaveTestResult("remove-me-extra_tcp", model.TestResult{Tag: "remove-me-extra", TestType: "tcp", Success: true}); err != nil {
		t.Fatal(err)
	}
	if err := db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(resultBucket).Put([]byte("remove-me"), []byte(`{"results":{"legacy":{"tag":"remove-me","test_type":"legacy"}}}`))
	}); err != nil {
		t.Fatal(err)
	}
	if err := nm.Delete("remove-me"); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"remove-me", "remove-me_tcp", "remove-me_http"} {
		if _, ok := nm.GetAllTestResults()[key]; ok {
			t.Fatalf("latest test result %q was not removed", key)
		}
	}
	if _, ok := nm.GetAllTestResults()["remove-me-extra_tcp"]; !ok {
		t.Fatal("unrelated test result was removed")
	}
}
