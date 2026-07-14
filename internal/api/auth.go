package api

import (
	"encoding/json"
	"net/http"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

type AuthHandler struct {
	username string
	verify   func(string) bool
	secrets  core.SecretProvider
	limiter  *loginRateLimiter
	now      func() time.Time
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	key := clientIP(r.RemoteAddr)
	if h.limiter != nil && !h.limiter.allow(key, h.now()) {
		writeJSONErrorCode(w, http.StatusTooManyRequests, model.ErrorRateLimited, "too many login attempts")
		return
	}

	var req model.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request body")
		return
	}

	if req.Username != h.username || !h.verify(req.Password) {
		if h.limiter != nil {
			h.limiter.recordFailure(key, h.now())
		}
		writeJSONErrorCode(w, http.StatusUnauthorized, model.ErrorUnauthorized, "invalid credentials")
		return
	}

	if h.limiter != nil {
		h.limiter.recordSuccess(key)
	}

	secret := h.secrets.JWTSecret()
	expiresAt := h.now().Add(24 * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": req.Username,
		"exp": expiresAt.Unix(),
		"iat": h.now().Unix(),
	})

	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "token generation failed")
		return
	}

	resp := model.AuthResponse{Token: tokenStr, ExpiresAt: expiresAt}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, nil)
}
