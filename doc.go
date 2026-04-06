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
//     state with steady glow).
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
//	reg := graph.NewRegistry()
//	reg.Register(graph.NodeType{
//	    Name:  "echo",
//	    Label: "Echo",
//	    Slots: []graph.Slot{
//	        {ID: "in", Name: "Input", Direction: graph.Input, DataType: "any"},
//	        {ID: "out", Name: "Output", Direction: graph.Output, DataType: "any"},
//	    },
//	    Script: `return { out = inputs["in"] }`,
//	})
//
//	srv := server.New(
//	    server.WithStaticFS(frontend.FS()),
//	    server.WithRegistry(reg),
//	    server.WithStore(store.NewMemoryStore()),
//	)
//	log.Fatal(srv.ListenAndServe(":8080"))
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
// file serving.
package gograph
