package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

// RuleSetService keeps installed definitions in desired state while status
// updates go to a separate repository.
type RuleSetService struct {
	profile    *ProfileService
	statuses   control.RuleSetRepository
	downloader control.Downloader
	now        func() time.Time
}

var _ control.RuleSetAPI = (*RuleSetService)(nil)

func NewRuleSetService(profile *ProfileService, statuses control.RuleSetRepository, downloader control.Downloader) *RuleSetService {
	return &RuleSetService{profile: profile, statuses: statuses, downloader: downloader, now: time.Now}
}

func (s *RuleSetService) List(ctx context.Context) ([]control.RuleSet, error) {
	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	statusByID, err := s.statusMap(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]control.RuleSet, 0, len(state.RuleSets))
	for _, definition := range state.RuleSets {
		result = append(result, ruleSetView(definition, statusByID[definition.Name]))
	}
	return result, nil
}

func (s *RuleSetService) Get(ctx context.Context, name string) (control.RuleSet, error) {
	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return control.RuleSet{}, err
	}
	for _, definition := range state.RuleSets {
		if definition.Name != name {
			continue
		}
		status, found, statusErr := s.getStatus(ctx, name)
		if statusErr != nil {
			return control.RuleSet{}, statusErr
		}
		if !found {
			status = control.RuleSetStatus{Name: name}
		}
		return ruleSetView(definition, status), nil
	}
	return control.RuleSet{}, notFound("rule set", name)
}

func (s *RuleSetService) Save(ctx context.Context, definition control.RuleSetDefinition) (control.RuleSet, error) {
	definition.Name = strings.TrimSpace(definition.Name)
	definition.URL = strings.TrimSpace(definition.URL)
	if definition.Format == "" {
		definition.Format = control.RuleSetFormatBinary
	}
	definition.Builtin = false
	state, err := s.profile.update(ctx, func(state *control.DesiredState) error {
		for index := range state.RuleSets {
			if state.RuleSets[index].Name != definition.Name {
				continue
			}
			if state.RuleSets[index].Builtin {
				return kernel.NewError(kernel.ErrCodeConflict, "rulesets.save", fmt.Sprintf("built-in rule set %q cannot be modified", definition.Name))
			}
			state.RuleSets[index] = definition
			return nil
		}
		state.RuleSets = append(state.RuleSets, definition)
		return nil
	})
	if err != nil {
		return control.RuleSet{}, err
	}
	var saved control.RuleSetDefinition
	for _, candidate := range state.RuleSets {
		if candidate.Name == definition.Name {
			saved = candidate
			break
		}
	}
	if saved.Name == "" {
		return control.RuleSet{}, notFound("rule set", definition.Name)
	}
	status, found, err := s.getStatus(ctx, saved.Name)
	if err != nil {
		return control.RuleSet{}, err
	}
	if !found {
		status = control.RuleSetStatus{Name: saved.Name}
	}
	return ruleSetView(saved, status), nil
}

func (s *RuleSetService) Delete(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	_, err := s.profile.update(ctx, func(state *control.DesiredState) error {
		for index := range state.RuleSets {
			if state.RuleSets[index].Name != name {
				continue
			}
			if state.RuleSets[index].Builtin {
				return kernel.NewError(kernel.ErrCodeConflict, "rulesets.delete", fmt.Sprintf("built-in rule set %q cannot be deleted", name))
			}
			state.RuleSets = append(state.RuleSets[:index], state.RuleSets[index+1:]...)
			return nil
		}
		return notFound("rule set", name)
	})
	if err != nil {
		return err
	}
	if s.statuses != nil {
		if err := s.statuses.Delete(ctx, name); err != nil {
			return wrapStoreError("rulesets.delete", "delete rule set status", err)
		}
	}
	return nil
}

