package web

import "embed"

// Files contains embedded web templates and static assets.
//go:embed templates/*.html static/*/*.js
var Files embed.FS
