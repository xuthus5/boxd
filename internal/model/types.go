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
	Running bool   `json:"running"`
	Uptime  string `json:"uptime,omitempty"`
	Memory  int64  `json:"memory,omitempty"`
	Version string `json:"version,omitempty"`
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

type Subscription struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	URL         string     `json:"url"`
	IntervalMin int        `json:"interval_min"`
	LastUpdated time.Time  `json:"last_updated"`
	Error       string     `json:"error,omitempty"`
	Outbounds   []Outbound `json:"outbounds,omitempty"`
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
	Tag       string  `json:"tag"`
	TestType  string  `json:"test_type"`
	Success   bool    `json:"success"`
	LatencyMs float64 `json:"latency_ms,omitempty"`
	Error     string  `json:"error,omitempty"`
}