func (s *RuleSetService) Refresh(ctx context.Context, name string) (control.RuleSet, error) {
	if s.downloader == nil {
		return control.RuleSet{}, capabilityUnavailable("rulesets.refresh", "rule set downloader")
	}
	if s.statuses == nil {
		return control.RuleSet{}, capabilityUnavailable("rulesets.refresh", "rule set status repository")
	}
	name = strings.TrimSpace(name)
	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return control.RuleSet{}, err
	}
	var definition control.RuleSetDefinition
	found := false
	for _, candidate := range state.RuleSets {
		if candidate.Name == name {
			definition = candidate
			found = true
			break
		}
	}
	if !found {
		return control.RuleSet{}, notFound("rule set", name)
	}
	previous, hasPrevious, err := s.getStatus(ctx, name)
	if err != nil {
		return control.RuleSet{}, err
	}
	if !hasPrevious {
		previous = control.RuleSetStatus{Name: name}
	}
	result, err := s.downloader.Get(ctx, control.DownloadRequest{
		URL:          definition.URL,
		ETag:         previous.ETag,
		LastModified: previous.LastModified,
		Timeout:      defaultDownloadTimeout * time.Second,
		MaxBytes:     defaultDownloadBytes,
	})
	if err != nil {
		return control.RuleSet{}, s.recordRefreshError(ctx, previous, err)
	}
	status := previous
	status.UpdatedAt = s.now().UTC()
	status.Error = ""
	if result.ETag != "" {
		status.ETag = result.ETag
	}
	if result.LastModified != "" {
		status.LastModified = result.LastModified
	}
	if !result.NotModified {
		if len(result.Body) == 0 {
			return control.RuleSet{}, s.recordRefreshError(ctx, previous, kernel.NewError(kernel.ErrCodeInvalidConfig, "rulesets.refresh", "downloaded rule set is empty"))
		}
		digest := sha256.Sum256(result.Body)
		status.SHA256 = hex.EncodeToString(digest[:])
	}
	if err := s.statuses.Upsert(ctx, status); err != nil {
		return control.RuleSet{}, wrapStoreError("rulesets.refresh", "save rule set status", err)
	}
	return ruleSetView(definition, status), nil
}

func (s *RuleSetService) RefreshAll(ctx context.Context) ([]control.RuleSet, error) {
	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]control.RuleSet, 0, len(state.RuleSets))
	var failures []error
	for _, definition := range state.RuleSets {
		refreshed, refreshErr := s.Refresh(ctx, definition.Name)
		if refreshErr != nil {
			failures = append(failures, fmt.Errorf("%s: %w", definition.Name, refreshErr))
			continue
		}
		result = append(result, refreshed)
	}
	return result, errors.Join(failures...)
}

func (s *RuleSetService) statusMap(ctx context.Context) (map[string]control.RuleSetStatus, error) {
	result := make(map[string]control.RuleSetStatus)
	if s.statuses == nil {
		return result, nil
	}
	statuses, err := s.statuses.List(ctx)
	if err != nil {
		return nil, wrapStoreError("rulesets.list", "list rule set status", err)
	}
	for _, status := range statuses {
		result[status.Name] = status
	}
	return result, nil
}

func (s *RuleSetService) getStatus(ctx context.Context, name string) (control.RuleSetStatus, bool, error) {
	if s.statuses == nil {
		return control.RuleSetStatus{}, false, nil
	}
	status, found, err := s.statuses.Get(ctx, name)
	if err != nil {
		return control.RuleSetStatus{}, false, wrapStoreError("rulesets.status", "load rule set status", err)
	}
	return status, found, nil
}

func (s *RuleSetService) recordRefreshError(ctx context.Context, status control.RuleSetStatus, cause error) error {
	status.UpdatedAt = s.now().UTC()
	status.Error = cause.Error()
	if err := s.statuses.Upsert(ctx, status); err != nil {
		return errors.Join(cause, wrapStoreError("rulesets.refresh", "save rule set failure status", err))
	}
	return cause
}

func ruleSetView(definition control.RuleSetDefinition, status control.RuleSetStatus) control.RuleSet {
	if status.Name == "" {
		status.Name = definition.Name
	}
	return control.RuleSet{
		Name:    definition.Name,
		URL:     definition.URL,
		Format:  definition.Format,
		Enabled: definition.Enabled,
		Builtin: definition.Builtin,
		Status:  status,
	}
}
