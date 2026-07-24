package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

func TestImportSaveNodeReturnsSyncFailure(t *testing.T) {
	nodeManager, subscriptionManager, _, configPath := newAPIManagers(t)
	restarter := &fakeRestartable{errs: []error{errors.New("restart unavailable")}}
	handler := NewImportHandler(nodeManager, subscriptionManager, configPath, restarter)

	recorder := httptest.NewRecorder()
	handler.SaveNode(recorder, jsonRequest(
		http.MethodPost,
		"/api/import/save",
		`{"tag":"sync-failure","type":"vless","server":"example.com","port":443}`,
	))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	envelope := decodeEnvelope(t, recorder)
	if envelope.Error == nil || envelope.Error.Code != model.ErrorNodeUpdateFailed {
		t.Fatalf("error = %#v", envelope.Error)
	}
	if !strings.Contains(envelope.Error.Message, "synchronize node configuration") {
		t.Fatalf("error message = %q", envelope.Error.Message)
	}
}

func TestNodeUpdateMapsSyncFailureToNodeError(t *testing.T) {
	nodeManager, subscriptionManager, _, configPath := newAPIManagers(t)
	if err := nodeManager.Add(model.Outbound{Tag: "before", Type: "vless"}); err != nil {
		t.Fatal(err)
	}
	restarter := &fakeRestartable{errs: []error{errors.New("restart unavailable")}}
	handler := NewNodesHandler(nodeManager, subscriptionManager, configPath, restarter)

	recorder := httptest.NewRecorder()
	request := withURLParam(jsonRequest(
		http.MethodPut,
		"/api/nodes/before",
		`{"tag":"after","type":"vless","server":"example.com","port":443}`,
	), "tag", "before")
	handler.Update(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	envelope := decodeEnvelope(t, recorder)
	if envelope.Error == nil || envelope.Error.Code != model.ErrorNodeUpdateFailed {
		t.Fatalf("error = %#v", envelope.Error)
	}
}
