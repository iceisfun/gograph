package server

import (
	"io/fs"
	"net/http"
	"strings"
)

// staticHandler returns an http.Handler that serves files from the given
// filesystem. Requests for the root path serve index.html. Paths starting
// with /api/ are not handled (they should be routed before this handler).
func staticHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't serve API paths through the static handler.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Try to serve the requested file. If it doesn't exist, serve
		// index.html so the SPA router can handle client-side routing.
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Check if the file exists; if not, fall back to index.html.
		if _, err := fs.Stat(fsys, path); err != nil {
			r.URL.Path = "/"
		}

		fileServer.ServeHTTP(w, r)
	})
}
