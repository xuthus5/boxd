package controlplane

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

// ProfileService serializes desired-state mutations and keeps SQLite, the
// persisted kernel configuration, and the running engine on one code path.
type ProfileService struct {
	mu       sync.Mutex
	manager  kernel.Manager
	store    control.ProfileRepository
	compiler control.ConfigCompiler
	audit    control.AuditRecorder
	now      func() time.Time
	id       func(prefix string) (string, error)
}

// SetAuditor installs the optional audit recorder used for successful profile
// mutations.
func (s *ProfileService) SetAuditor(recorder control.AuditRecorder) {
	if s == nil {
		return
	}
	s.audit = recorder
}

// NewProfileService constructs the managed-state coordinator.
func NewProfileService(manager kernel.Manager, store control.ProfileRepository, compiler control.ConfigCompiler) *ProfileService {
	return &ProfileService{
		manager:  manager,
		store:    store,
		compiler: compiler,
		now:      time.Now,
		id:       newStableID,
	}
}

// Snapshot returns a detached copy of the current managed state.
func (s *ProfileService) Snapshot(ctx context.Context) (control.DesiredState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked(ctx)
}

// EnsureDefaults seeds the first-run profile and makes sure a kernel
// configuration exists even when an older database has only the profile row.
func (s *ProfileService) EnsureDefaults(ctx context.Context) (control.DesiredState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load(ctx)
	if errors.Is(err, kernel.ErrConfigNotFound) {
		state = control.DefaultDesiredState()
		if _, applyErr := s.persistAndApplyLocked(ctx, state, kernel.ApplyOptions{Source: "setup"}); applyErr != nil {
			return control.DesiredState{}, applyErr
		}
		recordAudit(ctx, s.audit, "profile.ensure_defaults", map[string]any{"revision": state.Revision})
		return cloneState(state)
	}
	if err != nil {
		return control.DesiredState{}, wrapStoreError("profile.ensure_defaults", "load desired state", err)
	}
	if _, readErr := s.manager.ReadConfig(ctx); errors.Is(readErr, kernel.ErrConfigNotFound) {
		if _, applyErr := s.applyLocked(ctx, state, kernel.ApplyOptions{Source: "setup"}); applyErr != nil {
			return control.DesiredState{}, applyErr
		}
		recordAudit(ctx, s.audit, "profile.ensure_defaults", map[string]any{"revision": state.Revision})
	} else if readErr != nil {
		return control.DesiredState{}, readErr
	}
	return cloneState(state)
}

// Reconcile compiles and applies the current desired state without changing it.
func (s *ProfileService) Reconcile(ctx context.Context, options kernel.ApplyOptions) (kernel.ApplyResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.loadLocked(ctx)
	if err != nil {
		return kernel.ApplyResult{}, err
	}
	if options.Source == "" {
		options.Source = "control"
	}
	result, err := s.applyLocked(ctx, state, options)
	if err == nil {
		recordAudit(ctx, s.audit, "profile.reconcile", map[string]any{"revision": result.Revision})
	}
	return result, err
}

// ApplyProfile is used by setup and tests to validate and persist a complete
// candidate profile. Product APIs should prefer targeted mutations.
func (s *ProfileService) ApplyProfile(ctx context.Context, candidate control.DesiredState) (control.DesiredState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := normalizeAndValidateState(&candidate); err != nil {
		return control.DesiredState{}, err
	}
	applied, err := s.persistAndApplyLocked(ctx, candidate, kernel.ApplyOptions{Source: "control"})
	if err == nil {
		recordAudit(ctx, s.audit, "profile.apply", map[string]any{"revision": applied.Revision})
	}
	return applied, err
}

