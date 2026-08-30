package main

import (
	"os"
	"testing"
	"time"
)

func newTestRecovery(now func() time.Time) *webProcessRecovery {
	r := newWebProcessRecovery(nil)
	r.now = now
	return r
}

func TestWebProcessRecoveryReloads(t *testing.T) {
	current := time.Unix(0, 0)
	r := newTestRecovery(func() time.Time { return current })
	reloads := 0
	r.setReloadFn(func() { reloads++ })

	if !r.onTerminated() {
		t.Fatal("first terminate should reload")
	}
	if reloads != 1 {
		t.Fatalf("reloads = %d, want 1", reloads)
	}
}

func TestWebProcessRecoveryCooldown(t *testing.T) {
	current := time.Unix(0, 0)
	r := newTestRecovery(func() time.Time { return current })
	reloads := 0
	r.setReloadFn(func() { reloads++ })

	if !r.onTerminated() {
		t.Fatal("first terminate should reload")
	}
	// 冷却期内第二次终止不触发重载。
	if r.onTerminated() {
		t.Fatal("reload within cooldown should be suppressed")
	}
	if reloads != 1 {
		t.Fatalf("reloads = %d, want 1", reloads)
	}
	// 冷却期过后恢复触发。
	current = current.Add(defaultReloadCooldown + time.Second)
	if !r.onTerminated() {
		t.Fatal("terminate after cooldown should reload")
	}
	if reloads != 2 {
		t.Fatalf("reloads = %d, want 2", reloads)
	}
}

func TestWebProcessRecoveryRateLimit(t *testing.T) {
	current := time.Unix(0, 0)
	r := newTestRecovery(func() time.Time { return current })
	reloads := 0
	r.setReloadFn(func() { reloads++ })

	// 达到窗口内上限后停止自动重载。
	for i := 0; i < defaultReloadRateLimit; i++ {
		if !r.onTerminated() {
			t.Fatalf("reload %d should be allowed", i+1)
		}
		current = current.Add(defaultReloadCooldown + time.Second)
	}
	if r.onTerminated() {
		t.Fatal("reload beyond rate limit should be suspended")
	}
	if reloads != defaultReloadRateLimit {
		t.Fatalf("reloads = %d, want %d", reloads, defaultReloadRateLimit)
	}

	// 窗口滑出后恢复。
	current = current.Add(defaultRateWindow)
	if !r.onTerminated() {
		t.Fatal("terminate after rate window should reload")
	}
}

func TestWebProcessRecoveryNilReloadFn(t *testing.T) {
	r := newTestRecovery(func() time.Time { return time.Unix(0, 0) })
	if r.onTerminated() {
		t.Fatal("without reload fn should not report reload")
	}
}

func TestWebProcessRecoverySetReloadFn(t *testing.T) {
	r := newWebProcessRecovery(nil)
	called := false
	r.setReloadFn(func() { called = true })
	r.reloadFn()
	if !called {
		t.Fatal("reload fn should be settable")
	}
}

func TestEnableWebKitWorkarounds(t *testing.T) {
	const key = "WEBKIT_DISABLE_DMABUF_RENDERER"
	old, had := os.LookupEnv(key)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})

	// 未设置时默认开启。
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	enableWebKitWorkarounds()
	if got := os.Getenv(key); got != "1" {
		t.Fatalf("%s = %q, want 1", key, got)
	}

	// 已显式设置时不覆盖用户环境。
	if err := os.Setenv(key, "0"); err != nil {
		t.Fatal(err)
	}
	enableWebKitWorkarounds()
	if got := os.Getenv(key); got != "0" {
		t.Fatalf("%s = %q, want 0", key, got)
	}
}
