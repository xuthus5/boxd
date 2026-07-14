package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xuthus5/boxui/internal/core"
)

type fakeRuntimeInstance struct {
	groups    []core.OutboundGroupInfo
	err       error
	called    bool
	delays    map[string]uint16
	testErr   error
	dnsErr    error
	fakeipErr error
	delayRet  uint16
	delayErr  error
	delayTag  string
}

func (f *fakeRuntimeInstance) OutboundGroups() []core.OutboundGroupInfo {
	return f.groups
}

func (f *fakeRuntimeInstance) SelectOutbound(groupTag, outTag string) error {
	f.called = true
	return f.err
}

func (f *fakeRuntimeInstance) URLTestDelays(ctx context.Context, groupTag string) (map[string]uint16, error) {
	return f.delays, f.testErr
}

func (f *fakeRuntimeInstance) FlushDNS() error {
	f.called = true
	return f.dnsErr
}

func (f *fakeRuntimeInstance) FlushFakeIP() error {
	f.called = true
	return f.fakeipErr
}

func (f *fakeRuntimeInstance) OutboundDelay(ctx context.Context, tag, link string, timeout time.Duration) (uint16, error) {
	f.delayTag = tag
	return f.delayRet, f.delayErr
}

func TestRuntimeHandlerOutboundGroups(t *testing.T) {
	inst := &fakeRuntimeInstance{
		groups: []core.OutboundGroupInfo{
			{Type: "selector", Tag: "proxy", Now: "a", All: []string{"a", "b"}},
		},
	}
	handler := NewRuntimeHandler(inst)

	rr := httptest.NewRecorder()
	handler.OutboundGroups(rr, httptest.NewRequest(http.MethodGet, "/api/nodes/groups", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d %s", rr.Code, rr.Body.String())
	}
	if !containsAll(rr.Body.String(), `"tag":"proxy"`, `"now":"a"`, `"groups"`) {
		t.Fatalf("groups body = %q", rr.Body.String())
	}
}

func TestRuntimeHandlerOutboundGroupsEmpty(t *testing.T) {
	handler := NewRuntimeHandler(&fakeRuntimeInstance{})
	rr := httptest.NewRecorder()
	handler.OutboundGroups(rr, httptest.NewRequest(http.MethodGet, "/api/nodes/groups", nil))
	if rr.Code != http.StatusOK || !containsAll(rr.Body.String(), `"groups":[]`) {
		t.Fatalf("empty groups body = %q", rr.Body.String())
	}
}

func TestRuntimeHandlerSelectOutbound(t *testing.T) {
	tests := []struct {
		name    string
		group   string
		body    string
		instErr error
		want    int
	}{
		{name: "missing group param", group: "", body: `{"tag":"b"}`, want: http.StatusBadRequest},
		{name: "invalid json", group: "proxy", body: `{`, want: http.StatusBadRequest},
		{name: "missing tag", group: "proxy", body: `{}`, want: http.StatusBadRequest},
		{name: "not running", group: "proxy", body: `{"tag":"b"}`, instErr: core.ErrNotRunning, want: http.StatusServiceUnavailable},
		{name: "group not found", group: "proxy", body: `{"tag":"b"}`, instErr: core.ErrGroupNotFound, want: http.StatusNotFound},
		{name: "not selectable", group: "proxy", body: `{"tag":"b"}`, instErr: core.ErrNotSelectable, want: http.StatusBadRequest},
		{name: "tag not in group", group: "proxy", body: `{"tag":"c"}`, instErr: core.ErrTagNotInGroup, want: http.StatusBadRequest},
		{name: "unknown error", group: "proxy", body: `{"tag":"b"}`, instErr: errors.New("boom"), want: http.StatusInternalServerError},
		{name: "ok", group: "proxy", body: `{"tag":"b"}`, instErr: nil, want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &fakeRuntimeInstance{err: tt.instErr}
			handler := NewRuntimeHandler(inst)
			req := withURLParam(jsonRequest(http.MethodPost, "/api/nodes/selectors/"+tt.group+"/select", tt.body), "group", tt.group)
			rr := httptest.NewRecorder()
			handler.SelectOutbound(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("status = %d, want %d (%s)", rr.Code, tt.want, rr.Body.String())
			}
			// 仅成功用例才真正调用实例并返回选中结果
			if tt.want == http.StatusOK {
				if !inst.called {
					t.Fatal("SelectOutbound should be called")
				}
				resp := decodeBody[map[string]string](t, rr)
				if resp["selected"] != "b" {
					t.Fatalf("ok body = %q", rr.Body.String())
				}
				envelope := decodeEnvelope(t, rr)
				meta := envelope.Meta.(map[string]any)
				if meta["group"] != "proxy" {
					t.Fatalf("meta = %#v", meta)
				}
			}
		})
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

func TestRuntimeHandlerURLTestDelays(t *testing.T) {
	tests := []struct {
		name    string
		group   string
		delays  map[string]uint16
		testErr error
		want    int
	}{
		{name: "missing group", group: "", want: http.StatusBadRequest},
		{name: "not running", group: "auto", testErr: core.ErrNotRunning, want: http.StatusServiceUnavailable},
		{name: "group not found", group: "auto", testErr: core.ErrGroupNotFound, want: http.StatusNotFound},
		{name: "not urltest", group: "auto", testErr: core.ErrNotSelectable, want: http.StatusBadRequest},
		{name: "inner error", group: "auto", testErr: errors.New("boom"), want: http.StatusInternalServerError},
		{name: "ok", group: "auto", delays: map[string]uint16{"a": 120}, want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &fakeRuntimeInstance{delays: tt.delays, testErr: tt.testErr}
			handler := NewRuntimeHandler(inst)
			req := withURLParam(jsonRequest(http.MethodPost, "/api/nodes/groups/"+tt.group+"/urltest", ""), "group", tt.group)
			rr := httptest.NewRecorder()
			handler.URLTestDelays(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("status = %d, want %d (%s)", rr.Code, tt.want, rr.Body.String())
			}
			if tt.want == http.StatusOK {
				resp := decodeBody[map[string]uint16](t, rr)
				if resp["a"] != 120 {
					t.Fatalf("body = %q", rr.Body.String())
				}
				envelope := decodeEnvelope(t, rr)
				meta := envelope.Meta.(map[string]any)
				if meta["group"] != "auto" {
					t.Fatalf("meta = %#v", meta)
				}
			}
		})
	}
}

func TestRuntimeHandlerFlushDNS(t *testing.T) {
	tests := []struct {
		name   string
		dnsErr error
		want   int
	}{
		{name: "not running", dnsErr: core.ErrNotRunning, want: http.StatusServiceUnavailable},
		{name: "unknown error", dnsErr: errors.New("boom"), want: http.StatusInternalServerError},
		{name: "ok", dnsErr: nil, want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &fakeRuntimeInstance{dnsErr: tt.dnsErr}
			handler := NewRuntimeHandler(inst)
			rr := httptest.NewRecorder()
			handler.FlushDNS(rr, httptest.NewRequest(http.MethodPost, "/api/runtime/dns/flush", nil))
			if rr.Code != tt.want {
				t.Fatalf("status = %d, want %d (%s)", rr.Code, tt.want, rr.Body.String())
			}
		})
	}
}

