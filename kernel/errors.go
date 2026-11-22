package kernel

import "fmt"

// ErrorCode is a stable machine-readable error identifier.
type ErrorCode string

const (
	ErrCodeNotRunning         ErrorCode = "kernel_not_running"
	ErrCodeInvalidState       ErrorCode = "kernel_invalid_state"
	ErrCodeConflict           ErrorCode = "kernel_conflict"
	ErrCodeInvalidConfig      ErrorCode = "kernel_invalid_config"
	ErrCodeApplyFailed        ErrorCode = "kernel_apply_failed"
	ErrCodeFeatureDisabled    ErrorCode = "kernel_feature_disabled"
	ErrCodeEngineIncompatible ErrorCode = "kernel_engine_incompatible"
	ErrCodeTimeout            ErrorCode = "kernel_timeout"
	ErrCodeCanceled           ErrorCode = "kernel_canceled"
	ErrCodeNotFound           ErrorCode = "kernel_not_found"
	ErrCodeUnauthorized       ErrorCode = "kernel_unauthorized"
	ErrCodeInternal           ErrorCode = "kernel_internal"
)

// Error is the only error type callers should inspect across the kernel boundary.
type Error struct {
	Code      ErrorCode
	Op        string
	Message   string
	Hint      string
	Retryable bool
	Temporary bool
	Err       error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is matches by stable error code so callers never need string matching.
func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	return ok && e != nil && other != nil && e.Code == other.Code
}

// NewError creates a kernel error without an underlying cause.
func NewError(code ErrorCode, op, message string) *Error {
	return &Error{Code: code, Op: op, Message: message}
}

// WrapError converts an infrastructure failure into the stable kernel model.
func WrapError(code ErrorCode, op, message string, cause error) *Error {
	return &Error{Code: code, Op: op, Message: message, Err: cause}
}

// WithHint adds an actionable hint to an error.
func (e *Error) WithHint(hint string) *Error {
	if e != nil {
		e.Hint = hint
	}
	return e
}

// ErrConfigNotFound indicates that no configuration has been persisted yet.
var ErrConfigNotFound = &Error{
	Code:    ErrCodeNotFound,
	Op:      "config.load",
	Message: "no configuration is available",
	Hint:    "apply an initial configuration before starting the kernel",
}

// NewValidationError returns a stable invalid-config error with diagnostics.
func NewValidationError(diagnostics []Diagnostic) *Error {
	message := "configuration validation failed"
	hint := ""
	if len(diagnostics) > 0 {
		hint = diagnostics[0].Message
	}
	return &Error{
		Code:    ErrCodeInvalidConfig,
		Op:      "config.validate",
		Message: fmt.Sprintf("%s (%d diagnostics)", message, len(diagnostics)),
		Hint:    hint,
	}
}
