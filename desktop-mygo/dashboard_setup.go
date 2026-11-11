package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

func (a *application) dashboardSetup(c *ui.Context) {
	a.ensureSetup()
	t := c.Theme()
	card(c, func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			sectionHeading(c, "Setup", "Module readiness, proxy access and first-run configuration")
			ui.Spacer(c)
			if ui.Button(c, "Refresh").Key("setup-refresh").Disabled(a.setup.loading).Clicked() {
				a.loadSetup(true)
			}
			if ui.PrimaryButton(c, "Configure modules").Key("setup-open").Clicked() {
				a.openSetupDialog()
			}
		})
		if a.setup.loading && !a.setup.loaded {
			loadingRow(c, "Inspecting configuration modules…")
			return
		}
		if !a.setup.loaded {
			emptyState(c, "Setup status is unavailable", "Choose Configure modules to start the interactive setup.")
			return
		}
		ready, total := setupReadyModules(a.setup.status)
		ui.Row(c).Gap(16).Children(func() {
			keyValue(c, "Modules", fmt.Sprintf("%d/%d ready", ready, total))
			keyValue(c, "Listeners", fmt.Sprint(len(a.setup.status.Listeners)))
			keyValue(c, "Proxy", readyLabel(a.setup.status.ProxyReady))
			keyValue(c, "Kernel", readyLabel(a.setup.status.KernelRunning))
		})
		ui.Textf(c, "Platform %s · TUN %s · IPv6 %s · default listen %s",
			firstText(a.setup.status.Capabilities.Platform, "unknown"),
			capabilityLabel(a.setup.status.Capabilities.TUNAvailable, a.setup.status.Capabilities.TUNReason),
			capabilityLabel(a.setup.status.Capabilities.IPv6Available, a.setup.status.Capabilities.IPv6Reason),
			firstText(a.setup.status.Capabilities.DefaultListen, "unknown")).
			FontSize(11).TextColor(t.TextMuted)
		if a.setup.status.ConfigError != "" {
			warningBanner(c, "Configuration error: "+shortErrorText(a.setup.status.ConfigError))
		}
		warningBanner(c, a.setup.err)
		a.dashboardSetupModules(c)
	})
}

func (a *application) dashboardSetupModules(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).FillWidth().Gap(8).Children(func() {
		for _, module := range a.setup.status.Modules {
			color := t.TextMuted
			switch setupTone(module.State) {
			case "ready":
				color = t.Success
			case "invalid":
				color = t.Danger
			default:
				color = t.Warning
			}
			ui.Column(c).Gap(2).Padding(8).Radius(8).Background(t.Background).
				Border(1, t.Border).Children(func() {
				ui.Text(c, setupModuleLabel(module.ID)).FontSize(11).Bold()
				ui.Textf(c, "%s · %d", module.State, module.Count).FontSize(10).TextColor(color)
			})
		}
	})
}

func readyLabel(ready bool) string {
	if ready {
		return "ready"
	}
	return "not ready"
}

func capabilityLabel(available bool, reason string) string {
	if available {
		return "available"
	}
	if reason != "" {
		return "unavailable (" + reason + ")"
	}
	return "unavailable"
}
