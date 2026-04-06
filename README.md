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

The simplest way to embed GoGraph in an existing application is with
`gograph.Mount()`, which wires up the server, frontend, and route prefix
in a single call:

```go
mux := http.NewServeMux()
srv := gograph.Mount(mux, "/graph", gograph.MountOptions{
    Store:    store.NewMemoryStore(),
    Registry: myRegistry,
})
// srv is a *server.Server — attach an engine, customize further, etc.
srv.SetEngine(eng)
log.Fatal(http.ListenAndServe(":8080", mux))
```

`MountOptions.StaticFS` defaults to the embedded frontend. The `/config`
endpoint returns `{"apiBase": "/graph/api", "mode": "edit"}` so the
frontend discovers its API root automatically.

The lower-level `server.New()` pattern still works if you need full control:

```go
srv := server.New(
    server.WithStaticFS(frontend.FS()),
    server.WithRegistry(reg),
    server.WithStore(store.NewMemoryStore()),
)
log.Fatal(srv.ListenAndServe(":8080"))
```

### Frontend Integration

The embedded frontend exports a `GoGraph` class that can mount the graph
editor into any DOM element.

**Data attributes (zero JS):**

```html
<div data-gograph data-graph-id="my-graph" data-api="/graph/api"
     style="width:800px;height:600px"></div>
<script type="module" src="/graph/assets/gograph.js"></script>
```

**Programmatic init:**

```html
<div id="editor" style="width:100%;height:600px"></div>
<script type="module">
import { GoGraph } from '/graph/assets/gograph.js';
// or use the window.GoGraph global

const g = await GoGraph.create(document.getElementById('editor'), {
    graphId:  'my-graph',
    apiBase:  '/graph/api',
    readOnly: false,
    darkMode: true,
});

// Later: g.destroy() to clean up
</script>
```

If no `data-gograph` elements are found, the frontend falls back to the
`#graph-canvas` parent for backward compatibility.

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

Nodes can display rich visual content using 8 slot types: text, progress
bars, LED indicators, spinners, badges, sparklines, images, and SVGs.

```lua
self:display("bar", { type = "progress", value = 0.75, color = "#4CAF50" })
self:display("leds", { type = "led", states = {true, false, true} })
self:display("chart", { type = "sparkline", values = {1.2, 1.5, 1.3} })
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
