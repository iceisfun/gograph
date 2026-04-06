---
name: gograph
description: GoGraph — graph/node execution engine with Go+TypeScript via SSE, event and state connections, goroutine-per-node architecture, and Lua scripting for node logic.
license: MIT
compatibility: claude-code, opencode
metadata:
  language: go
  domain: graph-engine
---

# GoGraph Skill

Use this when helping someone who imported `github.com/iceisfun/gograph` and wants to build graph-based node execution systems, or who is writing Lua scripts for node logic.

## SKILLS

Copy-paste block for an AI assistant:

```text
SKILLS:
- GoGraph is a canvas-based graph/node execution engine. Go backend + embedded TypeScript frontend, communicating via SSE (JSON).
- Module: github.com/iceisfun/gograph. Lua scripting via github.com/iceisfun/golua.
- Graph model: Graph, Node, Slot, Connection (interface), NodeType, Registry.
- Two connection types: EventConnection (discrete messages, dot animation, duration) and StateConnection (continuous state, steady glow, change detection).
- Goroutine-per-node engine architecture with channel-based wire connections (EventWire/StateWire).
- Connection is an interface. EventConnection for discrete messages; StateConnection for continuous state (coils, registers, discrete I/O).
- Slot DataType determines connection kind: "state"/"bool"/"coil"/"register"/"numeric"/"value" -> StateConnection; "any"/"string"/"number" -> EventConnection.
- Constructors: graph.NewEventConnection(id, from, fromSlot, to, toSlot, config), graph.NewStateConnection(id, from, fromSlot, to, toSlot, dataType, config).
- Lua define phase: node:set_label(), node:set_category(), node:add_input(id, name, dataType), node:add_output(id, name, dataType), node:define_config(key, default, label).
- Lua event handlers: on_event(e), on_tick(), on_click(), on_init(), on_config(), on_connect(e), on_disconnect(e).
- Lua state handlers: on_change(e) with e.value/e.prev/e.slot/e.source, on_high(e), on_low(e).
- Lua methods: self:emit(slot, val), self:set(slot, val), self:display(text) or self:display(slotName, text, opts) or self:display(slotName, opts), self:glow(ms), self:set_config(k,v), self:set_label(label), self:log(msg), self:init_tick(ms), self:schedule_tick(ms).
- ContentSlot is an interface with 8 concrete types: TextSlot, ProgressSlot, LedSlot, SpinnerSlot, BadgeSlot, SparklineSlot, ImageSlot, SvgSlot. All share BaseSlot (Type, Color, Animate, Duration). Polymorphic JSON with "type" discriminator.
- Display slot types: text (default styled text), progress (bar 0..1), led (indicator circles), spinner (rotating arc), badge (colored pill), sparkline (inline chart), image (data URI), svg (blob URL). Content state is cached on Node.Content and seeded to new SSE clients on connect.
- Lua state: self.inputs, self.config, self.state (persistent across handler calls), self.incoming, self.outgoing.
- REST API at /api/graphs/{id}/... for CRUD. SSE at /api/graphs/{id}/events for real-time updates.
- Embeddable via gograph.Mount(mux, "/graph", MountOptions{Store, Registry, StaticFS}) convenience or lower-level server.WithRoutePrefix("/graph") + server.WithStaticFS(frontend.FS()).
- Frontend GoGraph class: GoGraph.create(element, {graphId?, apiBase?, readOnly?, darkMode?, theme?}). Data attribute auto-init: <div data-gograph data-graph-id="x" data-api="/graph/api">. window.GoGraph global. destroy() for cleanup.
- Packages: graph/ (core types, Connection interface, Registry, SSE events), engine/ (goroutine-per-node, WireRunner), lua/ (sandboxed Lua scripting), server/ (HTTP, REST, SSE), store/ (MemoryStore, JSONStore), frontend/ (embedded TS canvas renderer).
```

## What You Usually Need To Know

GoGraph is a graph execution engine where nodes run as goroutines and communicate through typed connections. The two connection types model fundamentally different communication patterns:

- **EventConnection**: fire-and-forget discrete messages. Think button clicks, triggers, data packets. Animated with a traveling dot. Use `self:emit(slot, val)`.
- **StateConnection**: continuous shared state with change detection. Think PLC coils, registers, sensor values. Shown with a steady glow. Use `self:set(slot, val)`. Listeners receive `on_change(e)`, `on_high(e)`, `on_low(e)`.

The slot's `DataType` determines which connection type is created: state-like types ("state", "bool", "coil", "register", "numeric", "value") produce StateConnections; event-like types ("any", "string", "number") produce EventConnections.

## Packages

