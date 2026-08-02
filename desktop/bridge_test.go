package main

import (
	"testing"
)

func TestBridgeServiceNotReady(t *testing.T) {
	rt := &desktopRuntime{}
	svc := newBoxdBridgeService(rt)
	_, err := svc.Call(BridgeRequest{Path: "/api/service/status"})
	if err == nil {
		t.Fatal("expected not ready error")
	}
}

func TestBridgeServiceStatus(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{Path: "/api/service/status"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestBridgeServiceStartStop(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	// config 缺失时 start 报错
	resp, err := svc.Call(BridgeRequest{Path: "/api/service/start"})
	if err == nil {
		t.Fatalf("expected start error, got %+v", resp)
	}
	if resp.Status != "error" {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestBridgeConfigGet(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{Path: "/api/config/"})
	if err == nil {
		t.Fatalf("expected error for missing config, got %+v", resp)
	}
}

func TestBridgeSettings(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{Path: "/api/settings/preferences"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestBridgeRuntimeVersion(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{Path: "/api/runtime/version"})
	if err != nil {
		t.Fatal(err)
	}
	data, ok := resp.Data.(map[string]string)
	if !ok {
		t.Fatalf("data type = %T", resp.Data)
	}
	if data["version"] == "" {
		t.Fatal("version empty")
	}
}

func TestBridgeHealth(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	resp, err := svc.Call(BridgeRequest{Path: "/readyz"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q", resp.Status)
	}
	resp, err = svc.Call(BridgeRequest{Path: "/healthz"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestBridgeUnknownPath(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	svc := newBoxdBridgeService(rt)
	_, err := svc.Call(BridgeRequest{Path: "/api/unknown"})
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestErrResultNil(t *testing.T) {
	resp, err := errResult(nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestErrorfBridge(t *testing.T) {
	err := ErrorfBridge(404, "not_found", "unknown path %q", "/x")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "not_found: unknown path \"/x\"" {
		t.Fatalf("err = %q", err.Error())
	}
}
