// Package data provides embedded man page data.
package data

import (
	_ "embed"
)

// ManPagesDB is the embedded SQLite database containing all man pages.
// This is populated at build time by the loader tool.
//
//go:embed manpages.db
var ManPagesDB []byte
