package main

import (
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xuthus5/boxd/internal/model"
)

func (a *application) view(c *ui.Context) {
	a.applyTheme(c)
	if a.startupErr != nil {
		a.startupErrorView(c)
		return
	}
	a.normalizeRoute()
	t := c.Theme()
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		a.sidebar(c)
		ui.Column(c).Grow(1).FillHeight().Background(t.Background).Children(func() {
			a.topbar(c)
			a.globalMessages(c)
			ui.Scroll(c).Grow(1).Children(func() {
				a.router.View(c, func(r *ui.Route) {
					switch {
					case r.Match("/dashboard"):
						r.Title("Dashboard")
						a.dashboardView(c)
					case r.Match("/nodes"):
						r.Title("Nodes")
						a.nodesView(c)
					case r.Match("/subscriptions"):
						r.Title("Subscriptions")
						a.subscriptionsView(c)
					case r.Match("/connections"):
						r.Title("Connections")
						a.connectionsView(c)
					case r.Match("/logs"):
						r.Title("Logs")
						a.logsView(c)
					case r.Match("/config"):
						r.Title("Config")
						a.configView(c)
					case r.Match("/advanced/raw"):
						r.Title("Raw configuration")
						a.configView(c)
					case r.Match("/settings"):
						r.Title("Settings")
						a.settingsView(c)
					case r.Match("/terminal"):
						r.Title("Terminal")
						a.terminalView(c)
					case r.Match("/proxy/inbounds"), r.Match("/proxy/outbounds"),
						r.Match("/policy/route"), r.Match("/policy/dns"),
						r.Match("/advanced/endpoints"), r.Match("/advanced/services"),
						r.Match("/advanced/certificate"), r.Match("/advanced/log"),
						r.Match("/advanced/ntp"), r.Match("/advanced/experimental"),
						r.Match("/advanced/network-namespaces"), r.Match("/advanced/http-clients"),
						r.Match("/advanced/certificate-providers"):
						path := a.router.Path()
						r.Title(routeTitle(path))
						a.configSectionView(c, path)
					case r.Match("/advanced/history"):
						r.Title("Config history")
						a.configHistoryView(c)
					case r.Match("/advanced/rule-sets"):
						r.Title("Rule sets")
						a.ruleSetsView(c)
					default:
						r.Title("Dashboard")
						a.dashboardView(c)
					}
				})
			})
		})
	})
	a.managementDialogs(c)
	a.configHistoryDialog(c)
	a.configSectionItemDialog(c)
}

func (a *application) sidebar(c *ui.Context) {
	t := c.Theme()
	selected := a.router.Path()
	sidebar := ui.Sidebar(c, &selected, func() {
		ui.Text(c, "boxd").FontSize(22).Bold()
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Native control plane").FontSize(11).TextColor(t.TextMuted)
			ui.Badge(c, "MyGo")
		})
		ui.Divider(c)
		ui.SidebarSection(c, "Overview", nil, func() {
			ui.SidebarItem(c, "/dashboard", iconDashboard, "Dashboard")
		})
		ui.SidebarSection(c, "Proxy & policy", nil, func() {
			ui.SidebarItem(c, "/proxy/inbounds", iconConns, "Inbounds")
			ui.SidebarItem(c, "/proxy/outbounds", iconNodes, "Outbounds")
			ui.SidebarItem(c, "/policy/route", iconSubs, "Route")
			ui.SidebarItem(c, "/policy/dns", iconDashboard, "DNS")
		})
		ui.SidebarSection(c, "Resources", nil, func() {
			ui.SidebarItem(c, "/nodes", iconNodes, "Nodes")
			ui.SidebarItem(c, "/subscriptions", iconSubs, "Subscriptions")
		})
		ui.SidebarSection(c, "Observability", nil, func() {
			ui.SidebarItem(c, "/connections", iconConns, "Connections")
			ui.SidebarItem(c, "/logs", iconLogs, "Logs")
		})
		ui.SidebarSection(c, "Advanced", &a.advancedOpen, func() {
			ui.SidebarItem(c, "/advanced/endpoints", iconConns, "Endpoints")
			ui.SidebarItem(c, "/advanced/services", iconNodes, "Services")
			ui.SidebarItem(c, "/advanced/certificate", iconConfig, "Certificate")
			ui.SidebarItem(c, "/advanced/network-namespaces", iconConns, "Network namespaces")
			ui.SidebarItem(c, "/advanced/http-clients", iconSubs, "HTTP clients")
			ui.SidebarItem(c, "/advanced/certificate-providers", iconConfig, "Certificate providers")
			ui.SidebarItem(c, "/advanced/log", iconLogs, "Kernel log")
			ui.SidebarItem(c, "/advanced/ntp", iconDashboard, "NTP")
			ui.SidebarItem(c, "/advanced/experimental", iconSettings, "Experimental")
			ui.SidebarItem(c, "/advanced/rule-sets", iconSubs, "Rule sets")
			ui.SidebarItem(c, "/advanced/history", iconLogs, "Config history")
			ui.SidebarItem(c, "/advanced/raw", iconConfig, "Raw configuration")
		})
		ui.SidebarSection(c, "System", nil, func() {
			ui.SidebarItem(c, "/config", iconConfig, "Config")
			ui.SidebarItem(c, "/terminal", iconConns, "Terminal")
			ui.SidebarItem(c, "/settings", iconSettings, "Settings")
		})
	}).Width(sidebarWidth).FillHeight().Padding(14).Gap(8).
		Background(t.Surface).BorderWidth(0, 1, 0, 0).BorderColor(t.Border)
	if sidebar.Changed() && selected != a.router.Path() {
		a.router.Push(selected)
	}
}

