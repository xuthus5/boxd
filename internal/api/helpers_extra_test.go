package api

import (
	"net/http"
	"testing"
	"time"
)

func TestDefaultErrorCode(t *testing.T) {
	tests := []struct {
		statusCode int
		want       string
	}{
		{http.StatusBadRequest, "invalid_request"},
		{http.StatusUnauthorized, "unauthorized"},
		{http.StatusForbidden, "forbidden"},
		{http.StatusTooManyRequests, "rate_limited"},
		{http.StatusNotFound, "not_found"},
		{http.StatusConflict, "conflict"},
		{http.StatusServiceUnavailable, "unavailable"},
		{http.StatusBadGateway, "bad_gateway"},
		{http.StatusInternalServerError, "internal_error"},
		{http.StatusTeapot, "internal_error"}, // 未知状态码走 default
	}

	for _, tt := range tests {
		got := defaultErrorCode(tt.statusCode)
		if got != tt.want {
			t.Errorf("defaultErrorCode(%d) = %q, want %q", tt.statusCode, got, tt.want)
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("a", "b"); got != "a" {
		t.Errorf("got %q, want 'a'", got)
	}
	if got := firstNonEmpty("", "b"); got != "b" {
		t.Errorf("got %q, want 'b'", got)
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Errorf("got %q, want ''", got)
	}
}

func TestNonEmpty(t *testing.T) {
	if got := nonEmpty("a", "fallback"); got != "a" {
		t.Errorf("got %q, want 'a'", got)
	}
	if got := nonEmpty("", "fallback"); got != "fallback" {
		t.Errorf("got %q, want 'fallback'", got)
	}
}

func TestNewTrafficHistoryBufferDefault(t *testing.T) {
	// limit <= 0 时使用默认值
	buf := newTrafficHistoryBuffer(0)
	if buf == nil {
		t.Fatal("expected non-nil buffer")
	}
	if buf.limit != defaultTrafficHistoryLimit {
		t.Errorf("limit = %d, want %d", buf.limit, defaultTrafficHistoryLimit)
	}
}

func TestNewTrafficHistoryBufferNegative(t *testing.T) {
	buf := newTrafficHistoryBuffer(-1)
	if buf.limit != defaultTrafficHistoryLimit {
		t.Errorf("limit = %d, want %d", buf.limit, defaultTrafficHistoryLimit)
	}
}

func TestTrafficHistoryBufferFullSnapshot(t *testing.T) {
	buf := newTrafficHistoryBuffer(3)
	buf.add(TrafficHistoryPoint{Timestamp: time.Now(), UploadBytes: 1})
	buf.add(TrafficHistoryPoint{Timestamp: time.Now(), UploadBytes: 2})
	buf.add(TrafficHistoryPoint{Timestamp: time.Now(), UploadBytes: 3})
	buf.add(TrafficHistoryPoint{Timestamp: time.Now(), UploadBytes: 4})

	snap := buf.snapshot()
	if len(snap) != 3 {
		t.Fatalf("expected 3 points, got %d", len(snap))
	}
	// 环形缓冲：最早的数据被覆盖
	if snap[0].UploadBytes != 2 {
		t.Errorf("first point upload = %d, want 2", snap[0].UploadBytes)
	}
}

func TestClientIPEdgeCases(t *testing.T) {
	// 无端口
	got := clientIP("1.2.3.4")
	if got != "1.2.3.4" {
		t.Errorf("got %q, want '1.2.3.4'", got)
	}
	// 正常 host:port
	got = clientIP("1.2.3.4:8080")
	if got != "1.2.3.4" {
		t.Errorf("got %q, want '1.2.3.4'", got)
	}
}

func TestLoginRateLimiterCleanup(t *testing.T) {
	l := newLoginRateLimiter()
	old := time.Now().Add(-(loginEntryTTL + time.Minute))
	recent := time.Now()

	// 添加旧条目（已过期）和近期条目
	for i := 0; i < loginCleanupThreshold+10; i++ {
		key := "ip-" + string(rune('a'+i%26)) + string(rune('a'+i/26%26)) + string(rune('a'+i/26/26%26))
		// 前 loginCleanupThreshold 个用旧时间，触发 cleanup
		if i < loginCleanupThreshold {
			l.recordFailure(key, old)
		} else {
			l.recordFailure(key, recent)
		}
	}

	l.mu.Lock()
	count := len(l.entries)
	l.mu.Unlock()

	// 清理后旧条目应被移除
	if count >= loginCleanupThreshold {
		t.Errorf("entries count = %d, should be < %d after cleanup", count, loginCleanupThreshold)
	}
}
