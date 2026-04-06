package gograph

import (
	"io/fs"
	"net/http"

	"github.com/iceisfun/gograph/frontend"
	"github.com/iceisfun/gograph/graph"
	"github.com/iceisfun/gograph/server"
	"github.com/iceisfun/gograph/store"
)

// MountOptions configures a GoGraph instance mounted on an existing mux.
type MountOptions struct {
	Store    store.GraphStore
	Registry *graph.Registry
	StaticFS fs.FS // nil = use embedded frontend
}

// Mount attaches GoGraph to an existing http.ServeMux at the given path.
// It returns the server for further configuration (e.g. SetEngine).
//
//	mux := http.NewServeMux()
//	srv := gograph.Mount(mux, "/graph", gograph.MountOptions{
//	    Store:    myStore,
//	    Registry: myRegistry,
//	})
//	srv.SetEngine(eng)
func Mount(mux *http.ServeMux, path string, opts MountOptions) *server.Server {
	sopts := []server.Option{
		server.WithRoutePrefix(path),
	}
	if opts.Store != nil {
		sopts = append(sopts, server.WithStore(opts.Store))
	}
	if opts.Registry != nil {
		sopts = append(sopts, server.WithRegistry(opts.Registry))
	}
	if opts.StaticFS != nil {
		sopts = append(sopts, server.WithStaticFS(opts.StaticFS))
	} else {
		sopts = append(sopts, server.WithStaticFS(frontend.FS()))
	}
	srv := server.New(sopts...)
	mux.Handle(path+"/", srv.Handler())
	return srv
}
