package manager

import (
	"context"
	"errors"

	"github.com/xuthus5/boxd/kernel"
)

// ReadConfig returns the current persisted configuration snapshot.
func (m *Manager) ReadConfig(ctx context.Context) (kernel.ConfigDocument, error) {
	if err := ctx.Err(); err != nil {
		return kernel.ConfigDocument{}, mapOperationError(kernel.ErrCodeCanceled, "config.read", "read configuration", err)
	}
	document, exists := m.currentConfig()
	if !exists {
		return kernel.ConfigDocument{}, kernel.ErrConfigNotFound
	}
	return document, nil
}

// ValidateConfig validates through the running engine or a temporary engine
// opened for this operation.
func (m *Manager) ValidateConfig(ctx context.Context, document kernel.ConfigDocument) (kernel.ValidationResult, error) {
	m.opMu.Lock()
	defer m.opMu.Unlock()
	if err := m.ensureUsableLocked(); err != nil {
		return kernel.ValidationResult{}, err
	}
	operationCtx, cancel := context.WithTimeout(ctx, m.timeouts.Apply)
	defer cancel()
	return m.validateLocked(operationCtx, document)
}

// ApplyConfig validates, persists, and applies a candidate configuration. A
// failed live apply restores both the store and the last running configuration.
func (m *Manager) ApplyConfig(ctx context.Context, document kernel.ConfigDocument, opts kernel.ApplyOptions) (kernel.ApplyResult, error) {
	m.opMu.Lock()
	defer m.opMu.Unlock()
	if err := m.ensureUsableLocked(); err != nil {
		return kernel.ApplyResult{}, err
	}

	previous, hadPrevious := m.currentConfig()
	if opts.ExpectedRevision != "" && opts.ExpectedRevision != previous.Revision {
		return kernel.ApplyResult{}, kernel.NewError(
			kernel.ErrCodeConflict,
			"config.apply",
			"configuration revision changed; reload and retry",
		).WithHint("the current revision is " + previous.Revision)
	}
	if document.Revision == "" {
		document.Revision = m.nextRevision()
	}
	if document.Format == "" {
		document.Format = "json"
	}
	document = cloneDocument(document)

	operationCtx, cancel := context.WithTimeout(ctx, m.timeouts.Apply)
	defer cancel()
	validation, err := m.validateLocked(operationCtx, document)
	if err != nil {
		return kernel.ApplyResult{}, err
	}
	if !validation.Valid {
		return kernel.ApplyResult{}, kernel.NewValidationError(validation.Diagnostics)
	}
	if len(validation.Normalized) > 0 {
		document.Body = append([]byte(nil), validation.Normalized...)
	}

	if saveErr := m.store.SaveAtomic(operationCtx, document); saveErr != nil {
		return kernel.ApplyResult{}, kernel.WrapError(
			kernel.ErrCodeApplyFailed,
			"config.apply",
			"persist candidate configuration",
			saveErr,
		)
	}

	engine, state := m.currentEngine()
	if state != kernel.StateRunning {
		m.setConfig(document, true)
		m.publishConfigApplied(document.Revision, previous.Revision, opts.Source, false, false, nil)
		return kernel.ApplyResult{Revision: document.Revision}, nil
	}
	if engine == nil {
		return kernel.ApplyResult{}, kernel.NewError(kernel.ErrCodeInvalidState, "config.apply", "running state has no engine instance")
	}

	result, applyErr := engine.ApplyConfig(operationCtx, document, opts)
	if applyErr == nil {
		if result.Revision == "" {
			result.Revision = document.Revision
		}
		m.setConfig(document, true)
		m.publishConfigApplied(document.Revision, previous.Revision, opts.Source, result.Restarted, false, nil)
		return result, nil
	}

	wrappedApply := mapOperationError(kernel.ErrCodeApplyFailed, "config.apply", "apply candidate configuration", applyErr)
	rollbackCtx, rollbackCancel := context.WithTimeout(context.WithoutCancel(ctx), m.timeouts.Apply)
	defer rollbackCancel()
	rollbackErr := m.rollbackStore(rollbackCtx, previous, hadPrevious)
	restoreErr := m.restoreAfterFailedApply(rollbackCtx, previous, hadPrevious)
	m.publishConfigApplied(document.Revision, previous.Revision, opts.Source, false, true, wrappedApply)
	return kernel.ApplyResult{}, errors.Join(wrappedApply, rollbackErr, restoreErr)
}

func (m *Manager) validateLocked(ctx context.Context, document kernel.ConfigDocument) (kernel.ValidationResult, error) {
	engine, state := m.currentEngine()
	if state == kernel.StateRunning && engine != nil {
		return m.validateWithEngine(ctx, engine, document)
	}

	openOptions := m.open
	openOptions.Config = document
	engine, err := m.factory.Open(ctx, openOptions)
	if err != nil {
		return kernel.ValidationResult{}, mapOperationError(kernel.ErrCodeInternal, "config.validate", "open validation engine", err)
	}
	result, validateErr := m.validateWithEngine(ctx, engine, document)
	closeErr := engine.Close()
	if validateErr != nil {
		return kernel.ValidationResult{}, validateErr
	}
	if closeErr != nil {
		return kernel.ValidationResult{}, mapOperationError(kernel.ErrCodeInternal, "config.validate", "close validation engine", closeErr)
	}
	return result, nil
}

func (m *Manager) validateWithEngine(ctx context.Context, engine kernel.Engine, document kernel.ConfigDocument) (kernel.ValidationResult, error) {
	result, err := engine.ValidateConfig(ctx, document)
	if err != nil {
		return kernel.ValidationResult{}, mapOperationError(kernel.ErrCodeInvalidConfig, "config.validate", "validate configuration", err)
	}
	return result, nil
}

func (m *Manager) rollbackStore(ctx context.Context, previous kernel.ConfigDocument, hadPrevious bool) error {
	if hadPrevious {
		if err := m.store.SaveAtomic(ctx, previous); err != nil {
			return kernel.WrapError(kernel.ErrCodeApplyFailed, "config.rollback", "restore previous configuration", err)
		}
		return nil
	}
	remover, ok := m.store.(kernel.ConfigRemover)
	if !ok {
		return kernel.NewError(kernel.ErrCodeApplyFailed, "config.rollback", "config store cannot remove the first failed configuration")
	}
	if err := remover.DeleteCurrent(ctx); err != nil {
		return kernel.WrapError(kernel.ErrCodeApplyFailed, "config.rollback", "remove failed first configuration", err)
	}
	return nil
}

func (m *Manager) publishConfigApplied(revision, previousRevision, source string, restarted, rolledBack bool, applyErr *kernel.Error) {
	m.events.Publish(kernel.Event{
		Type: kernel.EventConfigApplied,
		Payload: kernel.ConfigApplied{
			Revision:         revision,
			PreviousRevision: previousRevision,
			Source:           source,
			Restarted:        restarted,
			RolledBack:       rolledBack,
			Error:            cloneError(applyErr),
		},
	})
}
