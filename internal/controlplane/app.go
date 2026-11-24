// Package controlplane implements the public control application contract.
package controlplane

import (
	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

// App composes the initial control-plane use cases. Additional business modules
// are attached here as their public APIs become stable.
type App struct {
	kernel        control.KernelAPI
	config        control.ConfigAPI
	setup         control.SetupAPI
	inbounds      control.InboundAPI
	egress        control.EgressAPI
	subscriptions control.SubscriptionAPI
	scheduler     control.SchedulerAPI
	nodes         control.NodeAPI
	policies      control.PolicyAPI
	routes        control.RouteAPI
	dns           control.DNSAPI
	ruleSets      control.RuleSetAPI
	settings      control.SettingsAPI
	observability control.ObservabilityAPI
	security      control.SecurityAPI
	audit         control.AuditAPI
}

var _ control.Application = (*App)(nil)

// Dependencies contains optional outbound ports used by application modules.
// A missing dependency keeps CRUD available and returns an explicit error only
// for operations that require the missing capability.
type Dependencies struct {
	Downloader           control.Downloader
	SubscriptionParser   control.SubscriptionParser
	NodeCodec            control.NodeCodec
	NodeProber           control.NodeProber
	DNSProber            control.DNSProber
	RuleSetRepository    control.RuleSetRepository
	TestResultRepository control.TestResultRepository
	SecurityRepository   control.SecurityRepository
	AuditRepository      control.AuditRepository
}

// New constructs the application facade through constructor injection.
func New(kernelManager kernel.Manager, configStore kernel.ConfigStore, settingsRepository control.SettingsRepository, profileService *ProfileService) *App {
	return NewWithDependencies(kernelManager, configStore, settingsRepository, profileService, Dependencies{})
}

// NewWithDependencies constructs the full application facade. The composition
// root calls this function; focused tests can use New for a smaller surface.
func NewWithDependencies(kernelManager kernel.Manager, configStore kernel.ConfigStore, settingsRepository control.SettingsRepository, profileService *ProfileService, dependencies Dependencies) *App {
	routes := NewRouteService(profileService)
	dns := NewDNSService(profileService, kernelManager)
	audit := NewAuditService(dependencies.AuditRepository)
	settingsService := NewSettingsService(settingsRepository)
	settingsService.SetAuditor(audit)
	subscriptions := NewSubscriptionService(profileService, dependencies.Downloader, dependencies.SubscriptionParser, settingsService)
	if profileService != nil {
		profileService.SetAuditor(audit)
	}
	security := NewSecurityService(dependencies.SecurityRepository)
	security.SetAuditor(audit)
	kernelService := NewKernelService(kernelManager)
	kernelService.SetAuditor(audit)
	configService := NewConfigService(kernelManager, configStore, profileService)
	configService.SetAuditor(audit)
	ruleSetService := NewRuleSetService(profileService, dependencies.RuleSetRepository, dependencies.Downloader)
	ruleSetService.SetAuditor(audit)
	nodeService := NewNodeService(profileService, kernelManager, dependencies.TestResultRepository, dependencies.NodeCodec, dependencies.NodeProber)
	nodeService.SetAuditor(audit)
	dns.SetAuditor(audit)
	dns.SetProber(dependencies.DNSProber)
	return &App{
		kernel:        kernelService,
		config:        configService,
		setup:         NewSetupService(profileService, kernelManager),
		inbounds:      NewInboundService(profileService),
		egress:        NewEgressService(profileService, kernelManager),
		subscriptions: subscriptions,
		scheduler:     NewSchedulerService(subscriptions, SchedulerOptions{Settings: settingsService}),
		nodes:         nodeService,
		policies:      NewPolicyService(routes, dns),
		routes:        routes,
		dns:           dns,
		ruleSets:      ruleSetService,
		settings:      settingsService,
		observability: NewObservabilityService(kernelManager),
		security:      security,
		audit:         audit,
	}
}

func (a *App) Kernel() control.KernelAPI               { return a.kernel }
func (a *App) Config() control.ConfigAPI               { return a.config }
func (a *App) Setup() control.SetupAPI                 { return a.setup }
func (a *App) Inbounds() control.InboundAPI            { return a.inbounds }
func (a *App) Egress() control.EgressAPI               { return a.egress }
func (a *App) Subscriptions() control.SubscriptionAPI  { return a.subscriptions }
func (a *App) Scheduler() control.SchedulerAPI         { return a.scheduler }
func (a *App) Nodes() control.NodeAPI                  { return a.nodes }
func (a *App) Policies() control.PolicyAPI             { return a.policies }
func (a *App) Routes() control.RouteAPI                { return a.routes }
func (a *App) DNS() control.DNSAPI                     { return a.dns }
func (a *App) RuleSets() control.RuleSetAPI            { return a.ruleSets }
func (a *App) Settings() control.SettingsAPI           { return a.settings }
func (a *App) Observability() control.ObservabilityAPI { return a.observability }
func (a *App) Security() control.SecurityAPI           { return a.security }
func (a *App) Audit() control.AuditAPI                 { return a.audit }
