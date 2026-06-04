package web

import "embed"

// Files contains embedded static assets.
//
//go:embed all:static
var Files embed.FS
