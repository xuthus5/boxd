package main

import "github.com/egoist/mygo/ui"

const (
	cardRadius   = float32(12)
	cardPadding  = float32(18)
	pagePadding  = float32(28)
	sectionGap   = float32(18)
	compactGap   = float32(10)
	sidebarWidth = float32(228)
)

func (a *application) applyTheme(c *ui.Context) {
	c.SetTheme(shadcnTheme(a.effectiveDark()))
}

func (a *application) effectiveDark() bool {
	switch a.model.prefs.Theme {
	case themeDark:
		return true
	case themeLight:
		return false
	default:
		return a.systemDark
	}
}

func shadcnTheme(dark bool) *ui.Theme {
	theme := ui.LightTheme()
	if dark {
		theme = ui.DarkTheme()
		theme.Background = ui.Hex("#09090b")
		theme.Surface = ui.Hex("#18181b")
		theme.SurfaceHover = ui.Hex("#27272a")
		theme.SurfacePressed = ui.Hex("#3f3f46")
		theme.Border = ui.Hex("#27272a")
		theme.Text = ui.Hex("#fafafa")
		theme.TextMuted = ui.Hex("#a1a1aa")
		theme.Accent = ui.Hex("#3b82f6")
		theme.AccentHover = ui.Hex("#60a5fa")
		theme.AccentPressed = ui.Hex("#2563eb")
	} else {
		theme.Background = ui.Hex("#fafafa")
		theme.Surface = ui.Hex("#ffffff")
		theme.SurfaceHover = ui.Hex("#f4f4f5")
		theme.SurfacePressed = ui.Hex("#e4e4e7")
		theme.Border = ui.Hex("#e4e4e7")
		theme.Text = ui.Hex("#18181b")
		theme.TextMuted = ui.Hex("#71717a")
		theme.Accent = ui.Hex("#2563eb")
		theme.AccentHover = ui.Hex("#1d4ed8")
		theme.AccentPressed = ui.Hex("#1e40af")
	}
	theme.Radius = 10
	theme.Spacing = 4
	theme.FontSize = 13
	theme.FocusRingWidth = 2
	theme.FocusRingOffset = 2
	return theme
}
