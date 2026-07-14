//go:build embed_ui

package main

import (
	"embed"
	"io/fs"
)

//go:embed all:ui/dist/*
var embeddedStaticFS embed.FS

var staticFS fs.FS = embeddedStaticFS
