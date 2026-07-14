package model

const (
	StatusOK         = "ok"
	StatusError      = "error"
	StatusPartial    = "partial"
	StatusRolledBack = "rolled_back"
)

const (
	ErrorInvalidRequest       = "invalid_request"
	ErrorUnauthorized         = "unauthorized"
	ErrorForbidden            = "forbidden"
	ErrorRateLimited          = "rate_limited"
	ErrorNotFound             = "not_found"
	ErrorConflict             = "conflict"
	ErrorUnavailable          = "unavailable"
	ErrorInternal             = "internal_error"
	ErrorBadGateway           = "bad_gateway"
	ErrorConfigInvalidRuntime = "config_invalid_runtime"
	ErrorConfigRestartFailed  = "config_restart_failed"
	ErrorConfigRollbackFailed = "config_rollback_failed"
	ErrorNodeNotFound         = "node_not_found"
	ErrorNodeUpdateFailed     = "node_update_failed"
	ErrorSubscriptionNotFound = "subscription_not_found"
	ErrorSubscriptionRefresh  = "subscription_refresh_failed"
	ErrorRuntimeGroupNotFound = "runtime_group_not_found"
	ErrorRuntimeNotSelectable = "runtime_not_selectable"
	ErrorRuntimeDelayFailed   = "runtime_delay_failed"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIResponse struct {
	Status string    `json:"status"`
	Data   any       `json:"data"`
	Error  *APIError `json:"error"`
	Meta   any       `json:"meta"`
}
