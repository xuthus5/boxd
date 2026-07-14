package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"time"

	"github.com/xuthus5/boxui/internal/model"
)

func writeJSONEnvelope(w http.ResponseWriter, statusCode int, status string, data any, apiErr *model.APIError, meta any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(model.APIResponse{
		Status: status,
		Data:   data,
		Error:  apiErr,
		Meta:   meta,
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	writeJSONEnvelope(w, statusCode, model.StatusOK, data, nil, nil)
}

func writeJSONWithMeta(w http.ResponseWriter, statusCode int, data any, meta any) {
	writeJSONEnvelope(w, statusCode, model.StatusOK, data, nil, meta)
}

func writeJSONStatus(w http.ResponseWriter, statusCode int, status string, data any, apiErr *model.APIError, meta any) {
	writeJSONEnvelope(w, statusCode, status, data, apiErr, meta)
}

func writeJSONErrorCode(w http.ResponseWriter, statusCode int, code, msg string) {
	writeJSONEnvelope(w, statusCode, model.StatusError, nil, &model.APIError{
		Code:    code,
		Message: msg,
	}, nil)
}

func defaultErrorCode(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return model.ErrorInvalidRequest
	case http.StatusUnauthorized:
		return model.ErrorUnauthorized
	case http.StatusForbidden:
		return model.ErrorForbidden
	case http.StatusTooManyRequests:
		return model.ErrorRateLimited
	case http.StatusNotFound:
		return model.ErrorNotFound
	case http.StatusConflict:
		return model.ErrorConflict
	case http.StatusServiceUnavailable:
		return model.ErrorUnavailable
	case http.StatusBadGateway:
		return model.ErrorBadGateway
	default:
		return model.ErrorInternal
	}
}

func writeJSONError(w http.ResponseWriter, statusCode int, msg string) {
	writeJSONErrorCode(w, statusCode, defaultErrorCode(statusCode), msg)
}

// statusCaptureWriter 包裹 ResponseWriter 以捕获最终写入的 HTTP 状态码。
// 同时透传 http.Flusher / http.Hijacker / io.ReaderFrom 等接口，保证 SSE、WebSocket、sendfile 等正常工作。
type statusCaptureWriter struct {
	http.ResponseWriter
	statusCode int
}

func (sc *statusCaptureWriter) WriteHeader(code int) {
	sc.statusCode = code
	sc.ResponseWriter.WriteHeader(code)
}

// Flush 透传 http.Flusher，确保 SSE 等需要即时刷新的场景正常工作。
func (sc *statusCaptureWriter) Flush() {
	if f, ok := sc.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack 透传 http.Hijacker，确保 WebSocket 等连接升级正常工作。
func (sc *statusCaptureWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := sc.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("response writer does not implement http.Hijacker")
}

// LoggerMiddleware 记录每个 HTTP 请求的结构化日志，包含方法、路径、状态码、耗时。
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sc := &statusCaptureWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(sc, r)
		slog.InfoContext(r.Context(), "http request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
			"status", sc.statusCode,
			"duration", time.Since(start),
		)
	})
}

func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowMethods := "GET, POST, PUT, DELETE, OPTIONS"
	allowHeaders := "Content-Type, Authorization"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowed := origin != "" && slices.Contains(allowedOrigins, origin)

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", allowMethods)
				w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
				w.Header().Add("Vary", "Origin")
			}

			if r.Method == http.MethodOptions {
				if !allowed {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered", "err", err)
				writeJSONError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// maxRequestBodyBytes 限制请求体大小，防止恶意大体积请求耗尽内存。
const maxRequestBodyBytes = 2 * 1024 * 1024 // 2MB

// BodyLimitMiddleware 限制请求体大小，超过限制返回 413。
func BodyLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware 添加安全相关的 HTTP 响应头。
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
}
