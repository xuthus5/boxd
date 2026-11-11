package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/xuthus5/boxd/internal/core"
)

func (a *application) setupDialog(c *ui.Context) {
	ui.Modal(c, &a.setup.dialogOpen, func() {
		dialogPanel(c, "Configure modules", "Preview the selected modules before writing configuration or restarting the kernel.", func() {
			if !a.setup.loaded {
				if a.setup.loading {
					loadingRow(c, "Inspecting configuration modules…")
				} else {
					emptyState(c, "Setup status is unavailable", "Close this dialog and refresh the setup status before trying again.")
				}
				a.setupCloseButton(c)
				return
			}
			a.setupCapabilitySummary(c)
			a.setupResetConsent(c)
			a.setupModuleChoices(c)
			a.setupModeChoices(c)
			a.setupPreviewSummary(c)
			errorBanner(c, a.setup.err)
			a.setupDialogButtons(c)
		})
	})
}

func (a *application) setupCapabilitySummary(c *ui.Context) {
	status := a.setup.status
	ui.Row(c).FillWidth().Gap(12).Children(func() {
		keyValue(c, "Platform", firstText(status.Capabilities.Platform, "unknown"))
		keyValue(c, "TUN", capabilityLabel(status.Capabilities.TUNAvailable, status.Capabilities.TUNReason))
		keyValue(c, "IPv6", capabilityLabel(status.Capabilities.IPv6Available, status.Capabilities.IPv6Reason))
		keyValue(c, "Listeners", fmt.Sprint(len(status.Listeners)))
	})
}

func (a *application) setupResetConsent(c *ui.Context) {
	if a.setup.status.ConfigError == "" {
		return
	}
	warningBanner(c, "Configuration error: "+shortErrorText(a.setup.status.ConfigError))
	ui.Text(c, "Replacing an invalid configuration requires explicit confirmation.").FontSize(12)
	if ui.Checkbox(c.Key("setup-reset"), &a.setup.reset, "Replace the invalid configuration").
		Disabled(a.setupBusy()).Changed() {
		a.setupDraftChanged()
	}
}

func (a *application) setupModuleChoices(c *ui.Context) {
	ui.Text(c, "Modules").FontSize(12).Bold()
	ui.Text(c, "Dependencies are installed together and the selected modules replace their existing sections.").
		FontSize(11).TextColor(c.Theme().TextMuted)
	for start := 0; start < len(setupModuleIDs); start += 3 {
		end := min(start+3, len(setupModuleIDs))
		ui.Row(c).FillWidth().Gap(12).Children(func() {
			for _, id := range setupModuleIDs[start:end] {
				checked := a.setup.modules[id]
				if ui.Checkbox(c.Key("setup-module-"+id), &checked, setupModuleChoiceLabel(a.setup.status, id)).
					Disabled(a.setupBusy()).Changed() {
					a.setup.modules[id] = checked
					if id == "inbounds" && !checked {
						a.setup.inbound = setupInboundPreserve
					}
					a.setupDraftChanged()
				}
			}
		})
	}
}

func (a *application) setupModeChoices(c *ui.Context) {
	inboundsSelected := a.setup.modules["inbounds"]
	ui.Row(c).FillWidth().Gap(16).Children(func() {
		ui.Field(c, "Inbound mode", func() {
			if ui.Select(c.Key("setup-inbound"), &a.setup.inbound, setupInboundOptions).
				Disabled(a.setupBusy() || !inboundsSelected).Changed() {
				a.setupDraftChanged()
			}
		}).Grow(1)
		ui.Field(c, "IPv6", func() {
			if ui.Select(c.Key("setup-ipv6"), &a.setup.ipv6, setupIPv6Options).
				Disabled(a.setupBusy()).Changed() {
				a.setupDraftChanged()
			}
		}).Grow(1)
	})
	if !inboundsSelected {
		ui.Text(c, "Existing inbound listeners are preserved because Inbounds is not selected.").
			FontSize(11).TextColor(c.Theme().TextMuted)
	}
}

func (a *application) setupPreviewSummary(c *ui.Context) {
	if a.setup.preview == nil {
		return
	}
	plan := a.setup.preview
	successBannerLabel(c, "Ready", "Preview ready. Review the changes before applying.")
	keyValue(c, "Modules", strings.Join(plan.Modules, ", "))
	keyValue(c, "Restart", yesNo(plan.WillRestart))
	for _, warning := range plan.Warnings {
		warningBanner(c, setupWarningLabel(warning))
	}
}

func (a *application) setupDialogButtons(c *ui.Context) {
	ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
		a.setupCloseButton(c)
		previewLabel := "Preview"
		if a.setup.previewing {
			previewLabel = "Previewing…"
		}
		if ui.Button(c, previewLabel).Key("setup-preview").Disabled(a.setupBusy()).Clicked() {
			a.previewSetup()
		}
		applyLabel := "Apply"
		if a.setup.applying {
			applyLabel = "Applying…"
		}
		if ui.PrimaryButton(c, applyLabel).Key("setup-apply").
			Disabled(a.setupBusy() || a.setup.preview == nil).Clicked() {
			a.applySetup()
		}
	})
}

func (a *application) setupCloseButton(c *ui.Context) {
	if ui.Button(c, "Cancel").Key("setup-cancel").Disabled(a.setupBusy()).Clicked() {
		a.closeSetupDialog()
	}
}

func (a *application) setupBusy() bool {
	return a.model.busy || a.setup.previewing || a.setup.applying
}

func setupModuleChoiceLabel(status core.SetupStatus, id string) string {
	label := setupModuleLabel(id)
	for _, module := range status.Modules {
		if module.ID == id {
			return fmt.Sprintf("%s · %s · %d", label, firstText(module.State, "unknown"), module.Count)
		}
	}
	return label + " · missing · 0"
}

func setupWarningLabel(code string) string {
	switch code {
	case "config_reset":
		return "The invalid configuration will be replaced by a generated baseline."
	case "container_port_mapping":
		return "Container port mapping may affect access to new listeners."
	case "proxy_nodes_required":
		return "No proxy outbound is ready. Add a node or subscription after setup."
	case "tun_ipv4_only":
		return "TUN will be configured with IPv4 only."
	default:
		return code
	}
}
