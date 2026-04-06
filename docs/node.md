# Node Binding

Every Lua script receives a metatable-backed `node` global. Scripts define
event handlers on it — the engine calls the right handler when things happen.

## Event Handlers

Scripts override these methods. The base implementations are noops.

### function node:on_event(e)

Called when anything happens to the node: a source tick, data arriving on
an input slot, or a re-evaluation after a click or full Execute cycle.
The `e` parameter is a table describing what triggered the call (see
[event.md](event.md) for the full schema). `self.inputs` always contains
all latest input values, not just the one that arrived.

```lua
function node:on_event(e)
    local data = e.value or self.inputs["in"]
    self.outputs.out = string.lower(tostring(data or ""))
    self:display(data)
end
```

### function node:on_click()

Called when a user clicks an interactive node. Use `self:set_config()`
to change state. The engine persists config changes and triggers
re-evaluation afterward (which calls `on_event` with `e.type == "eval"`).

```lua
function node:on_click()
    if self.config.state == "on" then
        self:set_config("state", "off")
    else
        self:set_config("state", "on")
    end
end
```

## Identity

| Field         | Type   | Description                  |
|---------------|--------|------------------------------|
| `self.id`     | string | Instance ID (e.g. `"osc1"`)  |
| `self.type`   | string | Node type name (e.g. `"oscillator"`) |
| `self.label`  | string | Display label                |

Read-only snapshots set before the script runs.

## Inputs

```lua
self.inputs.slotName
self.inputs["in"]       -- bracket notation for reserved words
```

Read-only table of input values keyed by slot ID. Values arrive from
upstream connections. If no upstream is connected, the value is `nil`.

`self.inputs` always reflects ALL latest input values across every slot,
regardless of which slot triggered the current event. To inspect only the
value that just arrived, use `e.value` (which is `nil` for tick and eval
events).

## Config

```lua
self.config.key
self.config["period"]
```

Read-only table of node config values (user-editable key/value strings).
All values are strings — use `tonumber()` to convert numeric config.

Use `self:set_config(key, value)` in `on_click` to change config values.
Changes are persisted by the engine after the handler returns.

## Outputs

```lua
self.outputs.out = "hello"
self.outputs["b0"] = "1"
```

Write-capture table backed by a Go metatable. Writes are intercepted via
`__newindex` and stored in a Go-side map. Reads via `__index` return
previously written values. Only string keys are captured.

## Methods

### self:display(text)

Sets the text content rendered inside the node body on the canvas.

```lua
self:display("ON")
self:display(tostring(count))
```

Accepts strings and numbers. Triggers a `node.content` SSE event with
change detection (only emits when the display value actually changes).

### self:glow(duration_ms)

Triggers a glow animation on the node border for the given duration.

```lua
self:glow(500)  -- glow for 500ms
```

Emits a `node.active` SSE event.

### self:set_config(key, value)

Updates a config value. The change is reflected immediately in
`self.config` and persisted by the engine after the handler returns.

```lua
self:set_config("state", "on")
self:set_config("period", "2000")
```

Typically used in `on_click` to toggle interactive state.

### self:log(msg)

Prints a log message prefixed with the node ID.

```lua
self:log("processing started")
-- output: [osc1] processing started
```

## Connections

```lua
for i = 1, #self.incoming do
    local conn = self.incoming[i]
    print(conn.from_node, conn.from_slot)
end

for i = 1, #self.outgoing do
    local conn = self.outgoing[i]
    print(conn.to_node, conn.duration)
end
```

`self.incoming` and `self.outgoing` are read-only arrays of
[Connection](connection.md) objects.

## Complete Examples

### Source node

```lua
-- oscillator.lua
function node:on_event(e)
    local period = tonumber(self.config.period) or 2000
    local phase = math.floor(time.now() / period) % 2
    self.outputs.out = phase == 0 and "1" or "0"
    self:display(phase == 0 and "ON" or "OFF")
end
```

### Transform node

```lua
-- lowercase.lua
function node:on_event(e)
    local data = e.value or self.inputs["in"]
    self.outputs.out = string.lower(tostring(data or ""))
end
```

### Interactive node with click + event

```lua
-- toggle.lua
function node:on_click()
    if self.config.state == "on" then
        self:set_config("state", "off")
    else
        self:set_config("state", "on")
    end
end

function node:on_event(e)
    local on = self.config.state == "on"
    self.outputs.out = on and "1" or "0"
    self:display(on and "ON" or "OFF")
end
```

### Multi-output node

```lua
-- shift_register.lua
function node:on_event(e)
    local bits = tonumber(self.config.bits) or 8
    for i = 0, bits - 1 do
        self.outputs["b" .. i] = (i == pos) and "1" or "0"
    end
    self:display(table.concat(display))
end
```

## Top-Level Code

Code outside handlers runs once during script setup. Use it for helper
functions or shared constants:

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
