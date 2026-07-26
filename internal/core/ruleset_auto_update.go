package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/xuthus5/boxd/internal/model"
)

// RuleSetAutoUpdater 仅更新内置 local 规则集，默认关闭。
type RuleSetAutoUpdater struct {
	mu       sync.Mutex
	settings *SettingsManager
	updater  ruleSetAutoUpdateRunner
	cancel   context.CancelFunc
	done     chan struct{}
	running  bool
}

type ruleSetAutoUpdateRunner interface {
	Update(context.Context, RuleSetUpdateRequest) (model.RuleSetUpdateResponse, error)
}

type ruleSetAutoUpdateFailureDetail struct {
	Tag  string `json:"tag"`
	Code string `json:"code"`
}

const ruleSetAutoUpdateFailureLimit = 3

func NewRuleSetAutoUpdater(settings *SettingsManager, updater *RuleSetUpdater) *RuleSetAutoUpdater {
	autoUpdater := &RuleSetAutoUpdater{settings: settings}
	if updater != nil {
		autoUpdater.updater = updater
	}
	return autoUpdater
}

func (a *RuleSetAutoUpdater) Start() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.running {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.done = make(chan struct{})
	a.running = true
	done := a.done
	go func() {
		defer close(done)
		a.loop(ctx)
	}()
}

func (a *RuleSetAutoUpdater) Stop() {
	a.mu.Lock()
	if !a.running {
		a.mu.Unlock()
		return
	}
	cancel, done := a.cancel, a.done
	a.mu.Unlock()
	cancel()
	<-done
	a.mu.Lock()
	if a.done == done {
		a.cancel = nil
		a.done = nil
		a.running = false
	}
	a.mu.Unlock()
}

func (a *RuleSetAutoUpdater) loop(ctx context.Context) {
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			a.tick(ctx)
			timer.Reset(a.nextDelay())
		}
	}
}

func (a *RuleSetAutoUpdater) nextDelay() time.Duration {
	cfg, err := a.settings.RuleSetAutoUpdate()
	if err != nil || !cfg.Enabled {
		return time.Hour
	}
	d, err := time.ParseDuration(cfg.Interval)
	if err != nil || d <= 0 {
		return time.Hour
	}
	if d < time.Minute {
		return time.Minute
	}
	return d
}

func (a *RuleSetAutoUpdater) tick(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	cfg, err := a.settings.RuleSetAutoUpdate()
	if err != nil || !cfg.Enabled || a.updater == nil {
		return
	}
	// 仅更新内置 local，避免自动重启内核。
	result, err := a.updater.Update(ctx, RuleSetUpdateRequest{
		Tags:  BuiltinLocalRuleSetTags(),
		Types: []string{"local"},
	})
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		slog.Warn("ruleset auto update failed", "err", err)
		return
	}
	failureDetails, detailsErr := marshalRuleSetAutoUpdateFailures(result.Results)
	if detailsErr != nil {
		slog.Error("ruleset auto update failure details unavailable", "err", detailsErr)
	}
	args := []any{
		"updated", result.UpdatedCount,
		"failed", result.FailedCount,
		"skipped", result.SkippedCount,
	}
	if failureDetails != "" {
		args = append(args, "failed_details", failureDetails)
	}
	level := slog.LevelInfo
	if result.FailedCount > 0 {
		level = slog.LevelWarn
	}
	slog.Log(ctx, level, "ruleset auto update finished", args...)
}

func marshalRuleSetAutoUpdateFailures(results []model.RuleSetUpdateResult) (string, error) {
	failures := make([]ruleSetAutoUpdateFailureDetail, 0, min(len(results), ruleSetAutoUpdateFailureLimit))
	for _, result := range results {
		if result.OK || len(failures) >= ruleSetAutoUpdateFailureLimit {
			continue
		}
		code := strings.TrimSpace(result.ErrorCode)
		if code == "" {
			code = RuleSetErrorUnknown
		}
		failures = append(failures, ruleSetAutoUpdateFailureDetail{
			Tag:  strings.TrimSpace(result.Tag),
			Code: code,
		})
	}
	if len(failures) == 0 {
		return "", nil
	}
	data, err := json.Marshal(failures)
	if err != nil {
		return "", fmt.Errorf("marshal rule-set auto-update failures: %w", err)
	}
	return string(data), nil
}

func (a *RuleSetAutoUpdater) Config() (model.RuleSetAutoUpdate, error) {
	return a.settings.RuleSetAutoUpdate()
}
