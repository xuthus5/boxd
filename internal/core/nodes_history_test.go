package core

import (
	"testing"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

func TestNodeManagerLatencyHistory(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()
	nm := NewNodeManager(db)

	base := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		result := model.TestResult{
			Tag: "hk-01", TestType: "tcp", Success: true, LatencyMs: float64(10 + i),
			Timestamp: base.Add(time.Duration(i) * time.Minute),
		}
		if err := nm.SaveTestResult("hk-01_tcp", result); err != nil {
			t.Fatal(err)
		}
	}
	if err := nm.SaveTestResult("hk-01_tcp", model.TestResult{
		Tag: "hk-01", TestType: "tcp", Success: false, Error: "timeout",
		Timestamp: base.Add(3 * time.Minute),
	}); err != nil {
		t.Fatal(err)
	}

	points := nm.GetTestHistory("hk-01")["tcp"]
	if len(points) != 4 {
		t.Fatalf("points = %d, want 4", len(points))
	}
	if !points[0].Success || points[0].LatencyMs != 10 {
		t.Fatalf("first = %+v", points[0])
	}
	if points[3].Success || points[3].Error != "timeout" {
		t.Fatalf("last = %+v", points[3])
	}
	if len(nm.GetAllTestHistory()["hk-01"]["tcp"]) != 4 {
		t.Fatal("all history missing")
	}

	for i := 0; i < defaultLatencyHistoryLimit+5; i++ {
		if err := nm.SaveTestResult("hk-01_tcp", model.TestResult{
			Tag: "hk-01", TestType: "tcp", Success: true, LatencyMs: float64(i),
			Timestamp: base.Add(time.Duration(10+i) * time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(nm.GetTestHistory("hk-01")["tcp"]); got != defaultLatencyHistoryLimit {
		t.Fatalf("capped len = %d", got)
	}

	if err := nm.Delete("hk-01"); err != nil {
		t.Fatal(err)
	}
	if len(nm.GetTestHistory("hk-01")) != 0 {
		t.Fatal("history should be deleted with node")
	}
}

func TestGetTestHistoryEmptyTag(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()
	nm := NewNodeManager(db)
	if len(nm.GetTestHistory("")) != 0 {
		t.Fatal("empty tag should return empty")
	}
	if err := nm.AppendTestHistory("", "tcp", model.LatencyPoint{Success: true}); err != nil {
		t.Fatal(err)
	}
	if err := nm.AppendTestHistory("x", "", model.LatencyPoint{Success: true}); err != nil {
		t.Fatal(err)
	}
}

func TestAppendTestHistoryCorruptAndDelete(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()
	nm := NewNodeManager(db)

	// corrupt payload should be overwritten
	if err := db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(historyBucket)
		return b.Put([]byte("bad"), []byte("{"))
	}); err != nil {
		t.Fatal(err)
	}
	if err := nm.AppendTestHistory("bad", "tcp", model.LatencyPoint{Success: true, LatencyMs: 1}); err != nil {
		t.Fatal(err)
	}
	if len(nm.GetTestHistory("bad")["tcp"]) != 1 {
		t.Fatal("expected recovery from corrupt history")
	}
	if err := nm.DeleteTestHistory("bad"); err != nil {
		t.Fatal(err)
	}
	if len(nm.GetTestHistory("bad")) != 0 {
		t.Fatal("delete history failed")
	}
	// delete missing is fine
	if err := nm.DeleteTestHistory("missing"); err != nil {
		t.Fatal(err)
	}
}

func TestSaveTestResultInfersTagAndTimestamp(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()
	nm := NewNodeManager(db)
	if err := nm.SaveTestResult("proxy_tcp", model.TestResult{TestType: "tcp", Success: true, LatencyMs: 9}); err != nil {
		t.Fatal(err)
	}
	points := nm.GetTestHistory("proxy")["tcp"]
	if len(points) != 1 || points[0].LatencyMs != 9 || points[0].Timestamp.IsZero() {
		t.Fatalf("inferred history = %+v", points)
	}
	all := nm.GetAllTestResults()
	if all["proxy_tcp"]["tcp"].Timestamp.IsZero() {
		t.Fatal("latest result missing timestamp")
	}
}

func TestGetAllTestHistorySkipsCorrupt(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()
	nm := NewNodeManager(db)
	if err := db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(historyBucket)
		if err := b.Put([]byte("good"), []byte(`{"by_type":{"tcp":[{"success":true,"latency_ms":1,"timestamp":"2026-07-23T00:00:00Z"}]}}`)); err != nil {
			return err
		}
		return b.Put([]byte("bad"), []byte(`{`))
	}); err != nil {
		t.Fatal(err)
	}
	all := nm.GetAllTestHistory()
	if len(all["good"]["tcp"]) != 1 {
		t.Fatalf("good missing: %+v", all)
	}
	if _, ok := all["bad"]; ok {
		t.Fatal("corrupt entry should be skipped")
	}
	// empty bucket view path via fresh manager after delete all
	if err := nm.DeleteTestHistory("good"); err != nil {
		t.Fatal(err)
	}
}

func TestHistoryMissingBucketAndNilByType(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()
	nm := NewNodeManager(db)

	// nil by_type payload
	if err := db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(historyBucket).Put([]byte("niltype"), []byte(`{"by_type":null}`))
	}); err != nil {
		t.Fatal(err)
	}
	if got := nm.GetTestHistory("niltype"); len(got) != 0 {
		t.Fatalf("nil by_type should be empty, got %#v", got)
	}
	if err := nm.AppendTestHistory("niltype", "http", model.LatencyPoint{Success: true, LatencyMs: 2}); err != nil {
		t.Fatal(err)
	}
	if len(nm.GetTestHistory("niltype")["http"]) != 1 {
		t.Fatal("append after nil by_type failed")
	}

	// drop history bucket to hit missing-bucket branches
	if err := db.Update(func(tx *bbolt.Tx) error {
		return tx.DeleteBucket(historyBucket)
	}); err != nil {
		t.Fatal(err)
	}
	if len(nm.GetTestHistory("x")) != 0 {
		t.Fatal("missing bucket GetTestHistory")
	}
	if len(nm.GetAllTestHistory()) != 0 {
		t.Fatal("missing bucket GetAllTestHistory")
	}
	if err := nm.DeleteTestHistory("x"); err != nil {
		t.Fatal(err)
	}
	// Append recreates bucket
	if err := nm.AppendTestHistory("again", "tcp", model.LatencyPoint{Success: false, Error: "e"}); err != nil {
		t.Fatal(err)
	}
	if len(nm.GetTestHistory("again")["tcp"]) != 1 {
		t.Fatal("recreate bucket failed")
	}
}
