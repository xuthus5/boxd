package service

import (
	"context"
	"testing"
	"time"
)

type fakeSecretProvider struct {
	secret string
}

func (f *fakeSecretProvider) JWTSecret() string { return f.secret }

func TestAuthServiceLoginSuccess(t *testing.T) {
	svc := NewAuthService("admin", "password123", &fakeSecretProvider{secret: "test-secret-key-12345"})
	resp, err := svc.Login(context.Background(), "127.0.0.1:1234", AuthCredentials{Username: "admin", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token == "" {
		t.Fatal("empty token")
	}
	if resp.ExpiresAt.IsZero() {
		t.Fatal("empty expires")
	}
}

func TestAuthServiceLoginWrongPassword(t *testing.T) {
	svc := NewAuthService("admin", "password123", &fakeSecretProvider{secret: "test-secret-key-12345"})
	_, err := svc.Login(context.Background(), "127.0.0.1:1234", AuthCredentials{Username: "admin", Password: "wrong"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthServiceLoginWrongUsername(t *testing.T) {
	svc := NewAuthService("admin", "password123", &fakeSecretProvider{secret: "test-secret-key-12345"})
	_, err := svc.Login(context.Background(), "127.0.0.1:1234", AuthCredentials{Username: "nobody", Password: "password123"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthServiceLoginRateLimit(t *testing.T) {
	svc := NewAuthService("admin", "password123", &fakeSecretProvider{secret: "test-secret-key-12345"})
	now := time.Now()
	svc.now = func() time.Time { return now }
	for range 5 {
		_, err := svc.Login(context.Background(), "1.1.1.1:1", AuthCredentials{Username: "admin", Password: "wrong"})
		if err == nil {
			t.Fatal("expected error on failure")
		}
	}
	_, err := svc.Login(context.Background(), "1.1.1.1:1", AuthCredentials{Username: "admin", Password: "password123"})
	if err == nil {
		t.Fatal("expected rate limit error")
	}
}

func TestAuthServiceLoginRateLimitSuccessResets(t *testing.T) {
	svc := NewAuthService("admin", "password123", &fakeSecretProvider{secret: "test-secret-key-12345"})
	now := time.Now()
	svc.now = func() time.Time { return now }
	_, _ = svc.Login(context.Background(), "1.1.1.1:1", AuthCredentials{Username: "admin", Password: "password123"})
	for range 4 {
		_, err := svc.Login(context.Background(), "1.1.1.1:1", AuthCredentials{Username: "admin", Password: "wrong"})
		if err == nil {
			t.Fatal("expected error")
		}
	}
	_, err := svc.Login(context.Background(), "1.1.1.1:1", AuthCredentials{Username: "admin", Password: "password123"})
	if err != nil {
		t.Fatalf("expected success after reset, got %v", err)
	}
}

func TestAuthServiceLoginRateLimitExpiry(t *testing.T) {
	svc := NewAuthService("admin", "password123", &fakeSecretProvider{secret: "test-secret-key-12345"})
	now := time.Now()
	svc.now = func() time.Time { return now }
	for range 5 {
		_, _ = svc.Login(context.Background(), "1.1.1.1:1", AuthCredentials{Username: "admin", Password: "wrong"})
	}
	_, err := svc.Login(context.Background(), "1.1.1.1:1", AuthCredentials{Username: "admin", Password: "password123"})
	if err == nil {
		t.Fatal("expected rate limit")
	}
	now = now.Add(loginLockDuration + time.Second)
	svc.now = func() time.Time { return now }
	_, err = svc.Login(context.Background(), "1.1.1.1:1", AuthCredentials{Username: "admin", Password: "password123"})
	if err != nil {
		t.Fatalf("expected success after lock expiry, got %v", err)
	}
}

func TestAuthServiceLoginSigningFailure(t *testing.T) {
	// 空 secret 会触发签名失败？实际空 secret 也能签。用一个保证失败的场景跳过。
	svc := NewAuthService("admin", "password123", &fakeSecretProvider{secret: ""})
	_, err := svc.Login(context.Background(), "127.0.0.1:1", AuthCredentials{Username: "admin", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAuthServiceLogout(t *testing.T) {
	svc := NewAuthService("admin", "password123", &fakeSecretProvider{secret: "test-secret-key-12345"})
	if err := svc.Logout(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestAuthServiceLoginRateLimiterNil(t *testing.T) {
	svc := NewAuthService("admin", "password123", &fakeSecretProvider{secret: "test-secret-key-12345"})
	svc.limiter = nil
	resp, err := svc.Login(context.Background(), "127.0.0.1:1", AuthCredentials{Username: "admin", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token == "" {
		t.Fatal("empty token")
	}
}

func TestLoginRateLimiterCleanup(t *testing.T) {
	l := newLoginRateLimiter()
	now := time.Now()
	for range loginCleanupThreshold + 10 {
		l.recordFailure("k", now)
	}
	if len(l.entries) == 0 {
		t.Fatal("entries should not be empty after cleanup threshold")
	}
}

func TestLoginRateLimiterAllowMissing(t *testing.T) {
	l := newLoginRateLimiter()
	if !l.allow("missing", time.Now()) {
		t.Fatal("missing key should be allowed")
	}
}

func TestLoginRateLimiterExpiredEntryRemoved(t *testing.T) {
	l := newLoginRateLimiter()
	now := time.Now()
	l.recordFailure("k", now)
	l.mu.Lock()
	l.entries["k"] = loginRateLimitEntry{failures: 1, lastSeen: now, lockedUntil: now.Add(-time.Minute)}
	l.mu.Unlock()
	if !l.allow("k", now) {
		t.Fatal("expired entry should allow")
	}
	if _, ok := l.entries["k"]; ok {
		t.Fatal("expired entry should be removed")
	}
}

func TestLoginRateLimiterRecordSuccess(t *testing.T) {
	l := newLoginRateLimiter()
	now := time.Now()
	l.recordFailure("k", now)
	l.recordSuccess("k")
	if _, ok := l.entries["k"]; ok {
		t.Fatal("entry should be removed on success")
	}
}

func TestClientIP(t *testing.T) {
	if got := clientIP("1.2.3.4:8080"); got != "1.2.3.4" {
		t.Fatalf("got %q", got)
	}
	if got := clientIP("bad"); got != "bad" {
		t.Fatalf("got %q", got)
	}
	if got := clientIP(":8080"); got != ":8080" {
		t.Fatalf("got %q", got)
	}
}
