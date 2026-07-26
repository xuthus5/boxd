package core

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

type ruleSetAutoUpdateFunc func(context.Context, RuleSetUpdateRequest) (model.RuleSetUpdateResponse, error)

func (update ruleSetAutoUpdateFunc) Update(
	ctx context.Context,
	request RuleSetUpdateRequest,
) (model.RuleSetUpdateResponse, error) {
	return update(ctx, request)
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
			updater: ruleSetAutoUpdateFunc(func(
				context.Context,
				RuleSetUpdateRequest,
			) (model.RuleSetUpdateResponse, error) {
				calls++
				return model.RuleSetUpdateResponse{}, nil
			}),
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
			updater: ruleSetAutoUpdateFunc(func(
				ctx context.Context,
				_ RuleSetUpdateRequest,
			) (model.RuleSetUpdateResponse, error) {
				close(started)
				<-ctx.Done()
				return model.RuleSetUpdateResponse{FailedCount: 3}, ctx.Err()
			}),
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
		updater: ruleSetAutoUpdateFunc(func(
			context.Context,
			RuleSetUpdateRequest,
		) (model.RuleSetUpdateResponse, error) {
			return model.RuleSetUpdateResponse{
				UpdatedCount: 1,
				FailedCount:  2,
				Results: []model.RuleSetUpdateResult{
					{Tag: "loyalsoldier-direct", OK: true},
					{Tag: "loyalsoldier-proxy", ErrorCode: RuleSetErrorNetwork, Error: "GET https://example.com/rules?token=secret"},
					{Tag: "loyalsoldier-reject", ErrorCode: RuleSetErrorHTTP, Error: "unexpected status 500"},
				},
			}, nil
		}),
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
