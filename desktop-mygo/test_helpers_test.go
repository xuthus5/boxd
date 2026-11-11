package main

import (
	"context"
	"sync"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"

	"github.com/xuthus5/boxd/desktop-mygo/internal/runtime"
)

type fakeBackend struct {
	mu               sync.Mutex
	snapshot         snapshot
	snapshotErr      error
	actionErr        error
	configText       string
	configErr        error
	configBody       string
	sectionText      string
	sectionErr       error
	diagnostics      model.ConfigDiagnostics
	history          []model.ConfigApplyEvent
	snapshotText     string
	ruleSets         []model.RuleSetStatusItem
	ruleSetAuto      model.RuleSetAutoUpdate
	setupStatus      core.SetupStatus
	supportBundle    supportBundleCollection
	supportBundleErr error
	calls            []string
	lastPrefs        model.UIPreferences
	lastNode         service.NodeInput
	lastOldTag       string
	lastBatch        service.TestBatchRequest
	lastSub          service.SubscriptionInput
	lastSubID        string
	lastSelect       string
	lastFilters      service.ConnectionCloseFilters
	lastSection      string
	lastBody         string
	lastRuleSet      core.RuleSetUpdateRequest
	lastAuto         model.RuleSetAutoUpdate
	lastPath         string
	lastCurrent      string
	lastNew          string
}

func (f *fakeBackend) ValidateConfigSection(_ context.Context, section, body string) error {
	f.record("validate-config-section")
	f.mu.Lock()
	f.lastSection = section
	f.lastBody = body
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) Snapshot(context.Context) (snapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.snapshot, f.snapshotErr
}

func (f *fakeBackend) LoadConfig(context.Context) (string, error) {
	f.record("load-config")
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.configText, f.configErr
}

func (f *fakeBackend) LoadConfigSection(_ context.Context, section string) (string, error) {
	f.record("load-config-section")
	f.mu.Lock()
	f.lastSection = section
	f.mu.Unlock()
	return f.sectionText, f.sectionErr
}

func (f *fakeBackend) ValidateConfig(_ context.Context, body string) error {
	f.record("validate-config")
	f.mu.Lock()
	f.configBody = body
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) ApplyConfig(_ context.Context, body string) error {
	f.record("apply-config")
	f.mu.Lock()
	f.configBody = body
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) ApplyConfigSection(_ context.Context, section, body string) error {
	f.record("apply-config-section")
	f.mu.Lock()
	f.lastSection = section
	f.lastBody = body
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) InstallDefaultConfigSection(context.Context, string) error {
	f.record("install-config-defaults")
	return f.actionErr
}

func (f *fakeBackend) ConfigDiagnostics(context.Context) (model.ConfigDiagnostics, error) {
	f.record("config-diagnostics")
	return f.diagnostics, f.actionErr
}

func (f *fakeBackend) ConfigApplyHistory(context.Context) ([]model.ConfigApplyEvent, error) {
	f.record("config-history")
	return f.history, f.actionErr
}

func (f *fakeBackend) ConfigSnapshot(context.Context, string) (string, error) {
	f.record("config-snapshot")
	return f.snapshotText, f.actionErr
}

func (f *fakeBackend) RestoreConfigSnapshot(context.Context, string) error {
	f.record("restore-config")
	return f.actionErr
}

func (f *fakeBackend) SetupStatus(context.Context) (core.SetupStatus, error) {
	f.record("setup-status")
	return f.setupStatus, f.actionErr
}

func (f *fakeBackend) RuleSetStatus(context.Context) ([]model.RuleSetStatusItem, error) {
	f.record("ruleset-status")
	return f.ruleSets, f.actionErr
}

func (f *fakeBackend) UpdateRuleSets(_ context.Context, req core.RuleSetUpdateRequest) (model.RuleSetUpdateResponse, error) {
	f.record("ruleset-update")
	f.mu.Lock()
	f.lastRuleSet = req
	f.mu.Unlock()
	return model.RuleSetUpdateResponse{UpdatedCount: 1}, f.actionErr
}

func (f *fakeBackend) RuleSetAutoUpdate(context.Context) (model.RuleSetAutoUpdate, error) {
	f.record("ruleset-auto")
	return f.ruleSetAuto, f.actionErr
}

func (f *fakeBackend) SetRuleSetAutoUpdate(_ context.Context, cfg model.RuleSetAutoUpdate) error {
	f.record("set-ruleset-auto")
	f.mu.Lock()
	f.lastAuto = cfg
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) StartKernel(context.Context) error {
	f.record("start")
	return f.actionErr
}