func (s *ProfileService) update(ctx context.Context, action string, mutate func(*control.DesiredState) error) (control.DesiredState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	previous, err := s.loadLocked(ctx)
	if err != nil {
		return control.DesiredState{}, err
	}
	next, err := cloneState(previous)
	if err != nil {
		return control.DesiredState{}, kernel.WrapError(kernel.ErrCodeInternal, "profile.clone", "clone desired state", err)
	}
	if err := mutate(&next); err != nil {
		return control.DesiredState{}, err
	}
	if err := normalizeAndValidateState(&next); err != nil {
		return control.DesiredState{}, err
	}
	if err := s.saveLocked(ctx, &next); err != nil {
		return control.DesiredState{}, err
	}
	document, err := s.compiler.Compile(ctx, next)
	if err != nil {
		_ = s.store.Save(ctx, previous)
		return control.DesiredState{}, err
	}
	if _, err := s.manager.ApplyConfig(ctx, document, kernel.ApplyOptions{
		Source:           "control",
		RestartIfRunning: true,
	}); err != nil {
		restoreErr := s.store.Save(ctx, previous)
		if restoreErr != nil {
			return control.DesiredState{}, errors.Join(err, kernel.WrapError(
				kernel.ErrCodeApplyFailed,
				"profile.rollback",
				"restore desired state after failed config apply",
				restoreErr,
			))
		}
		return control.DesiredState{}, err
	}
	recordAudit(ctx, s.audit, action, map[string]any{
		"revision": next.Revision,
	})
	return cloneState(next)
}

func (s *ProfileService) persistAndApplyLocked(ctx context.Context, candidate control.DesiredState, options kernel.ApplyOptions) (control.DesiredState, error) {
	if err := normalizeAndValidateState(&candidate); err != nil {
		return control.DesiredState{}, err
	}
	document, err := s.compiler.Compile(ctx, candidate)
	if err != nil {
		return control.DesiredState{}, err
	}
	if err := s.saveLocked(ctx, &candidate); err != nil {
		return control.DesiredState{}, err
	}
	if options.Source == "" {
		options.Source = "control"
	}
	if _, err := s.manager.ApplyConfig(ctx, document, options); err != nil {
		return control.DesiredState{}, err
	}
	return cloneState(candidate)
}

func (s *ProfileService) applyLocked(ctx context.Context, state control.DesiredState, options kernel.ApplyOptions) (kernel.ApplyResult, error) {
	document, err := s.compiler.Compile(ctx, state)
	if err != nil {
		return kernel.ApplyResult{}, err
	}
	return s.manager.ApplyConfig(ctx, document, options)
}

func (s *ProfileService) loadLocked(ctx context.Context) (control.DesiredState, error) {
	state, err := s.store.Load(ctx)
	if errors.Is(err, kernel.ErrConfigNotFound) {
		return control.DefaultDesiredState(), nil
	}
	if err != nil {
		return control.DesiredState{}, wrapStoreError("profile.load", "load desired state", err)
	}
	return state, nil
}

func (s *ProfileService) saveLocked(ctx context.Context, state *control.DesiredState) error {
	state.UpdatedAt = s.now().UTC()
	state.Revision = profileRevision(*state)
	if err := s.store.Save(ctx, *state); err != nil {
		return wrapStoreError("profile.save", "save desired state", err)
	}
	return nil
}

func (s *ProfileService) nextID(prefix string) (string, error) {
	if s.id == nil {
		return "", kernel.NewError(kernel.ErrCodeInternal, "profile.id", "ID generator is unavailable")
	}
	return s.id(prefix)
}

func newStableID(prefix string) (string, error) {
	var random [6]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate identifier: %w", err)
	}
	return prefix + "-" + hex.EncodeToString(random[:]), nil
}

func profileRevision(state control.DesiredState) string {
	state.Revision = ""
	state.UpdatedAt = time.Time{}
	body, err := json.Marshal(state)
	if err != nil {
		return "profile:invalid"
	}
	digest := sha256.Sum256(body)
	return "profile:" + hex.EncodeToString(digest[:])
}

func cloneState(state control.DesiredState) (control.DesiredState, error) {
	body, err := json.Marshal(state)
	if err != nil {
		return control.DesiredState{}, err
	}
	var result control.DesiredState
	if err := json.Unmarshal(body, &result); err != nil {
		return control.DesiredState{}, err
	}
	return result, nil
}

func wrapStoreError(op, message string, err error) error {
	if err == nil {
		return nil
	}
	var kernelError *kernel.Error
	if errors.As(err, &kernelError) {
		return err
	}
	return kernel.WrapError(kernel.ErrCodeInternal, op, message, err)
}
