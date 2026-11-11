package main

import (
	"github.com/egoist/mygo/ui/editor"

	"github.com/xuthus5/boxd/internal/model"
)

type configSectionSpec struct {
	Route       string
	Key         string
	Title       string
	Subtitle    string
	CanDefaults bool
}

var configSectionSpecs = []configSectionSpec{
	{Route: "/proxy/inbounds", Key: sectionInbounds, Title: "Inbounds", Subtitle: "Listeners exposed by the shared sing-box configuration.", CanDefaults: true},
	{Route: "/proxy/outbounds", Key: sectionOutbounds, Title: "Outbounds", Subtitle: "Protocols and proxy groups used to reach destinations.", CanDefaults: true},
	{Route: "/policy/route", Key: sectionRoute, Title: "Route", Subtitle: "Routing rules, final outbound and rule-set references.", CanDefaults: true},
	{Route: "/policy/dns", Key: sectionDNS, Title: "DNS", Subtitle: "DNS servers, rules and resolver strategy.", CanDefaults: true},
	{Route: "/advanced/endpoints", Key: sectionEndpoints, Title: "Endpoints", Subtitle: "WireGuard and other sing-box endpoints.", CanDefaults: false},
	{Route: "/advanced/services", Key: sectionServices, Title: "Services", Subtitle: "Kernel services such as Clash API, SSH and TUN helpers.", CanDefaults: false},
	{Route: "/advanced/certificate", Key: sectionCertificate, Title: "Certificate", Subtitle: "Certificate providers and kernel TLS defaults.", CanDefaults: false},
	{Route: "/advanced/log", Key: sectionLog, Title: "Log", Subtitle: "Kernel log level, timestamp and output options.", CanDefaults: false},
	{Route: "/advanced/ntp", Key: sectionNTP, Title: "NTP", Subtitle: "Clock synchronization and NTP detour settings.", CanDefaults: false},
	{Route: "/advanced/experimental", Key: sectionExperimental, Title: "Experimental", Subtitle: "Cache file, Clash API and experimental kernel features.", CanDefaults: true},
	{Route: "/advanced/network-namespaces", Key: sectionNetworkNamespaces, Title: "Network namespaces", Subtitle: "Shared network namespace resources.", CanDefaults: false},
	{Route: "/advanced/http-clients", Key: sectionHTTPClients, Title: "HTTP clients", Subtitle: "Shared HTTP clients for remote rule sets and downloads.", CanDefaults: false},
	{Route: "/advanced/certificate-providers", Key: sectionCertificateProvider, Title: "Certificate providers", Subtitle: "Shared certificate provider resources.", CanDefaults: false},
}

func configSectionSpecForRoute(route string) (configSectionSpec, bool) {
	for _, spec := range configSectionSpecs {
		if spec.Route == route {
			return spec, true
		}
	}
	return configSectionSpec{}, false
}

type configSectionState struct {
	spec          configSectionSpec
	editor        *editor.State
	baseline      string
	loaded        bool
	loading       bool
	attempted     bool
	itemEditOpen  bool
	itemEditIndex int
	itemPath      string
	itemDraft     string
	itemErr       string
}

func newConfigSectionState(spec configSectionSpec) *configSectionState {
	return &configSectionState{spec: spec}
}

func (s *configSectionState) editorState() *editor.State {
	if s.editor == nil {
		buffer := editor.NewBuffer("")
		buffer.SetLanguage(editor.LanguageFor("config.json"))
		s.editor = &editor.State{Buffer: buffer}
	}
	return s.editor
}

type configHistoryState struct {
	events    []model.ConfigApplyEvent
	loaded    bool
	loading   bool
	attempted bool

	previewOpen   bool
	previewID     string
	previewSource string
	preview       *editor.State
	previewLoad   bool
}

func (s *configHistoryState) previewState() *editor.State {
	if s.preview == nil {
		buffer := editor.NewBuffer("")
		buffer.SetLanguage(editor.LanguageFor("config.json"))
		s.preview = &editor.State{Buffer: buffer}
	}
	return s.preview
}

type ruleSetPageState struct {
	items     []model.RuleSetStatusItem
	loaded    bool
	loading   bool
	attempted bool

	savedAuto  model.RuleSetAutoUpdate
	draftAuto  model.RuleSetAutoUpdate
	autoLoaded bool
}

type securityFormState struct {
	current string
	next    string
	confirm string
}

func (a *application) configSectionForRoute(route string) (*configSectionState, bool) {
	spec, ok := configSectionSpecForRoute(route)
	if !ok {
		return nil, false
	}
	if a.configSections == nil {
		a.configSections = make(map[string]*configSectionState)
	}
	state := a.configSections[spec.Key]
	if state == nil {
		state = newConfigSectionState(spec)
		a.configSections[spec.Key] = state
	}
	return state, true
}
