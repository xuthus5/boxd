package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

func newSettingsHandler(t *testing.T) *SettingsHandler {
	t.Helper()
	db := newTestDB(t)
	sm := core.NewSettingsManager(db)
	if err := sm.SetJWTSecret("initial-secret-value-0001"); err != nil {
		t.Fatal(err)
	}
	return NewSettingsHandler(sm)
}

func TestGetJWTSecretReturnsMasked(t *testing.T) {
	h := newSettingsHandler(t)
	rr := httptest.NewRecorder()
	h.GetJWTSecret(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	body := decodeBody[map[string]any](t, rr)
	masked, _ := body["masked"].(string)
	if masked == "" || masked == "initial-secret-value-0001" {
		t.Fatalf("secret should be masked, got %q", masked)
	}
	if masked == "initial-secret-value-0001" {
		t.Fatal("plain secret leaked")
	}
	if present, _ := body["present"].(bool); !present {
		t.Fatal("expected present=true")
	}
	if length, _ := body["length"].(float64); int(length) != len("initial-secret-value-0001") {
		t.Fatalf("length = %v", length)
	}
}

func TestSetJWTSecretRotatesImmediately(t *testing.T) {
	h := newSettingsHandler(t)

	// 用初始密钥登录拿 token
	auth := NewAuthHandler("admin", "pass", staticSecretProvider("initial-secret-value-0001"))
	loginRR := httptest.NewRecorder()
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "pass"})
	auth.Login(loginRR, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody)))
	if loginRR.Code != http.StatusOK {
		t.Fatalf("login status = %d", loginRR.Code)
	}
	authResp := decodeBody[model.AuthResponse](t, loginRR)
	if authResp.Token == "" {
		t.Fatal("expected token")
	}

	// 轮换密钥
	newSecret := "rotated-secret-value-9999"
	rotateRR := httptest.NewRecorder()
	rotateBody, _ := json.Marshal(map[string]string{"secret": newSecret})
	h.SetJWTSecret(rotateRR, httptest.NewRequest(http.MethodPut, "/api/settings/jwt-secret", bytes.NewReader(rotateBody)))
	if rotateRR.Code != http.StatusOK {
		t.Fatalf("rotate status = %d, body=%s", rotateRR.Code, rotateRR.Body.String())
	}
	rb := decodeBody[map[string]any](t, rotateRR)
	masked, _ := rb["masked"].(string)
	if masked == "" || masked == newSecret {
		t.Fatalf("rotate response should mask secret, got %q", masked)
	}

	// 旧 token 应在 AuthMiddleware 下失效
	mw := AuthMiddleware(h.settings)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+authResp.Token)
	invalidRR2 := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(invalidRR2, req)
	if invalidRR2.Code != http.StatusUnauthorized {
		t.Fatalf("old token should be invalid after rotation, got %d", invalidRR2.Code)
	}

	// 用新密钥登录拿到有效 token
	auth2 := NewAuthHandler("admin", "pass", staticSecretProvider(newSecret))
	login2RR := httptest.NewRecorder()
	auth2.Login(login2RR, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody)))
	if login2RR.Code != http.StatusOK {
		t.Fatalf("login with new secret status = %d", login2RR.Code)
	}
	authResp = decodeBody[model.AuthResponse](t, login2RR)
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Authorization", "Bearer "+authResp.Token)
	validRR := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(validRR, req2)
	if validRR.Code != http.StatusOK {
		t.Fatalf("new token should be valid, got %d", validRR.Code)
	}
}

func TestSetJWTSecretRejectsEmpty(t *testing.T) {
	h := newSettingsHandler(t)
	rr := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"secret": ""})
	h.SetJWTSecret(rr, httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(body)))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestSetJWTSecretRejectsShort(t *testing.T) {
	h := newSettingsHandler(t)
	rr := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"secret": "short"})
	h.SetJWTSecret(rr, httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(body)))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestSetJWTSecretRejectsInvalidBody(t *testing.T) {
	h := newSettingsHandler(t)
	rr := httptest.NewRecorder()
	h.SetJWTSecret(rr, httptest.NewRequest(http.MethodPut, "/", bytes.NewReader([]byte("not-json"))))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestMaskJWTSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		want   string
	}{
		{"empty", "", ""},
		{"short", "ab", "********"},
		{"boundary", "abcd", "********"},
		{"normal", "1234567890abcdef", "12********ef"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskJWTSecret(tt.secret); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetJWTSecretEmptyWhenNotSet(t *testing.T) {
	db := newTestDB(t)
	sm := core.NewSettingsManager(db)
	h := NewSettingsHandler(sm)
	rr := httptest.NewRecorder()
	h.GetJWTSecret(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	body := decodeBody[map[string]any](t, rr)
	if present, _ := body["present"].(bool); present {
		t.Fatal("expected present=false")
	}
	if masked, _ := body["masked"].(string); masked != "" {
		t.Fatalf("expected empty masked, got %q", masked)
	}
}
