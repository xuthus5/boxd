package mygo

import (
	"embed"

	"github.com/egoist/mygo/ui"
)

//go:embed assets/icon-64.png assets/icon-256.png
var iconAssets embed.FS

var (
	logoBitmap = mustBitmap("assets/icon-64.png")

	iconOverview      = mustIcon("M4 13h6V4H4v9Zm0 7h6v-5H4v5Zm10 0h6V11h-6v9Zm0-16v5h6V4h-6Z")
	iconSubscriptions = mustIcon("M5 5h14v14H5zM8 9h8M8 12h8M8 15h5")
	iconNodes         = mustIcon("M12 3v4m0 10v4M3 12h4m10 0h4M8.5 12a3.5 3.5 0 1 0 7 0 3.5 3.5 0 0 0-7 0Z")
	iconEgress        = mustIcon("M4 7h6m4 0h6M4 17h6m4 0h6M13 4l3 3-3 3M11 14l3 3-3 3")
	iconRoutes        = mustIcon("M5 5v9a3 3 0 0 0 3 3h11M5 9h6a3 3 0 0 1 3 3v5M16 14l3 3-3 3")
	iconDNS           = mustIcon("M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18Zm-8 9h16M12 3c2.2 2.2 3.5 5.3 3.5 9S14.2 18.8 12 21M12 3C9.8 5.2 8.5 8.3 8.5 12s1.3 6.8 3.5 9")
	iconRuleSets      = mustIcon("M6 4h12v16H6zM9 8h6M9 12h6M9 16h4")
	iconLogs          = mustIcon("M5 5h14v14H5zM8 9l2 2-2 2M12 15h4")
	iconConnections   = mustIcon("M8 12a4 4 0 0 1 4-4h3M16 12a4 4 0 0 1-4 4H9M14 6l2 2-2 2M10 14l-2 2 2 2")
	iconSettings      = mustIcon("M12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6Zm8 3-2-.7.7-2-1.4-1.4-2 .7L14.6 6H9.4L8.7 8.6l-2-.7L5.3 9.3l.7 2L4 12l.7 2-1.4 1.4 2 .7.7 2.6h5.2l.7-2.6 2 .7 1.4-1.4-.7-2L20 12Z")
	iconAbout         = mustIcon("M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18ZM12 10v6M12 7h.01")
	iconSun           = mustIcon("M12 7a5 5 0 1 0 0 10 5 5 0 0 0 0-10ZM12 2v2M12 20v2M2 12h2M20 12h2M5 5l1.5 1.5M17.5 17.5 19 19M19 5l-1.5 1.5M6.5 17.5 5 19")
	iconMoon          = mustIcon("M19 15.5A8 8 0 0 1 8.5 5 7 7 0 1 0 19 15.5Z")
	iconRefresh       = mustIcon("M20 7v5h-5M4 17v-5h5M6.1 9A7 7 0 0 1 18.7 7M17.9 15A7 7 0 0 1 5.3 17")
	iconPlus          = mustIcon("M12 5v14M5 12h14")
	iconEdit          = mustIcon("M4 20h4L19 9l-4-4L4 16v4ZM13.5 6.5l4 4")
	iconTrash         = mustIcon("M4 7h16M9 7V4h6v3M6 7l1 14h10l1-14M10 11v6M14 11v6")
	iconPlay          = mustIcon("M8 5v14l11-7L8 5Z")
	iconStop          = mustIcon("M6 6h12v12H6z")
	iconClose         = mustIcon("M6 6l12 12M18 6 6 18")
	iconChevronUp     = mustIcon("M6 15l6-6 6 6")
	iconChevronDown   = mustIcon("M6 9l6 6 6-6")
	iconCheck         = mustIcon("M5 12l5 5L19 7")
	iconAlert         = mustIcon("M12 8v5M12 17h.01M10.3 4.6 3.2 17a2 2 0 0 0 1.7 3h14.2a2 2 0 0 0 1.7-3L13.7 4.6a2 2 0 0 0-3.4 0Z")
	iconArrowRight    = mustIcon("M5 12h14M14 7l5 5-5 5")
	iconGrip          = mustIcon("M9 5h.01M9 12h.01M9 19h.01M15 5h.01M15 12h.01M15 19h.01")
)

func mustBitmap(path string) *ui.Bitmap {
	body, err := iconAssets.ReadFile(path)
	if err != nil {
		panic(err)
	}
	bitmap, err := ui.DecodeBitmap(body)
	if err != nil {
		panic(err)
	}
	return bitmap
}

func mustIcon(path string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="` + path + `"/></svg>`))
}

func navIcon(id string) *ui.SVG {
	switch id {
	case pageOverview:
		return iconOverview
	case pageSubscriptions:
		return iconSubscriptions
	case pageNodes:
		return iconNodes
	case pageEgress:
		return iconEgress
	case pageRoutes:
		return iconRoutes
	case pageDNS:
		return iconDNS
	case pageRuleSets:
		return iconRuleSets
	case pageLogs:
		return iconLogs
	case pageConnections:
		return iconConnections
	case pageSettings:
		return iconSettings
	case pageAbout:
		return iconAbout
	default:
		return iconAbout
	}
}
