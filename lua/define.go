package lua

import (
	"fmt"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"

	"github.com/iceisfun/gograph/graph"
)

// defineContext captures type declarations made by top-level script code.
type defineContext struct {
	label         string
	category      string
	contentHeight int
	interactive   bool
	slots         []graph.Slot
	configSchema  []graph.ConfigField
}

// Define runs a Lua script's top-level code to collect its type
// definition (label, category, slots, config schema). Returns a fully
// populated [graph.NodeType] ready for registration.
//
// The script's top-level code calls methods like node:set_label(),
// node:add_output(), node:define_config() to declare the type shape.
// Handler definitions (on_init, on_event, etc.) are parsed but not called.
func Define(name, script string) (graph.NodeType, error) {
	block, err := parser.Parse(name, script)
	if err != nil {
		return graph.NodeType{}, fmt.Errorf("parse %q: %w", name, err)
	}

	proto, err := compiler.Compile(name, block)
	if err != nil {
		return graph.NodeType{}, fmt.Errorf("compile %q: %w", name, err)
	}

	v := vm.New(vm.WithLimits(vm.Limits{
		MaxInstructions: 100_000,
		MaxCallDepth:    50,
		MaxStackSlots:   1000,
	}))
	v.SetTimeProvider(vm.NewDefaultTimeProvider())
	stdlib.Open(v)

	def := &defineContext{}
	nodeTbl := buildDefineBinding(def)
	v.SetGlobal("node", vm.NewTable(nodeTbl))

	if _, err := v.Run(proto); err != nil {
		return graph.NodeType{}, fmt.Errorf("define %q: %w", name, err)
	}

	nt := graph.NodeType{
		Name:          name,
		Label:         def.label,
		Category:      def.category,
		ContentHeight: def.contentHeight,
		Interactive:   def.interactive,
		Slots:         def.slots,
		ConfigSchema:  def.configSchema,
		Script:        script,
	}
	if nt.Label == "" {
		nt.Label = name
	}

	return nt, nil
}

// Register is a convenience that defines a node type from a Lua script
// and registers it in one step.
func Register(reg *graph.Registry, name, script string) error {
	nt, err := Define(name, script)
	if err != nil {
		return err
	}
	return reg.Register(nt)
}

// buildDefineBinding creates the node table used during the define phase.
// Type-declaration methods capture into defCtx. Runtime methods are noops.
func buildDefineBinding(def *defineContext) *vm.Table {
	node := vm.NewEmptyTable()

	// --- Type definition methods ---

	node.SetString("set_label", vm.NewNativeFunc(func(v *vm.VM) int {
		arg := v.Get(2)
		if arg.IsString() {
			def.label = arg.AsString()
		}
		return 0
	}))

	node.SetString("set_category", vm.NewNativeFunc(func(v *vm.VM) int {
		arg := v.Get(2)
		if arg.IsString() {
			def.category = arg.AsString()
		}
		return 0
	}))

	node.SetString("set_content_height", vm.NewNativeFunc(func(v *vm.VM) int {
		arg := v.Get(2)
		if arg.IsNumber() {
			def.contentHeight = int(arg.AsInt())
		}
		return 0
	}))

	node.SetString("set_interactive", vm.NewNativeFunc(func(v *vm.VM) int {
		arg := v.Get(2)
		if arg.IsBool() {
			def.interactive = arg.AsBool()
		}
		return 0
	}))

	node.SetString("add_input", vm.NewNativeFunc(func(v *vm.VM) int {
		id := v.Get(2)
		name := v.Get(3)
		dataType := v.Get(4)
		if !id.IsString() || !name.IsString() {
			return 0
		}
		dt := "any"
		if dataType.IsString() {
			dt = dataType.AsString()
		}
		def.slots = append(def.slots, graph.Slot{
			ID:        id.AsString(),
			Name:      name.AsString(),
			Direction: graph.Input,
			DataType:  dt,
		})
		return 0
	}))

	node.SetString("add_output", vm.NewNativeFunc(func(v *vm.VM) int {
		id := v.Get(2)
		name := v.Get(3)
		dataType := v.Get(4)
		if !id.IsString() || !name.IsString() {
			return 0
		}
		dt := "any"
		if dataType.IsString() {
			dt = dataType.AsString()
		}
		def.slots = append(def.slots, graph.Slot{
			ID:        id.AsString(),
			Name:      name.AsString(),
			Direction: graph.Output,
			DataType:  dt,
		})
		return 0
	}))

	node.SetString("define_config", vm.NewNativeFunc(func(v *vm.VM) int {
		key := v.Get(2)
		dflt := v.Get(3)
		label := v.Get(4)
		if !key.IsString() {
			return 0
		}
		cf := graph.ConfigField{Key: key.AsString()}
		if dflt.IsString() {
			cf.Default = dflt.AsString()
		} else if dflt.IsNumber() {
			cf.Default = vm.ValueToString(dflt)
		}
		if label.IsString() {
			cf.Label = label.AsString()
		} else {
			cf.Label = key.AsString()
		}
		def.configSchema = append(def.configSchema, cf)
		return 0
	}))

	// --- Runtime methods (noops during define) ---
	noop := vm.NewNativeFunc(func(v *vm.VM) int { return 0 })
	node.SetString("emit", noop)
	node.SetString("set", noop)
	node.SetString("display", noop)
	node.SetString("glow", noop)
	node.SetString("log", noop)
	node.SetString("set_config", noop)
	node.SetString("init_tick", noop)
	node.SetString("schedule_tick", noop)

	// --- Empty data tables (handlers reference self.inputs etc.) ---
	node.SetString("id", vm.NewString(""))
	node.SetString("type", vm.NewString(""))
	node.SetString("label", vm.NewString(""))
	node.SetString("inputs", vm.NewTable(vm.NewEmptyTable()))
	node.SetString("config", vm.NewTable(vm.NewEmptyTable()))
	node.SetString("outputs", vm.NewTable(vm.NewEmptyTable()))
	node.SetString("state", vm.NewTable(vm.NewEmptyTable()))
	node.SetString("incoming", vm.NewTable(vm.NewEmptyTable()))
	node.SetString("outgoing", vm.NewTable(vm.NewEmptyTable()))

	// --- Base handlers ---
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
