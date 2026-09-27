package main

import (
	"context"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/xuthus5/boxd/internal/model"
)

// trayStateLanguage 读取当前生效的托盘语言（与 refreshTrayLanguage 共用一把锁）。
func trayStateLanguage() string {
	trayMu.Lock()
	defer trayMu.Unlock()
	return activeTrayLanguage
}

func resetTrayState(t *testing.T) {
	t.Helper()
	setActiveTray(nil, "")
	t.Cleanup(func() { setActiveTray(nil, "") })
}

func TestTrayLabelsForLanguage(t *testing.T) {
	zh := trayLabelsFor("zh")
	en := trayLabelsFor("en")
	if zh.Show == en.Show || zh.Quit == en.Quit || zh.Tooltip == en.Tooltip {
		t.Fatalf("expected distinct zh/en labels, got %+v and %+v", zh, en)
	}
	for _, language := range []string{"", "fr", "ZH"} {
		if got := trayLabelsFor(language); got != zh {
			t.Fatalf("labels for %q = %+v, want default", language, got)
		}
	}
	for language, labels := range trayLabelsByLanguage {
		fields := map[string]string{
			"Tooltip":       labels.Tooltip,
			"Show":          labels.Show,
			"StartKernel":   labels.StartKernel,
			"StopKernel":    labels.StopKernel,
			"RestartKernel": labels.RestartKernel,
			"ReloadUI":      labels.ReloadUI,
			"Quit":          labels.Quit,
		}
		for name, value := range fields {
			if value == "" {
				t.Errorf("%s(%s) is empty", name, language)
			}
		}
	}
}

func TestTrayLanguageReadsPreferences(t *testing.T) {
	rt := newTestRuntimeWithService(t)
	if got := trayLanguage(rt); got != defaultTrayLanguage {
		t.Fatalf("default language = %q", got)
	}
	if _, err := rt.svc.Settings().SetUIPreferences(context.Background(), model.UIPreferences{
		Theme: "system", Language: "en", MinimumLogLevel: "all",
	}); err != nil {
		t.Fatal(err)
	}
	if got := trayLanguage(rt); got != "en" {
		t.Fatalf("language after save = %q", got)
	}
	if got := trayLanguage(nil); got != defaultTrayLanguage {
		t.Fatalf("nil runtime language = %q", got)
	}
	if got := trayLanguage(&desktopRuntime{}); got != defaultTrayLanguage {
		t.Fatalf("runtime without services language = %q", got)
	}
}

func TestTrayMenuUsesLabels(t *testing.T) {
	for _, language := range []string{"zh", "en"} {
		labels := trayLabelsFor(language)
		menu := trayMenu(nil, nil, labels)
		if menu == nil {
			t.Fatalf("%s: nil menu", language)
		}
		for _, label := range []string{
			labels.Show, labels.StartKernel, labels.StopKernel, labels.RestartKernel, labels.ReloadUI, labels.Quit,
		} {
			if menu.FindByLabel(label) == nil {
				t.Errorf("%s: menu item %q missing", language, label)
			}
		}
	}
}

func TestApplyTrayLabelsWithoutTray(t *testing.T) {
	applyTrayLabels(nil, nil, nil, "en")
}

func TestSetupTrayWithoutSystemTray(t *testing.T) {
	setupTray(&application.App{}, nil)
}

func TestRefreshTrayLanguage(t *testing.T) {
	resetTrayState(t)
	rt := newTestRuntimeWithService(t)

	// 托盘未创建时不应有任何副作用。
	refreshTrayLanguage(rt)
	if got := trayStateLanguage(); got != "" {
		t.Fatalf("language without tray = %q", got)
	}

	tray := &application.SystemTray{}
	setActiveTray(tray, defaultTrayLanguage)
	refreshTrayLanguage(rt)
	if got := trayStateLanguage(); got != defaultTrayLanguage {
		t.Fatalf("language without change = %q", got)
	}

	if _, err := rt.svc.Settings().SetUIPreferences(context.Background(), model.UIPreferences{
		Theme: "dark", Language: "en", MinimumLogLevel: "warn",
	}); err != nil {
		t.Fatal(err)
	}
	refreshTrayLanguage(rt)
	if got := trayStateLanguage(); got != "en" {
		t.Fatalf("language after switch = %q", got)
	}
}
