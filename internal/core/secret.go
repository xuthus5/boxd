package core

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const jwtSecretKey = "jwt_secret"

// minJWTSecretLen 是 JWT 签名密钥的最小长度（字节）。
const minJWTSecretLen = 16

// SecretProvider 提供运行时动态读取与轮换 JWT 签名密钥的能力。
// 替代在构造时固化密钥的方式，使配置页面可在不重启进程的情况下轮换密钥。
type SecretProvider interface {
	JWTSecret() string
	SetJWTSecret(secret string) error
}

// JWTSecret 返回当前存储的 JWT 签名密钥；不存在时返回空串。
func (m *SettingsManager) JWTSecret() string {
	return m.Get(jwtSecretKey)
}

// SetJWTSecret 更新 JWT 签名密钥并持久化到 bbolt。
func (m *SettingsManager) SetJWTSecret(secret string) error {
	if err := validateJWTSecret(secret); err != nil {
		return err
	}
	return m.Set(jwtSecretKey, secret)
}

// EnsureJWTSecret 确保数据库中存在合法的 JWT 密钥。
// 若数据库已有合法值则复用，否则随机生成并持久化。
// 返回最终使用的密钥与是否为本次生成（true 表示密钥由本函数随机生成）。
func (m *SettingsManager) EnsureJWTSecret() (string, bool, error) {
	if existing := m.JWTSecret(); existing != "" {
		if err := validateJWTSecret(existing); err != nil {
			return "", false, fmt.Errorf("stored jwt secret invalid: %w", err)
		}
		return existing, false, nil
	}

	secret, err := generateJWTSecret()
	if err != nil {
		return "", false, fmt.Errorf("generate jwt secret: %w", err)
	}
	if err := m.Set(jwtSecretKey, secret); err != nil {
		return "", false, fmt.Errorf("persist jwt secret: %w", err)
	}
	return secret, true, nil
}

// validateJWTSecret 校验密钥长度约束。
func validateJWTSecret(secret string) error {
	if secret == "" {
		return fmt.Errorf("jwt secret must not be empty")
	}
	if len(secret) < minJWTSecretLen {
		return fmt.Errorf("jwt secret must be at least %d characters", minJWTSecretLen)
	}
	return nil
}

// generateJWTSecret 生成 32 字节（256 bit）的随机 base64 密钥。
func generateJWTSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
