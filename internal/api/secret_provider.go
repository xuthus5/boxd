package api

import (
	"crypto/subtle"
	"time"

	"github.com/xuthus5/boxui/internal/core"
)

// NewAuthHandler 构造认证处理器，密钥由 provider 动态提供以支持运行时轮换。
func NewAuthHandler(username, password string, provider core.SecretProvider) *AuthHandler {
	verify := func(candidate string) bool {
		return subtle.ConstantTimeCompare([]byte(candidate), []byte(password)) == 1
	}
	if credentials, ok := provider.(interface{ VerifyAdminPassword(string) bool }); ok {
		verify = credentials.VerifyAdminPassword
	}
	return &AuthHandler{
		username: username,
		verify:   verify,
		secrets:  provider,
		limiter:  newLoginRateLimiter(),
		now:      time.Now,
	}
}
