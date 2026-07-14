package core

import (
	"testing"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxui/internal/model"
)

func setupNodesDB(t *testing.T) (*bbolt.DB, func()) {
	t.Helper()
	path := t.TempDir() + "/nodes_test.db"
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	return db, func() {
		_ = db.Close()
	}
}

func TestNewNodeManager(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()

	nm := NewNodeManager(db)
	if nm == nil {
		t.Fatal("expected non-nil NodeManager")
	}
}

func TestNodeManagerAddAndList(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()

	nm := NewNodeManager(db)
	ob := model.Outbound{Tag: "test-node", Type: "vless", Server: "1.2.3.4", Port: 443}

	if err := nm.Add(ob); err != nil {
		t.Fatal(err)
	}

	nodes := nm.List()
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Tag != "test-node" {
		t.Errorf("expected tag 'test-node', got '%s'", nodes[0].Tag)
	}
}

func TestNodeManagerGet(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()

	nm := NewNodeManager(db)
	ob := model.Outbound{Tag: "node1", Type: "vmess", Server: "5.6.7.8", Port: 80}
	if err := nm.Add(ob); err != nil {
		t.Fatal(err)
	}

	got := nm.Get("node1")
	if got == nil {
		t.Fatal("expected non-nil node")
	}
	if got.Server != "5.6.7.8" {
		t.Errorf("expected server '5.6.7.8', got '%s'", got.Server)
	}

	missing := nm.Get("non_existent")
	if missing != nil {
		t.Error("expected nil for missing node")
	}
}

func TestNodeManagerDelete(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()

	nm := NewNodeManager(db)
	if err := nm.Add(model.Outbound{Tag: "node1", Type: "vless", Server: "1.2.3.4", Port: 443}); err != nil {
		t.Fatal(err)
	}
	if err := nm.Add(model.Outbound{Tag: "node2", Type: "vmess", Server: "5.6.7.8", Port: 80}); err != nil {
		t.Fatal(err)
	}

	if err := nm.Delete("node1"); err != nil {
		t.Fatal(err)
	}

	nodes := nm.List()
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node after delete, got %d", len(nodes))
	}
	if nodes[0].Tag != "node2" {
		t.Errorf("expected remaining node 'node2', got '%s'", nodes[0].Tag)
	}
}

func TestNodeManagerSaveAndGetTestResults(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()

	nm := NewNodeManager(db)

	r1 := model.TestResult{Tag: "node1", TestType: "tcp", Success: true, LatencyMs: 10.5}
	if err := nm.SaveTestResult("node1_tcp", r1); err != nil {
		t.Fatal(err)
	}

	r2 := model.TestResult{Tag: "node1", TestType: "http", Success: true, LatencyMs: 20.3}
	if err := nm.SaveTestResult("node1_http", r2); err != nil {
		t.Fatal(err)
	}

	all := nm.GetAllTestResults()
	if len(all) != 2 {
		t.Fatalf("expected 2 results, got %d", len(all))
	}

	tcpResults, ok := all["node1_tcp"]
	if !ok {
		t.Fatal("node1_tcp not found")
	}
	if tcpResults["tcp"].LatencyMs != 10.5 {
		t.Errorf("expected latency 10.5, got %f", tcpResults["tcp"].LatencyMs)
	}
}

func TestNodeManagerEmptyList(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()

	nm := NewNodeManager(db)
	nodes := nm.List()
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}
}

func TestNodeManagerEmptyTestResults(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()

	nm := NewNodeManager(db)
	all := nm.GetAllTestResults()
	if len(all) != 0 {
		t.Errorf("expected 0 results, got %d", len(all))
	}
}

func TestNodeManagerDeleteWithRaw(t *testing.T) {
	db, cleanup := setupNodesDB(t)
	defer cleanup()

	nm := NewNodeManager(db)
	ob := model.Outbound{
		Tag: "node-raw", Type: "vless", Server: "1.2.3.4", Port: 443,
		Raw: map[string]any{"flow": "xtls-rprx-vision", "uuid": "abc123"},
	}
	if err := nm.Add(ob); err != nil {
		t.Fatal(err)
	}

	if err := nm.SaveTestResult("node-raw_tcp", model.TestResult{Tag: "node-raw", TestType: "tcp", Success: true, LatencyMs: 5}); err != nil {
		t.Fatal(err)
	}

	if err := nm.Delete("node-raw"); err != nil {
		t.Fatal(err)
	}

	if nm.Get("node-raw") != nil {
		t.Error("node should be nil after delete")
	}
}
