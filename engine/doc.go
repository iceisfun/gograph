// Package engine provides the graph execution engine.
//
// The engine uses a goroutine-per-node architecture. Each node runs in its
// own goroutine, receiving inputs via channel-based wire connections and
// producing outputs that are routed through wires to downstream nodes.
//
// Connections between nodes are managed by [WireRunner], an interface with
// two concrete implementations:
//
//   - [EventWire] handles discrete messages with optional traversal
//     animation (dot moving along the wire). Emits event.start / event.end
//     SSE events.
//   - [StateWire] handles continuous state values with change detection.
//     Only propagates when the value changes. Emits connection.state SSE
//     events (steady glow rendering, no dot animation).
//
// The [NewWire] constructor selects the appropriate type based on the
// connection's [graph.ConnectionKind].
//
// # Usage
//
//	eng := engine.New(
//	    engine.WithRegistry(reg),
//	    engine.WithLuaExecutor(luaExec),
//	)
//
//	sub := eng.Subscribe(64)
//	defer sub.Done()
//
//	go func() {
//	    for evt := range sub.Events() {
//	        server.Publish(g.ID, evt.Type, evt.Payload)
//	    }
//	}()
//
//	eng.LoadGraph(g)
//
// # Event Lifecycle
//
// For event connections: when a node emits a value via self:emit(),
//  1. An [EventStart] is emitted on the outgoing event wire.
//  2. After the configured duration, an [EventEnd] is emitted.
//  3. The downstream node receives the value as an arrival.
//
// For state connections: when a node sets a value via self:set(),
//  1. The [StateWire] compares with the previous value.
//  2. If changed, a connection.state SSE event is emitted.
//  3. The downstream node receives on_change / on_high / on_low handlers.
//
// The engine may emit [EventCancel] if execution is interrupted.
package engine
