package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"
)

func TestImportSaveBridgeRejectsInvalidCredentialsBeforeMutation(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	previous, err := os.ReadFile(rt.svc.Deps.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	body := json.RawMessage(`{"tag":"candidate","type":"shadowsocks","server":"192.0.2.1","port":443,` +
		`"config":{"method":"2022-blake3-aes-128-gcm","password":"private-test-key"}}`)
	_, err = importSaveBridge(rt, body)
	assertBridgeImportValidationError(t, err)
	after, err := os.ReadFile(rt.svc.Deps.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if rt.svc.Deps.NodeManager.Get("candidate") != nil {
		t.Fatal("invalid credentials were written to the node database")
	}
	if !bytes.Equal(previous, after) || rt.instance.Status().Running {
		t.Fatal("invalid credentials changed the active configuration or kernel state")
	}
}

func assertBridgeImportValidationError(t *testing.T, err error) {
	t.Helper()
	var domain *service.DomainError
	if !errors.As(err, &domain) {
		t.Fatalf("want validation domain error, got %v", err)
	}
	if domain.Status != 400 || domain.Code != model.ErrorInvalidRequest {
		t.Fatalf("want invalid_request status 400, got %#v", domain)
	}
	if !strings.HasPrefix(domain.Message, "config.password:") || strings.Contains(domain.Message, "private-test-key") {
		t.Fatalf("validation error has wrong field path or exposes a secret: %q", domain.Message)
	}
}
