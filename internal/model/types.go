package model

import "time"

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ServiceStatus struct {
	Running       bool       `json:"running"`
	Uptime        string     `json:"uptime,omitempty"`
	Memory        int64      `json:"memory,omitempty"`
	Version       string     `json:"version,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	ConfigPath    string     `json:"config_path,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	LastErrorCode string     `json:"last_error_code,omitempty"`
	LastErrorAt   *time.Time `json:"last_error_at,omitempty"`
}

type TrafficEvent struct {
	UploadBytes   int64     `json:"upload_bytes"`
	DownloadBytes int64     `json:"download_bytes"`
	Timestamp     time.Time `json:"timestamp"`
}

type LogEvent struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type ConnectionEvent struct {
	ActiveConnections int          `json:"active_connections"`
	List              []Connection `json:"list,omitempty"`
}

type Connection struct {
	ID       string `json:"id"`
	Target   string `json:"target"`
	Outbound string `json:"outbound"`
	Upload   int64  `json:"upload"`
	Download int64  `json:"download"`
	Duration string `json:"duration"`
	Rule     string `json:"rule,omitempty"`
}

type SubscriptionTraffic struct {
	Upload   int64      `json:"upload"`
	Download int64      `json:"download"`
	Total    int64      `json:"total"`
	Expire   *time.Time `json:"expire,omitempty"`
}

type Subscription struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	URL         string               `json:"url"`
	IntervalMin int                  `json:"interval_min"`
	URLTest     *URLTestOverrides    `json:"urltest,omitempty"`
	LastUpdated time.Time            `json:"last_updated"`
	Error       string               `json:"error,omitempty"`
	ErrorCode   string               `json:"error_code,omitempty"`
	ErrorAt     *time.Time           `json:"error_at,omitempty"`
	Traffic     *SubscriptionTraffic `json:"traffic,omitempty"`
	Outbounds   []Outbound           `json:"outbounds,omitempty"`
}

type URLTestDefaults struct {
	Enabled   bool   `json:"enabled"`
	URL       string `json:"url"`
	Interval  string `json:"interval"`
	Tolerance uint16 `json:"tolerance"`
}

type URLTestOverrides struct {
	Enabled   *bool   `json:"enabled,omitempty"`
	URL       *string `json:"url,omitempty"`
	Interval  *string `json:"interval,omitempty"`
	Tolerance *uint16 `json:"tolerance,omitempty"`
}

type RouteRuleMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Outbound struct {
	Tag    string `json:"tag"`
	Type   string `json:"type"`
	Server string `json:"server"`
	Port   int    `json:"port"`
	Raw    any    `json:"raw"`
}

type ImportResult struct {
	Tag    string `json:"tag"`
	Type   string `json:"type"`
	Server string `json:"server"`
	Port   int    `json:"port"`
	Config any    `json:"config"`
}

type TestResult struct {
	Tag       string    `json:"tag"`
	TestType  string    `json:"test_type"`
	Success   bool      `json:"success"`
	LatencyMs float64   `json:"latency_ms,omitempty"`
	Error     string    `json:"error,omitempty"`
	ErrorCode string    `json:"error_code,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// UIPreferences stores panel appearance preferences persisted in the database.
type UIPreferences struct {
	Theme           string `json:"theme"`
	Language        string `json:"language"`
	MinimumLogLevel string `json:"minimumLogLevel"`
}

// LatencyPoint is one historical node probe sample.
type LatencyPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	LatencyMs float64   `json:"latency_ms,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// ConfigApplyEvent is one config write/reload attempt for the ops timeline.
type ConfigApplyEvent struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	Status    string    `json:"status"`
	Hash      string    `json:"hash"`
	Size      int       `json:"size"`
	Error     string    `json:"error,omitempty"`
	ErrorCode string    `json:"error_code,omitempty"`
	AppliedAt time.Time `json:"applied_at"`
}
