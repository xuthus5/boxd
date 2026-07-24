package api

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

type restartFunc func() error

func (f restartFunc) Restart() error {
	return f()
}

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

func TestNodeSyncSkipsRestartWhenConfigIsUnchanged(t *testing.T) {
	nodeManager, subscriptionManager, _, configPath := newAPIManagers(t)
	if err := syncOutboundsToConfig(nodeManager, subscriptionManager, configPath); err != nil {
		t.Fatal(err)
	}
	restarter := &fakeRestartable{}
	handler := NewNodesHandler(nodeManager, subscriptionManager, configPath, restarter)

	recorder := httptest.NewRecorder()
	handler.SyncToConfig(recorder, httptest.NewRequest(http.MethodPost, "/api/nodes/sync-config", nil))

	if recorder.Code != http.StatusOK || restarter.calls != 0 {
		t.Fatalf("status = %d restart calls = %d body=%s", recorder.Code, restarter.calls, recorder.Body.String())
	}
}

func TestNodeSyncRestoresAppliedStateAfterRestartFailure(t *testing.T) {
	nodeManager, subscriptionManager, settings, configPath := newAPIManagers(t)
	previousConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := settings.SetURLTestManagedGroups([]string{"previous-group"}); err != nil {
		t.Fatal(err)
	}
	if err := nodeManager.Add(model.Outbound{Tag: "desired-node", Type: "vless", Server: "example.com", Port: 443}); err != nil {
		t.Fatal(err)
	}
	restarter := &fakeRestartable{errs: []error{errors.New("restart unavailable")}}
	handler := NewNodesHandler(nodeManager, subscriptionManager, configPath, restarter)

	recorder := httptest.NewRecorder()
	handler.SyncToConfig(recorder, httptest.NewRequest(http.MethodPost, "/api/nodes/sync-config", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if restarter.calls != 2 {
		t.Fatalf("restart calls = %d, want 2", restarter.calls)
	}
	afterConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(afterConfig, previousConfig) {
		t.Fatalf("config was not restored\nwant: %s\ngot: %s", previousConfig, afterConfig)
	}
	groups, err := settings.URLTestManagedGroups()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0] != "previous-group" {
		t.Fatalf("managed groups = %#v", groups)
	}
	envelope := decodeEnvelope(t, recorder)
	if envelope.Error == nil || !strings.Contains(envelope.Error.Message, "previous configuration restored") {
		t.Fatalf("error = %#v", envelope.Error)
	}
	if nodeManager.Get("desired-node") == nil {
		t.Fatal("desired node should remain persisted for a later retry")
	}
}

func TestNodeSyncReportsRollbackRestartFailure(t *testing.T) {
	nodeManager, subscriptionManager, _, configPath := newAPIManagers(t)
	if err := nodeManager.Add(model.Outbound{Tag: "desired-node", Type: "vless", Server: "example.com", Port: 443}); err != nil {
		t.Fatal(err)
	}
	restarter := &fakeRestartable{errs: []error{
		errors.New("new configuration restart failed"),
		errors.New("restored configuration restart failed"),
	}}
	handler := NewNodesHandler(nodeManager, subscriptionManager, configPath, restarter)

	recorder := httptest.NewRecorder()
	handler.SyncToConfig(recorder, httptest.NewRequest(http.MethodPost, "/api/nodes/sync-config", nil))

	if recorder.Code != http.StatusInternalServerError || restarter.calls != 2 {
		t.Fatalf("status = %d restart calls = %d body=%s", recorder.Code, restarter.calls, recorder.Body.String())
	}
	message := decodeEnvelope(t, recorder).Error.Message
	if !strings.Contains(message, "new configuration restart failed") || !strings.Contains(message, "restored configuration restart failed") {
		t.Fatalf("error message = %q", message)
	}
}

func TestNodeSyncRestartsPreviousConfigWhenGroupRollbackFails(t *testing.T) {
	nodeManager, subscriptionManager, _, configPath := newAPIManagers(t)
	previousConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := nodeManager.Add(model.Outbound{Tag: "desired-node", Type: "vless", Server: "example.com", Port: 443}); err != nil {
		t.Fatal(err)
	}
	restartCalls := 0
	restarter := restartFunc(func() error {
		restartCalls++
		if restartCalls == 1 {
			if err := subscriptionManager.DB().Close(); err != nil {
				return err
			}
			return errors.New("restart unavailable")
		}
		return nil
	})
	handler := NewNodesHandler(nodeManager, subscriptionManager, configPath, restarter)

	recorder := httptest.NewRecorder()
	handler.SyncToConfig(recorder, httptest.NewRequest(http.MethodPost, "/api/nodes/sync-config", nil))

	if recorder.Code != http.StatusInternalServerError || restartCalls != 2 {
		t.Fatalf("status = %d restart calls = %d body=%s", recorder.Code, restartCalls, recorder.Body.String())
	}
	afterConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(afterConfig, previousConfig) {
		t.Fatalf("config was not restored\nwant: %s\ngot: %s", previousConfig, afterConfig)
	}
	message := decodeEnvelope(t, recorder).Error.Message
	if !strings.Contains(message, "restoring previous managed groups") {
		t.Fatalf("error message = %q", message)
	}
}
