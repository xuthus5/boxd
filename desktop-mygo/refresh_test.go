package main

import (
	"testing"
	"time"
)

func TestRefreshLifecycleStopsBackgroundLoop(t *testing.T) {
	fake := &fakeBackend{snapshot: seededSnapshot(themeLight)}
	app := newApplication(fake, testConfig(), nil, false)
	app.startRefresh()
	deadline := time.Now().Add(2 * time.Second)
	for fake.callCount("snapshot") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if fake.callCount("snapshot") == 0 {
		t.Fatal("refresh loop did not request a snapshot")
	}
	app.stopRefresh()
	app.stopRefresh()
	select {
	case <-app.refreshDone:
	default:
		t.Fatal("refresh loop did not stop after cancellation")
	}
}
