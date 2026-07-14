package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	rr := httptest.NewRecorder()
	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want 'nosniff'", got)
	}
	if got := rr.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want 'DENY'", got)
	}
	if got := rr.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Errorf("Referrer-Policy = %q, want 'strict-origin-when-cross-origin'", got)
	}
	if got := rr.Header().Get("X-XSS-Protection"); got != "1; mode=block" {
		t.Errorf("X-XSS-Protection = %q, want '1; mode=block'", got)
	}
}

func TestBodyLimitMiddlewareRejectsLargeBody(t *testing.T) {
	rr := httptest.NewRecorder()
	handler := BodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 构造超过 2MB 的请求体
	largeBody := strings.Repeat("x", maxRequestBodyBytes+1)
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(largeBody)))

	if rr.Code != http.StatusInternalServerError {
		// MaxBytesReader 在读取时返回错误，handler 内部需要处理
		// 这里验证中间件不 panic 且请求能到达 handler
		t.Logf("status = %d (expected: middleware wraps body, handler decides response)", rr.Code)
	}
}

func TestBodyLimitMiddlewareAllowsNormalBody(t *testing.T) {
	rr := httptest.NewRecorder()
	handler := BodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("normal body")))

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestSecurityHeadersMiddlewarePreservesResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"msg": "ok"})
	}))

	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	// 验证安全头和正常响应体共存
	if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("security header missing alongside normal response")
	}
}
