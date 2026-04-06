# Event Binding

Scripts interact with the event system through handler methods and
side-effect methods on the `node` object.

## Script Events (Handlers)

The engine calls these methods on the node at the appropriate time.
Scripts define handlers by overriding them.

### on_event(e)

Called when anything happens to the node. The `e` parameter is a table
that describes what triggered the call. `self.inputs` always contains
all latest input values regardless of event type.

```lua
function node:on_event(e)
    local data = e.value or self.inputs["in"]
    self.outputs.out = string.lower(tostring(data or ""))
    self:display("processing")
end
```

#### Event table fields

| Field          | Type              | Description                                           |
|----------------|-------------------|-------------------------------------------------------|
| `e.type`       | string            | `"tick"`, `"arrival"`, or `"eval"`                    |
| `e.slot`       | string or nil     | Input slot that received data (arrival only)          |
| `e.value`      | any or nil        | The value that arrived (arrival only)                 |
| `e.source`     | string or nil     | Upstream node ID (arrival only)                       |
| `e.connection` | Connection or nil | The connection it came through (arrival only)         |

#### Event types

**tick** -- Source node periodic evaluation. No arrival info.

```lua
function node:on_event(e)
    -- e.type == "tick"
    -- e.slot, e.value, e.source, e.connection are nil
    local period = tonumber(self.config.period) or 2000
    local phase = math.floor(time.now() / period) % 2
    self.outputs.out = phase == 0 and "1" or "0"
    self:display(phase == 0 and "ON" or "OFF")
end
```

**arrival** -- Data arrived on a specific input slot via a connection.

```lua
function node:on_event(e)
    if e.type == "arrival" then
        -- e.slot: which input received data (e.g. "in")
        -- e.value: the value that arrived
        -- e.source: upstream node ID
        -- e.connection: Connection object (id, from_node, from_slot, etc.)
        self:log("got " .. tostring(e.value) .. " on slot " .. e.slot)
    end
    self.outputs.out = string.lower(tostring(e.value or self.inputs["in"] or ""))
end
```

**eval** -- Re-evaluation after a click or full Execute cycle. No arrival info.

```lua
function node:on_event(e)
    -- e.type == "eval"
    -- e.slot, e.value, e.source, e.connection are nil
    local on = self.config.state == "on"
    self.outputs.out = on and "1" or "0"
    self:display(on and "ON" or "OFF")
end
```

### on_change(e)

Called when a state input changes value. The `e` parameter is a table
describing the change.

#### Change event table fields

| Field      | Type   | Description                              |
|------------|--------|------------------------------------------|
| `e.slot`   | string | Input slot that changed                  |
| `e.value`  | any    | New value                                |
| `e.prev`   | any    | Previous value (nil on first change)     |
| `e.source` | string | Upstream node ID                         |

```lua
function node:on_change(e)
    self:log(e.slot .. " changed from " .. tostring(e.prev) .. " to " .. tostring(e.value))
    local a = self.inputs.a
    local b = self.inputs.b
    self:set("out", (a == "1" and b == "1") and "1" or "0")
end
```

### on_high(e)

Called when a state input transitions from falsy to truthy (same `e`
table as `on_change`). Fires before `on_change`.

```lua
function node:on_high(e)
    self:set("out", "1")
    self:display("ON")
end
```

### on_low(e)

Called when a state input transitions from truthy to falsy (same `e`
table as `on_change`). Fires before `on_change`.

```lua
function node:on_low(e)
    self:set("out", "0")
    self:display("OFF")
end
```

### on_click()

Called when a user clicks an interactive node. The handler typically
changes config state. The engine persists config changes and triggers
`on_event` with `e.type == "eval"` afterward.

```lua
function node:on_click()
    if self.config.state == "on" then
        self:set_config("state", "off")
    else
        self:set_config("state", "on")
    end
end
```

## Side-Effect Methods

### self:emit(slot, val)

Sends a value on an event output slot. Every call triggers propagation
regardless of the previous value. Use this for discrete messages.

```lua
self:emit("out", "hello")
self:emit("data", tostring(42))
```

### self:set(slot, val)

Sets a state output slot. The engine performs change detection and only
propagates when the value differs from the previous one. Use this for
continuous state (booleans, levels, registers).

```lua
self:set("out", "1")
self:set("level", tostring(voltage))
```

### self:set_label(label)

Updates the node's display label at runtime. The change is broadcast
via a `node.update` SSE event.

```lua
self:set_label("Switch (ON)")
```

### self:display(text)  /  self:display(slotName, text, opts)  /  self:display(slotName, opts)

Emits a `node.content` SSE event that updates visual content rendered
inside the node body on the canvas. The `ContentSlot` interface supports
8 concrete slot types, each with its own canvas renderer.

**Single-argument form** (default text slot):

```lua
self:display("ON")
self:display(tostring(42))
```

**Named text slot** with optional style table:

```lua
self:display("status", "ACTIVE", { color = "#0f0", animate = "pulse", duration = 500 })
self:display("value", tostring(reading))
```

**Typed slot** via opts table:

```lua
self:display("bar", { type = "progress", value = 0.75, duration = 2000, color = "#4CAF50" })
self:display("leds", { type = "led", states = {true, false, true} })
```

#### Text Slot Style Options

| Key        | Type   | Description                              |
|------------|--------|------------------------------------------|
| `color`    | string | CSS color                                |
| `size`     | number | Font size in px (0 = theme default)      |
| `align`    | string | `"left"`, `"center"`, or `"right"`       |
| `font`     | string | `"monospace"` or `"sans-serif"`          |
| `animate`  | string | `"flash"`, `"pulse"`, or `"none"`        |
| `duration` | number | Animation duration in ms                 |

