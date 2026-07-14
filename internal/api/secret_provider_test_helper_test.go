package api

// staticSecretProvider 仅用于测试：将固定字符串包装为 SecretProvider，
// 方便在不依赖 bbolt 的情况下测试签名与校验逻辑。
type staticSecretProvider string

func (s staticSecretProvider) JWTSecret() string         { return string(s) }
func (s staticSecretProvider) SetJWTSecret(string) error { return nil }
