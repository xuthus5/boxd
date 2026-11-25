package mygo

import "github.com/egoist/mygo/ui"

type palette struct {
	Background   ui.Color
	Surface      ui.Color
	SurfaceAlt   ui.Color
	SurfaceHover ui.Color
	Border       ui.Color
	BorderSoft   ui.Color
	Text         ui.Color
	TextMuted    ui.Color
	TextFaint    ui.Color
	Accent       ui.Color
	AccentSoft   ui.Color
	AccentText   ui.Color
	Indigo       ui.Color
	Violet       ui.Color
	Sky          ui.Color
	Success      ui.Color
	SuccessSoft  ui.Color
	Warning      ui.Color
	WarningSoft  ui.Color
	Danger       ui.Color
	DangerSoft   ui.Color
	Shadow       ui.Color
	HeroFrom     ui.Color
	HeroTo       ui.Color
	Sidebar      ui.Color
}

func lightPalette() palette {
	return palette{
		Background:   ui.Hex("#f6f7fb"),
		Surface:      ui.Hex("#ffffff"),
		SurfaceAlt:   ui.Hex("#f8f9fd"),
		SurfaceHover: ui.Hex("#eef1f8"),
		Border:       ui.Hex("#e3e7f0"),
		BorderSoft:   ui.Hex("#edf0f6"),
		Text:         ui.Hex("#141827"),
		TextMuted:    ui.Hex("#667085"),
		TextFaint:    ui.Hex("#98a2b3"),
		Accent:       ui.Hex("#6366f1"),
		AccentSoft:   ui.Hex("#eef0ff"),
		AccentText:   ui.Hex("#ffffff"),
		Indigo:       ui.Hex("#6366f1"),
		Violet:       ui.Hex("#8b5cf6"),
		Sky:          ui.Hex("#0ea5e9"),
		Success:      ui.Hex("#059669"),
		SuccessSoft:  ui.Hex("#e9f9f2"),
		Warning:      ui.Hex("#d97706"),
		WarningSoft:  ui.Hex("#fff7e8"),
		Danger:       ui.Hex("#dc3545"),
		DangerSoft:   ui.Hex("#fff0f1"),
		Shadow:       ui.RGBA(31, 41, 55, 0.06),
		HeroFrom:     ui.Hex("#eef0ff"),
		HeroTo:       ui.Hex("#f2f8ff"),
		Sidebar:      ui.Hex("#fbfcff"),
	}
}

func darkPalette() palette {
	return palette{
		Background:   ui.Hex("#0b0f19"),
		Surface:      ui.Hex("#121827"),
		SurfaceAlt:   ui.Hex("#172033"),
		SurfaceHover: ui.Hex("#1d2940"),
		Border:       ui.Hex("#263149"),
		BorderSoft:   ui.Hex("#1e283c"),
		Text:         ui.Hex("#f3f5fb"),
		TextMuted:    ui.Hex("#9aa6bd"),
		TextFaint:    ui.Hex("#64718a"),
		Accent:       ui.Hex("#818cf8"),
		AccentSoft:   ui.Hex("#20264a"),
		AccentText:   ui.Hex("#0b0f19"),
		Indigo:       ui.Hex("#818cf8"),
		Violet:       ui.Hex("#a78bfa"),
		Sky:          ui.Hex("#38bdf8"),
		Success:      ui.Hex("#34d399"),
		SuccessSoft:  ui.Hex("#123629"),
		Warning:      ui.Hex("#fbbf24"),
		WarningSoft:  ui.Hex("#3b2a0c"),
		Danger:       ui.Hex("#fb7185"),
		DangerSoft:   ui.Hex("#3a1722"),
		Shadow:       ui.RGBA(0, 0, 0, 0.24),
		HeroFrom:     ui.Hex("#18203d"),
		HeroTo:       ui.Hex("#102a3b"),
		Sidebar:      ui.Hex("#0f1421"),
	}
}

func (c *Controller) colors() palette {
	if c.dark {
		return darkPalette()
	}
	return lightPalette()
}

func (c *Controller) applyTheme(ctx *ui.Context) palette {
	p := c.colors()
	theme := ui.LightTheme()
	if c.dark {
		theme = ui.DarkTheme()
	}
	theme.Background = p.Background
	theme.Surface = p.Surface
	theme.SurfaceHover = p.SurfaceHover
	theme.SurfacePressed = p.SurfaceAlt
	theme.Border = p.Border
	theme.Text = p.Text
	theme.TextMuted = p.TextMuted
	theme.Accent = p.Accent
	theme.AccentHover = p.Indigo.Mix(p.Violet, 0.35)
	theme.AccentPressed = p.Accent.Mix(ui.Hex("#000000"), 0.16)
	theme.AccentText = p.AccentText
	theme.InactiveAccent = p.Accent
	theme.Danger = p.Danger
	theme.Warning = p.Warning
	theme.Success = p.Success
	theme.Selection = p.Accent.Alpha(0.22)
	theme.Focus = p.Accent.Alpha(0.6)
	theme.Inverse = p.Text
	theme.InverseText = p.Background
	theme.Scrollbar = p.TextMuted.Alpha(0.32)
	theme.ScrollbarWidth = 5
	theme.Radius = 10
	theme.Spacing = 4
	theme.FontSize = 14
	ctx.SetTheme(theme)
	return p
}

func (c *Controller) brandGradient(e ui.Element) ui.Element {
	p := c.colors()
	return e.LinearGradient(ui.LinearGradient{
		Angle: 120,
		Stops: []ui.GradientStop{
			{Color: p.Indigo, At: 0},
			{Color: p.Violet, At: 0.55},
			{Color: p.Sky, At: 1},
		},
		Oklab: true,
	})
}

func (c *Controller) heroGradient(e ui.Element) ui.Element {
	p := c.colors()
	return e.LinearGradient(ui.LinearGradient{
		Angle: 115,
		Stops: []ui.GradientStop{
			{Color: p.HeroFrom, At: 0},
			{Color: p.HeroTo, At: 1},
		},
		Oklab: true,
	})
}
