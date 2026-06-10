package static

import "embed"

// WebFS contains the built frontend static assets.
//go:embed dist/*
var WebFS embed.FS
