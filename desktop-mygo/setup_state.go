package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/service"
)

const (
	setupInboundProxy    = "proxy"
	setupInboundTUN      = "tun"
	setupInboundPreserve = "preserve"
	setupIPv6Auto        = "auto"
	setupIPv6On          = "on"
	setupIPv6Off         = "off"
)

var (
	setupModuleIDs      = []string{"outbounds", "rule_sets", "inbounds", "dns", "route", "experimental"}
	setupInboundOptions = []string{setupInboundProxy, setupInboundTUN, setupInboundPreserve}
	setupIPv6Options    = []string{setupIPv6Auto, setupIPv6Off, setupIPv6On}
)

type setupPageState struct {
	status    core.SetupStatus
	loaded    bool
	loading   bool
	attempted bool

	dialogOpen bool
	draftReady bool
	modules    map[string]bool
	inbound    string
	ipv6       string
	reset      bool
	preview    *core.SetupPlan
	previewing bool
	applying   bool
	err        string
}

func (a *application) ensureSetup() {
	if !a.setup.attempted {
		a.loadSetup(false)
	}
}

func (a *application) loadSetup(force bool) {
	if a.backend == nil || a.setup.loading || (a.setup.attempted && !force) {
		return
	}
	a.setup.loading = true
	a.setup.attempted = true
	if a.win == nil {
		status, err := a.fetchSetupStatus()
		a.applySetupStatus(status, err)
		return
	}
	go func() {
		status, err := a.fetchSetupStatus()
		a.win.Update(func() { a.applySetupStatus(status, err) })
	}()
}

func (a *application) applySetupStatus(status core.SetupStatus, err error) {
	a.setup.loading = false
	if err != nil {
		a.setup.err = shortError(err)
		a.model.setActionError(err)
		return
	}
	a.setup.status = status
	a.setup.loaded = true
	a.setup.err = ""
	if a.setup.dialogOpen && !a.setup.draftReady {
		a.resetSetupDraft()
	}
}

func (a *application) openSetupDialog() {
	a.ensureSetup()
	a.setup.dialogOpen = true
	if a.setup.loaded {
		a.resetSetupDraft()
	}
}

func (a *application) closeSetupDialog() {
	if a.setup.previewing || a.setup.applying {
		return
	}
	a.setup.dialogOpen = false
}

func (a *application) resetSetupDraft() {
	a.setup.modules = make(map[string]bool, len(setupModuleIDs))
	for _, id := range setupModuleIDs {
		a.setup.modules[id] = true
	}
	a.setup.inbound = setupInboundProxy
	if len(a.setup.status.Listeners) > 0 {
		a.setup.inbound = setupInboundPreserve
	}
	a.setup.ipv6 = setupIPv6Auto
	a.setup.reset = false
	a.setup.preview = nil
	a.setup.err = ""
	a.setup.draftReady = true
}

func (a *application) setupDraftChanged() {
	a.setup.preview = nil
	a.setup.err = ""
}

func (a *application) setupRequest() (core.SetupRequest, error) {
	modules := make([]string, 0, len(a.setup.modules))
	for _, id := range setupModuleIDs {
		if a.setup.modules[id] {
			modules = append(modules, id)
		}
	}
	if len(modules) == 0 {
		return core.SetupRequest{}, fmt.Errorf("select at least one module")
	}
	if a.setup.status.ConfigError != "" && !a.setup.reset {
		return core.SetupRequest{}, fmt.Errorf("confirm invalid configuration replacement before previewing")
	}
	inbound := a.setup.inbound
	if !a.setup.modules["inbounds"] {
		inbound = setupInboundPreserve
	}
	return core.SetupRequest{Modules: modules, InboundMode: inbound, IPv6: a.setup.ipv6, ResetInvalid: a.setup.reset}, nil
}

