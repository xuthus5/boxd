package controlplane_test

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/internal/controlplane"
	"github.com/xuthus5/boxd/internal/kernel/fake"
	"github.com/xuthus5/boxd/internal/kernel/manager"
	"github.com/xuthus5/boxd/internal/kernel/memstore"
	"github.com/xuthus5/boxd/kernel"
)

func TestApplicationFacadeAndSettings(t *testing.T) {
	ctx := context.Background()
	store := &memstore.ConfigStore{}
	kernelManager, err := manager.New(ctx, manager.Options{Factory: fake.NewFactory(), Store: store})
	if err != nil {
		t.Fatalf("manager.New() error = %v", err)
	}
	t.Cleanup(func() { _ = kernelManager.Close() })
	settingsRepository := newMemorySettings()
	profileService := controlplane.NewProfileService(kernelManager, newMemoryProfiles(), staticCompiler{})
	application := controlplane.New(kernelManager, store, settingsRepository, profileService)

	if application.Kernel() == nil || application.Config() == nil || application.Settings() == nil ||
		application.Setup() == nil || application.Inbounds() == nil || application.Egress() == nil ||
		application.Routes() == nil || application.DNS() == nil || application.Observability() == nil {
		t.Fatal("application facade returned a nil module")
	}
	if _, err := application.Setup().EnsureDefaults(ctx); err != nil {
		t.Fatalf("EnsureDefaults() error = %v", err)
	}
	if err := application.Kernel().Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if status, err := application.Kernel().Status(ctx); err != nil || status.State != kernel.StateRunning {
		t.Fatalf("Status() = %+v, %v", status, err)
	}
	if err := application.Kernel().Restart(ctx); err != nil {
		t.Fatalf("Restart() error = %v", err)
	}
	if err := application.Kernel().Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if _, err := application.Config().Validate(ctx, kernel.ConfigDocument{Format: "json", Body: []byte(`{}`)}); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	history, err := application.Config().History(ctx, 5)
	if err != nil || len(history) != 1 || history[0].Revision != "profile-rev" {
		t.Fatalf("History() = %+v, %v", history, err)
	}

	setting, err := application.Settings().Set(ctx, "  theme  ", "green")
	if err != nil || setting.Key != "theme" || setting.UpdatedAt.IsZero() {
		t.Fatalf("Set() = %+v, %v", setting, err)
	}
	got, found, err := application.Settings().Get(ctx, "theme")
	if err != nil || !found || got.Value != "green" {
		t.Fatalf("Get() = %+v, %v, %v", got, found, err)
	}
	all, err := application.Settings().List(ctx)
	if err != nil || len(all) != 1 {
		t.Fatalf("List() = %+v, %v", all, err)
	}
	if _, err := application.Settings().Set(ctx, " ", "value"); err == nil {
		t.Fatal("Set() accepted an empty key")
	} else {
		var kernelError *kernel.Error
		if !errors.As(err, &kernelError) || kernelError.Code != kernel.ErrCodeInvalidConfig {
			t.Fatalf("empty key error = %v", err)
		}
	}
}

type staticCompiler struct{}

func (staticCompiler) Compile(context.Context, control.DesiredState) (kernel.ConfigDocument, error) {
	return kernel.ConfigDocument{Format: "json", Body: []byte(`{}`), Revision: "profile-rev"}, nil
}

type memoryProfiles struct {
	mu    sync.Mutex
	state control.DesiredState
	set   bool
}

func newMemoryProfiles() *memoryProfiles { return &memoryProfiles{} }

func (r *memoryProfiles) Load(context.Context) (control.DesiredState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.set {
		return control.DesiredState{}, kernel.ErrConfigNotFound
	}
	return r.state, nil
}

func (r *memoryProfiles) Save(_ context.Context, state control.DesiredState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state = state
	r.set = true
	return nil
}

type memorySettings struct {
	mu       sync.RWMutex
	settings map[string]control.Setting
}

func newMemorySettings() *memorySettings {
	return &memorySettings{settings: make(map[string]control.Setting)}
}

func (r *memorySettings) List(context.Context) ([]control.Setting, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]string, 0, len(r.settings))
	for key := range r.settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]control.Setting, 0, len(keys))
	for _, key := range keys {
		result = append(result, r.settings[key])
	}
	return result, nil
}

func (r *memorySettings) Get(_ context.Context, key string) (control.Setting, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	setting, found := r.settings[key]
	return setting, found, nil
}

func (r *memorySettings) Upsert(_ context.Context, setting control.Setting) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if setting.UpdatedAt.IsZero() {
		setting.UpdatedAt = time.Now().UTC()
	}
	r.settings[setting.Key] = setting
	return nil
}
