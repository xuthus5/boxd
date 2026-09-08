package main

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/service"
)

func TestBridgeSetupStatusPreviewAndApply(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	bridge := newBoxdBridgeService(rt)
	status, err := bridge.Call(BridgeRequest{Path: "/api/config/setup"})
	if err != nil || status.Data.(core.SetupStatus).SourceHash == "" {
		t.Fatalf("setup status: %+v, %v", status, err)
	}
	request := core.SetupRequest{Modules: []string{"experimental"}, InboundMode: "preserve"}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := bridge.Call(BridgeRequest{Path: "/api/config/setup/preview", Method: "POST", Body: body})
	if err != nil {
		t.Fatal(err)
	}
	request.SourceHash = preview.Data.(*core.SetupPlan).SourceHash
	body, err = json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := bridge.Call(BridgeRequest{Path: "/api/config/setup/apply", Method: "POST", Body: body})
	if err != nil || applied.Data.(service.SetupApplyResult).Status != "ok" {
		t.Fatalf("setup apply: %+v, %v", applied, err)
	}
	if rt.instance.Status().Running {
		t.Fatal("desktop setup unexpectedly started a stopped kernel")
	}
}

func TestBridgeSetupRejectsStalePreview(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	bridge := newBoxdBridgeService(rt)
	request := core.SetupRequest{Modules: []string{"experimental"}, SourceHash: "old-preview"}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	response, err := bridge.Call(BridgeRequest{Path: "/api/config/setup/apply", Method: "POST", Body: body})
	var domain *service.DomainError
	if !errors.As(err, &domain) || domain.Status != 409 || domain.Code != "config_changed" || response.Status != "error" {
		t.Fatalf("stale preview must retain conflict semantics: %+v, %v", response, err)
	}
	original, err := os.ReadFile(rt.cfg.ConfigPath)
	if err != nil || string(original) != desktopTestConfigJSON {
		t.Fatalf("stale preview changed original config: %v", err)
	}
}

func TestBridgeSetupRejectsInvalidRequests(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	bridge := newBoxdBridgeService(rt)
	for _, tt := range []struct{ name, method, body string }{
		{name: "empty", method: "POST"},
		{name: "invalid JSON", method: "POST", body: "{"},
		{name: "unknown field", method: "POST", body: `{"unsupported":true}`},
		{name: "trailing JSON", method: "POST", body: `{} {}`},
		{name: "wrong method", method: "GET", body: `{}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			response, err := bridge.Call(BridgeRequest{Path: "/api/config/setup/preview", Method: tt.method, Body: json.RawMessage(tt.body)})
			if err == nil || response.Status != "error" {
				t.Fatalf("invalid setup request was accepted: %+v, %v", response, err)
			}
		})
	}
}
