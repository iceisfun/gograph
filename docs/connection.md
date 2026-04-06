# Connection Binding

Connection objects are read-only snapshots of the graph edges attached
to a node. They are accessible from scripts via `node.incoming` and
`node.outgoing` arrays.

## Fields

| Field             | Type   | Description                              |
|-------------------|--------|------------------------------------------|
| `conn.id`         | string | Connection ID                            |
| `conn.from_node`  | string | Source node ID                           |
| `conn.from_slot`  | string | Source slot ID                           |
| `conn.to_node`    | string | Target node ID                           |
| `conn.to_slot`    | string | Target slot ID                           |
| `conn.config`     | table  | Per-connection config (string key/value) |
| `conn.kind`       | string | `"event"` or `"state"`                   |
| `conn.duration`   | number | Traversal duration in ms (event only)    |

All fields are read-only. The `duration` field is a convenience shortcut
derived from `conn.config.duration` (only meaningful for event connections).

## Connection Kinds

`Connection` is an interface in Go with two concrete types. The kind is
determined by the source slot's `DataType` via `SlotConnectionKind()`:

- **State-like types** (`"state"`, `"bool"`, `"coil"`, `"register"`, `"numeric"`, `"value"`) produce a `StateConnection`.
- **Everything else** (e.g. `"string"`, `"any"`) produces an `EventConnection`.

### State Connections (kind = "state")

State connections carry a continuous value — something that "is" rather
than a message that "travels". The engine uses `connection.state` SSE
events on value change. The frontend renders a steady glow when the
value is truthy, dim when falsy. No dot animation.

State connections are created with `self:set(slot, val)` on the source
node. The engine performs change detection and only propagates when the
value actually changes.

### Event Connections (kind = "event")

Event connections carry discrete messages with optional traversal
animation. The engine emits `event.start` and `event.end` SSE events.
An animated dot traverses the wire over the configured duration.

Event connections are created with `self:emit(slot, val)` on the source
node. Every call triggers propagation regardless of the previous value.

## Accessing Connections

```lua
-- Check how many downstream connections this node has
local n = #node.outgoing
node:log("connected to " .. n .. " downstream nodes")

-- Find all state connections
for i = 1, #node.outgoing do
    local conn = node.outgoing[i]
    if conn.kind == "state" then
        node:log("state wire to " .. conn.to_node)
    end
end

-- Find event connections with traversal delay
for i = 1, #node.outgoing do
    local conn = node.outgoing[i]
    if conn.kind == "event" and conn.duration > 0 then
        node:log("delayed event wire to " .. conn.to_node)
    end
end

-- Read connection config
for i = 1, #node.incoming do
    local conn = node.incoming[i]
    local priority = conn.config.priority or "normal"
    node:log("input from " .. conn.from_node .. " priority=" .. priority)
end
```

## Use Cases

- **Routing decisions**: A node can check which downstream nodes are
  connected and route data accordingly.
- **Fan-in awareness**: A node can check how many inputs are connected
  to decide when it has enough data to proceed.
- **Topology-aware logic**: Scripts can adapt behavior based on connection
  properties like duration or custom config values.

## Go-Side Representation

`Connection` is an interface in `graph`:

```go
type Connection interface {
    GetID() string
    GetFromNode() string
    GetFromSlot() string
    GetToNode() string
    GetToSlot() string
    GetConfig() map[string]string
    Kind() ConnectionKind
}
```

Two concrete types implement it:

```go
// EventConnection carries discrete messages with optional traversal animation.
type EventConnection struct {
    BaseConnection
    Duration int `json:"duration,omitempty"`
}

// StateConnection carries continuous state. The wire publishes
// connection.state SSE events on value change rather than animating
// a traversal dot.
type StateConnection struct {
    BaseConnection
    StateDataType string `json:"stateDataType,omitempty"`
}
```

`ConnectionKind` is either `EventKind` or `StateKind`. Use
`SlotConnectionKind(dataType)` to determine which kind a slot implies:

```go
// "state", "bool", "coil", "register", "numeric", "value" → StateKind
// everything else → EventKind
func SlotConnectionKind(dataType string) ConnectionKind
```

The Lua binding creates a plain table snapshot for each connection with
the fields listed above. The `kind` field is set to `"event"` or
`"state"`. The `duration` field is parsed from `Config["duration"]` as
a convenience (event connections only).
