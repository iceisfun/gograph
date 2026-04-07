# Node Binding

Every Lua script receives a metatable-backed `node` global. The script
has two phases: a **define phase** (top-level code that declares the node
type's shape) and a **runtime phase** (handler functions called by the engine).

## Define Phase — Node Type Setup

Top-level code runs once when the node type is registered via
`golua.Register(reg, name, script)`. It declares the node's label,
category, slots, config schema, and visual properties.

### node:set_label(label)

Sets the default display label for the node title bar.

```lua
node:set_label("My Sensor")
```

### node:set_category(category)

Sets the category string. The frontend uses this to pick **node colors**
— each unique category gets its own fill, stroke, and title bar color.

```lua
node:set_category("sensor")
node:set_category("actuator")
node:set_category("transform")
```

Categories are arbitrary strings. Use whatever makes sense for your
domain. The default theme includes five categories (`source`,
`transform`, `output`, `delay`, `logic`) with predefined colors, but
**any string works** — unknown categories get auto-generated colors
derived from the category name.

To customize colors, pass `nodeCategories` in the theme options:

```javascript
GoGraph.create(element, {
    theme: {
        nodeCategories: {
            sensor:   { fill: '#1a3a2e', stroke: '#0f6040', titleBar: '#1a4a3e' },
            actuator: { fill: '#3e1a1a', stroke: '#a03030', titleBar: '#4e1a1a' },
        }
    }
});
```

This merges with the defaults — existing categories are preserved unless
you explicitly override them.

### node:set_content_height(pixels)

Sets the height of the content area below the title bar and slots.
Required for `display()` to render anything.

```lua
node:set_content_height(40)   -- enough for 2 text lines
node:set_content_height(120)  -- dashboard with progress, LEDs, sparkline
```

### node:set_interactive(bool)

Marks the node as clickable. Interactive nodes render a click button
in the content area and fire `on_click()` when the user clicks.

```lua
node:set_interactive(true)
```

### node:add_input(id, name, dataType)

Declares an input slot. The `dataType` determines connection kind:
state-like types (`"state"`, `"bool"`, `"coil"`) create state connections;
event-like types (`"any"`, `"string"`, `"number"`) create event connections.

```lua
node:add_input("in", "Input", "any")        -- event input
node:add_input("enable", "Enable", "state")  -- state input
```

### node:add_output(id, name, dataType)

Declares an output slot. Same dataType rules as inputs.

```lua
node:add_output("out", "Output", "any")      -- use self:emit()
node:add_output("coil", "Coil", "state")     -- use self:set()
```

### node:define_config(key, default, label)

Declares a user-editable config field. All config values are strings.

```lua
node:define_config("period", "2000", "Period (ms)")
node:define_config("threshold", "50", "Threshold")
node:define_config("message", "Hello", "Message")
```

### Complete define example

```lua
-- Top-level code: define phase
node:set_label("Rate Limit")
node:set_category("transform")
node:set_content_height(50)
node:add_input("in", "Input", "any")
node:add_output("out", "Output", "any")
node:define_config("rate", "2000", "Rate (ms)")

-- Handler definitions follow...
function node:on_init()
    -- ...
end
```

## Lifecycle Handlers

### function node:on_init()

Called once when the node runner starts. Use it to initialize persistent
state and start periodic ticks.

```lua
function node:on_init()
    self.state.count = 0
    self:init_tick(tonumber(self.config.period) or 5000)
    self:set_label("Counter 0")
end
```

### function node:on_shutdown()

Called when the node runner is stopped (graph removed, engine shut down).
Use for cleanup logging.

```lua
function node:on_shutdown()
    self:log("shutting down with count=" .. tostring(self.state.count))
end
```

### function node:on_config()

Called when the node's config changes at runtime (user edits via the UI).
Use it to reconfigure ticks, update labels, etc.

```lua
function node:on_config()
    self:init_tick(tonumber(self.config.period) or 5000)
    self:set_label("Timer " .. self.config.period .. "ms")
end
```

### function node:on_connect(e) / function node:on_disconnect(e)

Called when a connection is added or removed. The `e` table has
`e.connection` with the connection details.

```lua
function node:on_connect(e)
    self:log("connected: " .. e.connection.from_node .. " -> " .. e.connection.to_node)
end
```

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

### function node:on_change(e)

Called when a state input changes value. The `e` table contains:
`e.slot`, `e.value`, `e.prev`, `e.source`. Fires after `on_high`/`on_low`
(if applicable). See [event.md](event.md) for the full schema.

```lua
function node:on_change(e)
    local a = self.inputs.a
    local b = self.inputs.b
    self:set("out", (a == "1" and b == "1") and "1" or "0")
end
```

### function node:on_high(e)

Called when a state input transitions from falsy to truthy. Same `e`
table as `on_change`. Fires before `on_change`.

```lua
function node:on_high(e)
    self:set("out", "1")
    self:display("ON")
end
```

### function node:on_low(e)

Called when a state input transitions from truthy to falsy. Same `e`
table as `on_change`. Fires before `on_change`.

```lua
function node:on_low(e)
    self:set("out", "0")
    self:display("OFF")
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

### self:emit(slot, val)

Sends a value on an event output slot. Every call triggers propagation.

```lua
self:emit("out", "hello")
```

### self:set(slot, val)

Sets a state output slot with change detection. Only propagates when the
value differs from the previous one.

```lua
self:set("out", "1")
self:set("level", tostring(voltage))
```

### self:set_label(label)

Updates the node's display label at runtime. Broadcasts a `node.update`
SSE event.

```lua
self:set_label("Switch (ON)")
```

### self:display(text)  /  self:display(slotName, text, opts)  /  self:display(slotName, opts)

Sets visual content rendered inside the node body on the canvas. The
`ContentSlot` interface supports 8 concrete slot types, each with its
own canvas renderer.

**Single-argument form** (default text slot):

```lua
self:display("ON")
self:display(tostring(count))
```

**Named text slot** with optional style table:

```lua
self:display("status", "ACTIVE", { color = "#0f0", animate = "pulse", duration = 500 })
self:display("value", tostring(reading))
```

**Typed slot** via opts table (for non-text slot types):

```lua
self:display("progress", { type = "progress", value = 0.75, duration = 2000, color = "#4CAF50" })
self:display("leds", { type = "led", states = {true, false, true} })
self:display("loading", { type = "spinner", visible = true })
self:display("status", { type = "badge", text = "OK", color = "#fff", background = "#2ecc71" })
self:display("chart", { type = "sparkline", values = {1.2, 1.5, 1.3, 1.8, 1.1} })
self:display("icon", { type = "image", src = "data:image/png;base64,...", width = 24, height = 24 })
self:display("logo", { type = "svg", markup = "<svg>...</svg>", width = 32, height = 32 })
```

See [event.md](event.md) for the full list of slot types and their
fields. Accepts strings and numbers for text slots. Triggers a
`node.content` SSE event with change detection (only emits when the
display value actually changes).

#### Display Slot Types

All slot types share a `BaseSlot` with `Type`, `Color`, `Animate`, and
`Duration` fields. The `type` field in the opts table selects the
concrete type. JSON uses a `"type"` discriminator for polymorphic
encoding.

| Type | Description | Key Fields |
|------|-------------|------------|
| `text` | Styled text (default) | `color`, `size`, `align`, `font`, `animate`, `duration` |
| `progress` | Animated progress bar (0..1) | `value`, `duration`, `color` |
| `led` | Row of indicator circles | `states` (array of booleans) |
| `spinner` | Rotating arc animation | `visible` |
| `badge` | Colored pill label | `text`, `color`, `background` |
| `sparkline` | Inline mini-chart | `values` (array of numbers) |
| `image` | Inline raster image | `src` (data URI), `width`, `height` |
| `svg` | SVG rendered via blob URL | `markup`, `width`, `height` |

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

## Ticks and Scheduling

Ticks are the primary mechanism for source nodes that generate data on
their own schedule, and for deferred operations like delays and rate
limiting.

### self:init_tick(ms)

Starts a **recurring** periodic tick. The engine calls `on_tick()` every
`ms` milliseconds. Calling `init_tick` again replaces the previous
interval.

```lua
function node:on_init()
    self:init_tick(2000)  -- tick every 2 seconds
end

function node:on_tick()
    self:emit("out", tostring(time.now()))
end

function node:on_config()
    -- Reconfigure when user changes the interval
    self:init_tick(tonumber(self.config.interval) or 2000)
end
```

### self:schedule_tick(ms)

Schedules a **one-shot** tick after `ms` milliseconds. When it fires,
the engine calls `on_tick()` once. Use `0` for an immediate tick on the
next loop iteration.

Unlike `init_tick`, a scheduled tick does not repeat — you must call
`schedule_tick` again if you want another one.

```lua
function node:on_event(e)
    -- Received data, emit it after a delay
    self.state.pending = e.value
    local delay = tonumber(self.config.delay) or 1000
    self:schedule_tick(delay)
end

function node:on_tick()
    if self.state.pending then
        self:emit("out", self.state.pending)
        self.state.pending = nil
    end
end
```

### Combining recurring and one-shot ticks

Both can coexist. `init_tick` runs independently on its own timer;
`schedule_tick` fires through a separate channel. Both call `on_tick()`
— use `self.state` to distinguish what triggered the tick if needed.

```lua
function node:on_init()
    self:init_tick(5000)  -- periodic heartbeat
end

function node:on_event(e)
    -- Also schedule an immediate re-evaluation
    self:schedule_tick(0)
end

function node:on_tick()
    -- Called by both init_tick (periodic) and schedule_tick (one-shot)
    self:emit("out", tostring(self.state.count or 0))
end
```

### function node:on_tick()

Called by both `init_tick` (periodic) and `schedule_tick` (one-shot).
This is where source nodes generate data, and where deferred operations
complete.

```lua
function node:on_tick()
    local period = tonumber(self.config.period) or 2000
    local phase = math.floor(time.now() / period) % 2
    self:set("out", phase == 0 and "1" or "0")
    self:display(phase == 0 and "ON" or "OFF")
end
```

## Persistent State

`self.state` is a Lua table that persists across handler calls for the
lifetime of the node runner. It is **not** persisted to the store — it
resets when the engine restarts.

```lua
function node:on_init()
    self.state.count = 0
    self.state.history = {}
    self.state.last_emit = 0
end

function node:on_tick()
    self.state.count = self.state.count + 1
    self:display(tostring(self.state.count))
end
```

Use `self.state` for:
- Counters, accumulators
- Queue buffers (head/tail indices)
- Previous values for change detection
- Timekeeping (last emit time for rate limiting)

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

### Periodic source with dynamic label

```lua
node:set_label("Source")
node:set_category("source")
node:set_content_height(30)
node:add_output("out", "Output", "any")
node:define_config("message", "Hello", "Message")
node:define_config("interval", "5000", "Interval (ms)")

function node:on_init()
    self:init_tick(tonumber(self.config.interval) or 5000)
    self:set_label("Source: " .. (self.config.message or "Hello"))
end

function node:on_config()
    self:init_tick(tonumber(self.config.interval) or 5000)
    self:set_label("Source: " .. (self.config.message or "Hello"))
end

function node:on_tick()
    local msg = self.config.message or "Hello"
    self:emit("out", msg)
    self:display(msg)
end
```

### State oscillator

```lua
node:set_label("Oscillator")
node:set_category("source")
node:set_content_height(30)
node:add_output("out", "Output", "state")
node:define_config("period", "2000", "Period (ms)")

function node:on_init()
    self:init_tick(tonumber(self.config.period) or 2000)
end

function node:on_config()
    self:init_tick(tonumber(self.config.period) or 2000)
end

function node:on_tick()
    local period = tonumber(self.config.period) or 2000
    local phase = math.floor(time.now() / period) % 2
    local on = phase == 0
    self:set("out", on and "1" or "0")
    self:display(on and "ON" or "OFF")
end
```

### Transform with event passthrough

```lua
node:set_label("Lowercase")
node:set_category("transform")
node:add_input("in", "Input", "any")
node:add_output("out", "Output", "any")

function node:on_event(e)
    local data = e.value or self.inputs["in"]
    self:emit("out", string.lower(tostring(data or "")))
end
```

### Delay with queued events and one-shot ticks

```lua
node:set_label("Delay")
node:set_category("delay")
node:set_content_height(30)
node:add_input("in", "Input", "any")
node:add_output("out", "Output", "any")
node:define_config("duration", "1000", "Duration (ms)")

function node:on_init()
    self.state.qh = 1
    self.state.qt = 1
end

function node:on_event(e)
    local ms = tonumber(self.config.duration) or 1000
    local val = e.value or self.inputs["in"]
    self.state[self.state.qt] = { value = val, at = time.now() + ms }
    self.state.qt = self.state.qt + 1
    self:schedule_tick(ms)  -- one-shot: fire after delay
end

function node:on_tick()
    local now = time.now()
    while self.state.qh < self.state.qt do
        local entry = self.state[self.state.qh]
        if entry.at > now then
            self:schedule_tick(entry.at - now)  -- re-schedule for next item
            return
        end
        self:emit("out", entry.value)
        self.state[self.state.qh] = nil
        self.state.qh = self.state.qh + 1
    end
end
```

### Interactive toggle with state output

```lua
node:set_label("Toggle")
node:set_category("source")
node:set_interactive(true)
node:set_content_height(40)
node:add_output("out", "Output", "state")

function node:update_state()
    local on = self.config.state == "on"
    self:set("out", on and "1" or "0")
    self:display(on and "ON" or "OFF")
end

function node:on_init()
    self:update_state()
end

function node:on_click()
    if self.config.state == "on" then
        self:set_config("state", "off")
    else
        self:set_config("state", "on")
    end
    self:update_state()
end
```

### State logic gate (AND)

```lua
node:set_label("AND")
node:set_category("logic")
node:set_content_height(30)
node:add_input("a", "A", "state")
node:add_input("b", "B", "state")
node:add_output("out", "Output", "state")

local function truthy(v)
    return v == "1" or v == "true" or v == "on"
end

function node:on_change(e)
    local r = truthy(self.inputs.a) and truthy(self.inputs.b)
    self:set("out", r and "1" or "0")
    self:display(r and "1" or "0")
end
```

### Mixed node: state enable + event passthrough

```lua
node:set_label("Switch")
node:set_category("transform")
node:set_content_height(30)
node:add_input("en", "Enable", "state")
node:add_input("in", "Data", "any")
node:add_output("out", "Output", "any")
node:add_output("discard", "Discard", "any")

function node:on_change(e)
    -- State input changed — update display
    local enabled = self.inputs.en == "1"
    self:display(enabled and "OPEN" or "CLOSED")
end

function node:on_event(e)
    -- Event input arrived — route based on enable state
    local enabled = self.inputs.en == "1"
    local val = e.value or self.inputs["in"]
    if enabled then
        self:emit("out", val)
    else
        self:emit("discard", val)
    end
end
```

### Dashboard with rich display slots

```lua
node:set_label("Dashboard")
node:set_category("output")
node:set_content_height(120)
node:define_config("interval", "2000", "Interval (ms)")

function node:on_init()
    self.state.step = 0
    self:init_tick(tonumber(self.config.interval) or 2000)
    self:display("status", { type="badge", text="INIT", background="#3498db" })
    self:display("bar", { type="progress", value=0 })
end

function node:on_tick()
    self.state.step = (self.state.step or 0) + 1
    local step = self.state.step
    local interval = tonumber(self.config.interval) or 2000

    -- Progress bar cycles 0..1
    self:display("bar", { type="progress", value=(step % 8) / 8, duration=interval })

    -- Badge cycles through states
    local badges = {
        { text="OK",   bg="#2ecc71" },
        { text="BUSY", bg="#f39c12" },
        { text="WARN", bg="#e67e22" },
        { text="ERR",  bg="#e74c3c" },
    }
    local b = badges[(step % 4) + 1]
    self:display("status", { type="badge", text=b.text, color="#fff", background=b.bg })
end
```
