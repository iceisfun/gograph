// Package lua provides Lua script execution for graph nodes.
//
// It implements the [engine.Executor] interface using golua to run
// sandboxed Lua scripts. Each node execution creates a fresh VM with
// limited instruction count and call depth.
//
// # Script Interface
//
// A node's Lua script receives input values as a global table "inputs"
// keyed by slot ID, and must return a table of output values keyed by
// slot ID:
//
//	-- inputs: { ["data"] = "hello", ["count"] = 3 }
//	local data = inputs["data"]
//	local count = inputs["count"]
//	return { out = data:rep(count) }
//
// # Sandboxing
//
// Each execution uses a fresh VM with:
//   - MaxInstructions: 1,000,000 (configurable)
//   - MaxCallDepth: 200
//   - No IO, OS, or debug providers
//   - Standard library (string, table, math, etc.) is available
package lua
