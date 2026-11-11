package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/xuthus5/boxd/internal/core"
)

type terminalEndpoint struct {
	Host   string
	Port   int
	Scheme string
	URL    string
}

type terminalCommand struct {
	Label   string
	Command string
}

func (a *application) terminalView(c *ui.Context) {
	a.ensureTerminal()
	a.pageColumn(c, func() {
		pageHeading(c, "Terminal", "Copy shell proxy commands from the active mixed, HTTP or SOCKS listener.")
		card(c, func() {
			sectionHeading(c, "Proxy environment", "No shell command is executed; boxd only prepares copy-ready snippets")
			switch {
			case a.terminal.loading:
				loadingRow(c, "Inspecting configured listeners…")
			case !a.terminal.loaded:
				ui.Text(c, "Terminal status is not available.").TextColor(c.Theme().TextMuted)
			case a.terminal.status.ConfigError != "":
				ui.Text(c, a.terminal.status.ConfigError).TextColor(c.Theme().Danger)
			default:
				a.terminalEndpointContent(c)
			}
		})
	})
}

func (a *application) terminalEndpointContent(c *ui.Context) {
	endpoint, ok := resolveTerminalEndpoint(a.terminal.status.Listeners)
	if !ok {
		emptyState(c, "No proxy listener configured", "Add a mixed, HTTP or SOCKS inbound in the Inbounds page, then refresh Terminal.")
		return
	}
	ui.Textf(c, "Using %s from the shared configuration", endpoint.URL).
		FontSize(12).TextColor(c.Theme().TextMuted)
	for _, item := range buildTerminalCommands(endpoint) {
		terminalCommandRow(c, item, func() { a.copyTerminalCommand(item.Label, item.Command) })
	}
}

func terminalCommandRow(c *ui.Context, item terminalCommand, copy func()) {
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Gap(3).Children(func() {
			ui.Text(c, item.Label).FontSize(12).Bold()
			ui.Text(c, item.Command).FontSize(11).TextColor(c.Theme().TextMuted)
		})
		if ui.Button(c, "Copy").Key("terminal-copy-" + strings.ToLower(strings.ReplaceAll(item.Label, " ", "-"))).Clicked() {
			copy()
		}
	})
}

func loadingRow(c *ui.Context, text string) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Spinner(c)
		ui.Text(c, text).TextColor(c.Theme().TextMuted)
	})
}

func resolveTerminalEndpoint(listeners []core.SetupListener) (terminalEndpoint, bool) {
	for _, kind := range []string{"mixed", "http", "socks"} {
		for _, listener := range listeners {
			if listener.Type != kind || !validListenPort(listener.ListenPort) {
				continue
			}
			host := terminalHost(listener.Listen)
			scheme := "http"
			if kind == "socks" {
				scheme = "socks5h"
			}
			return terminalEndpoint{Host: host, Port: listener.ListenPort, Scheme: scheme,
				URL: scheme + "://" + net.JoinHostPort(host, strconv.Itoa(listener.ListenPort))}, true
		}
	}
	return terminalEndpoint{}, false
}

func validListenPort(port int) bool {
	return port >= 1 && port <= 65535
}

func terminalHost(listen string) string {
	host := strings.TrimSpace(listen)
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" || host == "localhost" {
		return "127.0.0.1"
	}
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	}
	return host
}

func buildTerminalCommands(endpoint terminalEndpoint) []terminalCommand {
	url := endpoint.URL
	return []terminalCommand{
		{Label: "Bash / Zsh", Command: fmt.Sprintf(`export http_proxy=%q https_proxy=%q HTTP_PROXY=%q HTTPS_PROXY=%q`, url, url, url, url)},
		{Label: "Fish", Command: fmt.Sprintf(`set -x http_proxy %q; set -x https_proxy %q; set -x HTTP_PROXY %q; set -x HTTPS_PROXY %q`, url, url, url, url)},
		{Label: "PowerShell", Command: fmt.Sprintf(`$env:HTTP_PROXY=%q; $env:HTTPS_PROXY=%q`, url, url)},
		{Label: "Command Prompt", Command: fmt.Sprintf(`set "http_proxy=%s" && set "https_proxy=%s"`, url, url)},
	}
}
