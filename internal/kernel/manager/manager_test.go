package manager_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/kernel/fake"
	"github.com/xuthus5/boxd/internal/kernel/manager"
	"github.com/xuthus5/boxd/internal/kernel/memstore"
	"github.com/xuthus5/boxd/kernel"
)

func TestManagerLifecycle(t *testing.T) {
	ctx := context.Background()
	store := storeWithRevision(t, "rev-1")
	factory := fake.NewFactory()
	subject := newManager(t, factory, store)

	if got := subject.Status().State; got != kernel.StateStopped {
		t.Fatalf("initial state = %s, want stopped", got)
	}
	if err := subject.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	status := subject.Status()
	if status.State != kernel.StateRunning || status.Version != "fake-1" || status.Revision != "rev-1" {
		t.Fatalf("running status = %+v", status)
	}
	if len(factory.Opened()) != 1 {
		t.Fatalf("opened engines = %d, want 1", len(factory.Opened()))
	}
	if err := subject.Start(ctx); err != nil {
		t.Fatalf("second Start() error = %v", err)
	}
	if len(factory.Opened()) != 1 {
		t.Fatalf("idempotent Start opened another engine")
	}

	if err := subject.Restart(ctx); err != nil {
		t.Fatalf("Restart() error = %v", err)
	}
	if len(factory.Opened()) != 2 {
		t.Fatalf("Restart opened %d engines, want 2", len(factory.Opened()))
	}
	if !factory.Opened()[0].Closed() {
		t.Fatal("old engine was not closed on restart")
	}

	if err := subject.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if got := subject.Status().State; got != kernel.StateStopped {
		t.Fatalf("state after Stop = %s", got)
	}
	if !factory.Opened()[1].Closed() {
		t.Fatal("engine was not closed on stop")
	}
	if err := subject.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := subject.Start(ctx); err == nil {
		t.Fatal("Start after Close succeeded")
	}
}

func TestManagerApplyStoppedPersistsWithoutStarting(t *testing.T) {
	ctx := context.Background()
	store := storeWithRevision(t, "rev-1")
	factory := fake.NewFactory()
	subject := newManager(t, factory, store)

	result, err := subject.ApplyConfig(ctx, document("rev-2"), kernel.ApplyOptions{ExpectedRevision: "rev-1"})
	if err != nil {
		t.Fatalf("ApplyConfig() error = %v", err)
	}
	if result.Revision != "rev-2" {
		t.Fatalf("result revision = %q", result.Revision)
	}
	if got := subject.Status().State; got != kernel.StateStopped {
		t.Fatalf("state = %s, want stopped", got)
	}
	current, err := store.Load(ctx)
	if err != nil || current.Revision != "rev-2" {
		t.Fatalf("store current = %+v, err = %v", current, err)
	}
	opened := factory.Opened()
	if len(opened) != 1 || !opened[0].Closed() {
		t.Fatalf("validation engine should be opened and closed once, got %d", len(opened))
	}
}

func TestManagerApplyRejectsRevisionConflict(t *testing.T) {
	ctx := context.Background()
	store := storeWithRevision(t, "rev-1")
	subject := newManager(t, fake.NewFactory(), store)

	_, err := subject.ApplyConfig(ctx, document("rev-2"), kernel.ApplyOptions{ExpectedRevision: "stale"})
	expectCode(t, err, kernel.ErrCodeConflict)
	current, loadErr := store.Load(ctx)
	if loadErr != nil || current.Revision != "rev-1" {
		t.Fatalf("store changed on conflict: %+v, %v", current, loadErr)
	}
}

func TestManagerApplyRunningRollsBackOnFailure(t *testing.T) {
	ctx := context.Background()
	store := storeWithRevision(t, "rev-1")
	factory := fake.NewFactory()
	opens := 0
	factory.NewEngine = func(options kernel.OpenOptions) (*fake.Engine, error) {
		opens++
		engine := fake.New(options.Config)
		if opens == 1 {
			engine.ApplyErr = errors.New("driver rejected live apply")
		}
		return engine, nil
	}
	subject := newManager(t, factory, store)
	if err := subject.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	_, err := subject.ApplyConfig(ctx, document("rev-2"), kernel.ApplyOptions{ExpectedRevision: "rev-1"})
	expectCode(t, err, kernel.ErrCodeApplyFailed)

	current, loadErr := store.Load(ctx)
	if loadErr != nil || current.Revision != "rev-1" {
		t.Fatalf("store after rollback = %+v, err = %v", current, loadErr)
	}
	status := subject.Status()
	if status.State != kernel.StateRunning || status.Revision != "rev-1" {
		t.Fatalf("status after rollback = %+v", status)
	}
	opened := factory.Opened()
	if len(opened) != 2 {
		t.Fatalf("opened engines = %d, want 2", len(opened))
	}
	if !opened[0].Closed() {
		t.Fatal("broken engine was not closed")
	}
	if opened[1].Status().State != kernel.StateRunning {
		t.Fatal("previous configuration was not restarted")
	}
}

