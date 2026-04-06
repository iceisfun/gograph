package lua

import (
	"context"
	"fmt"
	"strconv"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"

	"github.com/iceisfun/gograph/graph"
)

// NodeCallbacks are Go functions provided by the node runner. They're
// called directly from Lua during handler execution.
type NodeCallbacks struct {
	Emit         func(slot string, value any)                // send value on output slot
	Set          func(slot string, value any)                // set state output (with change detection)
	Display      func(slotName string, slot graph.ContentSlot) // set display content in named slot
	Glow         func(durationMs int)         // trigger glow animation
	Log          func(msg string)             // log with node ID prefix
	SetConfig    func(key, value string)      // update config (persisted)
	SetLabel     func(label string)           // update node label at runtime
	InitTick     func(ms int)                 // start periodic tick
	ScheduleTick func(ms int)                 // schedule one-shot tick (0=immediate)
}

// NodeVM holds the persistent VM and key tables for a running node.
type NodeVM struct {
	VM        *vm.VM
	NodeTbl   *vm.Table // the "node" global table
	InputsTbl *vm.Table // self.inputs (mutated in-place on arrivals)
	ConfigTbl *vm.Table // self.config (mutated in-place on configure)
}

// CreateNodeVM builds a persistent VM for a node. It parses and compiles
// the script once, creates the VM with a persistent node binding, and
// runs the top-level code (type declarations + handler definitions).
//
// The returned NodeVM is owned by a single goroutine (the node runner).
// The VM is NOT thread-safe — all access must be from that goroutine.
func CreateNodeVM(
	ctx context.Context,
	nodeID, graphID string,
	nt graph.NodeType,
	config map[string]string,
	cb NodeCallbacks,
) (*NodeVM, error) {
	if nt.Script == "" {
		return nil, fmt.Errorf("node type %q has no script", nt.Name)
	}

	block, err := parser.Parse(nt.Name, nt.Script)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", nt.Name, err)
	}

	proto, err := compiler.Compile(nt.Name, block)
	if err != nil {
		return nil, fmt.Errorf("compile %q: %w", nt.Name, err)
	}

	v := vm.New(
		vm.WithContext(ctx),
		vm.WithLimits(vm.Limits{
			MaxInstructions: 10_000_000, // higher limit for persistent VMs
			MaxCallDepth:    200,
			MaxStackSlots:   10000,
		}),
	)

	v.SetTimeProvider(vm.NewDefaultTimeProvider())
	stdlib.Open(v)

	// Build persistent node table.
	inputsTbl := vm.NewEmptyTable()
	configTbl := vm.NewEmptyTable()
	for k, val := range config {
		configTbl.SetString(k, vm.NewString(val))
	}

	nodeTbl := buildPersistentNodeBinding(nodeID, nt.Name, "", inputsTbl, configTbl, cb)
	v.SetGlobal("node", vm.NewTable(nodeTbl))

	// Run top-level code — type declarations + handler definitions.
	if _, err := v.Run(proto); err != nil {
		return nil, fmt.Errorf("init %q: %w", nt.Name, err)
	}

	return &NodeVM{
		VM:        v,
		NodeTbl:   nodeTbl,
		InputsTbl: inputsTbl,
		ConfigTbl: configTbl,
	}, nil
}

