package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/xuthus5/boxd/kernel"
)

type errorBody struct {
	Code    string `json:"code"`
	Op      string `json:"op,omitempty"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type apiError struct {
	status  int
	code    string
	op      string
	message string
	hint    string
	err     error
}

func (e *apiError) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

func newAPIError(status int, code, op, message, hint string) *apiError {
	return &apiError{status: status, code: code, op: op, message: message, hint: hint}
}

func (s *Server) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	apiErr := mapServiceError(err)
	if apiErr.status >= http.StatusInternalServerError {
		s.logger.Error("HTTP API operation failed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("error", describeErrorChain(err)),
		)
		s.logs.add("error", "api", "control-plane operation failed", map[string]any{
			"method": r.Method,
			"path":   r.URL.Path,
			"code":   apiErr.code,
		})
	}
	writeError(w, r, apiErr)
}

func describeErrorChain(err error) string {
	parts := make([]string, 0, 4)
	for err != nil {
		parts = append(parts, err.Error())
		err = errors.Unwrap(err)
	}
	return strings.Join(parts, ": ")
}

func mapServiceError(err error) *apiError {
	if err == nil {
		return nil
	}
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	var serviceErr *kernel.Error
	if !errors.As(err, &serviceErr) {
		return newAPIError(http.StatusInternalServerError, "internal", "http.operation", "internal server error", "retry the request")
	}
	status := http.StatusInternalServerError
	switch serviceErr.Code {
	case kernel.ErrCodeUnauthorized:
		status = http.StatusUnauthorized
	case kernel.ErrCodeNotFound:
		status = http.StatusNotFound
	case kernel.ErrCodeConflict, kernel.ErrCodeInvalidState, kernel.ErrCodeNotRunning, kernel.ErrCodeFeatureDisabled, kernel.ErrCodeEngineIncompatible:
		status = http.StatusConflict
	case kernel.ErrCodeInvalidConfig:
		status = http.StatusBadRequest
	case kernel.ErrCodeApplyFailed:
		status = http.StatusUnprocessableEntity
	case kernel.ErrCodeTimeout:
		status = http.StatusGatewayTimeout
	case kernel.ErrCodeCanceled:
		status = http.StatusRequestTimeout
	case kernel.ErrCodeInternal:
		status = http.StatusInternalServerError
	}
	message := serviceErr.Message
	hint := serviceErr.Hint
	if status == http.StatusInternalServerError {
		message = "internal server error"
		if hint == "" {
			hint = "check the application logs and retry"
		}
	}
	return newAPIError(status, string(serviceErr.Code), serviceErr.Op, message, hint)
}

func writeError(w http.ResponseWriter, _ *http.Request, err *apiError) {
	if err == nil {
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(err.status)
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: errorBody{
		Code:    err.code,
		Op:      err.op,
		Message: err.message,
		Hint:    err.hint,
	}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return newAPIError(http.StatusBadRequest, "invalid_json", "http.decode", "request body is not valid JSON", "check the request schema and content type")
	}
	return nil
}
