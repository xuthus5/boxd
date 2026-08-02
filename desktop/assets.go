//go:build desktop

package main

import (
	"embed"
	"io/fs"
)

//go:embed all:ui/dist
var embeddedAssets embed.FS

// assets 桌面构建时嵌入前端产物。
var assets fs.FS = embeddedAssets