func TestRuntimeHandlerFlushFakeIP(t *testing.T) {
	tests := []struct {
		name      string
		fakeipErr error
		want      int
	}{
		{name: "not running", fakeipErr: core.ErrNotRunning, want: http.StatusServiceUnavailable},
		{name: "not enabled", fakeipErr: core.ErrFeatureNotEnabled, want: http.StatusBadRequest},
		{name: "unknown error", fakeipErr: errors.New("boom"), want: http.StatusInternalServerError},
		{name: "ok", fakeipErr: nil, want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &fakeRuntimeInstance{fakeipErr: tt.fakeipErr}
			handler := NewRuntimeHandler(inst)
			rr := httptest.NewRecorder()
			handler.FlushFakeIP(rr, httptest.NewRequest(http.MethodPost, "/api/runtime/fakeip/flush", nil))
			if rr.Code != tt.want {
				t.Fatalf("status = %d, want %d (%s)", rr.Code, tt.want, rr.Body.String())
			}
		})
	}
}

func TestRuntimeHandlerOutboundDelay(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		query    string
		delayRet uint16
		delayErr error
		want     int
	}{
		{name: "missing tag", tag: "", want: http.StatusBadRequest},
		{name: "not running", tag: "p1", delayErr: core.ErrNotRunning, want: http.StatusServiceUnavailable},
		{name: "outbound not found", tag: "p1", delayErr: core.ErrOutboundNotFound, want: http.StatusNotFound},
		{name: "delay error", tag: "p1", delayErr: errors.New("dial fail"), want: http.StatusBadGateway},
		{name: "zero delay", tag: "p1", delayRet: 0, want: http.StatusBadGateway},
		{name: "ok", tag: "p1", query: "url=http://x/generate_204&timeout=5000", delayRet: 120, want: http.StatusOK},
		{name: "bad timeout", tag: "p1", query: "timeout=abc", want: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &fakeRuntimeInstance{delayRet: tt.delayRet, delayErr: tt.delayErr}
			handler := NewRuntimeHandler(inst)
			path := "/api/nodes/" + tt.tag + "/delay"
			if tt.query != "" {
				path += "?" + tt.query
			}
			req := withURLParam(httptest.NewRequest(http.MethodGet, path, nil), "tag", tt.tag)
			rr := httptest.NewRecorder()
			handler.OutboundDelay(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("status = %d, want %d (%s)", rr.Code, tt.want, rr.Body.String())
			}
			if tt.want == http.StatusOK {
				if inst.delayTag != tt.tag {
					t.Fatalf("delay tag = %q, want %q", inst.delayTag, tt.tag)
				}
				resp := decodeBody[map[string]any](t, rr)
				if resp["tag"] != "p1" || int(resp["delay"].(float64)) != 120 {
					t.Fatalf("ok body = %q", rr.Body.String())
				}
			}
		})
	}
}
