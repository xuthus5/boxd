package controlplane

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/xuthus5/boxd/control"
)

const defaultSchedulerInterval = time.Minute

// SchedulerOptions tunes the subscription refresh worker. Tests may shorten
// the interval; production uses the default one-minute scan cadence.
type SchedulerOptions struct {
	Interval time.Duration
	Now      func() time.Time
}

// SchedulerService periodically refreshes subscriptions whose configured
// interval has elapsed. It owns no transport and can run in any host.
type SchedulerService struct {
	subscriptions *SubscriptionService

	interval time.Duration
	now      func() time.Time

	runMu sync.Mutex
	mu    sync.Mutex

	cancel    context.CancelFunc
	done      chan struct{}
	running   bool
	lastRunAt time.Time
	lastError string
}

var _ control.SchedulerAPI = (*SchedulerService)(nil)

// NewSchedulerService constructs a stopped scheduler for the given
// subscription use case.
func NewSchedulerService(subscriptions *SubscriptionService, options SchedulerOptions) *SchedulerService {
	interval := options.Interval
	if interval <= 0 {
		interval = defaultSchedulerInterval
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &SchedulerService{
		subscriptions: subscriptions,
		interval:      interval,
		now:           now,
	}
}

func (s *SchedulerService) Status(context.Context) (control.SchedulerStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := control.SchedulerStatus{
		Running:         s.running,
		IntervalSeconds: int(s.interval / time.Second),
		LastRunAt:       s.lastRunAt,
		LastError:       s.lastError,
	}
	if s.running {
		base := s.lastRunAt
		if base.IsZero() {
			base = s.now().UTC()
		}
		status.NextRunAt = base.Add(s.interval)
	}
	return status, nil
}

func (s *SchedulerService) Start(ctx context.Context) error {
	if err := s.ensureAvailable("scheduler.start"); err != nil {
		return err
	}
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	loopContext, cancel := context.WithCancel(context.WithoutCancel(ctx))
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done
	s.running = true
	s.mu.Unlock()
	go s.loop(loopContext, done)
	return nil
}

func (s *SchedulerService) Stop(ctx context.Context) error {
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	s.mu.Unlock()
	if cancel == nil || done == nil {
		return nil
	}
	cancel()
	var stopErr error
	select {
	case <-done:
	case <-ctx.Done():
		stopErr = ctx.Err()
	}
	return stopErr
}

func (s *SchedulerService) RunOnce(ctx context.Context) ([]control.SubscriptionRefreshResult, error) {
	if err := s.ensureAvailable("scheduler.run"); err != nil {
		return nil, err
	}
	s.runMu.Lock()
	defer s.runMu.Unlock()

	startedAt := s.now().UTC()
	results, runErr := s.refreshDue(ctx, startedAt)
	s.mu.Lock()
	s.lastRunAt = startedAt
	if runErr != nil {
		s.lastError = runErr.Error()
	} else {
		s.lastError = ""
	}
	s.mu.Unlock()
	return results, runErr
}

func (s *SchedulerService) refreshDue(ctx context.Context, now time.Time) ([]control.SubscriptionRefreshResult, error) {
	subscriptions, err := s.subscriptions.List(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]control.SubscriptionRefreshResult, 0)
	var failures []error
	for _, subscription := range subscriptions {
		if !dueForRefresh(subscription, now) {
			continue
		}
		result, refreshErr := s.subscriptions.Refresh(ctx, subscription.ID)
		if refreshErr != nil {
			failures = append(failures, fmt.Errorf("%s: %w", subscription.ID, refreshErr))
			continue
		}
		results = append(results, result)
	}
	return results, errors.Join(failures...)
}

func (s *SchedulerService) loop(ctx context.Context, done chan struct{}) {
	defer s.finish(done)
	_, _ = s.RunOnce(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.RunOnce(ctx)
		}
	}
}

func (s *SchedulerService) finish(done chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.done == done {
		s.running = false
		s.cancel = nil
		s.done = nil
	}
	close(done)
}

func (s *SchedulerService) ensureAvailable(op string) error {
	if s.subscriptions == nil || s.subscriptions.downloader == nil || s.subscriptions.parser == nil {
		return capabilityUnavailable(op, "subscription downloader and parser")
	}
	return nil
}

func dueForRefresh(subscription control.Subscription, now time.Time) bool {
	if !subscription.Enabled || subscription.IntervalSeconds <= 0 {
		return false
	}
	if subscription.LastRefreshedAt.IsZero() {
		return true
	}
	interval := time.Duration(subscription.IntervalSeconds) * time.Second
	return !subscription.LastRefreshedAt.Add(interval).After(now)
}
