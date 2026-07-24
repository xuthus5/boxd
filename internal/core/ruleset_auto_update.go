package core

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/xuthus5/boxd/internal/model"
)

// RuleSetAutoUpdater 仅更新内置 local 规则集，默认关闭。
type RuleSetAutoUpdater struct {
	mu       sync.Mutex
	settings *SettingsManager
	updater  *RuleSetUpdater
	cancel   context.CancelFunc
	done     chan struct{}
	running  bool
}

func NewRuleSetAutoUpdater(settings *SettingsManager, updater *RuleSetUpdater) *RuleSetAutoUpdater {
	return &RuleSetAutoUpdater{settings: settings, updater: updater}
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
	cfg, err := a.settings.RuleSetAutoUpdate()
	if err != nil || !cfg.Enabled || a.updater == nil {
		return
	}
	// 仅更新内置 local，避免自动重启内核。
	result, err := a.updater.Update(ctx, RuleSetUpdateRequest{
		Tags:  BuiltinLocalRuleSetTags(),
		Types: []string{"local"},
	})
	if err != nil {
		slog.Warn("ruleset auto update failed", "err", err)
		return
	}
	slog.Info("ruleset auto update finished",
		"updated", result.UpdatedCount,
		"failed", result.FailedCount,
		"skipped", result.SkippedCount,
	)
}

func (a *RuleSetAutoUpdater) Config() (model.RuleSetAutoUpdate, error) {
	return a.settings.RuleSetAutoUpdate()
}