func (f *fakeBackend) StopKernel(context.Context) error {
	f.record("stop")
	return f.actionErr
}

func (f *fakeBackend) RestartKernel(context.Context) error {
	f.record("restart")
	return f.actionErr
}

func (f *fakeBackend) CloseConnection(context.Context, int64) error {
	f.record("close-connection")
	return f.actionErr
}

func (f *fakeBackend) CloseConnections(_ context.Context, filters service.ConnectionCloseFilters) (map[string]any, error) {
	f.record("close-connections")
	f.mu.Lock()
	f.lastFilters = filters
	f.mu.Unlock()
	return map[string]any{"closed": 1}, f.actionErr
}

func (f *fakeBackend) RefreshSubscription(context.Context, string) error {
	f.record("refresh-subscription")
	return f.actionErr
}

func (f *fakeBackend) RefreshAllSubscriptions(context.Context) (service.RefreshAllResult, error) {
	f.record("refresh-all-subscriptions")
	return service.RefreshAllResult{}, f.actionErr
}

func (f *fakeBackend) CreateSubscription(_ context.Context, input service.SubscriptionInput) (*model.Subscription, error) {
	f.record("create-subscription")
	f.mu.Lock()
	f.lastSub = input
	f.mu.Unlock()
	return &model.Subscription{ID: "created"}, f.actionErr
}

func (f *fakeBackend) UpdateSubscription(_ context.Context, id string, input service.SubscriptionInput) error {
	f.record("update-subscription")
	f.mu.Lock()
	f.lastSubID = id
	f.lastSub = input
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) DeleteSubscription(_ context.Context, id string) error {
	f.record("delete-subscription")
	f.mu.Lock()
	f.lastSubID = id
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) SelectSubscription(_ context.Context, id, tag string) error {
	f.record("select-subscription")
	f.mu.Lock()
	f.lastSubID = id
	f.lastSelect = tag
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) ImportNode(_ context.Context, link string) (model.ImportResult, error) {
	f.record("import-node")
	return model.ImportResult{Tag: link, Type: "vless", Server: "example.com", Port: 443}, f.actionErr
}

func (f *fakeBackend) SaveNode(_ context.Context, input service.NodeInput) error {
	f.record("save-node")
	f.mu.Lock()
	f.lastNode = input
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) UpdateNode(_ context.Context, oldTag string, input service.NodeInput) error {
	f.record("update-node")
	f.mu.Lock()
	f.lastOldTag = oldTag
	f.lastNode = input
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) DeleteNode(context.Context, string) error {
	f.record("delete-node")
	return f.actionErr
}

func (f *fakeBackend) TestNode(context.Context, service.TestRequest) (model.TestResult, error) {
	f.record("test-node")
	return model.TestResult{Success: true, LatencyMs: 12}, f.actionErr
}

func (f *fakeBackend) TestNodes(_ context.Context, input service.TestBatchRequest) ([]model.TestResult, error) {
	f.record("test-nodes")
	f.mu.Lock()
	f.lastBatch = input
	f.mu.Unlock()
	return []model.TestResult{{Success: true, LatencyMs: 12}}, f.actionErr
}

func (f *fakeBackend) SetKernelAutostart(context.Context, bool) error {
	f.record("kernel-autostart")
	return f.actionErr
}

func (f *fakeBackend) SetUIPreferences(_ context.Context, prefs model.UIPreferences) error {
	f.record("preferences")
	f.mu.Lock()
	f.lastPrefs = prefs
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) SetTestURL(context.Context, string) error {
	f.record("test-url")
	return f.actionErr
}

func (f *fakeBackend) ChangePassword(_ context.Context, currentPassword, newPassword string) error {
	f.record("change-password")
	f.mu.Lock()
	f.lastCurrent = currentPassword
	f.lastNew = newPassword
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) CreateBackup(_ context.Context, path string) error {
	f.record("create-backup")
	f.mu.Lock()
	f.lastPath = path
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) ImportLink(context.Context, string) error {
	f.record("import-link")
	return f.actionErr
}

func (f *fakeBackend) SupportBundle(context.Context) (supportBundleCollection, error) {
	f.record("support-bundle")
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.supportBundle, f.supportBundleErr
}

func (f *fakeBackend) record(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, name)
}

func (f *fakeBackend) callCount(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, call := range f.calls {
		if call == name {
			count++
		}
	}
	return count
}

func testConfig() runtime.Config {
	return runtime.Config{
		Mode: "embedded", DataDir: "/tmp/boxd-test", ConfigPath: "/tmp/boxd-test/config.json",
		RefreshInterval: 1,
	}
}
