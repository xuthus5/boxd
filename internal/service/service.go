package service

import (
	"context"
	"errors"
	"net"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/core"
)

// Deps 聚合 Service 所需全部传输无关依赖，供 HTTP 与 Wails 两种形态共享。
type Deps struct {
	DB               *bbolt.DB
	ConfigPath       string
	DataDir          string
	Username         string
	Version          string
	Settings         *core.SettingsManager
	Instance         *core.SBInstance
	NodeManager      *core.NodeManager
	SubManager       *core.SubscriptionManager
	RuleSetInstaller *core.LoyalsoldierRuleSetInstaller
	RuleSetUpdater   *core.RuleSetUpdater
	KernelLogWriter  *core.LogWriter
	AppLogWriter     *core.LogWriter
	ApplyHistory     *core.ConfigApplyHistoryManager
	RouteMetadata    *core.RouteRuleMetadataManager
	TestURL          func() string
}

// ServiceSet 聚合各域子服务，供 handler 与 Wails service 复用。
type ServiceSet struct {
	Deps     Deps
	config   *Config
	network  *Network
	kernel   *Kernel
	service  *ServiceControl
	dnsProbe *DNSProbe
}

// New 基于依赖构造服务集合。
func New(deps Deps) *ServiceSet {
	installers := ConfigInstaller{
		RuleSetInstaller:      deps.RuleSetInstaller,
		OutboundInstaller:     core.NewDefaultOutboundsInstaller(),
		InboundInstaller:      core.NewDefaultInboundsInstaller(),
		RouteInstaller:        core.NewDefaultRouteInstaller(),
		DNSInstaller:          core.NewDefaultDNSInstaller(),
		ExperimentalInstaller: core.NewDefaultExperimentalInstaller(),
		ApplyHistory:          deps.ApplyHistory,
		RouteMetadata:         deps.RouteMetadata,
	}
	svc := &ServiceSet{Deps: deps}
	svc.config = newConfig(deps.ConfigPath, restartAdapter{deps.Instance}, installers)
	svc.network = &Network{}
	svc.kernel = &Kernel{version: deps.Version}
	svc.service = &ServiceControl{instance: deps.Instance}
	svc.dnsProbe = &DNSProbe{}
	return svc
}

type restartAdapter struct{ instance *core.SBInstance }

func (a restartAdapter) Restart() error {
	if a.instance == nil {
		return nil
	}
	return a.instance.Restart()
}

// Config 返回配置用例服务。
func (s *ServiceSet) Config() *Config { return s.config }

// Network 返回网卡用例服务。
func (s *ServiceSet) Network() *Network { return s.network }

// Kernel 返回内核信息用例服务。
func (s *ServiceSet) Kernel() *Kernel { return s.kernel }

// Service 返回内核控制用例服务。
func (s *ServiceSet) Service() *ServiceControl { return s.service }

// DNSProbe 返回 DNS 探测用例服务。
func (s *ServiceSet) DNSProbe() *DNSProbe { return s.dnsProbe }

// Test 返回节点测速用例服务。
func (s *ServiceSet) Test() *TestService {
	return NewTestService(s.Deps.TestURL, s.Deps.NodeManager, testDialer{s.Deps.Instance})
}

// Settings 返回应用设置用例服务。
func (s *ServiceSet) Settings() *SettingsService {
	return NewSettingsService(s.Deps.Settings, s.Deps.Username)
}

// Subscriptions 返回订阅管理用例服务。
func (s *ServiceSet) Subscriptions() *SubscriptionService {
	return NewSubscriptionService(s.Deps.SubManager, s.Deps.NodeManager, s.Deps.ConfigPath, restartAdapter{s.Deps.Instance})
}

// Auth 返回认证用例服务。
func (s *ServiceSet) Auth() *AuthService {
	return NewAuthService(s.Deps.Username, "", s.Deps.Settings)
}

// Runtime 返回运行时交互用例服务。
func (s *ServiceSet) Runtime() *RuntimeService {
	return NewRuntimeService(s.Deps.Instance)
}

// Import 返回节点导入用例服务。
func (s *ServiceSet) Import() *ImportService {
	return NewImportService(s.Deps.NodeManager, s.Deps.SubManager, s.Deps.ConfigPath, restartAdapter{s.Deps.Instance})
}

// Health 返回健康检查用例服务。
func (s *ServiceSet) Health() *HealthService {
	return &HealthService{}
}

// Stats 返回流量/连接用例服务。
func (s *ServiceSet) Stats() *StatsService {
	return NewStatsService(s.Deps.Instance)
}

// Backup 返回备份导出用例服务。
func (s *ServiceSet) Backup() *BackupService {
	return NewBackupService(s.Deps.DB, s.Deps.ConfigPath, s.Deps.Version)
}

// RuleSets 返回规则集状态/更新/自动更新用例服务。
func (s *ServiceSet) RuleSets() *RuleSetService {
	return NewRuleSetService(s.Deps.RuleSetUpdater, s.Deps.Settings)
}

// testDialer 适配 *core.SBInstance 为 outboundDialer。
type testDialer struct{ instance *core.SBInstance }

func (d testDialer) DialOutbound(ctx context.Context, tag, network, addr string) (net.Conn, error) {
	if d.instance == nil {
		return nil, errors.New("service is not running")
	}
	return d.instance.DialOutbound(ctx, tag, network, addr)
}

func (d testDialer) OutboundDelay(ctx context.Context, tag, link string, timeout time.Duration) (uint16, error) {
	if d.instance == nil {
		return 0, errors.New("service is not running")
	}
	return d.instance.OutboundDelay(ctx, tag, link, timeout)
}