| Package | Purpose |
|---------|---------|
| `graph/` | Core types: Graph, Node, Slot, Connection interface, EventConnection, StateConnection, Registry, SSE event types |
| `engine/` | Goroutine-per-node execution engine, WireRunner (EventWire/StateWire), channel-based wire connections |
| `lua/` | Sandboxed Lua scripting for node logic via github.com/iceisfun/golua |
| `server/` | HTTP server, REST API, SSE event streaming |
| `store/` | Persistence backends: MemoryStore, JSONStore |
| `frontend/` | Embedded TypeScript canvas renderer |

## Connection System

### Connection Interface

`Connection` is the interface. Concrete types are `EventConnection` and `StateConnection`.

```go
// Event connection — discrete messages
conn := graph.NewEventConnection(id, fromNodeID, fromSlotID, toNodeID, toSlotID, config)

// State connection — continuous state with change detection
conn := graph.NewStateConnection(id, fromNodeID, fromSlotID, toNodeID, toSlotID, dataType, config)
```

### DataType to Connection Kind Mapping

| DataType | Connection Kind |
|----------|----------------|
| `"state"`, `"bool"`, `"coil"`, `"register"`, `"numeric"`, `"value"` | StateConnection |
| `"any"`, `"string"`, `"number"` | EventConnection |

## Lua Node Scripting

### Define Phase

Called once when the node type is registered. Sets up the node's metadata, slots, and config schema.

```lua
node:set_label("My Node")
node:set_category("Logic")
node:add_input("in1", "Input 1", "any")
node:add_output("out1", "Output 1", "any")
node:define_config("threshold", 10, "Threshold")
```

### Event Handlers

```lua
function node:on_init()
    self:init_tick(1000)
end

function node:on_event(e)
    -- e.value, e.slot, e.source
    self:emit("out1", e.value)
end

function node:on_tick()
    self:emit("out1", tostring(time.now()))
end

function node:on_click()
    self:glow(500)
end

function node:on_config()
    local threshold = tonumber(self.config.threshold) or 50
end

function node:on_connect(e) end
function node:on_disconnect(e) end
```

### State Handlers

```lua
function node:on_change(e)
    -- e.value  = new value
    -- e.prev   = previous value
    -- e.slot   = slot ID
    -- e.source = source node ID
    self:set("out1", e.value)
end

function node:on_high(e)
    -- Boolean state went truthy
    self:glow(300)
end

function node:on_low(e)
    -- Boolean state went falsy
end
```

### Node Methods

```lua
-- Emit a discrete event on an output slot
self:emit("out1", value)

-- Set continuous state on an output slot
self:set("out1", value)

-- Display text on the node body (default text slot)
self:display("Hello")

-- Display text in a named slot with options
self:display("slotName", "text", { color = "#fff" })

-- Display typed slots (progress, led, spinner, badge, sparkline, image, svg)
self:display("bar", { type = "progress", value = 0.75, duration = 2000, color = "#4CAF50" })
self:display("leds", { type = "led", states = {true, false, true} })
self:display("loading", { type = "spinner", visible = true })
self:display("status", { type = "badge", text = "OK", color = "#fff", background = "#2ecc71" })
self:display("chart", { type = "sparkline", values = {1.2, 1.5, 1.3} })
self:display("icon", { type = "image", src = "data:...", width = 24, height = 24 })
self:display("logo", { type = "svg", markup = "<svg>...</svg>", width = 32, height = 32 })

-- Visual glow effect (milliseconds)
self:glow(500)

-- Config management
self:set_config("key", "value")
self:set_label("New Label")

-- Logging
self:log("debug message")

-- Tick scheduling
self:init_tick(1000)       -- recurring tick every 1000ms
self:schedule_tick(500)    -- one-shot tick after 500ms
```

### Node State

```lua
-- Available in all handlers:
self.inputs    -- table of current input values
self.config    -- table of config key/value pairs
self.state     -- persistent table across handler calls
self.incoming  -- incoming connections
self.outgoing  -- outgoing connections
```

## Server / API

### Setting Up the Server

The recommended approach is `gograph.Mount()`, which attaches the server and
embedded frontend to an existing mux:

```go
mux := http.NewServeMux()
srv := gograph.Mount(mux, "/graph", gograph.MountOptions{
    Store:    store.NewMemoryStore(),
    Registry: reg,
})
srv.SetEngine(eng)
http.ListenAndServe(":8080", mux)
```

The lower-level `server.New()` pattern is still available:

```go
srv := server.New(
    server.WithRoutePrefix("/graph"),
    server.WithStaticFS(frontend.FS()),
)
srv.ListenAndServe(":8080")
```

### REST API

