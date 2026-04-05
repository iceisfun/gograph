// Package frontend provides the embedded frontend filesystem for the
// GoGraph web UI.
package frontend

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var distFS embed.FS

// FS returns the frontend filesystem rooted at the dist/ directory.
func FS() fs.FS {
	sub, _ := fs.Sub(distFS, "dist")
	return sub
}
