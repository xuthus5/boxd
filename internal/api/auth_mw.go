package api

import (
	"context"
	"net/http"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

type contextKey string

const userKey contextKey = "user"

// AuthMiddleware 基于可动态轮换的 SecretProvider 构造认证中间件。
// 每次校验 token 时实时读取当前密钥，轮换后旧 token 立即失效。
func AuthMiddleware(provider core.SecretProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenStr string

			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
					tokenStr = parts[1]
				}
			}

			if tokenStr == "" {
				writeJSONErrorCode(w, http.StatusUnauthorized, model.ErrorUnauthorized, "missing authorization")
				return
			}

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(provider.JWTSecret()), nil
			})

			if err != nil || !token.Valid {
				writeJSONErrorCode(w, http.StatusUnauthorized, model.ErrorUnauthorized, "invalid or expired token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeJSONErrorCode(w, http.StatusUnauthorized, model.ErrorUnauthorized, "invalid token claims")
				return
			}

			ctx := context.WithValue(r.Context(), userKey, claims["sub"])
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
