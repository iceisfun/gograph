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
| `conn.duration`   | number | Traversal duration in ms (0 = instant)   |

All fields are read-only. The `duration` field is a convenience shortcut
derived from `conn.config.duration`.

## Connection Types

### Instant (duration = 0)

Instant connections propagate values forward immediately when any upstream
value changes. The engine uses `connection.state` SSE events for these —
wires glow when the value is truthy, dim when falsy.

### Timed (duration > 0)

Timed connections animate a dot traversing the wire over the specified
duration. The engine emits `event.start` and `event.end` SSE events.
Downstream nodes wait for the traversal to complete before executing.

## Accessing Connections

```lua
-- Check how many downstream connections this node has
local n = #node.outgoing
node:log("connected to " .. n .. " downstream nodes")

-- Find all instant downstream connections
for i = 1, #node.outgoing do
    local conn = node.outgoing[i]
    if conn.duration == 0 then
        node:log("instant wire to " .. conn.to_node)
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

Connections are defined in `graph.Connection`:

```go
type Connection struct {
    ID       string            `json:"id"`
    FromNode string            `json:"fromNode"`
    FromSlot string            `json:"fromSlot"`
    ToNode   string            `json:"toNode"`
    ToSlot   string            `json:"toSlot"`
    Config   map[string]string `json:"config,omitempty"`
}
```

The Lua binding creates a plain table snapshot for each connection with
the fields listed above. The `duration` field is parsed from
`Config["duration"]` as a convenience.
