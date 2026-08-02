package service

import (
	"context"
	"crypto/subtle"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"

	"github.com/xuthus5/boxd/internal/model"
)

// SecretProvider 提供动态 JWT 密钥，支持运行时轮换。
type SecretProvider interface {
	JWTSecret() string
}

// AuthCredentials 描述登录凭据。
type AuthCredentials struct {
	Username string
	Password string
}

// AuthService 提供登录/登出用例逻辑。
type AuthService struct {
	username string
	verify   func(string) bool
	secrets  SecretProvider
	limiter  *loginRateLimiter
	now      func() time.Time
}

// NewAuthService 构造认证用例服务。
func NewAuthService(username, password string, provider SecretProvider) *AuthService {
	verify := func(candidate string) bool {
		return subtle.ConstantTimeCompare([]byte(candidate), []byte(password)) == 1
	}
	if credentials, ok := provider.(interface{ VerifyAdminPassword(string) bool }); ok {
		verify = credentials.VerifyAdminPassword
	}
	return &AuthService{
		username: username,
		verify:   verify,
		secrets:  provider,
		limiter:  newLoginRateLimiter(),
		now:      time.Now,
	}
}

// Login 校验凭据并签发 JWT。
func (s *AuthService) Login(_ context.Context, remoteAddr string, creds AuthCredentials) (model.AuthResponse, error) {
	key := clientIP(remoteAddr)
	if s.limiter != nil && !s.limiter.allow(key, s.now()) {
		return model.AuthResponse{}, Errorf(429, model.ErrorRateLimited, "too many login attempts")
	}

	if creds.Username != s.username || !s.verify(creds.Password) {
		if s.limiter != nil {
			s.limiter.recordFailure(key, s.now())
		}
		return model.AuthResponse{}, Errorf(401, model.ErrorUnauthorized, "invalid credentials")
	}

	if s.limiter != nil {
		s.limiter.recordSuccess(key)
	}

	secret := s.secrets.JWTSecret()
	expiresAt := s.now().Add(24 * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": creds.Username,
		"exp": expiresAt.Unix(),
		"iat": s.now().Unix(),
	})

	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		return model.AuthResponse{}, Errorf(500, model.ErrorInternal, "token generation failed")
	}

	return model.AuthResponse{Token: tokenStr, ExpiresAt: expiresAt}, nil
}

// Logout 登出（无服务端状态需要清理）。
func (s *AuthService) Logout(_ context.Context) error {
	return nil
}

// Secret 返回当前 JWT 签名密钥。
func (s *AuthService) Secret() string {
	if s.secrets == nil {
		return ""
	}
	return s.secrets.JWTSecret()
}