func (a *application) previewSetup() {
	if a.setup.previewing || a.setup.applying || a.model.busy {
		return
	}
	request, err := a.setupRequest()
	if err != nil {
		a.setup.err = err.Error()
		return
	}
	a.setup.previewing = true
	a.setup.preview = nil
	a.setup.err = ""
	a.invalidate()
	if a.win == nil {
		plan, planErr := a.fetchSetupPreview(request)
		a.applySetupPreview(plan, planErr)
		return
	}
	go func() {
		plan, planErr := a.fetchSetupPreview(request)
		a.win.Update(func() { a.applySetupPreview(plan, planErr) })
	}()
}

func (a *application) fetchSetupPreview(request core.SetupRequest) (*core.SetupPlan, error) {
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
	defer cancel()
	return a.backend.PreviewSetup(ctx, request)
}

func (a *application) applySetupPreview(plan *core.SetupPlan, err error) {
	a.setup.previewing = false
	if err == nil && plan == nil {
		err = errors.New("setup preview returned no plan")
	}
	if err != nil {
		a.setup.err = shortError(err)
		a.invalidate()
		return
	}
	a.setup.preview = plan
	a.invalidate()
}

func (a *application) applySetup() {
	if a.setup.preview == nil || a.setup.previewing || a.setup.applying || a.model.busy {
		return
	}
	request, err := a.setupRequest()
	if err != nil {
		a.setup.err = err.Error()
		return
	}
	request.SourceHash = a.setup.preview.SourceHash
	a.setup.applying = true
	a.setup.err = ""
	a.invalidate()
	if a.win == nil {
		result, applyErr := a.fetchSetupApply(request)
		a.finishSetupApply(result, applyErr)
		return
	}
	go func() {
		result, applyErr := a.fetchSetupApply(request)
		a.win.Update(func() { a.finishSetupApply(result, applyErr) })
	}()
}

func (a *application) fetchSetupApply(request core.SetupRequest) (service.SetupApplyResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
	defer cancel()
	return a.backend.ApplySetup(ctx, request)
}

func (a *application) finishSetupApply(result service.SetupApplyResult, err error) {
	a.setup.applying = false
	a.setup.preview = nil
	if err != nil {
		a.setup.err = shortError(err)
		a.invalidate()
		return
	}
	if result.RolledBack {
		message := "configuration was rolled back"
		if result.APIError != nil && result.APIError.Message != "" {
			message = result.APIError.Message
		}
		a.setup.err = message
		a.invalidate()
		return
	}
	a.setup.dialogOpen = false
	a.setup.draftReady = false
	a.setup.err = ""
	a.model.actionError = ""
	a.model.notice = "Modules configured"
	a.invalidateConfigAfterSetup()
	a.loadConfig()
	a.loadSetup(true)
	a.requestRefresh()
	a.invalidate()
}

func (a *application) invalidateConfigAfterSetup() {
	a.config.loaded = false
	a.config.loading = false
	a.config.attempted = false
	a.config.diagnosticsTried = false
	a.config.diagnosticsLoaded = false
	a.configHistory.attempted = false
	a.configHistory.loaded = false
	for _, state := range a.configSections {
		state.attempted = false
		state.loaded = false
	}
}

func setupModuleLabel(id string) string {
	switch id {
	case "inbounds":
		return "Inbounds"
	case "outbounds":
		return "Outbounds"
	case "rule_sets":
		return "Rule sets"
	case "dns":
		return "DNS"
	case "route":
		return "Route"
	case "experimental":
		return "Experimental"
	default:
		return strings.ReplaceAll(id, "_", " ")
	}
}

func setupReadyModules(status core.SetupStatus) (ready, total int) {
	total = len(status.Modules)
	for _, module := range status.Modules {
		if module.State == "ready" {
			ready++
		}
	}
	return ready, total
}

func setupTone(state string) string {
	switch state {
	case "ready":
		return "ready"
	case "invalid":
		return "invalid"
	default:
		return "missing"
	}
}