- `GET /api/graphs/{id}` — fetch a graph
- `POST /api/graphs` — create a graph
- `PUT /api/graphs/{id}` — update a graph
- `DELETE /api/graphs/{id}` — delete a graph
- CRUD endpoints for nodes, connections, etc. under `/api/graphs/{id}/...`

### SSE Events

- `GET /api/graphs/{id}/events` — real-time event stream
- Events include `connection.state` for state connection updates

### Frontend Integration

The embedded frontend exports a `GoGraph` class for mounting the editor into
any DOM element.

**Data attribute auto-init (zero JS):**

```html
<div data-gograph data-graph-id="x" data-api="/graph/api"></div>
<script type="module" src="/graph/assets/gograph.js"></script>
```

**Programmatic init:**

```js
import { GoGraph } from '/graph/assets/gograph.js';
const g = await GoGraph.create(element, {
    graphId: 'my-graph',
    apiBase: '/graph/api',
    readOnly: false,
    darkMode: true,
    theme: { /* partial Theme overrides */ },
});
g.destroy(); // cleanup
```

`window.GoGraph` is also available as a global for `<script type="module">` usage.
Falls back to `#graph-canvas` parent if no `data-gograph` elements exist.

## Complete Lua Node Example

```lua
-- Define phase
node:set_label("Threshold Gate")
node:set_category("Logic")
node:add_input("value", "Value", "numeric")
node:add_output("above", "Above", "any")
node:add_output("below", "Below", "any")
node:define_config("threshold", 50, "Threshold")

-- Handlers
function node:on_init()
    self.state.last = nil
end

function node:on_change(e)
    local threshold = tonumber(self.config.threshold) or 50
    local val = tonumber(e.value) or 0

    if val >= threshold then
        self:emit("above", val)
        self:display("ABOVE: " .. tostring(val))
    else
        self:emit("below", val)
        self:display("BELOW: " .. tostring(val))
    end

    self:glow(200)
    self.state.last = val
end

function node:on_config()
    if self.state.last then
        self:on_change({ value = self.state.last, prev = self.state.last, slot = "value", source = "" })
    end
end
```

## Display Slot Types

The `display()` method supports 8 visual slot types via the `ContentSlot` interface.
Each type has its own canvas renderer. The `type` field in the opts table selects the type.

| Type | Lua Usage | Visual |
|------|-----------|--------|
| `text` | `self:display("name", "text", {color, size, align, font, animate, duration})` | Styled text (default) |
| `progress` | `self:display("name", {type="progress", value=0.75, duration=2000, color="#4CAF50"})` | Animated bar 0..1 |
| `led` | `self:display("name", {type="led", states={true, false, true}})` | Row of indicator circles |
| `spinner` | `self:display("name", {type="spinner", visible=true})` | Rotating arc |
| `badge` | `self:display("name", {type="badge", text="OK", color="#fff", background="#2ecc71"})` | Colored pill |
| `sparkline` | `self:display("name", {type="sparkline", values={1.2, 1.5, 1.3}})` | Inline chart |
| `image` | `self:display("name", {type="image", src="data:...", width=24, height=24})` | Inline image |
| `svg` | `self:display("name", {type="svg", markup="<svg>...</svg>", width=32, height=32})` | SVG via blob URL |

All share `BaseSlot` with `Type`, `Color`, `Animate`, `Duration`. Polymorphic JSON with `"type"` discriminator.

## Common Patterns

### Timer Node

```lua
node:set_label("Timer")
node:set_category("Utility")
node:add_output("tick", "Tick", "any")
node:define_config("interval", "1000", "Interval (ms)")

function node:on_init()
    self:init_tick(tonumber(self.config.interval) or 1000)
end

function node:on_tick()
    self:emit("tick", tostring(time.now()))
end

function node:on_config()
    self:init_tick(tonumber(self.config.interval) or 1000)
end
```

### State Passthrough with Logging

```lua
node:set_label("Monitor")
node:set_category("Debug")
node:add_input("in", "Input", "state")
node:add_output("out", "Output", "state")

function node:on_change(e)
    self:log("Changed: " .. tostring(e.prev) .. " -> " .. tostring(e.value))
    self:set("out", e.value)
    self:display(tostring(e.value))
end
```

### Boolean Latch

```lua
node:set_label("Latch")
node:set_category("Logic")
node:add_input("set", "Set", "state")
node:add_input("reset", "Reset", "state")
node:add_output("q", "Q", "state")

function node:on_init()
    self.state.latched = false
end

function node:on_high(e)
    if e.slot == "set" then
        self.state.latched = true
        self:set("q", "1")
        self:glow(300)
    end
end

function node:on_low(e)
    if e.slot == "reset" and self.state.latched then
        self.state.latched = false
        self:set("q", "0")
    end
end
```
