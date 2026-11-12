package main

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestMainWindowUsesCustomChrome(t *testing.T) {
	options := mainWindowOptions(true)
	if options.TitleBarStyle != mygo.TitleBarHidden {
		t.Fatalf("title bar style = %q, want %q", options.TitleBarStyle, mygo.TitleBarHidden)
	}
	if !options.HideWindowButtons {
		t.Fatal("system window buttons should be hidden")
	}
	if options.Frameless {
		t.Fatal("custom chrome should keep native shadow and resize edges")
	}
	if !options.Hidden {
		t.Fatal("hidden option was not preserved")
	}
}

func TestWindowControlsRenderInMainBody(t *testing.T) {
	app := newTestApplication(seededModel(themeLight))
	tt := ui.NewTester(app.view, 1440, 900)
	for _, label := range []string{"Minimize window", "Maximize or restore window", "Close window"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("window control %q is missing; texts=%v", label, tt.Texts())
		}
		if err := tt.Click(label); err != nil {
			t.Fatalf("click %s: %v", label, err)
		}
	}
}

func TestStartupErrorViewKeepsWindowControls(t *testing.T) {
	app := newApplication(nil, testConfig(), errTestStartup, false)
	tt := ui.NewTester(app.view, 1200, 800)
	for _, label := range []string{"Minimize window", "Maximize or restore window", "Close window"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("startup view is missing %q", label)
		}
	}
}
