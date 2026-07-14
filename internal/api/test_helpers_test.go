package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	chi "github.com/go-chi/chi/v5"
	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

func newTestDB(t *testing.T) *bbolt.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func newAPIManagers(t *testing.T) (*core.NodeManager, *core.SubscriptionManager, *core.SettingsManager, string) {
	t.Helper()

	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{},
	})

	return core.NewNodeManager(db),
		core.NewSubscriptionManager(db, t.TempDir()),
		core.NewSettingsManager(db),
		configPath
}

func writeConfigFile(t *testing.T, path string, value any) {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func jsonRequest(method, target string, body string) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func decodeBody[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()

	var envelope model.APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}

	raw, err := json.Marshal(envelope.Data)
	if err != nil {
		t.Fatal(err)
	}

	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func decodeEnvelope(t *testing.T, rr *httptest.ResponseRecorder) model.APIResponse {
	t.Helper()

	var envelope model.APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope
}

func decodeConfigFile(t *testing.T, path string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func withURLParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}
