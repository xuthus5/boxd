package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxui/internal/core"
)

func TestPasswordSettingsChangeInvalidatesOldToken(t *testing.T) {
	db := openTestBoltDB(t)
	settings := core.NewSettingsManager(db)
	if _, err := settings.EnsureAdminCredential("admin", "current-password-123"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := settings.EnsureJWTSecret(); err != nil {
		t.Fatal(err)
	}

	auth := NewAuthHandler("admin", "", settings)
	oldToken := loginToken(t, auth, "current-password-123")
	handler := NewSettingsHandler(settings, "admin")

	rr := httptest.NewRecorder()
	handler.GetPasswordStatus(rr, httptest.NewRequest(http.MethodGet, "/api/settings/password", nil))
	if rr.Code != http.StatusOK || rr.Body.String() == "" {
		t.Fatalf("password status = %d %q", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	handler.ChangePassword(rr, jsonRequest(http.MethodPut, "/api/settings/password", `{"currentPassword":"wrong","newPassword":"replacement-password-456"}`))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("wrong current password status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.ChangePassword(rr, jsonRequest(http.MethodPut, "/api/settings/password", `{`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.ChangePassword(rr, jsonRequest(http.MethodPut, "/api/settings/password", `{"currentPassword":"current-password-123","newPassword":"short"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("weak password status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.ChangePassword(rr, jsonRequest(http.MethodPut, "/api/settings/password", `{"currentPassword":"current-password-123","newPassword":"replacement-password-456"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("change password status = %d body = %s", rr.Code, rr.Body.String())
	}

	protected := AuthMiddleware(settings)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+oldToken)
	rr = httptest.NewRecorder()
	protected.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("old token status = %d, want 401", rr.Code)
	}

	if token := loginToken(t, auth, "replacement-password-456"); token == "" {
		t.Fatal("login with changed password returned empty token")
	}
}

func loginToken(t *testing.T, handler *AuthHandler, password string) string {
	t.Helper()
	rr := httptest.NewRecorder()
	handler.Login(rr, jsonRequest(http.MethodPost, "/api/auth/login", `{"username":"admin","password":"`+password+`"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("login status = %d body = %s", rr.Code, rr.Body.String())
	}
	body := decodeBody[struct {
		Token string `json:"token"`
	}](t, rr)
	return body.Token
}

func openTestBoltDB(t *testing.T) *bbolt.DB {
	t.Helper()
	db, err := bbolt.Open(t.TempDir()+"/settings.db", 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	return db
}
