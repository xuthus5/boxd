package api

import (
	"net"
	"sync"
	"time"
)

const (
	maxLoginFailures  = 5
	loginLockDuration = time.Minute
	// loginEntryTTL 控制已过期的失败条目存活时间，超过后清理。
	loginEntryTTL = 10 * time.Minute
	// loginCleanupThreshold 触发清理扫描的条目数量阈值。
	loginCleanupThreshold = 500
)

type loginRateLimiter struct {
	mu      sync.Mutex
	entries map[string]loginRateLimitEntry
}

type loginRateLimitEntry struct {
	failures    int
	lockedUntil time.Time
	// lastSeen 记录最后一次失败时间，用于过期清理。
	lastSeen time.Time
}

func newLoginRateLimiter() *loginRateLimiter {
	return &loginRateLimiter{
		entries: make(map[string]loginRateLimitEntry),
	}
}

func (l *loginRateLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[key]
	if !ok {
		return true
	}
	if entry.lockedUntil.After(now) {
		return false
	}
	if !entry.lockedUntil.IsZero() && !entry.lockedUntil.After(now) {
		delete(l.entries, key)
	}
	return true
}

func (l *loginRateLimiter) recordFailure(key string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 定期清理过期条目，防止内存无限增长
	if len(l.entries) > loginCleanupThreshold {
		l.cleanupLocked(now)
	}

	entry := l.entries[key]
	entry.failures++
	entry.lastSeen = now
	if entry.failures >= maxLoginFailures {
		entry.failures = 0
		entry.lockedUntil = now.Add(loginLockDuration)
	}
	l.entries[key] = entry
}

func (l *loginRateLimiter) recordSuccess(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

// cleanupLocked 清理已超过 TTL 的条目，调用方需持有 mu 锁。
func (l *loginRateLimiter) cleanupLocked(now time.Time) {
	cutoff := now.Add(-loginEntryTTL)
	for k, e := range l.entries {
		if e.lastSeen.Before(cutoff) && !e.lockedUntil.After(now) {
			delete(l.entries, k)
		}
	}
}

func clientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil || host == "" {
		return remoteAddr
	}
	return host
}