func TestManagerApplyRunningUpdatesRevisionAndPublishesEvent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := storeWithRevision(t, "rev-1")
	subject := newManager(t, fake.NewFactory(), store)
	if err := subject.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	events, err := subject.Subscribe(ctx, kernel.EventConfigApplied)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	result, err := subject.ApplyConfig(ctx, document("rev-2"), kernel.ApplyOptions{ExpectedRevision: "rev-1", Source: "test"})
	if err != nil {
		t.Fatalf("ApplyConfig() error = %v", err)
	}
	if result.Revision != "rev-2" {
		t.Fatalf("result revision = %q", result.Revision)
	}
	if status := subject.Status(); status.Revision != "rev-2" || status.State != kernel.StateRunning {
		t.Fatalf("status = %+v", status)
	}
	current, loadErr := store.Load(ctx)
	if loadErr != nil || current.Revision != "rev-2" {
		t.Fatalf("store current = %+v, err = %v", current, loadErr)
	}
	select {
	case event := <-events:
		payload, ok := event.Payload.(kernel.ConfigApplied)
		if !ok || payload.Revision != "rev-2" || payload.PreviousRevision != "rev-1" {
			t.Fatalf("event payload = %+v", event.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("config.applied event was not published")
	}
}

func TestManagerApplyRejectsInvalidConfiguration(t *testing.T) {
	ctx := context.Background()
	store := storeWithRevision(t, "rev-1")
	factory := fake.NewFactory()
	factory.NewEngine = func(options kernel.OpenOptions) (*fake.Engine, error) {
		engine := fake.New(options.Config)
		engine.Validation = kernel.ValidationResult{
			Valid:       false,
			Diagnostics: []kernel.Diagnostic{{Severity: "error", Message: "invalid inbound"}},
		}
		return engine, nil
	}
	subject := newManager(t, factory, store)

	_, err := subject.ApplyConfig(ctx, document("rev-2"), kernel.ApplyOptions{ExpectedRevision: "rev-1"})
	expectCode(t, err, kernel.ErrCodeInvalidConfig)
	current, loadErr := store.Load(ctx)
	if loadErr != nil || current.Revision != "rev-1" {
		t.Fatalf("store changed on invalid config: %+v, %v", current, loadErr)
	}
}

func TestManagerRuntimeRequiresRunningState(t *testing.T) {
	ctx := context.Background()
	subject := newManager(t, fake.NewFactory(), storeWithRevision(t, "rev-1"))
	_, err := subject.OutboundGroups(ctx)
	expectCode(t, err, kernel.ErrCodeNotRunning)
}

func TestManagerDelegatesRuntimeAndObserver(t *testing.T) {
	ctx := context.Background()
	store := storeWithRevision(t, "rev-1")
	factory := fake.NewFactory()
	factory.NewEngine = func(options kernel.OpenOptions) (*fake.Engine, error) {
		engine := fake.New(options.Config)
		engine.Descriptor.Capabilities = kernel.Capabilities{HotReload: true}
		engine.Groups = []kernel.OutboundGroup{{Tag: "proxy"}}
		engine.Mode = kernel.ClashMode{Current: "rule", Available: []string{"rule", "direct"}}
		engine.Connections = []kernel.Connection{{ID: 7, Target: "example"}}
		engine.Traffic = kernel.TrafficSnapshot{UploadTotal: 10, DownloadTotal: 20, At: time.Now().UTC()}
		return engine, nil
	}
	subject := newManager(t, factory, store)
	if err := subject.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if subject.Version() != "fake-1" || !subject.Capabilities().HotReload {
		t.Fatalf("engine info = %s, %+v", subject.Version(), subject.Capabilities())
	}
	if groups, err := subject.OutboundGroups(ctx); err != nil || len(groups) != 1 {
		t.Fatalf("OutboundGroups() = %+v, %v", groups, err)
	}
	if err := subject.SelectOutbound(ctx, "proxy", "node"); err != nil {
		t.Fatalf("SelectOutbound() error = %v", err)
	}
	if _, err := subject.OutboundDelay(ctx, "node", "probe"); err != nil {
		t.Fatalf("OutboundDelay() error = %v", err)
	}
	if mode, err := subject.ClashMode(ctx); err != nil || mode.Current != "rule" {
		t.Fatalf("ClashMode() = %+v, %v", mode, err)
	}
	if err := subject.SetClashMode(ctx, "direct"); err != nil {
		t.Fatalf("SetClashMode() error = %v", err)
	}
	if err := subject.FlushDNS(ctx); err != nil {
		t.Fatalf("FlushDNS() error = %v", err)
	}
	if connections, err := subject.ListConnections(ctx); err != nil || len(connections) != 1 {
		t.Fatalf("ListConnections() = %+v, %v", connections, err)
	}
	if err := subject.CloseConnection(ctx, 7); err != nil {
		t.Fatalf("CloseConnection() error = %v", err)
	}
	if _, err := subject.CloseConnections(ctx, kernel.ConnectionFilter{Process: "test"}); err != nil {
		t.Fatalf("CloseConnections() error = %v", err)
	}
	if len(subject.RecentLogs(10)) != 0 {
		t.Fatal("unexpected recent logs")
	}
	if subject.TrafficSnapshot().UploadTotal != 10 {
		t.Fatal("traffic snapshot was not delegated")
	}
	if _, err := subject.ValidateConfig(ctx, document("rev-1")); err != nil {
		t.Fatalf("ValidateConfig() error = %v", err)
	}
	result, err := subject.ApplyConfig(ctx, kernel.ConfigDocument{Format: "json", Body: []byte(`{}`)}, kernel.ApplyOptions{})
	if err != nil || result.Revision == "" {
		t.Fatalf("ApplyConfig() with generated revision = %+v, %v", result, err)
	}
	if err := subject.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := subject.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestManagerStartOpenFailure(t *testing.T) {
	ctx := context.Background()
	factory := fake.NewFactory()
	factory.OpenErr = errors.New("open failed")
	subject := newManager(t, factory, storeWithRevision(t, "rev-1"))
	err := subject.Start(ctx)
	expectCode(t, err, kernel.ErrCodeInternal)
	if subject.Status().State != kernel.StateFailed {
		t.Fatalf("state = %s, want failed", subject.Status().State)
	}
}

func TestManagerStartEngineFailureClosesEngine(t *testing.T) {
	ctx := context.Background()
	factory := fake.NewFactory()
	factory.NewEngine = func(options kernel.OpenOptions) (*fake.Engine, error) {
		engine := fake.New(options.Config)
		engine.StartErr = errors.New("start failed")
		return engine, nil
	}
	subject := newManager(t, factory, storeWithRevision(t, "rev-1"))
	err := subject.Start(ctx)
	expectCode(t, err, kernel.ErrCodeInternal)
	if len(factory.Opened()) != 1 || !factory.Opened()[0].Closed() {
		t.Fatal("failed engine was not closed")
	}
}

func TestManagerStopFailureMarksFailed(t *testing.T) {
	ctx := context.Background()
	factory := fake.NewFactory()
	factory.NewEngine = func(options kernel.OpenOptions) (*fake.Engine, error) {
		engine := fake.New(options.Config)
		engine.StopErr = errors.New("stop failed")
		return engine, nil
	}
	subject := newManager(t, factory, storeWithRevision(t, "rev-1"))
	if err := subject.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	err := subject.Stop(ctx)
	expectCode(t, err, kernel.ErrCodeInternal)
	if subject.Status().State != kernel.StateFailed {
		t.Fatalf("state = %s, want failed", subject.Status().State)
	}
	if !factory.Opened()[0].Closed() {
		t.Fatal("engine was not closed after stop failure")
	}
}

func TestManagerFirstApplyFailureRestoresEmptyState(t *testing.T) {
	ctx := context.Background()
	store := &memstore.ConfigStore{}
	factory := fake.NewFactory()
	factory.NewEngine = func(options kernel.OpenOptions) (*fake.Engine, error) {
		engine := fake.New(options.Config)
		engine.ApplyErr = errors.New("apply failed")
		return engine, nil
	}
	subject := newManager(t, factory, store)
	if err := subject.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	_, err := subject.ApplyConfig(ctx, document("rev-1"), kernel.ApplyOptions{})
	expectCode(t, err, kernel.ErrCodeApplyFailed)
	if _, loadErr := store.Load(ctx); !errors.Is(loadErr, kernel.ErrConfigNotFound) {
		t.Fatalf("store after rollback error = %v, want not found", loadErr)
	}
	if subject.Status().State != kernel.StateStopped {
		t.Fatalf("state = %s, want stopped", subject.Status().State)
	}
}

func newManager(t *testing.T, factory kernel.Factory, store kernel.ConfigStore) *manager.Manager {
	t.Helper()
	subject, err := manager.New(context.Background(), manager.Options{
		Factory: factory,
		Store:   store,
		Open:    kernel.OpenOptions{Name: "test", WorkDir: t.TempDir()},
	})
	if err != nil {
		t.Fatalf("manager.New() error = %v", err)
	}
	t.Cleanup(func() { _ = subject.Close() })
	return subject
}

func storeWithRevision(t *testing.T, revision string) *memstore.ConfigStore {
	t.Helper()
	store := &memstore.ConfigStore{}
	if err := store.SaveAtomic(context.Background(), document(revision)); err != nil {
		t.Fatalf("seed store: %v", err)
	}
	return store
}

func document(revision string) kernel.ConfigDocument {
	return kernel.ConfigDocument{Format: "json", Body: []byte(`{}`), Revision: revision}
}

func expectCode(t *testing.T, err error, code kernel.ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want code %s", code)
	}
	var kernelError *kernel.Error
	if !errors.As(err, &kernelError) {
		t.Fatalf("error = %T %v, want *kernel.Error", err, err)
	}
	if kernelError.Code != code {
		t.Fatalf("error code = %s, want %s (%v)", kernelError.Code, code, err)
	}
}
