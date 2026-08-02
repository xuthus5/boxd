package service

import (
	"context"
	"errors"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
)

func TestImportParseLink(t *testing.T) {
	svc := NewImportService(nil, nil, "", nil)
	result, err := svc.ParseLink(context.Background(), "vless://00000000-0000-0000-0000-000000000000@example.com:443")
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag == "" {
		t.Fatal("empty tag")
	}
}

func TestImportParseLinkEmpty(t *testing.T) {
	svc := NewImportService(nil, nil, "", nil)
	if _, err := svc.ParseLink(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty link")
	}
}

func TestImportParseLinkInvalid(t *testing.T) {
	svc := NewImportService(nil, nil, "", nil)
	if _, err := svc.ParseLink(context.Background(), "not-a-link"); err == nil {
		t.Fatal("expected error for invalid link")
	}
}

func TestImportSaveNode(t *testing.T) {
	db := newTestDB(t)
	configPath := t.TempDir() + "/config.json"
	if err := writeTestJSONFile(configPath, map[string]any{"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}}}); err != nil {
		t.Fatal(err)
	}
	nodeMgr := core.NewNodeManager(db)
	subMgr := core.NewSubscriptionManager(db, t.TempDir())
	svc := NewImportService(nodeMgr, subMgr, configPath, &fakeRestart{})
	if err := svc.SaveNode(context.Background(), NodeInput{Tag: "n", Type: "vless", Server: "1.1.1.1", Port: 443}); err != nil {
		t.Fatal(err)
	}
	if nodeMgr.Get("n") == nil {
		t.Fatal("node not saved")
	}
}

func TestImportSaveNodeNilManager(t *testing.T) {
	svc := NewImportService(nil, nil, "", nil)
	if err := svc.SaveNode(context.Background(), NodeInput{Tag: "n", Type: "vless"}); err == nil {
		t.Fatal("expected error for nil manager")
	}
}

func TestImportSaveNodeAddFailure(t *testing.T) {
	db := newTestDB(t)
	nodeMgr := core.NewNodeManager(db)
	_ = db.Close()
	subMgr := core.NewSubscriptionManager(db, t.TempDir())
	svc := NewImportService(nodeMgr, subMgr, "", &fakeRestart{})
	if err := svc.SaveNode(context.Background(), NodeInput{Tag: "n", Type: "vless"}); err == nil {
		t.Fatal("expected add error")
	}
}

func TestImportSyncConfigNilManagers(t *testing.T) {
	svc := NewImportService(nil, nil, "", nil)
	if err := svc.syncConfig(); err != nil {
		t.Fatal(err)
	}
}

func TestHealthLiveness(t *testing.T) {
	svc := NewHealthService(func() error { return nil })
	status, err := svc.Liveness(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status["status"] != "ok" {
		t.Fatalf("status = %v", status)
	}
}

func TestHealthReadinessOK(t *testing.T) {
	svc := NewHealthService(func() error { return nil })
	status, err := svc.Readiness(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status["status"] != "ready" {
		t.Fatalf("status = %v", status)
	}
}

func TestHealthReadinessFailure(t *testing.T) {
	svc := NewHealthService(func() error { return errors.New("down") })
	_, err := svc.Readiness(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHealthReadinessNilFn(t *testing.T) {
	svc := NewHealthService(nil)
	status, err := svc.Readiness(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status["status"] != "ready" {
		t.Fatalf("status = %v", status)
	}
}
