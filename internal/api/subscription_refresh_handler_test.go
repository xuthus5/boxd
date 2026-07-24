package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

const validSubscriptionBody = `{"outbounds":[{"tag":"node","type":"trojan","server":"example.com","port":443}]}`

type refreshAllSyncExpectation struct {
	errorCode   string
	failedCount int
}

func TestSubscriptionSyncErrorMessage(t *testing.T) {
	if got := subscriptionSyncErrorMessage(errors.New("restart unavailable")); !strings.Contains(got, "restart unavailable") {
		t.Fatalf("message = %q", got)
	}
	if got := subscriptionSyncErrorMessage(errors.New("   ")); got != "subscription refreshed but configuration sync failed" {
		t.Fatalf("empty detail message = %q", got)
	}
}

func TestSubscriptionHandlerSyncConfig(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewSubscriptionHandler(subMgr, nodeMgr, configPath)

	if err := handler.SyncConfig(); err != nil {
		t.Fatalf("sync config error = %v", err)
	}
	config := decodeConfigFile(t, configPath)
	if _, ok := config["outbounds"]; !ok {
		t.Fatalf("synced config = %#v", config)
	}
}

func TestSubscriptionRefreshReportsConfigSyncFailure(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t, newSubscriptionTestClient(validSubscriptionBody))
	restarter := &fakeRestartable{errs: []error{errors.New("restart unavailable")}}
	handler := NewSubscriptionHandler(subMgr, nodeMgr, configPath, restarter)
	subscription, err := subMgr.Create(core.SubscriptionParams{Name: "sync-failure", URL: "https://example.test/sub", IntervalMin: 60})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	request := withURLParam(httptest.NewRequest(http.MethodPost, "/api/subscriptions/"+subscription.ID+"/refresh", nil), "id", subscription.ID)
	handler.Refresh(recorder, request)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	envelope := decodeEnvelope(t, recorder)
	if envelope.Error == nil || envelope.Error.Code != model.ErrorSubscriptionSync {
		t.Fatalf("error = %#v", envelope.Error)
	}
	if !strings.Contains(envelope.Error.Message, "configuration sync failed") {
		t.Fatalf("error message = %q", envelope.Error.Message)
	}
	if restarter.calls != 2 {
		t.Fatalf("restart calls = %d, want 2", restarter.calls)
	}
}

func TestSubscriptionRefreshAllReportsConfigSyncFailure(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t, newSubscriptionTestClient(validSubscriptionBody))
	restarter := &fakeRestartable{errs: []error{errors.New("restart unavailable")}}
	handler := NewSubscriptionHandler(subMgr, nodeMgr, configPath, restarter)
	if _, err := subMgr.Create(core.SubscriptionParams{Name: "batch-sync-failure", URL: "https://example.test/sub", IntervalMin: 60}); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler.RefreshAll(recorder, httptest.NewRequest(http.MethodPost, "/api/subscriptions/refresh-all", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	assertRefreshAllSyncFailure(t, decodeEnvelope(t, recorder), refreshAllSyncExpectation{
		errorCode: model.ErrorSubscriptionSync,
	})
}

func TestSubscriptionRefreshAllReportsRefreshAndSyncFailures(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t, newSubscriptionTestClient(validSubscriptionBody))
	restarter := &fakeRestartable{errs: []error{errors.New("restart unavailable")}}
	handler := NewSubscriptionHandler(subMgr, nodeMgr, configPath, restarter)
	if _, err := subMgr.Create(core.SubscriptionParams{Name: "valid", URL: "https://example.test/sub", IntervalMin: 60}); err != nil {
		t.Fatal(err)
	}
	insertLegacySubscription(t, subMgr, model.Subscription{ID: "legacy-invalid", Name: "invalid", URL: "://bad-url", IntervalMin: 60})

	recorder := httptest.NewRecorder()
	handler.RefreshAll(recorder, httptest.NewRequest(http.MethodPost, "/api/subscriptions/refresh-all", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	assertRefreshAllSyncFailure(t, decodeEnvelope(t, recorder), refreshAllSyncExpectation{
		errorCode:   model.ErrorSubscriptionRefresh,
		failedCount: 1,
	})
}

func assertRefreshAllSyncFailure(t *testing.T, envelope model.APIResponse, want refreshAllSyncExpectation) {
	t.Helper()
	if envelope.Status != model.StatusPartial || envelope.Error == nil || envelope.Error.Code != want.errorCode {
		t.Fatalf("envelope = %#v", envelope)
	}
	data, ok := envelope.Data.(map[string]any)
	if !ok || data["sync_error"] == nil {
		t.Fatalf("data = %#v", envelope.Data)
	}
	failed, ok := data["failed"].([]any)
	if !ok || len(failed) != want.failedCount {
		t.Fatalf("failed = %#v", data["failed"])
	}
	meta, ok := envelope.Meta.(map[string]any)
	if !ok {
		t.Fatalf("meta = %#v", envelope.Meta)
	}
	failedCount, ok := meta["failed_count"].(float64)
	if !ok || int(failedCount) != want.failedCount {
		t.Fatalf("failed_count = %#v", meta["failed_count"])
	}
	syncFailed, ok := meta["sync_failed"].(bool)
	if !ok || !syncFailed {
		t.Fatalf("meta = %#v", envelope.Meta)
	}
}
