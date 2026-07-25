package core

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/xuthus5/boxd/internal/model"
)

const (
	defaultSubscriptionRefreshInterval = time.Hour
	subscriptionRefreshScanInterval    = time.Minute
)

type SubscriptionAutoRefresher struct {
	manager         *SubscriptionManager
	syncConfig      func() error
	fallbackMinutes int
	scanInterval    time.Duration
	now             func() time.Time

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	running bool

	pendingSync bool
}

// NewSubscriptionAutoRefresher creates a stoppable subscription refresh loop.
func NewSubscriptionAutoRefresher(
	manager *SubscriptionManager,
	syncConfig func() error,
	fallbackMinutes int,
) *SubscriptionAutoRefresher {
	return newSubscriptionAutoRefresher(manager, syncConfig, fallbackMinutes)
}

func newSubscriptionAutoRefresher(
	manager *SubscriptionManager,
	syncConfig func() error,
	fallbackMinutes int,
) *SubscriptionAutoRefresher {
	if fallbackMinutes <= 0 {
		fallbackMinutes = int(defaultSubscriptionRefreshInterval / time.Minute)
	}
	return &SubscriptionAutoRefresher{
		manager:         manager,
		syncConfig:      syncConfig,
		fallbackMinutes: fallbackMinutes,
		scanInterval:    subscriptionRefreshScanInterval,
		now:             time.Now,
	}
}

// Start starts the refresh loop once; repeated calls are harmless.
func (a *SubscriptionAutoRefresher) Start() {
	if a == nil {
		return
	}
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	a.cancel = cancel
	a.done = done
	a.running = true
	a.mu.Unlock()
	go func() {
		defer close(done)
		a.loop(ctx)
	}()
}

// Stop cancels the refresh loop and waits for any in-flight refresh to finish.
func (a *SubscriptionAutoRefresher) Stop() {
	if a == nil {
		return
	}
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

func (a *SubscriptionAutoRefresher) loop(ctx context.Context) {
	a.tick(ctx)
	interval := a.scanInterval
	if interval <= 0 {
		interval = subscriptionRefreshScanInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.tick(ctx)
		}
	}
}

func (a *SubscriptionAutoRefresher) tick(ctx context.Context) {
	if a.manager == nil || ctx.Err() != nil {
		return
	}
	subs, err := a.manager.List()
	if err != nil {
		slog.Warn("subscription auto-refresh list failed", "err", err)
		a.trySync(ctx)
		return
	}
	now := a.now()
	due := make([]model.Subscription, 0, len(subs))
	for _, sub := range subs {
		if ctx.Err() != nil {
			return
		}
		interval := subscriptionRefreshInterval(sub, a.fallbackMinutes)
		if !subscriptionRefreshDue(sub, now, interval) {
			continue
		}
		due = append(due, sub)
	}
	if len(due) > 0 {
		result := a.manager.refreshSubscriptionsConcurrently(ctx, due)
		if result.successes > 0 {
			a.pendingSync = true
		}
		if ctx.Err() != nil {
			return
		}
		for _, failure := range result.failures {
			slog.Warn("subscription auto-refresh failed", "id", failure.ID, "name", failure.Name, "code", failure.Code, "err", failure.Message)
		}
	}
	a.trySync(ctx)
}

func (a *SubscriptionAutoRefresher) trySync(ctx context.Context) {
	if !a.pendingSync || a.syncConfig == nil || ctx.Err() != nil {
		return
	}
	if err := a.syncConfig(); err != nil {
		slog.Warn("subscription auto-refresh config sync failed", "err", err)
		return
	}
	a.pendingSync = false
	slog.Info("subscription auto-refresh config synced")
}

func subscriptionRefreshInterval(sub model.Subscription, fallbackMinutes int) time.Duration {
	fallback, ok := subscriptionRefreshDuration(fallbackMinutes)
	if !ok {
		fallback = defaultSubscriptionRefreshInterval
	}
	if sub.IntervalMin <= 0 {
		return fallback
	}
	interval, ok := subscriptionRefreshDuration(sub.IntervalMin)
	if !ok {
		return fallback
	}
	return interval
}

func subscriptionRefreshDuration(minutes int) (time.Duration, bool) {
	maxMinutes := int64(^uint64(0)>>1) / int64(time.Minute)
	if minutes <= 0 || int64(minutes) > maxMinutes {
		return 0, false
	}
	return time.Duration(minutes) * time.Minute, true
}

func subscriptionRefreshDue(sub model.Subscription, now time.Time, interval time.Duration) bool {
	if interval <= 0 {
		interval = defaultSubscriptionRefreshInterval
	}
	lastAttempt := sub.LastUpdated
	if sub.ErrorAt != nil && sub.ErrorAt.After(lastAttempt) {
		lastAttempt = *sub.ErrorAt
	}
	return lastAttempt.IsZero() || !lastAttempt.Add(interval).After(now)
}
