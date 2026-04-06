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
	Emit         func(slot string, value any) // send value on output slot
	Display      func(text string)            // set node display content
	Glow         func(durationMs int)         // trigger glow animation
	Log          func(msg string)             // log with node ID prefix
	SetConfig    func(key, value string)      // update config (persisted)
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
func BuildEventTable(eventType string, slot string, value any, source string, conn *graph.Connection) *vm.Table {
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
func BuildConnectEventTable(conn *graph.Connection) *vm.Table {
	e := vm.NewEmptyTable()
	if conn != nil {
		e.SetString("connection", vm.NewTable(BuildConnectionBinding(conn)))
	}
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

	// --- display ---
	node.SetString("display", vm.NewNativeFunc(func(v *vm.VM) int {
		text := v.Get(2)
		if text.IsString() {
			cb.Display(text.AsString())
		} else if text.IsNumber() {
			cb.Display(vm.ValueToString(text))
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

	// --- Type definition methods (noops at runtime) ---
	noop := vm.NewNativeFunc(func(v *vm.VM) int { return 0 })
	node.SetString("set_label", noop)
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

// BuildConnectionBinding creates a Lua table representing a connection.
func BuildConnectionBinding(c *graph.Connection) *vm.Table {
	tbl := vm.NewEmptyTable()
	tbl.SetString("id", vm.NewString(c.ID))
	tbl.SetString("from_node", vm.NewString(c.FromNode))
	tbl.SetString("from_slot", vm.NewString(c.FromSlot))
	tbl.SetString("to_node", vm.NewString(c.ToNode))
	tbl.SetString("to_slot", vm.NewString(c.ToSlot))

	cfgTbl := vm.NewEmptyTable()
	for k, val := range c.Config {
		cfgTbl.SetString(k, vm.NewString(val))
	}
	tbl.SetString("config", vm.NewTable(cfgTbl))

	var dur int64
	if c.Config != nil {
		if d, ok := c.Config["duration"]; ok {
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