func (a *application) topbar(c *ui.Context) {
	t := c.Theme()
	title := routeTitle(a.router.Path())
	running := a.model.hasStatus && a.model.status.Running
	color := t.Danger
	if running {
		color = t.Success
	}
	ui.Row(c).FillWidth().Height(64).Padding(0, pagePadding).Gap(12).
		AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).
		Background(t.Background).Children(func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(15).Bold()
			updated := noDataPlaceholder
			if a.model.loaded {
				updated = "Updated " + formatRelative(a.model.lastUpdated, time.Now())
			}
			ui.Text(c, updated).FontSize(11).TextColor(t.TextMuted)
		})
		ui.Spacer(c)
		if a.model.busy {
			ui.Spinner(c)
			ui.Text(c, a.model.busyLabel).FontSize(11).TextColor(t.TextMuted)
		}
		statusPill(c, kernelStatusLabel(running, a.model.busy), color)
		if ui.Button(c, "Refresh").Key("refresh-now").Clicked() {
			a.requestRefresh()
			a.invalidate()
		}
		if ui.Button(c, themeButtonLabel(a.model.prefs.Theme)).Key("theme-toggle").Clicked() {
			a.cycleTheme()
		}
	})
}

func (a *application) globalMessages(c *ui.Context) {
	message := ""
	if len(a.model.warnings) > 0 {
		message = a.model.warnings[0]
	}
	if a.model.stale || message != "" {
		ui.Column(c).FillWidth().Padding(10, pagePadding, 0, pagePadding).Children(func() {
			if message == "" {
				message = dataStaleWarning
			}
			warningBanner(c, message)
		})
	}
	if a.model.actionError != "" || a.model.notice != "" {
		ui.Column(c).FillWidth().Padding(10, pagePadding, 0, pagePadding).Gap(8).Children(func() {
			errorBanner(c, a.model.actionError)
			successBanner(c, a.model.notice)
		})
	}
}

func (a *application) pageColumn(c *ui.Context, fn func()) {
	ui.Column(c).FillWidth().Padding(pagePadding).Gap(sectionGap).Children(fn)
}

func (a *application) startupErrorView(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Center().Gap(14).Padding(40).Background(t.Background).Children(func() {
		ui.Text(c, "boxd could not start").FontSize(26).Bold()
		ui.Text(c, shortError(a.startupErr)).FontSize(13).TextColor(t.Danger).MaxWidth(680)
		ui.Textf(c, "Data directory: %s", a.cfg.DataDir).FontSize(12).TextColor(t.TextMuted)
		ui.Textf(c, "Config path: %s", a.cfg.ConfigPath).FontSize(12).TextColor(t.TextMuted)
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "Quit").Clicked() {
				mygo.App.Quit()
			}
		})
	})
}

func (a *application) cycleTheme() {
	if a.model.busy {
		return
	}
	next := a.model.prefs
	switch next.Theme {
	case themeSystem:
		next.Theme = themeLight
	case themeLight:
		next.Theme = themeDark
	default:
		next.Theme = themeSystem
	}
	a.updateUIPreferences(next)
}

func themeButtonLabel(theme string) string {
	switch theme {
	case themeLight:
		return "Light"
	case themeDark:
		return "Dark"
	default:
		return "System"
	}
}

func routeTitle(path string) string {
	if spec, ok := configSectionSpecForRoute(path); ok {
		return spec.Title
	}
	switch path {
	case "/nodes":
		return "Nodes"
	case "/subscriptions":
		return "Subscriptions"
	case "/connections":
		return "Connections"
	case "/logs":
		return "Logs"
	case "/config":
		return "Config"
	case "/settings":
		return "Settings"
	case "/terminal":
		return "Terminal"
	case "/advanced/history":
		return "Config history"
	case "/advanced/rule-sets":
		return "Rule sets"
	case "/advanced/raw":
		return "Raw configuration"
	default:
		return "Dashboard"
	}
}

func (a *application) normalizeRoute() {
	if _, ok := configSectionSpecForRoute(a.router.Path()); ok {
		return
	}
	switch a.router.Path() {
	case "/dashboard", "/nodes", "/subscriptions", "/connections", "/logs", "/config", "/terminal", "/settings":
		return
	case "/advanced/history", "/advanced/rule-sets", "/advanced/raw":
		return
	default:
		a.router.Replace("/dashboard")
	}
}

func prefsWithTheme(prefs model.UIPreferences, theme string) model.UIPreferences {
	prefs.Theme = theme
	return prefs
}
