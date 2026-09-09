package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuthus5/boxd/internal/model"
)

const (
	ruleSetAutoUpdateKey       = "ruleset_auto_update"
	defaultRuleSetAutoInterval = "24h"
	singBoxRuleSetBucket       = "rule_set"
)

var (
	ErrRuleSetNotFound      = errors.New("rule-set not found in config")
	ErrRuleSetNotUpdatable  = errors.New("rule-set is not updatable")
	ErrRuleSetCacheDisabled = errors.New("rule-set cache is unavailable")
)

// RuleSetUpdateRequest 选择要更新的规则集。
// Tags 为空表示更新全部可更新项；Types 可过滤 local/remote。
type RuleSetUpdateRequest struct {
	Tags  []string `json:"tags"`
	Types []string `json:"types"`
}

type RuleSetUpdater struct {
	configPath string
	cachePath  string
	installer  *LoyalsoldierRuleSetInstaller
	client     *http.Client
	stop       func() error
	start      func() error
}

func NewRuleSetUpdater(configPath, dataDir string, installer *LoyalsoldierRuleSetInstaller, stop, start func() error) *RuleSetUpdater {
	if installer == nil {
		installer = NewLoyalsoldierRuleSetInstaller(dataDir)
	}
	return &RuleSetUpdater{
		configPath: configPath,
		cachePath:  filepath.Join(dataDir, "cache.db"),
		installer:  installer,
		client:     newPublicHTTPClient(ruleSetHTTPTimeout),
		stop:       stop,
		start:      start,
	}
}

func DefaultRuleSetAutoUpdate() model.RuleSetAutoUpdate {
	return model.RuleSetAutoUpdate{Enabled: false, Interval: defaultRuleSetAutoInterval}
}

func ValidateRuleSetAutoUpdate(cfg model.RuleSetAutoUpdate) error {
	if strings.TrimSpace(cfg.Interval) == "" {
		return fmt.Errorf("interval is required")
	}
	d, err := time.ParseDuration(cfg.Interval)
	if err != nil || d <= 0 {
		return fmt.Errorf("interval must be a positive duration such as 24h")
	}
	return nil
}

func (m *SettingsManager) RuleSetAutoUpdate() (model.RuleSetAutoUpdate, error) {
	value := m.Get(ruleSetAutoUpdateKey)
	if value == "" {
		return DefaultRuleSetAutoUpdate(), nil
	}
	var cfg model.RuleSetAutoUpdate
	if err := json.Unmarshal([]byte(value), &cfg); err != nil {
		return model.RuleSetAutoUpdate{}, fmt.Errorf("decoding ruleset auto update: %w", err)
	}
	if cfg.Interval == "" {
		cfg.Interval = defaultRuleSetAutoInterval
	}
	return cfg, nil
}

func (m *SettingsManager) SetRuleSetAutoUpdate(cfg model.RuleSetAutoUpdate) error {
	if err := ValidateRuleSetAutoUpdate(cfg); err != nil {
		return err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encoding ruleset auto update: %w", err)
	}
	return m.Set(ruleSetAutoUpdateKey, string(data))
}

func (u *RuleSetUpdater) Status(ctx context.Context) ([]model.RuleSetStatusItem, error) {
	_ = ctx
	cfg, err := u.loadConfig()
	if err != nil {
		return nil, err
	}
	entries := ruleSetEntries(cfg)
	cache, _ := u.openCacheReadOnly()
	if cache != nil {
		defer func() { _ = cache.Close() }()
	}
	items := make([]model.RuleSetStatusItem, 0, len(entries))
	for _, entry := range entries {
		items = append(items, u.statusOf(entry, cache))
	}
	return items, nil
}

func (u *RuleSetUpdater) Update(ctx context.Context, req RuleSetUpdateRequest) (model.RuleSetUpdateResponse, error) {
	cfg, err := u.loadConfig()
	if err != nil {
		return model.RuleSetUpdateResponse{}, err
	}
	entries := ruleSetEntries(cfg)
	selected := selectRuleSets(entries, req)
	slog.Info("rule set update started", "total", len(entries), "selected", len(selected))
	resp := model.RuleSetUpdateResponse{Results: make([]model.RuleSetUpdateResult, 0, len(selected))}
	stopped, err := u.stopBeforeRuleSetUpdate(selected)
	if err != nil {
		return model.RuleSetUpdateResponse{}, err
	}
	batch := ruleSetUpdateBatch{updater: u, results: make(map[string]model.RuleSetUpdateResult)}
	for _, entry := range selected {
		recordRuleSetUpdateResult(&resp, batch.update(ctx, entry))
	}
	if stopped && u.start != nil {
		if err := u.start(); err != nil {
			return resp, fmt.Errorf("start kernel after rule-set update: %w", err)
		}
		resp.Restarted = true
	}
	slog.Info("rule set update completed",
		"updated", resp.UpdatedCount,
		"skipped", resp.SkippedCount,
		"failed", resp.FailedCount,
		"restarted", resp.Restarted,
	)
	return resp, nil
}

func (u *RuleSetUpdater) updateOne(ctx context.Context, entry map[string]any) model.RuleSetUpdateResult {
	tag, _ := entry["tag"].(string)
	typ, _ := entry["type"].(string)
	if typ == "" {
		typ = "inline"
	}
	result := model.RuleSetUpdateResult{Tag: tag, Type: typ}
	if ruleSetEntryHTTPPolicy(entry).managed {
		return failRuleSetResult(result, ErrRuleSetKernelManaged.Error(), ErrRuleSetKernelManaged)
	}
	switch typ {
	case "local":
		return u.updateLocal(ctx, entry, result)
	case "remote":
		return u.updateRemote(ctx, entry, result)
	default:
		return failRuleSetResult(result, ErrRuleSetNotUpdatable.Error(), ErrRuleSetNotUpdatable)
	}
}

func (u *RuleSetUpdater) updateLocal(ctx context.Context, entry map[string]any, result model.RuleSetUpdateResult) model.RuleSetUpdateResult {
	tag := result.Tag
	src, ok := u.installer.SourceByTag(tag)
	if !ok {
		return failRuleSetResult(result, "custom local rule-set files are not auto-updated", nil)
	}
	path, _ := entry["path"].(string)
	if path == "" {
		path = filepath.Join(u.installer.ruleSetDir, src.FileName)
	}
	now := time.Now()
	data, err := u.installer.downloadRuleSet(ctx, src)
	if err != nil {
		return failRuleSetResult(result, err.Error(), err)
	}
	if err := ctx.Err(); err != nil {
		return failRuleSetResult(result, err.Error(), err)
	}
	if err := atomicWriteFile0600(path, data); err != nil {
		return failRuleSetResult(result, err.Error(), err)
	}
	result.OK = true
	result.UpdatedAt = &now
	return result
}

func (u *RuleSetUpdater) loadConfig() (map[string]any, error) {
	data, err := os.ReadFile(u.configPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	return cfg, nil
}
