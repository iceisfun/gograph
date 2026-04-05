package server

import (
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/iceisfun/gograph/engine"
	"github.com/iceisfun/gograph/graph"
	"github.com/iceisfun/gograph/store"
)

// Server is the HTTP server for the GoGraph engine. It provides REST
// endpoints for graph management, SSE for real-time updates, and serves
// the embedded frontend.
type Server struct {
	mux      *http.ServeMux
	store    store.GraphStore
	registry *graph.Registry
	broker   *sseBroker
	engine   *engine.Engine
	staticFS fs.FS
	prefix   string
}

// Option configures a [Server].
type Option func(*Server)

// WithStaticFS sets the filesystem used to serve frontend static files.
func WithStaticFS(fsys fs.FS) Option {
	return func(s *Server) {
		s.staticFS = fsys
	}
}

// WithRoutePrefix sets a URL prefix for all routes (e.g. "/app").
// The prefix should not have a trailing slash.
func WithRoutePrefix(prefix string) Option {
	return func(s *Server) {
		s.prefix = strings.TrimRight(prefix, "/")
	}
}

// WithStore sets the graph persistence backend.
func WithStore(st store.GraphStore) Option {
	return func(s *Server) {
		s.store = st
	}
}

// WithRegistry sets the node type registry.
func WithRegistry(r *graph.Registry) Option {
	return func(s *Server) {
		s.registry = r
	}
}

// WithEngine sets the execution engine. When set, node clicks trigger
// immediate forward propagation through instant connections.
func WithEngine(eng *engine.Engine) Option {
	return func(s *Server) {
		s.engine = eng
	}
}

// New creates a new server with the given options. If no store is
// provided, an in-memory store is used. If no registry is provided, an
// empty one is created. If no static filesystem is provided, a minimal
// fallback is used.
func New(opts ...Option) *Server {
	s := &Server{
		mux:    http.NewServeMux(),
		broker: newSSEBroker(),
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.store == nil {
		s.store = store.NewMemoryStore()
	}
	if s.registry == nil {
		s.registry = graph.NewRegistry()
	}
	if s.staticFS == nil {
		s.staticFS = newFallbackFS()
	}
	s.registerRoutes()
	return s
}

// Handler returns the server's HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// ListenAndServe starts the HTTP server on the given address.
func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

// fallbackFS serves a minimal "not built" page when no frontend is provided.
type fallbackFS struct{}

func newFallbackFS() fs.FS {
	return &fallbackFS{}
}

func (f *fallbackFS) Open(name string) (fs.File, error) {
	if name == "." || name == "index.html" {
		return &fallbackFile{}, nil
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

type fallbackFile struct {
	data   []byte
	offset int
}

var fallbackHTML = []byte(`<!DOCTYPE html>
<html><head><title>GoGraph</title></head>
<body><p>No frontend built. Pass WithStaticFS to serve your frontend.</p></body>
</html>`)

func (f *fallbackFile) Stat() (fs.FileInfo, error) {
	return &fallbackFileInfo{size: int64(len(fallbackHTML))}, nil
}

func (f *fallbackFile) Read(b []byte) (int, error) {
	if f.data == nil {
		f.data = fallbackHTML
	}
	if f.offset >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(b, f.data[f.offset:])
	f.offset += n
	if f.offset >= len(f.data) {
		return n, io.EOF
	}
	return n, nil
}

func (f *fallbackFile) Close() error { return nil }

// fallbackFileInfo implements fs.FileInfo for the fallback HTML page.
type fallbackFileInfo struct {
	size int64
}

func (fi *fallbackFileInfo) Name() string      { return "index.html" }
func (fi *fallbackFileInfo) Size() int64        { return fi.size }
func (fi *fallbackFileInfo) Mode() fs.FileMode  { return 0o444 }
func (fi *fallbackFileInfo) ModTime() time.Time { return time.Time{} }
func (fi *fallbackFileInfo) IsDir() bool        { return false }
func (fi *fallbackFileInfo) Sys() any           { return nil }
