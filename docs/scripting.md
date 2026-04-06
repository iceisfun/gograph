# Scripting Guide

GoGraph nodes are scripted in Lua using an embedded
[GoLua](https://github.com/iceisfun/golua) runtime. Each node type can
have an attached Lua script that defines event handlers.

## Execution Model

The engine provides a `node` global with base noop handlers. The script
runs once to define/override handlers and helper functions. Then the
engine calls the appropriate handler:

1. **Script setup** — top-level code runs (define helpers, override handlers)
2. **Handler call** — engine calls the appropriate handler via ProtectedCall:
   - `on_event(e)` for ticks, arrivals, and re-evaluations
   - `on_click()` for user clicks
   - `on_change(e)`, `on_high(e)`, `on_low(e)` for state input changes
3. **Collect results** — outputs, display, glow, config updates are collected

Each execution creates a **fresh sandboxed VM** — no state persists
between runs. This ensures isolation and predictability.

**Limits**: 1M instructions, 200 call depth, 10K stack slots (configurable).

## Event Handlers

### on_event(e)

The main handler. Called on engine tick, data arrival, or re-evaluation.
The `e` parameter is a table describing what triggered the call:

| Field          | Type              | Present for                |
|----------------|-------------------|----------------------------|
| `e.type`       | string            | always (`"tick"`, `"arrival"`, `"eval"`) |
| `e.slot`       | string or nil     | arrival only               |
| `e.value`      | any or nil        | arrival only               |
| `e.source`     | string or nil     | arrival only               |
| `e.connection` | Connection or nil | arrival only               |

`self.inputs` always has ALL latest input values (not just the arriving
one). Use `e.value` for direct access to the value that just arrived.

```lua
function node:on_event(e)
    local data = e.value or self.inputs["in"]
    self.outputs.out = string.lower(tostring(data or ""))
end
```

### on_change(e) / on_high(e) / on_low(e)

State input handlers. Called when a state connection delivers a changed
value. The `e` table has: `e.slot`, `e.value`, `e.prev`, `e.source`.

- `on_high(e)` — state input went from falsy to truthy (fires before `on_change`)
- `on_low(e)` — state input went from truthy to falsy (fires before `on_change`)
- `on_change(e)` — any state input change (always fires)

```lua
function node:on_change(e)
    local a = self.inputs.a
    local b = self.inputs.b
    self:set("out", (a == "1" and b == "1") and "1" or "0")
end
```

### on_click()

Called when a user clicks an interactive node. Change config here.

```lua
function node:on_click()
    if self.config.state == "on" then
        self:set_config("state", "off")
    else
        self:set_config("state", "on")
    end
end
```

After `on_click` returns, the engine persists config changes and
triggers re-evaluation (which calls `on_event` with `e.type == "eval"`
and updated config).

## The `node` Object

```
-- Identity (read-only)
self.id             -- node instance ID
self.type           -- node type name
self.label          -- display label

-- Data
self.inputs         -- read-only: input values by slot ID (ALL latest values)
self.config         -- read-only: config key/value pairs (strings)
self.outputs        -- write-capture: set output values by slot ID

-- Connections (read-only)
self.incoming       -- array of Connection objects
self.outgoing       -- array of Connection objects

-- Methods
self:emit(slot, val)   -- send event output (discrete message)
self:set(slot, val)    -- set state output (change-detected)
self:display(text)     -- set node display content (default text slot)
self:display(slot, text, opts) -- named text slot with style options
self:display(slot, opts)       -- named typed slot (progress, led, etc.)
self:set_label(label)  -- update display label at runtime
self:glow(ms)          -- trigger glow animation
self:set_config(k, v)  -- update config (persisted by engine)
self:log(msg)          -- log with node ID prefix
```

Connection objects have: `id`, `from_node`, `from_slot`, `to_node`,
`to_slot`, `config`, `kind`, `duration`.

See [node.md](node.md), [connection.md](connection.md), and
[event.md](event.md) for full API documentation.

## Display Slot Types

The `display()` method supports 8 visual slot types via the
`ContentSlot` interface. Each type has its own canvas renderer and
fields. The `type` field in the opts table selects the concrete type.

### text (default)

Styled text. Used when calling `display(text)` or `display(name, text, opts)`.

```lua
self:display("ON")
self:display("status", "ACTIVE", { color = "#0f0", animate = "pulse", duration = 500 })
```

### progress

Animated progress bar with a value from 0 to 1.

```lua
self:display("bar", { type = "progress", value = 0.75, duration = 2000, color = "#4CAF50" })
```

### led

Row of indicator circles.

```lua
self:display("leds", { type = "led", states = {true, false, true} })
```

### spinner

Rotating arc animation.

```lua
self:display("loading", { type = "spinner", visible = true })
```

### badge

Colored pill label.

```lua
self:display("status", { type = "badge", text = "OK", color = "#fff", background = "#2ecc71" })
```

### sparkline

Inline mini-chart from an array of numbers.

```lua
self:display("chart", { type = "sparkline", values = {1.2, 1.5, 1.3, 1.8, 1.1} })
```

### image

Inline raster image from a data URI.

```lua
self:display("icon", { type = "image", src = "data:image/png;base64,...", width = 24, height = 24 })
```

### svg

SVG content rendered via blob URL.

```lua
self:display("logo", { type = "svg", markup = "<svg>...</svg>", width = 32, height = 32 })
```

All types share `BaseSlot` fields: `Type`, `Color`, `Animate`, `Duration`.
Polymorphic JSON uses the `"type"` discriminator.

## Script Patterns

### Source Node (no inputs)

Source nodes receive `e.type == "tick"` on each engine tick. The event
table has no arrival info — the node generates its own data.

```lua
function node:on_event(e)
    local period = tonumber(self.config.period) or 2000
    local phase = math.floor(time.now() / period) % 2
    self.outputs.out = phase == 0 and "1" or "0"
    self:display(phase == 0 and "ON" or "OFF")
end
```

### Transform Node (input -> output)

Transform nodes receive `e.type == "arrival"` when upstream data arrives.
Use `e.value` for direct access to the arriving value, or fall back to
`self.inputs` for the full picture.

```lua
function node:on_event(e)
    local data = e.value or self.inputs["in"]
    self.outputs.out = string.lower(tostring(data or ""))
end
```

### Logic Gate with Helpers (fan-in)

When a node has multiple inputs, use `self.inputs` to read all of them.
The gate fires correctly regardless of which input triggered the arrival.

```lua
local function truthy(v)
    return v == "1" or v == "true" or v == "on"
end

function node:on_event(e)
    local a = truthy(self.inputs.a)
    local b = truthy(self.inputs.b)
    self.outputs.out = (a and b) and "1" or "0"
end
```

### Interactive Toggle (state output)

The click handler toggles state. The engine then calls `on_event` with
`e.type == "eval"` so the node can update its state output and display.

```lua
function node:on_click()
    if self.config.state == "on" then
        self:set_config("state", "off")
    else
        self:set_config("state", "on")
    end
end

function node:on_event(e)
    local on = self.config.state == "on"
    self:set("out", on and "1" or "0")
    self:display(on and "ON" or "OFF")
end
```

### State Source

A source node that produces state output. Uses `self:set()` so
downstream state wires get change detection and steady-glow rendering.

```lua
function node:on_event(e)
    local period = tonumber(self.config.period) or 2000
    local phase = math.floor(time.now() / period) % 2
    self:set("out", phase == 0 and "1" or "0")
    self:display(phase == 0 and "ON" or "OFF")
end
```

### State Logic Gate

Reacts to state input changes with `on_change`. Reads all inputs and
produces a state output.

```lua
local function truthy(v)
    return v == "1" or v == "true" or v == "on"
end

function node:on_change(e)
    local a = truthy(self.inputs.a)
    local b = truthy(self.inputs.b)
    self:set("out", (a and b) and "1" or "0")
end
```

### Mixed Node (state enable + event data passthrough)

A switch that uses a state input for enable/disable and passes event
data through when enabled.

```lua
function node:on_high(e)
    self:display("OPEN")
end

function node:on_low(e)
    self:display("CLOSED")
end

function node:on_event(e)
    if e.type == "arrival" and self.inputs.enable == "1" then
        self:emit("out", e.value)
    end
end
```

### Interactive Gate (click + event with input passthrough)

```lua
function node:on_click()
    if self.config.state == "on" then
        self:set_config("state", "off")
    else
        self:set_config("state", "on")
    end
end

function node:on_event(e)
    if self.config.state == "on" then
        self:emit("out", self.inputs["in"])
        self:display("OPEN")
    else
        self:display("CLOSED")
    end
end
```

### Display-Only Sink

```lua
function node:on_event(e)
    local data = e.value or self.inputs["in"]
    self:display(tostring(data or ""))
end
```

### Multi-Output Node

```lua
function node:on_event(e)
    local bits = tonumber(self.config.bits) or 8
    for i = 0, bits - 1 do
        self.outputs["b" .. i] = (i == pos) and "1" or "0"
    end
    self:display(table.concat(display))
end
```

### Connection-Aware Node

```lua
function node:on_event(e)
    if #self.outgoing == 0 then
        self:log("no downstream connections")
        return
    end
    self.outputs.out = self.inputs["in"]
end
```

### Event-Type-Aware Node

Use `e.type` to vary behavior depending on what triggered the call.

```lua
function node:on_event(e)
    if e.type == "tick" then
        self.outputs.out = tostring(time.now())
    elseif e.type == "arrival" then
        self:log("received " .. tostring(e.value) .. " on " .. e.slot)
        self.outputs.out = e.value
    elseif e.type == "eval" then
        self:log("re-evaluated")
    end
end
```

## Available Globals

| Global  | Description                                         |
|---------|-----------------------------------------------------|
| `node`  | Metatable-backed node object (see above)            |
| `math`  | Standard Lua math library                           |
| `string`| Standard Lua string library                         |
| `table` | Standard Lua table library                          |
| `time`  | GoLua time library (`time.now()`, `time.tick()`)    |
| `print` | Prints to server stdout                             |
| `type`  | Standard Lua `type()` function                      |
| `tostring` / `tonumber` | Standard conversions               |
| `pcall` / `error` | Error handling                            |
| `pairs` / `ipairs` / `next` | Table iteration                 |

## Go Integration

Register a node type with an inline script:

```go
reg.Register(graph.NodeType{
    Name:     "source",
    Label:    "Source",
    Category: "source",
    Slots: []graph.Slot{
        {ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
    },
    Script: `function node:on_event(e) self.outputs.out = "hello" end`,
})
```

Or load from a file:

```go
Script: mustReadFile("scripts/oscillator.lua"),
```

For interactive nodes, set `Interactive: true`:

```go
reg.Register(graph.NodeType{
    Name:        "toggle",
    Label:       "Toggle",
    Interactive: true,
    ContentHeight: 40,
    Slots: []graph.Slot{
        {ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
    },
    Script: mustReadFile("scripts/toggle.lua"),
})
```

## How It Works (Go Side)

1. `Bindings.ExecuteNode()` receives `ExecInput` with event type
2. Creates a sandboxed VM, injects `node` global with base handlers
3. Runs the user script — defines/overrides `on_event`, `on_click`, etc.
4. Builds the event table `e` with type, slot, value, source, connection
5. Calls `on_event(e)` or `on_click()` via `ProtectedCall` with the node table as `self`
6. Collects outputs, display, glow, and config updates from the Go-side context
7. Returns `ExecOutput` to the engine
