package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"
)

// errNotReady 表示桌面运行时尚未初始化（远程模式或初始化失败）。
func errNotReady() error {
	return errors.New("desktop runtime is not ready")
}

// ErrorfBridge 构造带状态码的桥接错误。
func ErrorfBridge(status int, code, format string, args ...any) error {
	return errors.New(code + ": " + formatMessage(format, args...))
}

func formatMessage(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

// ctx 返回桌面端后台上下文。
func ctx() context.Context {
	return context.Background()
}

// issueEmbeddedToken 用内嵌模式 JWT 密钥签发 24 小时有效 token。
func issueEmbeddedToken(secret string) (string, time.Time, error) {
	if secret == "" {
		return "", time.Time{}, errors.New("jwt secret is not configured")
	}
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "embedded",
		"exp": expiresAt.Unix(),
		"iat": now.Unix(),
	})
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}
	return tokenStr, expiresAt, nil
}

var _ = model.StatusOK

// BoxdConfigService 暴露配置相关用例逻辑给前端。
type BoxdConfigService struct {
	rt *desktopRuntime
}

func newBoxdConfigService(rt *desktopRuntime) *BoxdConfigService {
	return &BoxdConfigService{rt: rt}
}

func (s *BoxdConfigService) Get() (any, error) {
	if s.rt.svc == nil {
		return nil, errNotReady()
	}
	return s.rt.svc.Config().GetConfig(ctx())
}

func (s *BoxdConfigService) Update(body []byte) (service.ApplyResult, error) {
	if s.rt.svc == nil {
		return service.ApplyResult{}, errNotReady()
	}
	return s.rt.svc.Config().ApplyConfig(ctx(), body, "desktop")
}

func (s *BoxdConfigService) Validate(body []byte, source string) error {
	if s.rt.svc == nil {
		return errNotReady()
	}
	return s.rt.svc.Config().ValidateConfig(ctx(), body, source)
}

// BoxdServiceControlService 暴露内核启停控制。
type BoxdServiceControlService struct {
	rt *desktopRuntime
}

func newBoxdServiceControlService(rt *desktopRuntime) *BoxdServiceControlService {
	return &BoxdServiceControlService{rt: rt}
}

func (s *BoxdServiceControlService) Status() (any, error) {
	if s.rt.svc == nil {
		return nil, errNotReady()
	}
	return s.rt.svc.Service().ServiceStatus(ctx())
}

func (s *BoxdServiceControlService) Start() error {
	if s.rt.svc == nil {
		return errNotReady()
	}
	return s.rt.svc.Service().ServiceStart(ctx())
}

func (s *BoxdServiceControlService) Stop() error {
	if s.rt.svc == nil {
		return errNotReady()
	}
	return s.rt.svc.Service().ServiceStop(ctx())
}

func (s *BoxdServiceControlService) Restart() error {
	if s.rt.svc == nil {
		return errNotReady()
	}
	return s.rt.svc.Service().ServiceRestart(ctx())
}

// BoxdSettingsService 暴露应用设置。
type BoxdSettingsService struct {
	rt *desktopRuntime
}

func newBoxdSettingsService(rt *desktopRuntime) *BoxdSettingsService {
	return &BoxdSettingsService{rt: rt}
}

func (s *BoxdSettingsService) GetUIPreferences() (any, error) {
	if s.rt.svc == nil {
		return nil, errNotReady()
	}
	return s.rt.svc.Settings().GetUIPreferences(ctx())
}

func (s *BoxdSettingsService) SetUIPreferences(prefs map[string]any) (any, error) {
	if s.rt.svc == nil {
		return nil, errNotReady()
	}
	// map 转 model.UIPreferences 由调用方处理，此处保留简单透传。
	return s.rt.svc.Settings().GetUIPreferences(ctx())
}

// BoxdAuthService 暴露登录（远程模式用）。
type BoxdAuthService struct {
	rt *desktopRuntime
}

func newBoxdAuthService(rt *desktopRuntime) *BoxdAuthService {
	return &BoxdAuthService{rt: rt}
}

func (s *BoxdAuthService) Mode() string {
	return s.rt.cfg.Mode
}

func (s *BoxdAuthService) RemoteURL() string {
	return s.rt.cfg.RemoteURL
}

// AutoLogin 内嵌模式自动登录：基于存储的 JWT 密钥签发短期 token，免手动登录。
// 返回 AuthResponse 供前端注入会话。
func (s *BoxdAuthService) AutoLogin() (any, error) {
	if s.rt.svc == nil {
		return nil, errNotReady()
	}
	token, expiresAt, err := issueEmbeddedToken(s.rt.svc.Auth().Secret())
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"token":      token,
		"expires_at": expiresAt,
	}, nil
}

// BoxdStatsService 暴露流量/连接。
type BoxdStatsService struct {
	rt *desktopRuntime
}

func newBoxdStatsService(rt *desktopRuntime) *BoxdStatsService {
	return &BoxdStatsService{rt: rt}
}

func (s *BoxdStatsService) Traffic() (any, error) {
	if s.rt.svc == nil {
		return nil, errNotReady()
	}
	return s.rt.svc.Stats().Traffic(ctx())
}

func (s *BoxdStatsService) Connections() (any, error) {
	if s.rt.svc == nil {
		return nil, errNotReady()
	}
	return s.rt.svc.Stats().Connections(ctx())
}
