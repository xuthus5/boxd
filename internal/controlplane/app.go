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
	nodes         control.NodeAPI
	policies      control.PolicyAPI
	routes        control.RouteAPI
	dns           control.DNSAPI
	ruleSets      control.RuleSetAPI
	settings      control.SettingsAPI
	observability control.ObservabilityAPI
	backup        control.BackupAPI
	security      control.SecurityAPI
}

var _ control.Application = (*App)(nil)

// Dependencies contains optional outbound ports used by application modules.
// A missing dependency keeps CRUD available and returns an explicit error only
// for operations that require the missing capability.
type Dependencies struct {
	Downloader           control.Downloader
	SubscriptionParser   control.SubscriptionParser
	RuleSetRepository    control.RuleSetRepository
	TestResultRepository control.TestResultRepository
	BackupStore          control.BackupStore
	BackupDir            string
	SecurityRepository   control.SecurityRepository
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
	return &App{
		kernel:        NewKernelService(kernelManager),
		config:        NewConfigService(kernelManager, configStore, profileService),
		setup:         NewSetupService(profileService, kernelManager),
		inbounds:      NewInboundService(profileService),
		egress:        NewEgressService(profileService, kernelManager),
		subscriptions: NewSubscriptionService(profileService, dependencies.Downloader, dependencies.SubscriptionParser),
		nodes:         NewNodeService(profileService, kernelManager, dependencies.TestResultRepository),
		policies:      NewPolicyService(routes, dns),
		routes:        routes,
		dns:           dns,
		ruleSets:      NewRuleSetService(profileService, dependencies.RuleSetRepository, dependencies.Downloader),
		settings:      NewSettingsService(settingsRepository),
		observability: NewObservabilityService(kernelManager),
		backup:        NewBackupService(dependencies.BackupStore, dependencies.BackupDir),
		security:      NewSecurityService(dependencies.SecurityRepository),
	}
}

func (a *App) Kernel() control.KernelAPI               { return a.kernel }
func (a *App) Config() control.ConfigAPI               { return a.config }
func (a *App) Setup() control.SetupAPI                 { return a.setup }
func (a *App) Inbounds() control.InboundAPI            { return a.inbounds }
func (a *App) Egress() control.EgressAPI               { return a.egress }
func (a *App) Subscriptions() control.SubscriptionAPI  { return a.subscriptions }
func (a *App) Nodes() control.NodeAPI                  { return a.nodes }
func (a *App) Policies() control.PolicyAPI             { return a.policies }
func (a *App) Routes() control.RouteAPI                { return a.routes }
func (a *App) DNS() control.DNSAPI                     { return a.dns }
func (a *App) RuleSets() control.RuleSetAPI            { return a.ruleSets }
func (a *App) Settings() control.SettingsAPI           { return a.settings }
func (a *App) Observability() control.ObservabilityAPI { return a.observability }
func (a *App) Backup() control.BackupAPI               { return a.backup }
func (a *App) Security() control.SecurityAPI           { return a.security }
