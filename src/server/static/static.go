// Package static provides embedded static assets for casman.
package static

import "embed"

//go:embed *.css *.js *.svg
var Files embed.FS
