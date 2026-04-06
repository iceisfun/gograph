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

### self:display(text)  /  self:display(slotName, text, opts)

Emits a `node.content` SSE event that updates text rendered inside
the node body on the canvas.

**Single-argument form** (default slot):

```lua
self:display("ON")
self:display(tostring(42))
```

**Named-slot form** with optional style table:

```lua
self:display("status", "ACTIVE", { color = "#0f0", animate = "pulse", duration = 500 })
self:display("value", tostring(reading))
```

Style options:

| Key        | Type   | Description                              |
|------------|--------|------------------------------------------|
| `color`    | string | CSS color                                |
| `size`     | number | Font size in px (0 = theme default)      |
| `align`    | string | `"left"`, `"center"`, or `"right"`       |
| `font`     | string | `"monospace"` or `"sans-serif"`          |
| `animate`  | string | `"flash"`, `"pulse"`, or `"none"`        |
| `duration` | number | Animation duration in ms                 |

**Change detection**: The engine tracks the last display value per slot.
If the script sets the same display text and options as the previous
execution, no SSE event is emitted.

**Type**: `node.content`
**Payload**: `{ nodeID, slots: { slotName: { text, color, ... } } }`

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

Emitted when the full graph state changes via the REST API.

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
| `node.content`      | script/engine  | Node display text changed          |
| `node.active`       | script/engine  | Node glow animation                |
| `node.update`       | REST API/click | Node added or modified             |
| `connection.state`  | engine         | Instant wire state changed         |
| `connection.update` | REST API       | Connection added or modified       |
| `event.start`       | engine         | Timed dot animation started        |
| `event.update`      | engine         | In-flight event modified           |
| `event.end`         | engine         | Timed dot arrived at target        |
| `event.cancel`      | engine         | In-flight events cancelled         |
| `graph.update`      | REST API       | Full graph state replaced          |
