// Package graph defines the core types for a canvas-based graph engine.
//
// The graph model is built around four primary types:
//
//   - [Graph] is the top-level container holding nodes and connections.
//   - [Node] is a positioned vertex that references a registered [NodeType].
//   - [Slot] is a typed input or output port defined on a [NodeType].
//   - [Connection] is a directed edge from an output slot to an input slot.
//
// Node types are managed through a [Registry]. Each [NodeType] declares its
// slots (inputs and outputs) and optionally a Lua script for execution.
// All nodes of the same type share the same slot definitions.
//
// The package also defines the SSE wire protocol types used to synchronize
// graph state and execution events between the Go server and TypeScript
// frontend. See [Envelope] and the event payload types for the protocol
// specification.
//
// # Extension Points
//
//   - Register custom [NodeType] values with a [Registry]
//   - Define slot [DataType] identifiers for connection validation
//   - Provide a custom [ConnectionValidator] to override default validation rules
//
// # Execution Model
//
// Graphs are executed by the engine package. The engine traverses nodes in
// topological order, executing each node's Lua script with its input values
// and propagating outputs along connections. Execution events (start, update,
// end, cancel) are emitted as SSE messages for the frontend to animate.
package graph
