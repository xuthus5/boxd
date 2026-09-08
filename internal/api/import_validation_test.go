package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func TestImportSaveNodeRejectsInvalidProtocolFields(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		path string
	}{
		{name: "vless", body: `"type":"vless","config":{"flow":"unsupported"}`, path: "config.flow"},
		{name: "vmess", body: `"type":"vmess","config":{"security":"unsupported"}`, path: "config.security"},
		{name: "tuic", body: `"type":"tuic","config":{"uuid":"private-test-key"}`, path: "config.uuid"},
		{name: "shadowsocks", body: `"type":"shadowsocks","config":{"method":"aes-128-gcm"}`, path: "config.password"},
		{name: "ss2022", path: "config.password",
			body: `"type":"shadowsocks","config":{"method":"2022-blake3-aes-128-gcm","password":"private-test-key"}`},
		{name: "raw override", body: `"type":"vless","config":{"type":"tuic"}`, path: "config.uuid"},
		{name: "raw must be object", body: `"type":"vless","config":["private-test-key"]`, path: "config"},
	} {
		t.Run(test.name, func(t *testing.T) { assertImportRejectsRequest(t, test.body, test.path) })
	}
}

func assertImportRejectsRequest(t *testing.T, fields, path string) {
	t.Helper()
	nodes, subs, _, configPath := newAPIManagers(t)
	previous, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	probe := &nodeSyncReloadProbe{running: true}
	handler := NewImportHandler(
		nodes,
		subs,
		configPath,
		probe,
	)
	request := `{"tag":"candidate","server":"192.0.2.1","port":443,` + fields + `}`
	recorder := httptest.NewRecorder()
	handler.SaveNode(recorder, jsonRequest(http.MethodPost, "/api/import/save", request))
	assertImportValidationResponse(t, recorder, path)
	if nodes.Get("candidate") != nil {
		t.Fatal("invalid input was written to the node database")
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	kernelUnchanged := probe.running && probe.reloadCalls == 0 && probe.restartCalls == 0
	if !bytes.Equal(previous, after) || !kernelUnchanged {
		t.Fatal("invalid input changed active configuration or kernel state")
	}
}

func assertImportValidationResponse(t *testing.T, recorder *httptest.ResponseRecorder, path string) {
	t.Helper()
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("want status 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
	apiErr := decodeEnvelope(t, recorder).Error
	if apiErr == nil {
		t.Fatal("expected validation error envelope")
	}
	if apiErr.Code != model.ErrorInvalidRequest || !strings.HasPrefix(apiErr.Message, path+":") {
		t.Fatalf("want invalid_request with path %s, got %#v", path, apiErr)
	}
	if strings.Contains(recorder.Body.String(), "private-test-key") {
		t.Fatal("validation response disclosed credentials")
	}
}

func TestImportSaveNodePreservesSupportedCredentialsWhileStopped(t *testing.T) {
	const key = "AAAAAAAAAAAAAAAAAAAAAA=="
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "vless empty identity", body: `"type":"vless","config":{}`},
		{name: "vmess named identity", body: `"type":"vmess","config":{"uuid":"named identity"}`},
		{name: "ss none empty password", body: `"type":"shadowsocks","config":{"method":"none"}`},
		{name: "ss2022 key chain",
			body: `"type":"shadowsocks","config":{"method":"2022-blake3-aes-128-gcm","password":"` + key + ":" + key + `"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			nodes, subs, _, configPath := newAPIManagers(t)
			instance := core.NewSBInstance(configPath, core.NewLogWriter(5))
			handler := NewImportHandler(
				nodes,
				subs,
				configPath,
				instance,
			)
			body := `{"tag":"accepted","server":"192.0.2.1","port":443,` + test.body + `}`
			recorder := httptest.NewRecorder()
			handler.SaveNode(recorder, jsonRequest(http.MethodPost, "/api/import/save", body))
			if recorder.Code != http.StatusOK {
				t.Fatalf("want supported credentials accepted, got %s", recorder.Body.String())
			}
			if nodes.Get("accepted") == nil || instance.Status().Running {
				t.Fatal("valid credentials must persist without starting a stopped kernel")
			}
		})
	}
}

func TestImportSaveNodeReportsStorageFailure(t *testing.T) {
	nodes, subs, _, configPath := newAPIManagers(t)
	if err := subs.DB().Close(); err != nil {
		t.Fatal(err)
	}
	probe := &nodeSyncReloadProbe{running: true}
	handler := NewImportHandler(
		nodes,
		subs,
		configPath,
		probe,
	)
	body := `{"tag":"candidate","type":"vless","server":"192.0.2.1","port":443}`
	recorder := httptest.NewRecorder()
	handler.SaveNode(recorder, jsonRequest(http.MethodPost, "/api/import/save", body))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("want storage failure status 500, got %d", recorder.Code)
	}
	apiErr := decodeEnvelope(t, recorder).Error
	if apiErr == nil || apiErr.Code != model.ErrorInternal {
		t.Fatalf("storage failure was mapped to invalid input: %#v", apiErr)
	}
	kernelUnchanged := probe.running && probe.reloadCalls == 0 && probe.restartCalls == 0
	if !kernelUnchanged {
		t.Fatal("storage failure changed kernel state")
	}
}
