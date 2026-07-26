package core

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

type ruleSetAutoUpdateRunnerStub struct {
	status func(context.Context) ([]model.RuleSetStatusItem, error)
	update func(context.Context, RuleSetUpdateRequest) (model.RuleSetUpdateResponse, error)
}

func (runner ruleSetAutoUpdateRunnerStub) Status(ctx context.Context) ([]model.RuleSetStatusItem, error) {
	return runner.status(ctx)
}

func (runner ruleSetAutoUpdateRunnerStub) Update(
	ctx context.Context,
	request RuleSetUpdateRequest,
) (model.RuleSetUpdateResponse, error) {
	return runner.update(ctx, request)
}

func TestRuleSetAutoUpdaterTickSkipsCanceledContext(t *testing.T) {
	settings := newEnabledRuleSetAutoUpdateSettings(t)

	var logs bytes.Buffer
	defaultLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(defaultLogger) })

	t.Run("before update", func(t *testing.T) {
		calls := 0
		autoUpdater := &RuleSetAutoUpdater{
			settings: settings,
			updater: ruleSetAutoUpdateRunnerStub{
				status: func(context.Context) ([]model.RuleSetStatusItem, error) {
					return missingBuiltinRuleSetStatuses(), nil
				},
				update: func(context.Context, RuleSetUpdateRequest) (model.RuleSetUpdateResponse, error) {
					calls++
					return model.RuleSetUpdateResponse{}, nil
				},
			},
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		autoUpdater.tick(ctx)

		if calls != 0 {
			t.Fatalf("update calls = %d, want 0", calls)
		}
		if logs.Len() != 0 {
			t.Fatalf("unexpected logs: %s", logs.String())
		}
	})

	t.Run("during update", func(t *testing.T) {
		logs.Reset()
		started := make(chan struct{})
		autoUpdater := &RuleSetAutoUpdater{
			settings: settings,
			updater: ruleSetAutoUpdateRunnerStub{
				status: func(context.Context) ([]model.RuleSetStatusItem, error) {
					return missingBuiltinRuleSetStatuses(), nil
				},
				update: func(ctx context.Context, _ RuleSetUpdateRequest) (model.RuleSetUpdateResponse, error) {
					close(started)
					<-ctx.Done()
					return model.RuleSetUpdateResponse{FailedCount: 3}, ctx.Err()
				},
			},
		}
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		done := make(chan struct{})
		go func() {
			autoUpdater.tick(ctx)
			close(done)
		}()

		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("updater did not start")
		}
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("tick did not stop after cancellation")
		}

		if strings.Contains(logs.String(), "ruleset auto update") {
			t.Fatalf("unexpected cancellation log: %s", logs.String())
		}
	})
}

func TestRuleSetAutoUpdaterTickLogsPartialFailures(t *testing.T) {
	settings := newEnabledRuleSetAutoUpdateSettings(t)
	var logs bytes.Buffer
	defaultLogger := slog.Default()
	slog.SetDefault(slog.New(NewAppLogHandler(&logs, nil, slog.LevelInfo)))
	t.Cleanup(func() { slog.SetDefault(defaultLogger) })
	autoUpdater := &RuleSetAutoUpdater{
		settings: settings,
		updater: ruleSetAutoUpdateRunnerStub{
			status: func(context.Context) ([]model.RuleSetStatusItem, error) {
				return missingBuiltinRuleSetStatuses(), nil
			},
			update: func(context.Context, RuleSetUpdateRequest) (model.RuleSetUpdateResponse, error) {
				return model.RuleSetUpdateResponse{
					UpdatedCount: 1,
					FailedCount:  2,
					Results: []model.RuleSetUpdateResult{
						{Tag: "loyalsoldier-direct", OK: true},
						{Tag: "loyalsoldier-proxy", ErrorCode: RuleSetErrorNetwork, Error: "GET https://example.com/rules?token=secret"},
						{Tag: "loyalsoldier-reject", ErrorCode: RuleSetErrorHTTP, Error: "unexpected status 500"},
					},
				}, nil
			},
		},
	}

	autoUpdater.tick(context.Background())

	output := logs.String()
	for _, expected := range []string{
		"WARN ruleset auto update finished updated=1 failed=2 skipped=0",
		`failed_details=[{"tag":"loyalsoldier-proxy","code":"network"},{"tag":"loyalsoldier-reject","code":"http_status"}]`,
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("log %q missing from %q", expected, output)
		}
	}
	if strings.Contains(output, "token=secret") {
		t.Fatalf("log leaked raw update error: %q", output)
	}
}

