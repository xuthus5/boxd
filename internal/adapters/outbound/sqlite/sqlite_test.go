package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

func TestOpenAppliesRequiredPragmasAndMigrations(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	var journalMode string
	if err := db.sql.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("read journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}
	var foreignKeys int
	if err := db.sql.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("read foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}
	var busyTimeout int
	if err := db.sql.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("read busy_timeout: %v", err)
	}
	if busyTimeout != busyTimeoutMS {
		t.Fatalf("busy_timeout = %d, want %d", busyTimeout, busyTimeoutMS)
	}

	var migrations int
	if err := db.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&migrations); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if migrations != 5 {
		t.Fatalf("migrations = %d, want 5", migrations)
	}
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}
	_ = db.Checkpoint(ctx)
}

func TestConfigRepositoryRoundTripAndRollback(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	repository := db.Configs()

	if _, err := repository.Load(ctx); !errors.Is(err, kernel.ErrConfigNotFound) {
		t.Fatalf("Load() error = %v, want config not found", err)
	}
	first := kernel.ConfigDocument{Format: "json", Body: []byte(`{"mixed":true}`), Revision: "rev-1"}
	if err := repository.SaveAtomic(ctx, first); err != nil {
		t.Fatalf("SaveAtomic(first) error = %v", err)
	}
	if err := repository.SaveAtomic(ctx, first); err != nil {
		t.Fatalf("idempotent SaveAtomic() error = %v", err)
	}
	loaded, err := repository.Load(ctx)
	if err != nil || loaded.Revision != first.Revision || string(loaded.Body) != string(first.Body) {
		t.Fatalf("Load() = %+v, %v", loaded, err)
	}

	conflicting := first
	conflicting.Body = []byte(`{"mixed":false}`)
	if err := repository.SaveAtomic(ctx, conflicting); err == nil {
		t.Fatal("SaveAtomic() accepted different content for an existing revision")
	}

	second := kernel.ConfigDocument{Format: "json", Body: []byte(`{"route":true}`), Revision: "rev-2"}
	if err := repository.SaveAtomic(ctx, second); err != nil {
		t.Fatalf("SaveAtomic(second) error = %v", err)
	}
	history, err := repository.History(ctx, 10)
	if err != nil {
		t.Fatalf("History() error = %v", err)
	}
	if len(history) != 2 || history[0].Revision != "rev-2" || history[1].Revision != "rev-1" {
		t.Fatalf("history = %+v", history)
	}
	if err := repository.SaveAtomic(ctx, first); err != nil {
		t.Fatalf("rollback SaveAtomic() error = %v", err)
	}
	loaded, err = repository.Load(ctx)
	if err != nil || loaded.Revision != "rev-1" {
		t.Fatalf("Load() after rollback = %+v, %v", loaded, err)
	}
	if err := repository.DeleteCurrent(ctx); err != nil {
		t.Fatalf("DeleteCurrent() error = %v", err)
	}
	if _, err := repository.Load(ctx); !errors.Is(err, kernel.ErrConfigNotFound) {
		t.Fatalf("Load() after delete error = %v", err)
	}
}

func TestSettingsRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	repository := db.Settings()

	if _, found, err := repository.Get(ctx, "theme"); err != nil || found {
		t.Fatalf("Get() = found %v, err %v", found, err)
	}
	setting := control.Setting{Key: "theme", Value: "green", UpdatedAt: time.Now().UTC().Truncate(time.Millisecond)}
	if err := repository.Upsert(ctx, setting); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	got, found, err := repository.Get(ctx, "theme")
	if err != nil || !found || got.Value != "green" {
		t.Fatalf("Get() = %+v, found %v, err %v", got, found, err)
	}
	all, err := repository.List(ctx)
	if err != nil || len(all) != 1 || all[0].Key != "theme" {
		t.Fatalf("List() = %+v, %v", all, err)
	}
}

func TestProfileRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	repository := db.Profiles()

	if _, err := repository.Load(ctx); !errors.Is(err, kernel.ErrConfigNotFound) {
		t.Fatalf("Load() error = %v, want profile not found", err)
	}
	state := control.DefaultDesiredState()
	state.Revision = "profile:test"
	state.Nodes = []control.Node{{ID: "hk", Name: "HK", Protocol: "shadowsocks", Server: "127.0.0.1", Port: 8388}}
	if err := repository.Save(ctx, state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := repository.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Revision != state.Revision || len(loaded.Inbounds) != 2 || len(loaded.Nodes) != 1 || loaded.Nodes[0].ID != "hk" {
		t.Fatalf("loaded state = %+v", loaded)
	}
}

func TestRuleSetStatusRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	repository := db.RuleSets()

	status := control.RuleSetStatus{
		Name:         "geosite-cn",
		ETag:         `"v1"`,
		LastModified: "Wed, 21 Oct 2015 07:28:00 GMT",
		SHA256:       "abc",
		SizeBytes:    4096,
		UpdatedAt:    time.Now().UTC().Truncate(time.Millisecond),
		Error:        "previous failure",
	}
	if err := repository.Upsert(ctx, status); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	got, found, err := repository.Get(ctx, status.Name)
	if err != nil || !found || got.ETag != status.ETag || got.Error != status.Error || got.SHA256 != status.SHA256 || got.SizeBytes != status.SizeBytes {
		t.Fatalf("Get() = %+v, found %v, err %v", got, found, err)
	}
	all, err := repository.List(ctx)
	if err != nil || len(all) != 1 || all[0].Name != status.Name {
		t.Fatalf("List() = %+v, %v", all, err)
	}
	if err := repository.Delete(ctx, status.Name); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, found, err := repository.Get(ctx, status.Name); err != nil || found {
		t.Fatalf("Get() after delete = found %v, err %v", found, err)
	}
}

func TestTestResultRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	repository := db.TestResults()
	node := control.Node{ID: "hk", Name: "HK", Protocol: "shadowsocks", Server: "127.0.0.1", Port: 8388}
	state := control.DefaultDesiredState()
	state.Revision = "profile:test"
	state.Nodes = []control.Node{node}
	if err := db.Profiles().Save(ctx, state); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	results := []control.NodeTestResult{
		{NodeID: node.ID, Kind: "delay", URL: "https://example.com/one", Latency: 20 * time.Millisecond, TestedAt: now.Add(-time.Second)},
		{NodeID: node.ID, Kind: "delay", URL: "https://example.com/two", Latency: 30 * time.Millisecond, TestedAt: now},
	}
	for _, result := range results {
		if err := repository.Save(ctx, result); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}
	history, err := repository.List(ctx, node.ID, 10)
	if err != nil || len(history) != 2 {
		t.Fatalf("List() = %+v, %v", history, err)
	}
	if history[0].URL != "https://example.com/two" || history[0].Latency != 30*time.Millisecond {
		t.Fatalf("newest result = %+v", history[0])
	}
}

func TestSecurityRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	repository := db.Security()
	if _, found, err := repository.Load(ctx); err != nil || found {
		t.Fatalf("Load() = found %v, err %v", found, err)
	}
	credential := control.SecurityCredential{
		Salt:       []byte("salt"),
		Hash:       []byte("hash"),
		Iterations: 600000,
		UpdatedAt:  time.Now().UTC().Truncate(time.Millisecond),
	}
	if err := repository.Save(ctx, credential); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, found, err := repository.Load(ctx)
	if err != nil || !found || string(got.Salt) != "salt" || string(got.Hash) != "hash" || got.Iterations != credential.Iterations {
		t.Fatalf("Load() = %+v, found %v, err %v", got, found, err)
	}
}

func TestAuditRepositoryRoundTripAndFilters(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	repository := db.Audit()
	now := time.Now().UTC().Truncate(time.Millisecond)
	events := []control.AuditEvent{
		{Action: "egress.save_node", Actor: "alice", Source: "http", Detail: map[string]any{"revision": "r1"}, CreatedAt: now.Add(-time.Second)},
		{Action: "routes.set_mode", Actor: "bob", Source: "cli", Detail: map[string]any{"mode": "rules"}, CreatedAt: now},
	}
	for _, event := range events {
		if _, err := repository.Append(ctx, event); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}
	all, err := repository.List(ctx, control.AuditFilter{Limit: 10})
	if err != nil || len(all) != 2 || all[0].Action != "routes.set_mode" {
		t.Fatalf("List() = %+v, %v", all, err)
	}
	if all[0].Detail["mode"] != "rules" || all[0].CreatedAt.IsZero() {
		t.Fatalf("decoded event = %+v", all[0])
	}
	filtered, err := repository.List(ctx, control.AuditFilter{Actor: "alice", Since: now.Add(-2 * time.Second), Limit: 10})
	if err != nil || len(filtered) != 1 || filtered[0].Action != "egress.save_node" {
		t.Fatalf("filtered List() = %+v, %v", filtered, err)
	}
}

func openTestDB(t *testing.T) *DB {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "boxd.sqlite")
	db, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	return db
}
