package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type blockingRestartable struct {
	entered chan int
	release chan struct{}
	mu      sync.Mutex
	calls   int
	active  int
	max     int
}

func newBlockingRestartable() *blockingRestartable {
	return &blockingRestartable{
		entered: make(chan int, 2),
		release: make(chan struct{}),
	}
}

func (instance *blockingRestartable) Restart() error {
	instance.mu.Lock()
	instance.calls++
	call := instance.calls
	instance.active++
	if instance.active > instance.max {
		instance.max = instance.active
	}
	instance.mu.Unlock()

	instance.entered <- call
	<-instance.release

	instance.mu.Lock()
	instance.active--
	instance.mu.Unlock()
	return nil
}

func (instance *blockingRestartable) maxConcurrent() int {
	instance.mu.Lock()
	defer instance.mu.Unlock()
	return instance.max
}

func updateConfigAsync(handler *ConfigHandler, body string) <-chan *httptest.ResponseRecorder {
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		recorder := httptest.NewRecorder()
		handler.UpdateConfig(recorder, jsonRequest(http.MethodPut, "/api/config", body))
		done <- recorder
	}()
	return done
}

func waitForConfigUpdate(t *testing.T, done <-chan *httptest.ResponseRecorder) *httptest.ResponseRecorder {
	t.Helper()
	select {
	case recorder := <-done:
		return recorder
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for config update")
		return nil
	}
}

func TestUpdateConfigSerializesApplyLifecycle(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, configPath, map[string]any{"outbounds": []any{}})
	instance := newBlockingRestartable()
	handler := NewConfigHandler(configPath, instance, nil, nil, nil, nil)
	release := sync.OnceFunc(func() { close(instance.release) })
	t.Cleanup(release)

	firstDone := updateConfigAsync(handler, `{"outbounds":[{"type":"direct","tag":"first"}]}`)
	select {
	case <-instance.entered:
	case <-time.After(time.Second):
		t.Fatal("first restart did not start")
	}

	secondDone := updateConfigAsync(handler, `{"outbounds":[{"type":"direct","tag":"second"}]}`)
	select {
	case call := <-instance.entered:
		release()
		t.Fatalf("restart call %d entered before the first apply completed", call)
	case <-time.After(100 * time.Millisecond):
	}

	release()
	select {
	case <-instance.entered:
	case <-time.After(time.Second):
		t.Fatal("second restart did not start after releasing the first apply")
	}

	first := waitForConfigUpdate(t, firstDone)
	second := waitForConfigUpdate(t, secondDone)
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("update statuses = %d, %d; want 200, 200", first.Code, second.Code)
	}
	if got := instance.maxConcurrent(); got != 1 {
		t.Fatalf("maximum concurrent restarts = %d, want 1", got)
	}
}
