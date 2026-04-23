package web

import "embed"

// Files contains embedded web templates and static assets.
//
//go:embed all:templates all:static
var Files embed.FS
