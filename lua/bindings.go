package lua

import (
	"context"
	"fmt"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"

	"github.com/iceisfun/gograph/graph"
)

// Bindings implements [engine.Executor] using golua for Lua script execution.
type Bindings struct {
	registry       *graph.Registry
	maxInstructions int64
}

// Option configures [Bindings].
type Option func(*Bindings)

// WithMaxInstructions sets the instruction limit per execution.
// Defaults to 1,000,000.
func WithMaxInstructions(n int64) Option {
	return func(b *Bindings) {
		b.maxInstructions = n
	}
}

// New creates Lua bindings with the given registry.
func New(registry *graph.Registry, opts ...Option) *Bindings {
	b := &Bindings{
		registry:       registry,
		maxInstructions: 1_000_000,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// ExecuteNode runs a node type's Lua script with the given inputs.
// A fresh sandboxed VM is created for each execution. The inputs map
// is set as a global Lua table "inputs". The script must return a table
// whose keys are output slot IDs and values are the output data.
func (b *Bindings) ExecuteNode(ctx context.Context, nt graph.NodeType, inputs map[string]any) (map[string]any, error) {
	if nt.Script == "" {
		return nil, fmt.Errorf("node type %q has no script", nt.Name)
	}

	// Parse the Lua source.
	block, err := parser.Parse(nt.Name, nt.Script)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", nt.Name, err)
	}

	// Compile to bytecode.
	proto, err := compiler.Compile(nt.Name, block)
	if err != nil {
		return nil, fmt.Errorf("compile %q: %w", nt.Name, err)
	}

	// Create a sandboxed VM.
	v := vm.New(
		vm.WithContext(ctx),
		vm.WithLimits(vm.Limits{
			MaxInstructions: b.maxInstructions,
			MaxCallDepth:    200,
			MaxStackSlots:   10000,
		}),
	)
	stdlib.Open(v)

	// Set the inputs global.
	inputTable := goMapToLuaTable(inputs)
	v.SetGlobal("inputs", vm.NewTable(inputTable))

	// Run the script.
	results, err := v.Run(proto)
	if err != nil {
		return nil, fmt.Errorf("execute %q: %w", nt.Name, err)
	}

	// Extract the returned table as outputs.
	if len(results) == 0 || results[0].IsNil() {
		return make(map[string]any), nil
	}

	if !results[0].IsTable() {
		return nil, fmt.Errorf("execute %q: expected return table, got %s", nt.Name, results[0].Type())
	}

	return luaTableToGoMap(results[0].AsTable()), nil
}

// goMapToLuaTable converts a Go map[string]any to a Lua table.
func goMapToLuaTable(m map[string]any) *vm.Table {
	t := vm.NewEmptyTable()
	for k, v := range m {
		t.SetString(k, goToLuaValue(v))
	}
	return t
}

// goToLuaValue converts a Go value to a Lua Value.
func goToLuaValue(v any) vm.Value {
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
	case map[string]any:
		return vm.NewTable(goMapToLuaTable(val))
	default:
		return vm.NewString(fmt.Sprintf("%v", val))
	}
}

// luaTableToGoMap converts a Lua table to a Go map[string]any.
func luaTableToGoMap(t vm.LuaTable) map[string]any {
	result := make(map[string]any)
	var key vm.Value = vm.Nil
	for {
		nextKey, value, err := t.Next(key)
		if err != nil || nextKey.IsNil() {
			break
		}
		if nextKey.IsString() {
			result[nextKey.AsString()] = luaToGoValue(value)
		}
		key = nextKey
	}
	return result
}

// luaToGoValue converts a Lua Value to a Go value.
func luaToGoValue(v vm.Value) any {
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
		return luaTableToGoMap(v.AsTable())
	default:
		return vm.ValueToString(v)
	}
}
