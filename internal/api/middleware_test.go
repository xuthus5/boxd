package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xuthus5/boxui/internal/model"
)

type middlewareHijackWriter struct {
	*httptest.ResponseRecorder
	called bool
}

func (w *middlewareHijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.called = true
	return nil, nil, nil
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"key": "value"})

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}

	var envelope map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	data := envelope["data"].(map[string]any)
	if data["key"] != "value" {
		t.Errorf("expected 'value', got '%v'", data["key"])
	}
	if envelope["status"] != model.StatusOK {
		t.Errorf("expected status ok, got %v", envelope["status"])
	}
}

func TestWriteJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSONError(w, http.StatusBadRequest, "something went wrong")

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}

	var envelope map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	errBody := envelope["error"].(map[string]any)
	if errBody["message"] != "something went wrong" {
		t.Errorf("expected 'something went wrong', got '%v'", errBody["message"])
	}
	if errBody["code"] != model.ErrorInvalidRequest {
		t.Errorf("expected invalid_request code, got %v", errBody["code"])
	}
}

func TestCORSMiddleware(t *testing.T) {
	handler := CORSMiddleware([]string{"https://allowed.example"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("preflight allowed", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "https://allowed.example")
		handler.ServeHTTP(w, req)
		if w.Result().StatusCode != http.StatusNoContent {
			t.Errorf("expected 204 for OPTIONS, got %d", w.Result().StatusCode)
		}
		if w.Result().Header.Get("Access-Control-Allow-Origin") != "https://allowed.example" {
			t.Errorf("unexpected allow origin %q", w.Result().Header.Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("preflight denied", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "https://denied.example")
		handler.ServeHTTP(w, req)
		if w.Result().StatusCode != http.StatusForbidden {
			t.Errorf("expected 403 for denied OPTIONS, got %d", w.Result().StatusCode)
		}
	})

	t.Run("actual request allowed", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://allowed.example")
		handler.ServeHTTP(w, req)
		resp := w.Result()
		if resp.Header.Get("Access-Control-Allow-Origin") != "https://allowed.example" {
			t.Errorf("unexpected CORS header %q", resp.Header.Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("same-origin no origin header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, req)
		resp := w.Result()
		if resp.Header.Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("expected empty CORS header, got %q", resp.Header.Get("Access-Control-Allow-Origin"))
		}
	})
}

func TestLoggerMiddleware(t *testing.T) {
	var output bytes.Buffer
	previousHandler := slog.Default()
	logger := slog.New(slog.NewTextHandler(&output, nil))
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(previousHandler) })

	handler := LoggerMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?token=secret&foo=bar", nil)
	handler.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
	logged := output.String()
	if strings.Contains(logged, "token=secret") || strings.Contains(logged, "?") {
		t.Fatalf("logger should not include raw query string: %q", logged)
	}
	if !strings.Contains(logged, "http request") {
		t.Fatalf("logger should include msg 'http request': %q", logged)
	}
	if !strings.Contains(logged, "status=200") {
		t.Fatalf("logger should include status=200: %q", logged)
	}
	if !strings.Contains(logged, "method=GET") {
		t.Fatalf("logger should include method=GET: %q", logged)
	}
}

func TestStatusCaptureWriterOptionalInterfaces(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := &statusCaptureWriter{ResponseWriter: recorder}
	writer.Flush()
	if _, _, err := writer.Hijack(); err == nil {
		t.Fatal("expected unsupported hijack error")
	}

	hijacker := &middlewareHijackWriter{ResponseRecorder: httptest.NewRecorder()}
	writer = &statusCaptureWriter{ResponseWriter: hijacker}
	if _, _, err := writer.Hijack(); err != nil {
		t.Fatalf("Hijack() error = %v", err)
	}
	if !hijacker.called {
		t.Fatal("underlying Hijacker was not called")
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	handler := RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestAuthMiddlewareMissingToken(t *testing.T) {
	handler := AuthMiddleware(staticSecretProvider("secret"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Result().StatusCode)
	}
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	handler := AuthMiddleware(staticSecretProvider("secret"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	handler.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Result().StatusCode)
	}
}

func TestAuthMiddlewareValidToken(t *testing.T) {
	authHandler := NewAuthHandler("admin", "pass", staticSecretProvider("my-secret-key-12345678"))
	handler := AuthMiddleware(staticSecretProvider("my-secret-key-12345678"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	token := getToken(t, authHandler)
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

func TestAuthMiddlewareRejectsQueryToken(t *testing.T) {
	authHandler := NewAuthHandler("admin", "pass", staticSecretProvider("my-secret-key-12345678"))
	handler := AuthMiddleware(staticSecretProvider("my-secret-key-12345678"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	token := getToken(t, authHandler)
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?token="+token, nil)
	handler.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Result().StatusCode)
	}
}

func getToken(t *testing.T, h *AuthHandler) string {
	t.Helper()
	body := strings.NewReader(`{"username":"admin","password":"pass"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/json")
	h.Login(w, req)
	resp := decodeBody[map[string]string](t, w)
	return resp["token"]
}

func TestLoggerMiddlewareCapturesStatus(t *testing.T) {
	var output bytes.Buffer
	previousHandler := slog.Default()
	logger := slog.New(slog.NewTextHandler(&output, nil))
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(previousHandler) })

	handler := LoggerMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/missing", nil)
	handler.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Result().StatusCode)
	}
	logged := output.String()
	if !strings.Contains(logged, "status=404") {
		t.Fatalf("logger should include status=404: %q", logged)
	}
	if !strings.Contains(logged, "method=POST") {
		t.Fatalf("logger should include method=POST: %q", logged)
	}
}
