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

### self:display(text)

Emits a `node.content` SSE event that updates the text rendered inside
the node body on the canvas.

```lua
self:display("ON")
self:display(tostring(42))
```

**Change detection**: The engine tracks the last display value per node.
If the script sets the same display text as the previous execution, no
SSE event is emitted.

**Type**: `node.content`
**Payload**: `{ nodeID, text }`

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

Emitted for instant connections (duration = 0) when the upstream value
changes. The frontend renders the wire as glowing (active) or dim
(inactive) based on the value truthiness.

**Active**: value is not empty, `"0"`, `"false"`, `"off"`, or `nil`

### event.start / event.end

Emitted for timed connections (duration > 0). An animated dot traverses
the wire over the specified duration. `event.start` spawns the dot,
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
