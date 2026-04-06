// Package graph defines the core types for a canvas-based graph engine.
//
// The graph model is built around four primary types:
//
//   - [Graph] is the top-level container holding nodes and connections.
//   - [Node] is a positioned vertex that references a registered [NodeType].
//   - [Slot] is a typed input or output port defined on a [NodeType].
//   - [Connection] is a directed edge interface from an output slot to an
//     input slot. Concrete types are [EventConnection] (discrete messages
//     with optional traversal animation) and [StateConnection] (continuous
//     state with change detection). [ConnectionKind] distinguishes them,
//     and [SlotConnectionKind] determines the kind from a slot's DataType.
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
// Graphs are executed by the engine package. The engine runs a goroutine per
// node, with channel-based wire connections between them. Event connections
// emit start/end SSE events with dot animation; state connections emit
// connection.state SSE events with steady glow rendering.
package graph
