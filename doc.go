// Package gograph is a canvas-based graph engine combining a Go backend
// with an embedded TypeScript frontend.
//
// # Architecture
//
// The system is organized into several packages:
//
//   - [graph] defines the core data model: Graph, Node, Slot, NodeType,
//     and the SSE wire protocol event types. [graph.Connection] is an
//     interface with two concrete types: [graph.EventConnection] (discrete
//     messages with dot animation) and [graph.StateConnection] (continuous
//     state with steady glow). [graph.ContentSlot] is an interface with
//     8 concrete slot types (TextSlot, ProgressSlot, LedSlot, SpinnerSlot,
//     BadgeSlot, SparklineSlot, ImageSlot, SvgSlot) for rich node display.
//   - [engine] provides goroutine-per-node graph execution with channel-based
//     connections and emits real-time events for visualization.
//   - [lua] implements node execution using embedded Lua (golua), providing
//     a sandboxed scripting environment.
//   - [server] provides an HTTP server with REST API for graph management,
//     SSE for real-time event streaming, and static file serving for the
//     embedded frontend.
//   - [store] defines the persistence interface and provides JSON file
//     and in-memory implementations.
//   - [frontend] contains the compiled TypeScript canvas renderer,
//     embedded into the Go binary via go:embed.
//
// # Quick Start
//
//	mux := http.NewServeMux()
//	srv := gograph.Mount(mux, "/graph", gograph.MountOptions{
//	    Store:    store.NewMemoryStore(),
//	    Registry: reg,
//	})
//	srv.SetEngine(eng)
//	log.Fatal(http.ListenAndServe(":8080", mux))
//
// [Mount] attaches the server and embedded frontend to an existing mux.
// The lower-level [server.New] pattern is still available for full control.
//
// # Extension Points
//
// Custom node types are registered with [graph.Registry]. Each type
// declares its input/output slots and optionally a Lua script for
// execution logic. Connection validation uses slot data types; provide
// a custom [graph.ConnectionValidator] for advanced rules.
//
// The frontend is fully themeable. The server accepts any [fs.FS] via
// [server.WithStaticFS], allowing custom frontends or development-mode
// file serving. The frontend exports a GoGraph class for programmatic
// mounting into any DOM element, with data attribute auto-init support
// and a destroy() method for cleanup.
package gograph
