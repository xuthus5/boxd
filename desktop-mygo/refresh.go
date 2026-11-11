package main

import (
	"context"
	"time"
)

func (a *application) startRefresh() {
	if a.backend == nil {
		return
	}
	a.refreshOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		a.refreshCancel = cancel
		a.refreshDone = make(chan struct{})
		go a.refreshLoop(ctx)
	})
}

func (a *application) refreshLoop(ctx context.Context) {
	defer close(a.refreshDone)
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()
	a.refresh(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.refresh(ctx)
		case <-a.refreshNow:
			a.refresh(ctx)
		}
	}
}

func (a *application) refresh(parent context.Context) {
	if a.backend == nil || parent.Err() != nil {
		return
	}
	ctx, cancel := context.WithTimeout(parent, actionTimeout)
	snap, err := a.backend.Snapshot(ctx)
	cancel()
	if parent.Err() != nil {
		return
	}
	if a.win == nil {
		return
	}
	a.win.Update(func() {
		a.applyRefresh(snap, err, time.Now())
	})
}

func (a *application) applyRefresh(snap snapshot, err error, now time.Time) {
	if err != nil {
		a.model.stale = a.model.loaded
		a.model.warnings = []string{shortError(err)}
		return
	}
	a.model.applySnapshot(snap, now)
}

func (a *application) requestRefresh() {
	select {
	case a.refreshNow <- struct{}{}:
	default:
	}
}

func (a *application) stopRefresh() {
	if a.refreshCancel != nil {
		a.refreshCancel()
	}
	if a.refreshDone != nil {
		<-a.refreshDone
	}
}
