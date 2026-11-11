package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/xuthus5/boxd/internal/core"
)

func TestResolveTerminalEndpoint(t *testing.T) {
	endpoint, ok := resolveTerminalEndpoint([]core.SetupListener{
		{Type: "socks", Listen: "0.0.0.0", ListenPort: 1081},
		{Type: "mixed", Listen: "::1", ListenPort: 1080},
		{Type: "http", Listen: "127.0.0.1", ListenPort: 0},
	})
	if !ok {
		t.Fatal("mixed listener was not found")
	}
	if endpoint.URL != "http://[::1]:1080" || endpoint.Host != "::1" {
		t.Fatalf("endpoint = %+v", endpoint)
	}
	if _, ok := resolveTerminalEndpoint([]core.SetupListener{{Type: "tun", ListenPort: 1080}}); ok {
		t.Fatal("unsupported listener was accepted")
	}
	if got := terminalHost("[::]"); got != "127.0.0.1" {
		t.Fatalf("wildcard host = %q", got)
	}
}

func TestResolveTerminalEndpointFallbackAndValidation(t *testing.T) {
	endpoint, ok := resolveTerminalEndpoint([]core.SetupListener{
		{Type: "http", Listen: "localhost", ListenPort: 8080},
		{Type: "socks", Listen: "::", ListenPort: 1080},
	})
	if !ok || endpoint.URL != "http://127.0.0.1:8080" {
		t.Fatalf("fallback endpoint = %+v, ok=%v", endpoint, ok)
	}
	endpoint, ok = resolveTerminalEndpoint([]core.SetupListener{{Type: "socks", Listen: "[::1]", ListenPort: 1081}})
	if !ok || endpoint.URL != "socks5h://[::1]:1081" {
		t.Fatalf("socks endpoint = %+v, ok=%v", endpoint, ok)
	}
	for _, port := range []int{-1, 0, 65536} {
		if _, ok := resolveTerminalEndpoint([]core.SetupListener{{Type: "mixed", ListenPort: port}}); ok {
			t.Fatalf("invalid port %d was accepted", port)
		}
	}
	if got := terminalHost("127.0.0.1"); got != "127.0.0.1" {
		t.Fatalf("loopback host = %q", got)
	}
}

func TestTerminalViewAndCommands(t *testing.T) {
	fake := &fakeBackend{setupStatus: core.SetupStatus{
		Listeners:  []core.SetupListener{{Tag: "mixed-in", Type: "mixed", Listen: "0.0.0.0", ListenPort: 1080}},
		ProxyReady: true,
	}}
	app := newApplication(fake, testConfig(), nil, false)
	tt := ui.NewTester(app.view, 1440, 900)
	app.router.Push("/terminal")
	settleTransitions(tt)
	for _, text := range []string{"Terminal", "http://127.0.0.1:1080", "Bash / Zsh", "PowerShell"} {
		if !tt.HasText(text) {
			t.Errorf("terminal page is missing %q; texts=%v", text, tt.Texts())
		}
	}
	commands := buildTerminalCommands(terminalEndpoint{URL: "http://127.0.0.1:1080"})
	if len(commands) != 4 || !strings.Contains(commands[0].Command, "http_proxy") {
		t.Fatalf("commands = %+v", commands)
	}
}

func TestTerminalViewStates(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		app := newApplication(&fakeBackend{setupStatus: core.SetupStatus{}}, testConfig(), nil, false)
		tt := ui.NewTester(app.view, 1440, 900)
		app.router.Push("/terminal")
		settleTransitions(tt)
		if !tt.HasText("No proxy listener configured") {
			t.Fatalf("empty state is missing: %v", tt.Texts())
		}
	})
	t.Run("config error", func(t *testing.T) {
		app := newApplication(&fakeBackend{setupStatus: core.SetupStatus{ConfigError: "invalid inbound"}}, testConfig(), nil, false)
		tt := ui.NewTester(app.view, 1440, 900)
		app.router.Push("/terminal")
		settleTransitions(tt)
		if !tt.HasText("invalid inbound") {
			t.Fatalf("config error is missing: %v", tt.Texts())
		}
	})
	t.Run("backend error", func(t *testing.T) {
		app := newApplication(&fakeBackend{actionErr: errors.New("setup unavailable")}, testConfig(), nil, false)
		tt := ui.NewTester(app.view, 1440, 900)
		app.router.Push("/terminal")
		settleTransitions(tt)
		if !tt.HasText("Terminal status is not available.") || !tt.HasText("setup unavailable") {
			t.Fatalf("backend error state is missing: %v", tt.Texts())
		}
	})
}
