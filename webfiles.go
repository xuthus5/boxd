// Package boxd exposes embedded web assets from the module root so the
// production binary is self-contained. The control-plane implementation lives
// in the kernel, control, internal, and cmd packages.
package boxd

import "embed"

// WebFiles contains the browser application served by the HTTP inbound adapter.
// Keep prototype/ as the single source of truth for the static UI.
//
//go:embed prototype/index.html prototype/app.js prototype/styles.css prototype/favicon.svg prototype/assets
var WebFiles embed.FS

// WebRoot is the path of the embedded browser application.
const WebRoot = "prototype"