// CallHandler looks up a handler on the node table and calls it via
// ProtectedCall. Returns nil if the handler doesn't exist.
func CallHandler(v *vm.VM, nodeTbl *vm.Table, name string, eventTbl *vm.Table) error {
	handler := nodeTbl.Get(vm.NewString(name))
	if !handler.IsCallable() {
		return nil
	}
	nodeVal := vm.NewTable(nodeTbl)
	var args []vm.Value
	if eventTbl != nil {
		args = []vm.Value{nodeVal, vm.NewTable(eventTbl)}
	} else {
		args = []vm.Value{nodeVal}
	}
	if _, err := v.ProtectedCall(handler, args); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// BuildEventTable creates the event table passed to on_event(e).
//
//	e.type        — "tick" | "arrival" | "eval"
//	e.slot        — string or nil
//	e.value       — any or nil
//	e.source      — string or nil
//	e.connection  — Connection table or nil
func BuildEventTable(eventType string, slot string, value any, source string, conn graph.Connection) *vm.Table {
	e := vm.NewEmptyTable()
	e.SetString("type", vm.NewString(eventType))

	if slot != "" {
		e.SetString("slot", vm.NewString(slot))
		e.SetString("value", GoToLuaValue(value))
		e.SetString("source", vm.NewString(source))
		if conn != nil {
			e.SetString("connection", vm.NewTable(BuildConnectionBinding(conn)))
		}
	}

	// __tostring
	meta := vm.NewEmptyTable()
	meta.SetString("__tostring", vm.NewNativeFunc(func(v *vm.VM) int {
		tbl := v.Get(1).AsTable()
		typ := tbl.Get(vm.NewString("type"))
		s := "event(" + typ.AsString()
		sl := tbl.Get(vm.NewString("slot"))
		if sl.IsString() {
			s += ", slot=" + sl.AsString()
		}
		val := tbl.Get(vm.NewString("value"))
		if !val.IsNil() {
			s += ", value=" + vm.ValueToString(val)
		}
		src := tbl.Get(vm.NewString("source"))
		if src.IsString() {
			s += ", source=" + src.AsString()
		}
		s += ")"
		v.Set(0, vm.NewString(s))
		return 1
	}))
	e.SetMetatable(meta)

	return e
}

// BuildConnectEventTable creates the event table passed to on_connect(e)
// and on_disconnect(e). Contains e.connection with the connection details.
func BuildConnectEventTable(conn graph.Connection) *vm.Table {
	e := vm.NewEmptyTable()
	if conn != nil {
		e.SetString("connection", vm.NewTable(BuildConnectionBinding(conn)))
	}
	return e
}

// BuildChangeEventTable creates the event table passed to on_change(e),
// on_high(e), and on_low(e) state handlers.
//
//	e.slot   — input slot name
//	e.value  — new value
//	e.prev   — previous value
//	e.source — source node ID
func BuildChangeEventTable(slot string, value, prev any, source string) *vm.Table {
	e := vm.NewEmptyTable()
	e.SetString("type", vm.NewString("change"))
	e.SetString("slot", vm.NewString(slot))
	e.SetString("value", GoToLuaValue(value))
	e.SetString("prev", GoToLuaValue(prev))
	e.SetString("source", vm.NewString(source))
	return e
}

// ---------------------------------------------------------------------------
// Persistent node binding
// ---------------------------------------------------------------------------

func buildPersistentNodeBinding(
	nodeID, typeName, label string,
	inputsTbl, configTbl *vm.Table,
	cb NodeCallbacks,
) *vm.Table {
	node := vm.NewEmptyTable()

	// --- Identity ---
	node.SetString("id", vm.NewString(nodeID))
	node.SetString("type", vm.NewString(typeName))
	node.SetString("label", vm.NewString(label))

	// --- Data tables (persistent, mutated in-place) ---
	node.SetString("inputs", vm.NewTable(inputsTbl))
	node.SetString("config", vm.NewTable(configTbl))
	node.SetString("state", vm.NewTable(vm.NewEmptyTable()))
	node.SetString("incoming", vm.NewTable(vm.NewEmptyTable()))
	node.SetString("outgoing", vm.NewTable(vm.NewEmptyTable()))

	// --- emit ---
	node.SetString("emit", vm.NewNativeFunc(func(v *vm.VM) int {
		slot := v.Get(2)
		val := v.Get(3)
		if !slot.IsString() {
			panic(&vm.LuaError{Value: vm.NewString("emit: slot name must be a string")})
		}
		cb.Emit(slot.AsString(), LuaToGoValue(val))
		return 0
	}))

	// --- set (state output) ---
	node.SetString("set", vm.NewNativeFunc(func(v *vm.VM) int {
		slot := v.Get(2)
		val := v.Get(3)
		if !slot.IsString() {
			panic(&vm.LuaError{Value: vm.NewString("set: slot name must be a string")})
		}
		cb.Set(slot.AsString(), LuaToGoValue(val))
		return 0
	}))

	// --- display (overloaded) ---
	// 1 arg: display(text)           → default slot
	// 2 args: display(slot, text)    → named slot
	// 3 args: display(slot, text, opts) → named slot with style
	node.SetString("display", vm.NewNativeFunc(func(v *vm.VM) int {
		arg1 := v.Get(2) // colon call: self=1, args start at 2
		arg2 := v.Get(3)
		arg3 := v.Get(4)

		if arg2.IsNil() {
			// 1 arg: display(text) → default slot
			var text string
			if arg1.IsString() {
				text = arg1.AsString()
			} else if arg1.IsNumber() {
				text = vm.ValueToString(arg1)
			} else {
				return 0
			}
			cb.Display("default", &graph.TextSlot{Text: text})
		} else if arg2.IsTable() {
			// 2 args: display(slot, opts) → named slot from opts table (no text arg)
			if !arg1.IsString() {
				return 0
			}
			cb.Display(arg1.AsString(), parseContentSlotOpts(arg2.AsTable(), ""))
		} else {
			// 2+ args: display(slot, text [, opts])
			if !arg1.IsString() {
				return 0
			}
			slotName := arg1.AsString()
			var text string
			if arg2.IsString() {
				text = arg2.AsString()
			} else if arg2.IsNumber() {
				text = vm.ValueToString(arg2)
			} else {
				return 0
			}
			if arg3.IsTable() {
				cb.Display(slotName, parseContentSlotOpts(arg3.AsTable(), text))
			} else {
				cb.Display(slotName, &graph.TextSlot{Text: text})
			}
		}
		return 0
	}))

	// --- glow ---
	node.SetString("glow", vm.NewNativeFunc(func(v *vm.VM) int {
		dur := v.Get(2)
		if dur.IsNumber() {
			cb.Glow(int(dur.AsInt()))
		}
		return 0
	}))

	// --- log ---
	node.SetString("log", vm.NewNativeFunc(func(v *vm.VM) int {
		msg := v.Get(2)
		if msg.IsString() {
			cb.Log(msg.AsString())
		} else {
			cb.Log(vm.ValueToString(msg))
		}
		return 0
	}))

	// --- set_config ---
	node.SetString("set_config", vm.NewNativeFunc(func(v *vm.VM) int {
		key := v.Get(2)
		val := v.Get(3)
		if !key.IsString() {
			return 0
		}
		k := key.AsString()
		var s string
		if val.IsString() {
			s = val.AsString()
		} else {
			s = vm.ValueToString(val)
		}
		cb.SetConfig(k, s)
		configTbl.SetString(k, vm.NewString(s))
		return 0
	}))

	// --- init_tick ---
	node.SetString("init_tick", vm.NewNativeFunc(func(v *vm.VM) int {
		ms := v.Get(2)
		if ms.IsNumber() && ms.AsInt() > 0 {
			cb.InitTick(int(ms.AsInt()))
		}
		return 0
	}))

	// --- schedule_tick ---
	node.SetString("schedule_tick", vm.NewNativeFunc(func(v *vm.VM) int {
		ms := v.Get(2)
		if ms.IsNumber() && ms.AsInt() > 0 {
			cb.ScheduleTick(int(ms.AsInt()))
		} else {
			cb.ScheduleTick(0)
		}
		return 0
	}))

	// --- set_label (works at runtime) ---
	node.SetString("set_label", vm.NewNativeFunc(func(v *vm.VM) int {
		arg := v.Get(2)
		if arg.IsString() {
			cb.SetLabel(arg.AsString())
		}
		return 0
	}))

	// --- Type definition methods (noops at runtime) ---
	noop := vm.NewNativeFunc(func(v *vm.VM) int { return 0 })
	node.SetString("set_category", noop)
	node.SetString("set_content_height", noop)
	node.SetString("set_interactive", noop)
	node.SetString("add_input", noop)
	node.SetString("add_output", noop)
	node.SetString("define_config", noop)

	// --- Base event handlers (noops, scripts override) ---
	node.SetString("on_event", noop)
	node.SetString("on_tick", noop)
	node.SetString("on_click", noop)
	node.SetString("on_config", noop)
	node.SetString("on_connect", noop)
	node.SetString("on_disconnect", noop)
	node.SetString("on_init", noop)
	node.SetString("on_shutdown", noop)

	return node
}

// ---------------------------------------------------------------------------
// Connection binding (read-only table per connection)
// ---------------------------------------------------------------------------

// luaTruthy returns true unless the value is nil or false.
func luaTruthy(v vm.Value) bool {
	if v.IsNil() {
		return false
	}
	if v.IsBool() {
		return v.AsBool()
	}
	return true
}

// parseBaseSlot reads shared style fields from a Lua table.
func parseBaseSlot(t vm.LuaTable) graph.BaseSlot {
	var b graph.BaseSlot
	if v := t.Get(vm.NewString("color")); v.IsString() {
		b.Color = v.AsString()
	}
	if v := t.Get(vm.NewString("animate")); v.IsString() {
		b.Animate = v.AsString()
	}
	if v := t.Get(vm.NewString("duration")); v.IsNumber() {
		b.Duration = int(v.AsInt())
	}
	return b
}

// parseContentSlotOpts reads a Lua options table and returns the appropriate
// concrete ContentSlot type based on the "type" field.
func parseContentSlotOpts(t vm.LuaTable, text string) graph.ContentSlot {
	slotType := ""
	if v := t.Get(vm.NewString("type")); v.IsString() {
		slotType = v.AsString()
	}
	base := parseBaseSlot(t)

	switch slotType {
	case "progress":
		s := &graph.ProgressSlot{BaseSlot: base}
		if v := t.Get(vm.NewString("value")); v.IsNumber() {
			s.Value = v.AsFloat()
		}
		return s

	case "led":
		s := &graph.LedSlot{BaseSlot: base}
		if v := t.Get(vm.NewString("states")); v.IsTable() {
			tbl := v.AsTable()
			for i := int64(1); ; i++ {
				elem := tbl.Get(vm.NewInt(i))
				if elem.IsNil() {
					break
				}
				s.States = append(s.States, luaTruthy(elem))
			}
		}
		return s

	case "spinner":
		s := &graph.SpinnerSlot{BaseSlot: base}
		if v := t.Get(vm.NewString("visible")); v.IsBool() {
			s.Visible = v.AsBool()
		}
		return s

	case "badge":
		s := &graph.BadgeSlot{BaseSlot: base, Text: text}
		if v := t.Get(vm.NewString("text")); v.IsString() {
			s.Text = v.AsString()
		}
		if v := t.Get(vm.NewString("background")); v.IsString() {
			s.Background = v.AsString()
		}
		return s

	case "sparkline":
		s := &graph.SparklineSlot{BaseSlot: base}
		if v := t.Get(vm.NewString("values")); v.IsTable() {
			tbl := v.AsTable()
			for i := int64(1); ; i++ {
				elem := tbl.Get(vm.NewInt(i))
				if elem.IsNil() {
					break
				}
				if elem.IsNumber() {
					s.Values = append(s.Values, elem.AsFloat())
				}
			}
		}
		if v := t.Get(vm.NewString("min")); v.IsNumber() {
			f := v.AsFloat()
			s.Min = &f
		}
		if v := t.Get(vm.NewString("max")); v.IsNumber() {
			f := v.AsFloat()
			s.Max = &f
		}
		return s

	case "image":
		s := &graph.ImageSlot{BaseSlot: base}
		if v := t.Get(vm.NewString("src")); v.IsString() {
			s.Src = v.AsString()
		}
		if v := t.Get(vm.NewString("width")); v.IsNumber() {
			s.Width = int(v.AsInt())
		}
		if v := t.Get(vm.NewString("height")); v.IsNumber() {
			s.Height = int(v.AsInt())
		}
		return s

	case "svg":
		s := &graph.SvgSlot{BaseSlot: base, Markup: text}
		if v := t.Get(vm.NewString("markup")); v.IsString() {
			s.Markup = v.AsString()
		}
		if v := t.Get(vm.NewString("width")); v.IsNumber() {
			s.Width = int(v.AsInt())
		}
		if v := t.Get(vm.NewString("height")); v.IsNumber() {
			s.Height = int(v.AsInt())
		}
		return s

	default: // "text" or ""
		s := &graph.TextSlot{BaseSlot: base, Text: text}
		if v := t.Get(vm.NewString("size")); v.IsNumber() {
			s.Size = int(v.AsInt())
		}
		if v := t.Get(vm.NewString("align")); v.IsString() {
			s.Align = v.AsString()
		}
		if v := t.Get(vm.NewString("font")); v.IsString() {
			s.Font = v.AsString()
		}
		return s
	}
}

// BuildConnectionBinding creates a Lua table representing a connection.
func BuildConnectionBinding(c graph.Connection) *vm.Table {
	tbl := vm.NewEmptyTable()
	tbl.SetString("id", vm.NewString(c.GetID()))
	tbl.SetString("from_node", vm.NewString(c.GetFromNode()))
	tbl.SetString("from_slot", vm.NewString(c.GetFromSlot()))
	tbl.SetString("to_node", vm.NewString(c.GetToNode()))
	tbl.SetString("to_slot", vm.NewString(c.GetToSlot()))

	cfg := c.GetConfig()
	cfgTbl := vm.NewEmptyTable()
	for k, val := range cfg {
		cfgTbl.SetString(k, vm.NewString(val))
	}
	tbl.SetString("config", vm.NewTable(cfgTbl))

	var dur int64
	if cfg != nil {
		if d, ok := cfg["duration"]; ok {
			if ms, err := strconv.ParseInt(d, 10, 64); err == nil && ms > 0 {
				dur = ms
			}
		}
	}
	tbl.SetString("duration", vm.NewInt(dur))

	return tbl
}

// ---------------------------------------------------------------------------
// Value conversion helpers (exported for use by engine package)
// ---------------------------------------------------------------------------

// GoMapToLuaTable converts a Go map to a Lua table.
func GoMapToLuaTable(m map[string]any) *vm.Table {
	t := vm.NewEmptyTable()
	for k, v := range m {
		t.SetString(k, GoToLuaValue(v))
	}
	return t
}

// GoToLuaValue converts a Go value to a Lua Value.
func GoToLuaValue(v any) vm.Value {
	switch val := v.(type) {
	case nil:
		return vm.Nil
	case bool:
		return vm.NewBool(val)
	case int:
		return vm.NewInt(int64(val))
	case int64:
		return vm.NewInt(val)
	case float64:
		return vm.NewFloat(val)
	case string:
		return vm.NewString(val)
	case []any:
		t := vm.NewEmptyTable()
		for i, item := range val {
			t.SetInt(i+1, GoToLuaValue(item))
		}
		return vm.NewTable(t)
	case map[string]any:
		return vm.NewTable(GoMapToLuaTable(val))
	default:
		return vm.NewString(fmt.Sprintf("%v", val))
	}
}

// LuaToGoValue converts a Lua Value to a Go value.
func LuaToGoValue(v vm.Value) any {
	switch {
	case v.IsNil():
		return nil
	case v.IsBool():
		return v.AsBool()
	case v.IsInt():
		return v.AsInt()
	case v.IsFloat():
		return v.AsFloat()
	case v.IsString():
		return v.AsString()
	case v.IsTable():
		return LuaTableToGoAny(v.AsTable())
	default:
		return vm.ValueToString(v)
	}
}

// LuaTableToGoAny converts a Lua table to either []any (if it's a pure
// integer sequence) or map[string]any (otherwise).
func LuaTableToGoAny(t vm.LuaTable) any {
	hasString := false
	maxInt := int64(0)
	intCount := 0

	var key vm.Value = vm.Nil
	for {
		nextKey, _, err := t.Next(key)
		if err != nil || nextKey.IsNil() {
			break
		}
		if nextKey.IsString() {
			hasString = true
		} else if nextKey.IsInt() && nextKey.AsInt() > 0 {
			intCount++
			if nextKey.AsInt() > maxInt {
				maxInt = nextKey.AsInt()
			}
		}
		key = nextKey
	}

	if !hasString && intCount > 0 && maxInt == int64(intCount) {
		arr := make([]any, intCount)
		for i := int64(1); i <= maxInt; i++ {
			arr[i-1] = LuaToGoValue(t.Get(vm.NewInt(i)))
		}
		return arr
	}

	return LuaTableToGoMap(t)
}

// LuaTableToGoMap converts a Lua table to a Go map (string keys only).
func LuaTableToGoMap(t vm.LuaTable) map[string]any {
	result := make(map[string]any)
	var key vm.Value = vm.Nil
	for {
		nextKey, value, err := t.Next(key)
		if err != nil || nextKey.IsNil() {
			break
		}
		if nextKey.IsString() {
			result[nextKey.AsString()] = LuaToGoValue(value)
		}
		key = nextKey
	}
	return result
}
