package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

var defaultTestURL = "https://cp.cloudflare.com/"

type SettingsHandler struct {
	settings *core.SettingsManager
	username string
}

func NewSettingsHandler(settings *core.SettingsManager, usernames ...string) *SettingsHandler {
	username := "admin"
	if len(usernames) > 0 && usernames[0] != "" {
		username = usernames[0]
	}
	return &SettingsHandler{settings: settings, username: username}
}

func (h *SettingsHandler) GetPasswordStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"defaultPassword": h.settings.AdminPasswordIsDefault()})
}

func (h *SettingsHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request")
		return
	}
	err := h.settings.ChangeAdminPassword(h.username, req.CurrentPassword, req.NewPassword)
	if errors.Is(err, core.ErrCurrentPasswordInvalid) {
		writeJSONErrorCode(w, http.StatusUnauthorized, model.ErrorUnauthorized, err.Error())
		return
	}
	if errors.Is(err, core.ErrWeakPassword) {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, err.Error())
		return
	}
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to change password")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"changed": true})
}

func (h *SettingsHandler) GetTestURL(w http.ResponseWriter, r *http.Request) {
	url := h.settings.Get("url_test")
	if url == "" {
		url = defaultTestURL
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

func (h *SettingsHandler) SetTestURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request")
		return
	}
	if req.URL == "" {
		req.URL = defaultTestURL
	}
	if err := h.settings.Set("url_test", req.URL); err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to save")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": req.URL})
}

const kernelAutostartKey = "kernel_autostart"

// GetKernelAutostart GET /api/settings/kernel-autostart
func (h *SettingsHandler) GetKernelAutostart(w http.ResponseWriter, r *http.Request) {
	val := h.settings.Get(kernelAutostartKey)
	enabled := val == "true"
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": enabled})
}

// SetKernelAutostart PUT /api/settings/kernel-autostart
func (h *SettingsHandler) SetKernelAutostart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request")
		return
	}
	val := "false"
	if req.Enabled {
		val = "true"
	}
	if err := h.settings.Set(kernelAutostartKey, val); err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to save")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": req.Enabled})
}

// maskJWTSecret 对密钥做脱敏展示：长度不足时仅返回掩码，其余保留首尾各 2 个字符。
func maskJWTSecret(secret string) string {
	n := len(secret)
	if n == 0 {
		return ""
	}
	if n <= 4 {
		return strings.Repeat("*", 8)
	}
	return secret[:2] + strings.Repeat("*", 8) + secret[n-2:]
}

// GetJWTSecret GET /api/settings/jwt-secret
// 返回脱敏后的密钥与是否存在，避免明文泄露签发能力。
func (h *SettingsHandler) GetJWTSecret(w http.ResponseWriter, r *http.Request) {
	secret := h.settings.JWTSecret()
	writeJSON(w, http.StatusOK, map[string]any{
		"masked":  maskJWTSecret(secret),
		"present": secret != "",
		"length":  len(secret),
	})
}

// SetJWTSecret PUT /api/settings/jwt-secret
// 轮换 JWT 签名密钥，轮换后已签发的 token 立即失效，需重新登录。
func (h *SettingsHandler) SetJWTSecret(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request")
		return
	}
	if req.Secret == "" {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "secret must not be empty")
		return
	}
	if err := h.settings.SetJWTSecret(req.Secret); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"masked": maskJWTSecret(req.Secret),
		"length": len(req.Secret),
	})
}
