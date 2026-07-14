package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xuthus5/boxui/internal/model"
)

func TestKernelHandlerVersion(t *testing.T) {
	h := NewKernelHandler("1.13.14-test")
	rr := httptest.NewRecorder()
	h.Version(rr, httptest.NewRequest(http.MethodGet, "/api/runtime/version", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d %s", rr.Code, rr.Body.String())
	}
	resp := decodeBody[map[string]string](t, rr)
	if resp["version"] != "1.13.14-test" {
		t.Fatalf("version = %q", resp["version"])
	}
}

func TestKernelHandlerMemoryShape(t *testing.T) {
	h := NewKernelHandler("v")
	rr := httptest.NewRecorder()
	h.Memory(rr, httptest.NewRequest(http.MethodGet, "/api/runtime/memory", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d %s", rr.Code, rr.Body.String())
	}
	resp := decodeBody[map[string]any](t, rr)
	for _, k := range []string{"alloc", "total", "sys", "num_gc", "heap_inuse", "stack_inuse"} {
		if _, ok := resp[k]; !ok {
			t.Fatalf("missing field %q in memory response: %v", k, resp)
		}
	}
}

func TestKernelHandlerGC(t *testing.T) {
	h := NewKernelHandler("v")
	rr := httptest.NewRecorder()
	h.GC(rr, httptest.NewRequest(http.MethodPost, "/api/runtime/gc", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d %s", rr.Code, rr.Body.String())
	}
	resp := decodeEnvelope(t, rr)
	if resp.Status != model.StatusOK {
		t.Fatalf("status = %q", resp.Status)
	}
}
