package core

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

type ruleSetUpdateBatch struct {
	updater *RuleSetUpdater
	results map[string]model.RuleSetUpdateResult
}

func (u *RuleSetUpdater) stopBeforeRuleSetUpdate(entries []map[string]any) (bool, error) {
	if u.stop == nil || !ruleSetUpdateNeedsCacheWrite(entries) {
		return false, nil
	}
	if err := u.stop(); err != nil {
		return false, fmt.Errorf("stop kernel before rule-set update: %w", err)
	}
	return true, nil
}

func ruleSetUpdateNeedsCacheWrite(entries []map[string]any) bool {
	for _, entry := range entries {
		if stringValue(entry["type"]) != "remote" || ruleSetEntryHTTPPolicy(entry).managed {
			continue
		}
		if err := ValidatePublicHTTPURL(stringValue(entry["url"])); err == nil {
			return true
		}
	}
	return false
}

func recordRuleSetUpdateResult(response *model.RuleSetUpdateResponse, result model.RuleSetUpdateResult) {
	response.Results = append(response.Results, result)
	switch {
	case result.OK && !result.NotModified:
		response.UpdatedCount++
	case result.OK && result.NotModified:
	case result.ErrorCode == RuleSetErrorNotUpdatable || result.ErrorCode == RuleSetErrorUnsupported || result.ErrorCode == RuleSetErrorKernelManaged ||
		strings.Contains(result.Error, "not auto-updated") || strings.Contains(result.Error, "not updatable"):
		response.SkippedCount++
	default:
		response.FailedCount++
	}
}

func (b *ruleSetUpdateBatch) update(ctx context.Context, entry map[string]any) model.RuleSetUpdateResult {
	key := ruleSetUpdateIdentity(entry)
	if previous, loaded := b.results[key]; loaded {
		return b.reuse(previous, entry)
	}
	result := b.updater.updateOne(ctx, entry)
	b.results[key] = result
	return result
}

func (b *ruleSetUpdateBatch) reuse(previous model.RuleSetUpdateResult, entry map[string]any) model.RuleSetUpdateResult {
	result := previous
	result.Tag = stringValue(entry["tag"])
	if !result.OK {
		return result
	}
	if result.Type == "remote" && previous.Tag != result.Tag {
		if err := b.updater.copyRemoteRuleSetCache(previous.Tag, result.Tag); err != nil {
			return failRuleSetResult(result, err.Error(), err)
		}
	}
	result.NotModified = true
	return result
}

func ruleSetUpdateIdentity(entry map[string]any) string {
	kind := stringValue(entry["type"])
	switch kind {
	case "local":
		if path := stringValue(entry["path"]); path != "" {
			return kind + ":" + filepath.Clean(path)
		}
	case "remote":
		if url := stringValue(entry["url"]); url != "" && !ruleSetEntryHTTPPolicy(entry).managed {
			return kind + ":" + url
		}
	}
	return kind + ":tag:" + stringValue(entry["tag"])
}

func (u *RuleSetUpdater) copyRemoteRuleSetCache(from, to string) error {
	db, err := u.openCacheReadOnly()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRuleSetCacheDisabled, err)
	}
	saved := loadRuleSetCache(db, from)
	if err := db.Close(); err != nil {
		return err
	}
	if saved == nil {
		return ErrRuleSetCacheDisabled
	}
	return u.saveRemoteCache(to, saved)
}
