package controlplane_test

import (
	"context"
	"sync"
	"testing"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/internal/adapters/outbound/nodelink"
	"github.com/xuthus5/boxd/internal/controlplane"
	"github.com/xuthus5/boxd/internal/kernel/fake"
	"github.com/xuthus5/boxd/internal/kernel/manager"
	"github.com/xuthus5/boxd/internal/kernel/memstore"
	"github.com/xuthus5/boxd/kernel"
)

func TestSubscriptionRefreshAtomicallyReplacesNodes(t *testing.T) {
	ctx := context.Background()
	downloader := &recordingDownloader{result: control.DownloadResult{Body: []byte(`{"outbounds":[]}`), ETag: `"v1"`}}
	parser := &staticSubscriptionParser{nodes: []control.Node{
		{Name: "HK", Protocol: "shadowsocks", Server: "127.0.0.1", Port: 8388},
		{Name: "JP", Protocol: "trojan", Server: "127.0.0.2", Port: 443},
	}}
	application := newApplicationWithDependencies(t, controlplane.Dependencies{
		Downloader:         downloader,
		SubscriptionParser: parser,
	})

	subscription, err := application.Subscriptions().Save(ctx, control.Subscription{
		Name: "Provider", URL: "https://example.com/subscription", Enabled: true, IntervalSeconds: 3600,
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	refresh, err := application.Subscriptions().Refresh(ctx, subscription.ID)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if refresh.Added != 2 || refresh.Updated != 0 || refresh.Removed != 0 || refresh.NotModified {
		t.Fatalf("first refresh = %+v", refresh)
	}
	if refresh.Subscription.ETag != `"v1"` || refresh.Subscription.LastRefreshedAt.IsZero() {
		t.Fatalf("subscription metadata = %+v", refresh.Subscription)
	}
	nodes, err := application.Nodes().List(ctx, subscription.ID)
	if err != nil || len(nodes) != 2 {
		t.Fatalf("List() = %+v, %v", nodes, err)
	}
	for _, node := range nodes {
		if node.SubscriptionID != subscription.ID || node.ID == "" {
			t.Fatalf("refreshed node = %+v", node)
		}
	}

	downloader.set(control.DownloadResult{NotModified: true, ETag: `"v1"`})
	refresh, err = application.Subscriptions().Refresh(ctx, subscription.ID)
	if err != nil {
		t.Fatalf("conditional Refresh() error = %v", err)
	}
	if !refresh.NotModified || refresh.Added != 0 || refresh.Updated != 0 || refresh.Removed != 0 {
		t.Fatalf("conditional refresh = %+v", refresh)
	}
	lastRequest := downloader.lastRequest()
	if lastRequest.ETag != `"v1"` {
		t.Fatalf("conditional ETag = %q", lastRequest.ETag)
	}

	downloader.set(control.DownloadResult{Body: []byte(`{"outbounds":[]}`), ETag: `"v2"`})
	refresh, err = application.Subscriptions().Refresh(ctx, subscription.ID)
	if err != nil {
		t.Fatalf("second Refresh() error = %v", err)
	}
	if refresh.Updated != 2 || refresh.Added != 0 || refresh.Removed != 0 {
		t.Fatalf("second refresh = %+v", refresh)
	}

	if err := application.Subscriptions().Delete(ctx, subscription.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	nodes, err = application.Nodes().List(ctx, subscription.ID)
	if err != nil || len(nodes) != 0 {
		t.Fatalf("nodes after delete = %+v, %v", nodes, err)
	}
}

func TestNodeImportSaveFilterAndDelay(t *testing.T) {
	ctx := context.Background()
	results := &memoryTestResults{}
	application := newApplicationWithDependencies(t, controlplane.Dependencies{TestResultRepository: results})
	if err := application.Kernel().Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	importResult, err := application.Nodes().Import(ctx, []control.Node{
		{Name: "HK", Protocol: "shadowsocks", Server: "127.0.0.1", Port: 8388, Options: map[string]any{"method": "aes-128-gcm"}},
		{Name: "JP", Protocol: "trojan", Server: "127.0.0.2", Port: 443},
	})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if importResult.Added != 2 || importResult.Updated != 0 || len(importResult.Nodes) != 2 {
		t.Fatalf("Import() = %+v", importResult)
	}
	for _, node := range importResult.Nodes {
		if node.ID == "" {
			t.Fatalf("imported node has no ID: %+v", node)
		}
	}
	importResult, err = application.Nodes().Import(ctx, importResult.Nodes)
	if err != nil || importResult.Updated != 2 || importResult.Added != 0 {
		t.Fatalf("second Import() = %+v, %v", importResult, err)
	}
	nodes, err := application.Nodes().List(ctx, "")
	if err != nil || len(nodes) != 2 {
		t.Fatalf("List() = %+v, %v", nodes, err)
	}

	delay, err := application.Nodes().Delay(ctx, nodes[0].ID, "")
	if err != nil {
		t.Fatalf("Delay() error = %v", err)
	}
	if delay.NodeID != nodes[0].ID || delay.URL == "" || delay.TestedAt.IsZero() {
		t.Fatalf("Delay() = %+v", delay)
	}
	history, err := application.Nodes().TestResults(ctx, nodes[0].ID, 10)
	if err != nil || len(history) != 1 || history[0].Kind != "delay" {
		t.Fatalf("TestResults() = %+v, %v", history, err)
	}
}

func TestSubscriptionSchedulerRefreshesDueSubscriptions(t *testing.T) {
	ctx := context.Background()
	downloader := &recordingDownloader{result: control.DownloadResult{Body: []byte(`{"outbounds":[]}`), ETag: `"scheduler-v1"`}}
	parser := &staticSubscriptionParser{nodes: []control.Node{
		{Name: "HK", Protocol: "shadowsocks", Server: "127.0.0.1", Port: 8388},
	}}
	application := newApplicationWithDependencies(t, controlplane.Dependencies{
		Downloader:         downloader,
		SubscriptionParser: parser,
	})
	subscription, err := application.Subscriptions().Save(ctx, control.Subscription{
		Name: "Scheduled", URL: "https://example.com/subscription", Enabled: true, IntervalSeconds: 3600,
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	status, err := application.Scheduler().Status(ctx)
	if err != nil || status.Running || status.IntervalSeconds <= 0 {
		t.Fatalf("initial Status() = %+v, %v", status, err)
	}
	results, err := application.Scheduler().RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(results) != 1 || results[0].Subscription.ID != subscription.ID || results[0].Added != 1 {
		t.Fatalf("first RunOnce() = %+v", results)
	}
	results, err = application.Scheduler().RunOnce(ctx)
	if err != nil || len(results) != 0 {
		t.Fatalf("second RunOnce() = %+v, %v, want no due subscriptions", results, err)
	}
	status, err = application.Scheduler().Status(ctx)
	if err != nil || status.LastRunAt.IsZero() || status.LastError != "" {
		t.Fatalf("final Status() = %+v, %v", status, err)
	}
	if err := application.Scheduler().Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if status, err := application.Scheduler().Status(ctx); err != nil || !status.Running || status.NextRunAt.IsZero() {
		t.Fatalf("running Status() = %+v, %v", status, err)
	}
	if err := application.Scheduler().Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if status, err := application.Scheduler().Status(ctx); err != nil || status.Running {
		t.Fatalf("stopped Status() = %+v, %v", status, err)
	}
}

func TestNodeLinkImportAndExport(t *testing.T) {
	ctx := context.Background()
	application := newApplicationWithDependencies(t, controlplane.Dependencies{NodeCodec: nodelink.New()})
	result, err := application.Nodes().ImportLinks(ctx, []string{
		"ss://YWVzLTEyOC1nY206c2VjcmV0@127.0.0.1:8388#SS%20Node",
		"trojan://secret@127.0.0.2:443?sni=example.com#Trojan",
	})
	if err != nil {
		t.Fatalf("ImportLinks() error = %v", err)
	}
	if result.Added != 2 || len(result.Nodes) != 2 {
		t.Fatalf("ImportLinks() = %+v", result)
	}
	link, err := application.Nodes().EncodeLink(ctx, result.Nodes[0])
	if err != nil || link == "" {
		t.Fatalf("EncodeLink() = %q, %v", link, err)
	}
	decoded, err := application.Nodes().DecodeLink(ctx, link)
	if err != nil || decoded.Protocol != result.Nodes[0].Protocol || decoded.Server != result.Nodes[0].Server {
		t.Fatalf("DecodeLink() = %+v, %v", decoded, err)
	}
}

func TestRuleSetDefinitionAndRefreshStatusAreSeparated(t *testing.T) {
	ctx := context.Background()
	downloader := &recordingDownloader{result: control.DownloadResult{Body: []byte("rule-set"), ETag: `"rules-v1"`}}
	statuses := newMemoryRuleSetStatuses()
	application := newApplicationWithDependencies(t, controlplane.Dependencies{
		Downloader:        downloader,
		RuleSetRepository: statuses,
	})

	defaults, err := application.RuleSets().List(ctx)
	if err != nil || len(defaults) != 3 {
		t.Fatalf("default rule sets = %+v, %v", defaults, err)
	}
	custom, err := application.RuleSets().Save(ctx, control.RuleSetDefinition{
		Name: "custom-ads", URL: "https://example.com/ads.srs", Format: control.RuleSetFormatBinary, Enabled: true,
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if custom.Builtin || custom.Name != "custom-ads" {
		t.Fatalf("custom rule set = %+v", custom)
	}
	refreshed, err := application.RuleSets().Refresh(ctx, custom.Name)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if refreshed.Status.SHA256 == "" || refreshed.Status.ETag != `"rules-v1"` || refreshed.Status.UpdatedAt.IsZero() {
		t.Fatalf("refreshed status = %+v", refreshed.Status)
	}
	if err := application.RuleSets().Delete(ctx, "geosite-cn"); !hasCode(err, kernel.ErrCodeConflict) {
		t.Fatalf("Delete(builtin) error = %v", err)
	}
	if err := application.RuleSets().Delete(ctx, custom.Name); err != nil {
		t.Fatalf("Delete(custom) error = %v", err)
	}
	if _, found, err := statuses.Get(ctx, custom.Name); err != nil || found {
		t.Fatalf("status after delete = found %v, err %v", found, err)
	}
}

func TestPolicyAggregateFacade(t *testing.T) {
	ctx := context.Background()
	application := newApplicationWithDependencies(t, controlplane.Dependencies{})
	policies, err := application.Policies().Get(ctx)
	if err != nil {
		t.Fatalf("Policies().Get() error = %v", err)
	}
	if policies.Routes.Mode != control.RouteModeRules || policies.DNS.Strategy == "" || application.Policies().Routes() == nil || application.Policies().DNS() == nil {
		t.Fatalf("policy snapshot = %+v", policies)
	}
}

func TestSecurityCredentialAndSessionLifecycle(t *testing.T) {
	ctx := context.Background()
	repository := &memorySecurityRepository{}
	application := newApplicationWithDependencies(t, controlplane.Dependencies{SecurityRepository: repository})

	status, err := application.Security().Status(ctx)
	if err != nil || status.Initialized || status.Mode != control.SecurityModeLocal {
		t.Fatalf("initial Status() = %+v, %v", status, err)
	}
	if _, err := application.Security().Initialize(ctx, "short"); !hasCode(err, kernel.ErrCodeInvalidConfig) {
		t.Fatalf("short password error = %v", err)
	}
	first, err := application.Security().Initialize(ctx, "correct horse battery staple")
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if first.Token == "" || first.ExpiresAt.IsZero() || len(repository.credential.Hash) == 0 {
		t.Fatalf("initialized session = %+v, credential = %+v", first, repository.credential)
	}
	if _, err := application.Security().Initialize(ctx, "another password"); !hasCode(err, kernel.ErrCodeConflict) {
		t.Fatalf("second Initialize() error = %v", err)
	}
	if _, err := application.Security().Login(ctx, "wrong password"); !hasCode(err, kernel.ErrCodeUnauthorized) {
		t.Fatalf("wrong password error = %v", err)
	}
	second, err := application.Security().Login(ctx, "correct horse battery staple")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	validated, err := application.Security().ValidateSession(ctx, second.Token)
	if err != nil || validated.Token != second.Token {
		t.Fatalf("ValidateSession() = %+v, %v", validated, err)
	}
	if err := application.Security().Logout(ctx, second.Token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := application.Security().ValidateSession(ctx, second.Token); !hasCode(err, kernel.ErrCodeUnauthorized) {
		t.Fatalf("ValidateSession() after logout error = %v", err)
	}
	status, err = application.Security().Status(ctx)
	if err != nil || !status.Initialized || status.Mode != control.SecurityModePassword || status.SessionCount != 1 {
		t.Fatalf("final Status() = %+v, %v", status, err)
	}
}

func newApplicationWithDependencies(t *testing.T, dependencies controlplane.Dependencies) control.Application {
	t.Helper()
	ctx := context.Background()
	store := &memstore.ConfigStore{}
	kernelManager, err := manager.New(ctx, manager.Options{Factory: fake.NewFactory(), Store: store})
	if err != nil {
		t.Fatalf("manager.New() error = %v", err)
	}
	t.Cleanup(func() { _ = kernelManager.Close() })
	profileService := controlplane.NewProfileService(kernelManager, newMemoryProfiles(), staticCompiler{})
	application := controlplane.NewWithDependencies(kernelManager, store, newMemorySettings(), profileService, dependencies)
	if _, err := application.Setup().EnsureDefaults(ctx); err != nil {
		t.Fatalf("EnsureDefaults() error = %v", err)
	}
	return application
}

type recordingDownloader struct {
	mu      sync.Mutex
	result  control.DownloadResult
	err     error
	request control.DownloadRequest
}

func (d *recordingDownloader) Get(_ context.Context, request control.DownloadRequest) (control.DownloadResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.request = request
	return d.result, d.err
}

func (d *recordingDownloader) set(result control.DownloadResult) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.result = result
}

func (d *recordingDownloader) lastRequest() control.DownloadRequest {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.request
}

type staticSubscriptionParser struct {
	nodes []control.Node
	err   error
}

func (p *staticSubscriptionParser) Parse(context.Context, control.DownloadResult) ([]control.Node, error) {
	return append([]control.Node(nil), p.nodes...), p.err
}

type memoryRuleSetStatuses struct {
	mu      sync.Mutex
	records map[string]control.RuleSetStatus
}

func newMemoryRuleSetStatuses() *memoryRuleSetStatuses {
	return &memoryRuleSetStatuses{records: make(map[string]control.RuleSetStatus)}
}

func (r *memoryRuleSetStatuses) List(context.Context) ([]control.RuleSetStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]control.RuleSetStatus, 0, len(r.records))
	for _, record := range r.records {
		result = append(result, record)
	}
	return result, nil
}

func (r *memoryRuleSetStatuses) Get(_ context.Context, name string) (control.RuleSetStatus, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, found := r.records[name]
	return record, found, nil
}

func (r *memoryRuleSetStatuses) Upsert(_ context.Context, status control.RuleSetStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[status.Name] = status
	return nil
}

func (r *memoryRuleSetStatuses) Delete(_ context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.records, name)
	return nil
}

type memoryTestResults struct {
	mu      sync.Mutex
	records []control.NodeTestResult
}

func (r *memoryTestResults) Save(_ context.Context, result control.NodeTestResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, result)
	return nil
}

func (r *memoryTestResults) List(_ context.Context, nodeID string, limit int) ([]control.NodeTestResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]control.NodeTestResult, 0)
	for index := len(r.records) - 1; index >= 0 && len(result) < limit; index-- {
		if r.records[index].NodeID == nodeID {
			result = append(result, r.records[index])
		}
	}
	return result, nil
}

type memorySecurityRepository struct {
	credential  control.SecurityCredential
	initialized bool
}

func (r *memorySecurityRepository) Load(context.Context) (control.SecurityCredential, bool, error) {
	return r.credential, r.initialized, nil
}

func (r *memorySecurityRepository) Save(_ context.Context, credential control.SecurityCredential) error {
	r.credential = credential
	r.initialized = true
	return nil
}
