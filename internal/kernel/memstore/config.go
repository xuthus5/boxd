// Package memstore provides an in-memory ConfigStore for tests and bootstrap.
package memstore

import (
	"context"
	"sync"
	"time"

	"github.com/xuthus5/boxd/kernel"
)

// ConfigStore stores one current revision plus immutable history.
type ConfigStore struct {
	mu      sync.RWMutex
	current kernel.ConfigDocument
	exists  bool
	history []kernel.ConfigRevision
}

var _ kernel.ConfigStore = (*ConfigStore)(nil)
var _ kernel.ConfigRemover = (*ConfigStore)(nil)

// Load returns the current configuration.
func (s *ConfigStore) Load(context.Context) (kernel.ConfigDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.exists {
		return kernel.ConfigDocument{}, kernel.ErrConfigNotFound
	}
	return clone(s.current), nil
}

// SaveAtomic updates the current revision and appends new history. Re-saving the
// same revision is idempotent only when the body is identical.
func (s *ConfigStore) SaveAtomic(_ context.Context, document kernel.ConfigDocument) error {
	if document.Revision == "" {
		return kernel.NewError(kernel.ErrCodeInvalidConfig, "config.store", "revision is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, revision := range s.history {
		if revision.Revision != document.Revision {
			continue
		}
		if s.exists && s.current.Revision == document.Revision && string(s.current.Body) != string(document.Body) {
			return kernel.NewError(kernel.ErrCodeConflict, "config.store", "revision already has different content")
		}
		s.current = clone(document)
		s.exists = true
		return nil
	}
	s.current = clone(document)
	s.exists = true
	s.history = append(s.history, kernel.ConfigRevision{
		Revision:  document.Revision,
		Source:    "memory",
		CreatedAt: time.Now().UTC(),
	})
	return nil
}

// History returns newest revisions first.
func (s *ConfigStore) History(_ context.Context, limit int) ([]kernel.ConfigRevision, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.history) {
		limit = len(s.history)
	}
	result := make([]kernel.ConfigRevision, 0, limit)
	for i := len(s.history) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, s.history[i])
	}
	return result, nil
}

// DeleteCurrent removes the current pointer while keeping history.
func (s *ConfigStore) DeleteCurrent(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = kernel.ConfigDocument{}
	s.exists = false
	return nil
}

func clone(document kernel.ConfigDocument) kernel.ConfigDocument {
	document.Body = append([]byte(nil), document.Body...)
	return document
}