#### Content Slot Types

All slot types share a `BaseSlot` with `Type`, `Color`, `Animate`, and
`Duration` fields. The `type` field in the opts table selects the
concrete type. Polymorphic JSON uses a `"type"` discriminator.

| Type | Go Struct | Lua Usage | Visual |
|------|-----------|-----------|--------|
| `text` | `TextSlot` | `self:display("name", "text", {color, size, align, font, animate, duration})` | Styled text (default) |
| `progress` | `ProgressSlot` | `self:display("name", {type="progress", value=0.75, duration=2000, color="#4CAF50"})` | Animated bar 0..1 |
| `led` | `LedSlot` | `self:display("name", {type="led", states={true, false, true}})` | Row of indicator circles |
| `spinner` | `SpinnerSlot` | `self:display("name", {type="spinner", visible=true})` | Rotating arc |
| `badge` | `BadgeSlot` | `self:display("name", {type="badge", text="OK", color="#fff", background="#2ecc71"})` | Colored pill |
| `sparkline` | `SparklineSlot` | `self:display("name", {type="sparkline", values={1.2, 1.5, 1.3}})` | Inline chart |
| `image` | `ImageSlot` | `self:display("name", {type="image", src="data:...", width=24, height=24})` | Inline image |
| `svg` | `SvgSlot` | `self:display("name", {type="svg", markup="<svg>...</svg>", width=32, height=32})` | SVG rendered via blob URL |

**Change detection**: The engine tracks the last display value per slot.
If the script sets the same display content and options as the previous
execution, no SSE event is emitted.

**Type**: `node.content`
**Payload**: `{ nodeID, slots: { slotName: { type, ...slot fields } } }`

### self:glow(duration_ms)

Emits a `node.active` SSE event that triggers a border glow animation
on the node for the specified duration in milliseconds.

```lua
self:glow(500)   -- half-second glow
self:glow(2000)  -- two-second glow
```

**Type**: `node.active`
**Payload**: `{ nodeID, duration }`

### self:set_config(key, value)

Updates a config value. The change is visible immediately in
`self.config` and persisted by the engine after the handler returns.
The engine also broadcasts a `node.update` SSE event.

```lua
self:set_config("state", "on")
```

**Scope**: Typically used in `on_click`. Can also be used in
`on_event` for self-modifying nodes.

## Engine-Generated Events

These are generated automatically by the engine, not directly
controllable from scripts:

### connection.state

Emitted by `StateWire` when the upstream state value changes. The
frontend renders the wire with a steady glow when the value is truthy,
dim when falsy.

**Active**: value is not empty, `"0"`, `"false"`, `"off"`, or `nil`

### event.start / event.end

Emitted for `EventConnection` wires only. An animated dot traverses
the wire over the configured duration. `event.start` spawns the dot,
`event.end` removes it on arrival.

### event.cancel

Emitted when the engine context is cancelled (e.g. shutdown). All
in-flight animated dots are removed immediately.

### node.update

Emitted when a node is modified — click handler changes config, REST
API modifies the node, etc.

### graph.update

Emitted when the full graph state changes via the REST API. Also sent
on SSE connect with the full graph snapshot. The engine injects current
`Node.Content` (display slot state) into the snapshot so new clients
see the current visual state immediately without waiting for Lua
handlers to fire again.

## Event Flow

### Engine tick
```
Engine timer fires
  -> evaluateSources()
    -> on_event(e) for each source node   [e.type = "tick"]
      -> self.outputs.X = value
      -> self:display("text")
    -> propagateFrom() cascades to downstream nodes
      -> on_event(e) for each downstream  [e.type = "arrival"]
        -> e.slot, e.value, e.source, e.connection populated
        -> self.inputs has ALL latest values
```

### User click
```
User clicks interactive node
  -> server.handleClickNode()
    -> engine.ClickNode()
      -> on_click() handler runs
        -> self:set_config("state", "on")
      -> engine persists config updates
    -> server broadcasts node.update
    -> engine.PropagateFrom()
      -> on_event(e) handler runs          [e.type = "eval"]
        -> self.outputs.out = "1"
        -> self:display("ON")
```

### Data arrival
```
Upstream node produces output
  -> engine propagates via connection
    -> on_event(e) on downstream node      [e.type = "arrival"]
      -> e.slot = "in"                     (which slot received data)
      -> e.value = "hello"                 (the arriving value)
      -> e.source = "node1"               (upstream node ID)
      -> e.connection = Connection         (the connection object)
      -> self.inputs has ALL latest values (not just the arriving one)
```

## SSE Protocol

All events follow the envelope format:

```json
{
    "v": 1,
    "ts": 1712345678000,
    ...payload fields
}
```

| Field  | Description                          |
|--------|--------------------------------------|
| `v`    | Protocol version (currently 1)       |
| `ts`   | Unix timestamp in milliseconds       |

## Full Event Type Reference

| Type                | Source          | Description                        |
|---------------------|----------------|------------------------------------|
| `node.content`      | script/engine  | Node display content changed (polymorphic slots) |
| `node.active`       | script/engine  | Node glow animation                |
| `node.update`       | REST API/click | Node added or modified             |
| `connection.state`  | engine         | Instant wire state changed         |
| `connection.update` | REST API       | Connection added or modified       |
| `event.start`       | engine         | Timed dot animation started        |
| `event.update`      | engine         | In-flight event modified           |
| `event.end`         | engine         | Timed dot arrived at target        |
| `event.cancel`      | engine         | In-flight events cancelled         |
| `graph.update`      | REST API       | Full graph state replaced          |
