package service

import (
	"context"
	"errors"
	"strings"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

// SettingsService 提供应用设置用例逻辑。
type SettingsService struct {
	settings *core.SettingsManager
	username string
}

// NewSettingsService 构造设置用例服务。
func NewSettingsService(settings *core.SettingsManager, username string) *SettingsService {
	return &SettingsService{settings: settings, username: username}
}

// GetPasswordStatus 返回是否仍在使用默认密码。
func (s *SettingsService) GetPasswordStatus(_ context.Context) (map[string]bool, error) {
	return map[string]bool{"defaultPassword": s.settings.AdminPasswordIsDefault()}, nil
}

// ChangePassword 修改管理员密码。
func (s *SettingsService) ChangePassword(_ context.Context, currentPassword, newPassword string) (map[string]bool, error) {
	err := s.settings.ChangeAdminPassword(s.username, currentPassword, newPassword)
	if errors.Is(err, core.ErrCurrentPasswordInvalid) {
		return nil, Errorf(401, model.ErrorUnauthorized, "%v", err)
	}
	if errors.Is(err, core.ErrWeakPassword) {
		return nil, Errorf(400, model.ErrorInvalidRequest, "%v", err)
	}
	if err != nil {
		return nil, Errorf(500, model.ErrorInternal, "failed to change password")
	}
	return map[string]bool{"changed": true}, nil
}

// GetTestURL 返回系统测速 URL，无效时回退默认。
func (s *SettingsService) GetTestURL(_ context.Context) (map[string]string, error) {
	url := s.settings.Get("url_test")
	invalidURL := false
	if url != "" {
		if err := core.ValidateHTTPURL(url); err != nil {
			invalidURL = true
		}
	}
	if url == "" || invalidURL {
		url = defaultTestURL
	}
	return map[string]string{"url": url}, nil
}

// SetTestURL 设置系统测速 URL。
func (s *SettingsService) SetTestURL(_ context.Context, url string) (map[string]string, error) {
	if url == "" {
		url = defaultTestURL
	}
	if err := core.ValidateHTTPURL(url); err != nil {
		return nil, Errorf(400, model.ErrorInvalidRequest, "%v", err)
	}
	if err := s.settings.Set("url_test", url); err != nil {
		return nil, Errorf(500, model.ErrorInternal, "failed to save")
	}
	return map[string]string{"url": url}, nil
}

// GetURLTestDefaults 返回 URLTest 默认配置。
func (s *SettingsService) GetURLTestDefaults(_ context.Context) (model.URLTestDefaults, error) {
	config, err := s.settings.URLTestDefaults()
	if err != nil {
		return model.URLTestDefaults{}, Errorf(500, model.ErrorInternal, "failed to load urltest defaults")
	}
	return config, nil
}

// SetURLTestDefaults 设置 URLTest 默认配置。
func (s *SettingsService) SetURLTestDefaults(_ context.Context, input URLTestDefaultsInput) (model.URLTestDefaults, error) {
	config, err := input.defaults()
	if err != nil {
		return model.URLTestDefaults{}, Errorf(400, model.ErrorInvalidRequest, "%v", err)
	}
	if err := core.ValidateURLTestDefaults(config); err != nil {
		return model.URLTestDefaults{}, Errorf(400, model.ErrorInvalidRequest, "%v", err)
	}
	if err := s.settings.SetURLTestDefaults(config); err != nil {
		return model.URLTestDefaults{}, Errorf(500, model.ErrorInternal, "failed to save urltest defaults")
	}
	return config, nil
}

// URLTestDefaultsInput 描述 URLTest 默认配置更新请求。
type URLTestDefaultsInput struct {
	Enabled   *bool   `json:"enabled"`
	URL       *string `json:"url"`
	Interval  *string `json:"interval"`
	Tolerance *uint16 `json:"tolerance"`
}

func (r URLTestDefaultsInput) defaults() (model.URLTestDefaults, error) {
	if r.Enabled == nil || r.URL == nil || r.Interval == nil || r.Tolerance == nil {
		return model.URLTestDefaults{}, errors.New("enabled, url, interval and tolerance are required")
	}
	return model.URLTestDefaults{
		Enabled:   *r.Enabled,
		URL:       *r.URL,
		Interval:  *r.Interval,
		Tolerance: *r.Tolerance,
	}, nil
}

// GetKernelAutostart 返回内核自启开关。
func (s *SettingsService) GetKernelAutostart(_ context.Context) (map[string]bool, error) {
	val := s.settings.Get("kernel_autostart")
	return map[string]bool{"enabled": val == "true"}, nil
}

// SetKernelAutostart 设置内核自启开关。
func (s *SettingsService) SetKernelAutostart(_ context.Context, enabled bool) (map[string]bool, error) {
	val := "false"
	if enabled {
		val = "true"
	}
	if err := s.settings.Set("kernel_autostart", val); err != nil {
		return nil, Errorf(500, model.ErrorInternal, "failed to save")
	}
	return map[string]bool{"enabled": enabled}, nil
}

// GetUIPreferences 返回界面偏好。
func (s *SettingsService) GetUIPreferences(_ context.Context) (model.UIPreferences, error) {
	prefs, err := s.settings.UIPreferences()
	if err != nil {
		return model.UIPreferences{}, Errorf(500, model.ErrorInternal, "failed to load preferences")
	}
	return prefs, nil
}

// SetUIPreferences 设置界面偏好。
func (s *SettingsService) SetUIPreferences(_ context.Context, prefs model.UIPreferences) (model.UIPreferences, error) {
	if err := core.ValidateUIPreferences(prefs); err != nil {
		return model.UIPreferences{}, Errorf(400, model.ErrorInvalidRequest, "%v", err)
	}
	if err := s.settings.SetUIPreferences(prefs); err != nil {
		return model.UIPreferences{}, Errorf(500, model.ErrorInternal, "failed to save preferences")
	}
	return prefs, nil
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

// GetJWTSecret 返回脱敏后的 JWT 密钥信息。
func (s *SettingsService) GetJWTSecret(_ context.Context) (map[string]any, error) {
	secret := s.settings.JWTSecret()
	return map[string]any{
		"masked":  maskJWTSecret(secret),
		"present": secret != "",
		"length":  len(secret),
	}, nil
}

// SetJWTSecret 轮换 JWT 签名密钥。
func (s *SettingsService) SetJWTSecret(_ context.Context, secret string) (map[string]any, error) {
	if secret == "" {
		return nil, Errorf(400, model.ErrorInvalidRequest, "secret must not be empty")
	}
	if err := s.settings.SetJWTSecret(secret); err != nil {
		return nil, Errorf(400, model.ErrorInvalidRequest, "%v", err)
	}
	return map[string]any{
		"masked": maskJWTSecret(secret),
		"length": len(secret),
	}, nil
}