func TestRuleSetAutoUpdaterTickUpdatesOnlyDueArtifacts(t *testing.T) {
	settings := newEnabledRuleSetAutoUpdateSettings(t)
	now := time.Now()
	var request RuleSetUpdateRequest
	var logs bytes.Buffer
	defaultLogger := slog.Default()
	slog.SetDefault(slog.New(NewAppLogHandler(&logs, nil, slog.LevelInfo)))
	t.Cleanup(func() { slog.SetDefault(defaultLogger) })
	autoUpdater := &RuleSetAutoUpdater{
		settings: settings,
		updater: ruleSetAutoUpdateRunnerStub{
			status: func(context.Context) ([]model.RuleSetStatusItem, error) {
				fresh := now.Add(-30 * time.Minute)
				stale := now.Add(-2 * time.Hour)
				return []model.RuleSetStatusItem{
					builtinLocalRuleSetStatus("loyalsoldier-direct", &fresh),
					builtinLocalRuleSetStatus("loyalsoldier-proxy", &stale),
					builtinLocalRuleSetStatus("loyalsoldier-reject", nil),
					{Tag: "custom", Type: "local", Updatable: true},
					{Tag: "remote", Type: "remote", Builtin: true, Updatable: true},
				}, nil
			},
			update: func(_ context.Context, value RuleSetUpdateRequest) (model.RuleSetUpdateResponse, error) {
				request = value
				return model.RuleSetUpdateResponse{UpdatedCount: 2}, nil
			},
		},
	}

	autoUpdater.tick(context.Background())

	if !slices.Equal(request.Tags, []string{"loyalsoldier-proxy", "loyalsoldier-reject"}) {
		t.Fatalf("update tags = %#v", request.Tags)
	}
	if !slices.Equal(request.Types, []string{"local"}) {
		t.Fatalf("update types = %#v", request.Types)
	}
	if output := logs.String(); !strings.Contains(output, "ruleset auto update finished updated=2 failed=0 skipped=1") {
		t.Fatalf("partial freshness log missing from %q", output)
	}
}

func TestRuleSetAutoUpdaterTickSkipsFreshArtifacts(t *testing.T) {
	settings := newEnabledRuleSetAutoUpdateSettings(t)
	updated := time.Now().Add(-30 * time.Minute)
	updateCalls := 0
	var logs bytes.Buffer
	defaultLogger := slog.Default()
	slog.SetDefault(slog.New(NewAppLogHandler(&logs, nil, slog.LevelInfo)))
	t.Cleanup(func() { slog.SetDefault(defaultLogger) })
	autoUpdater := &RuleSetAutoUpdater{
		settings: settings,
		updater: ruleSetAutoUpdateRunnerStub{
			status: func(context.Context) ([]model.RuleSetStatusItem, error) {
				statuses := missingBuiltinRuleSetStatuses()
				for index := range statuses {
					statuses[index].LastUpdated = &updated
				}
				return statuses, nil
			},
			update: func(context.Context, RuleSetUpdateRequest) (model.RuleSetUpdateResponse, error) {
				updateCalls++
				return model.RuleSetUpdateResponse{}, nil
			},
		},
	}

	autoUpdater.tick(context.Background())

	if updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", updateCalls)
	}
	if output := logs.String(); !strings.Contains(output, "ruleset auto update finished updated=0 failed=0 skipped=3") {
		t.Fatalf("fresh skip log missing from %q", output)
	}
}

func TestRuleSetAutoUpdaterTickReportsStatusFailure(t *testing.T) {
	settings := newEnabledRuleSetAutoUpdateSettings(t)
	updateCalls := 0
	var logs bytes.Buffer
	defaultLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(defaultLogger) })
	autoUpdater := &RuleSetAutoUpdater{
		settings: settings,
		updater: ruleSetAutoUpdateRunnerStub{
			status: func(context.Context) ([]model.RuleSetStatusItem, error) {
				return nil, errors.New("status unavailable")
			},
			update: func(context.Context, RuleSetUpdateRequest) (model.RuleSetUpdateResponse, error) {
				updateCalls++
				return model.RuleSetUpdateResponse{}, nil
			},
		},
	}

	autoUpdater.tick(context.Background())

	if updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", updateCalls)
	}
	if output := logs.String(); !strings.Contains(output, "ruleset auto update status failed") {
		t.Fatalf("status failure log missing from %q", output)
	}
}

func newEnabledRuleSetAutoUpdateSettings(t *testing.T) *SettingsManager {
	t.Helper()
	db, err := bbolt.Open(filepath.Join(t.TempDir(), "boxd.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	settings := NewSettingsManager(db)
	if err := settings.SetRuleSetAutoUpdate(model.RuleSetAutoUpdate{Enabled: true, Interval: "1h"}); err != nil {
		t.Fatal(err)
	}
	return settings
}

func missingBuiltinRuleSetStatuses() []model.RuleSetStatusItem {
	tags := BuiltinLocalRuleSetTags()
	statuses := make([]model.RuleSetStatusItem, 0, len(tags))
	for _, tag := range tags {
		statuses = append(statuses, builtinLocalRuleSetStatus(tag, nil))
	}
	return statuses
}

func builtinLocalRuleSetStatus(tag string, updatedAt *time.Time) model.RuleSetStatusItem {
	return model.RuleSetStatusItem{
		Tag:         tag,
		Type:        "local",
		Builtin:     true,
		Updatable:   true,
		LastUpdated: updatedAt,
	}
}
