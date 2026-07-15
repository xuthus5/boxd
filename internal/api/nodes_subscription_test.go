package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

func TestNodesHandlerListGetUpdateDeleteAndSync(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	if err := nodeMgr.Add(model.Outbound{Tag: "old", Type: "vless", Server: "1.2.3.4", Port: 443}); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.List(rr, httptest.NewRequest(http.MethodGet, "/api/nodes", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"old"`) {
		t.Fatalf("list body = %s", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/nodes/old", nil), "tag", "old")
	handler.Get(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = withURLParam(
		jsonRequest(http.MethodPut, "/api/nodes/old", `{"tag":"new","type":"vmess","server":"5.6.7.8","port":80,"config":{"uuid":"u"}}`),
		"tag",
		"old",
	)
	handler.Update(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("update status = %d body=%s", rr.Code, rr.Body.String())
	}
	if nodeMgr.Get("old") != nil || nodeMgr.Get("new") == nil {
		t.Fatal("node rename did not persist")
	}

	rr = httptest.NewRecorder()
	handler.SyncToConfig(rr, httptest.NewRequest(http.MethodPost, "/api/nodes/sync-config", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("sync status = %d", rr.Code)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"proxy"`) || !strings.Contains(string(data), `"new"`) {
		t.Fatalf("synced config = %s", string(data))
	}

	rr = httptest.NewRecorder()
	req = withURLParam(httptest.NewRequest(http.MethodDelete, "/api/nodes/new", nil), "tag", "new")
	handler.Delete(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status = %d", rr.Code)
	}
}

func TestNodesHandlerErrors(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodeMgr, subMgr, configPath)

	tests := []struct {
		name   string
		method func(*httptest.ResponseRecorder)
		want   int
	}{
		{
			name: "get missing tag",
			method: func(rr *httptest.ResponseRecorder) {
				handler.Get(rr, httptest.NewRequest(http.MethodGet, "/api/nodes/", nil))
			},
			want: http.StatusBadRequest,
		},
		{
			name: "get not found",
			method: func(rr *httptest.ResponseRecorder) {
				req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/nodes/nope", nil), "tag", "nope")
				handler.Get(rr, req)
			},
			want: http.StatusNotFound,
		},
		{
			name: "update invalid json",
			method: func(rr *httptest.ResponseRecorder) {
				if err := nodeMgr.Add(model.Outbound{Tag: "bad-json", Type: "vless"}); err != nil {
					t.Fatal(err)
				}
				req := withURLParam(jsonRequest(http.MethodPut, "/api/nodes/bad-json", `{`), "tag", "bad-json")
				handler.Update(rr, req)
			},
			want: http.StatusBadRequest,
		},
		{
			name: "update missing required fields",
			method: func(rr *httptest.ResponseRecorder) {
				if err := nodeMgr.Add(model.Outbound{Tag: "missing-fields", Type: "vless"}); err != nil {
					t.Fatal(err)
				}
				req := withURLParam(jsonRequest(http.MethodPut, "/api/nodes/missing-fields", `{"tag":"","type":""}`), "tag", "missing-fields")
				handler.Update(rr, req)
			},
			want: http.StatusBadRequest,
		},
		{
			name: "delete missing tag",
			method: func(rr *httptest.ResponseRecorder) {
				handler.Delete(rr, httptest.NewRequest(http.MethodDelete, "/api/nodes/", nil))
			},
			want: http.StatusBadRequest,
		},
		{
			name: "sync config error",
			method: func(rr *httptest.ResponseRecorder) {
				badHandler := NewNodesHandler(nodeMgr, subMgr, t.TempDir()+"/missing/config.json")
				badHandler.SyncToConfig(rr, httptest.NewRequest(http.MethodPost, "/api/nodes/sync-config", nil))
			},
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tt.method(rr)
			if rr.Code != tt.want {
				t.Fatalf("status = %d, want %d", rr.Code, tt.want)
			}
		})
	}
}

func TestSubscriptionHandlerCRUDAndRefresh(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewSubscriptionHandler(subMgr, nodeMgr, configPath)

	rr := httptest.NewRecorder()
	handler.Create(rr, jsonRequest(http.MethodPost, "/api/subscriptions", `{"name":"sub","url":"https://example.test/sub","interval_min":0}`))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", rr.Code, rr.Body.String())
	}
	created := decodeBody[model.Subscription](t, rr)
	if created.IntervalMin != 60 {
		t.Fatalf("default interval = %d", created.IntervalMin)
	}

	rr = httptest.NewRecorder()
	req := withURLParam(jsonRequest(http.MethodPut, "/api/subscriptions/"+created.ID, `{"name":"renamed","url":"","interval_min":30}`), "id", created.ID)
	handler.Update(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("update status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = withURLParam(httptest.NewRequest(http.MethodGet, "/api/subscriptions/"+created.ID, nil), "id", created.ID)
	handler.Get(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.List(rr, httptest.NewRequest(http.MethodGet, "/api/subscriptions", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = withURLParam(httptest.NewRequest(http.MethodDelete, "/api/subscriptions/"+created.ID, nil), "id", created.ID)
	handler.Delete(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status = %d", rr.Code)
	}
}

func TestSubscriptionHandlerErrorsAndRefreshAll(t *testing.T) {
	nodeMgr, subMgr, _, configPath := newAPIManagers(t)
	handler := NewSubscriptionHandler(subMgr, nodeMgr, configPath)

	rr := httptest.NewRecorder()
	handler.RefreshAll(rr, httptest.NewRequest(http.MethodPost, "/api/subscriptions/refresh-all", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("empty refresh all status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.Create(rr, jsonRequest(http.MethodPost, "/api/subscriptions", `{`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid create status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.Create(rr, jsonRequest(http.MethodPost, "/api/subscriptions", `{"name":"","url":""}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("missing fields status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req := withURLParam(jsonRequest(http.MethodPut, "/api/subscriptions/missing", `{`), "id", "missing")
	handler.Update(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid update status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = withURLParam(jsonRequest(http.MethodPut, "/api/subscriptions/missing", `{"name":"x"}`), "id", "missing")
	handler.Update(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("missing update status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = withURLParam(httptest.NewRequest(http.MethodDelete, "/api/subscriptions/missing", nil), "id", "missing")
	handler.Delete(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("missing delete status = %d", rr.Code)
	}

	if _, err := subMgr.Create(core.SubscriptionParams{Name: "bad", URL: "://bad-url", IntervalMin: 60}); err != nil {
		t.Fatal(err)
	}
	subscriptions, err := subMgr.List()
	if err != nil {
		t.Fatal(err)
	}
	bad := subscriptions[0]
	rr = httptest.NewRecorder()
	req = withURLParam(httptest.NewRequest(http.MethodPost, "/api/subscriptions/"+bad.ID+"/refresh", nil), "id", bad.ID)
	handler.Refresh(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("refresh status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.RefreshAll(rr, httptest.NewRequest(http.MethodPost, "/api/subscriptions/refresh-all", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("refresh all status = %d", rr.Code)
	}
	envelope := decodeEnvelope(t, rr)
	if envelope.Status != model.StatusPartial {
		t.Fatalf("expected partial status, got %#v", envelope.Status)
	}

	rr = httptest.NewRecorder()
	req = withURLParam(httptest.NewRequest(http.MethodGet, "/api/subscriptions/missing", nil), "id", "missing")
	handler.Get(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("missing get status = %d", rr.Code)
	}
}
