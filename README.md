# GoGraph

Canvas-based graph engine with a Go backend and embedded TypeScript frontend.

## Architecture

```
┌─────────────┐         ┌───────────────────┐
│  Go Server  │         │ TypeScript Client │
│             │  SSE    │                   │
│  graph/     ├────────►│  render/          │
│  engine/    │         │  interaction/     │
│  server/    │◄────────│  state/           │
│  lua/       │  REST   │  net/             │
│  store/     │         │  themes/          │
│  frontend/  │         │                   │
│  (embed)    │         │  (canvas)         │
└─────────────┘         └───────────────────┘
```

- **graph/** - Core types: Graph, Node, Slot, Connection, NodeType, SSE wire protocol
- **engine/** - Goroutine-per-node graph execution with event and state connections
- **lua/** - Sandboxed Lua node scripting via [golua](https://github.com/iceisfun/golua)
- **server/** - HTTP server with REST API, SSE streaming, static file serving
- **store/** - Persistence interface with JSON file and in-memory implementations
- **frontend/** - TypeScript canvas renderer embedded via `go:embed`

## Quick Start

```bash
make build
go run ./examples/basic
# Open http://127.0.0.1:8080
```

## Embedding in Your Application

```go
package main

import (
    "log"

    "github.com/iceisfun/gograph/frontend"
    "github.com/iceisfun/gograph/graph"
    "github.com/iceisfun/gograph/server"
    "github.com/iceisfun/gograph/store"
)

func main() {
    reg := graph.NewRegistry()
    reg.Register(graph.NodeType{
        Name:  "echo",
        Label: "Echo",
        Slots: []graph.Slot{
            {ID: "in", Name: "Input", Direction: graph.Input, DataType: "any"},
            {ID: "out", Name: "Output", Direction: graph.Output, DataType: "any"},
        },
        Script: `return { out = inputs["in"] }`,
    })

    srv := server.New(
        server.WithStaticFS(frontend.FS()),
        server.WithRegistry(reg),
        server.WithStore(store.NewMemoryStore()),
    )
    log.Fatal(srv.ListenAndServe(":8080"))
}
```

Mount at a sub-path within an existing application:

```go
graphServer := server.New(
    server.WithStaticFS(frontend.FS()),
    server.WithRoutePrefix("/graph"),
)
mux.Handle("/graph/", graphServer.Handler())
```

## Lua Node Scripting

Node types can include Lua scripts with event handlers. Connections come
in two kinds: **event connections** carry discrete messages (animated dot
traversal) and **state connections** carry continuous values (steady glow).

```lua
-- Event output: self:emit(slot, val) — discrete message
function node:on_event(e)
    self:emit("out", string.upper(e.value or ""))
end

-- State output: self:set(slot, val) — change-detected
function node:on_change(e)
    self:set("out", self.inputs.a == "1" and "1" or "0")
end
```

Each execution runs in a fresh sandboxed VM with limited instructions and
no file system or network access.

## Development

```bash
# Frontend watch + Go server in dev mode
make dev

# Build everything
make build

# Type-check and vet
make vet

# Run tests
make test
```

## Examples

- **[basic](examples/basic/)** - Minimal graph with source, transform, and sink nodes
- **[embedded](examples/embedded/)** - Mounting GoGraph within a larger HTTP application
- **[lua](examples/lua/)** - Lua-scripted node execution with animated events
- **[showcase](examples/showcase/)** - Full demo with oscillators, toggles, logic gates, state and event connections
