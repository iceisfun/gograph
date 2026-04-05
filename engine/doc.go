// Package engine provides the graph execution engine.
//
// The engine traverses a graph in topological order, executing each node's
// Lua script with input values from upstream connections and propagating
// outputs downstream. Execution events (start, update, end, cancel) are
// emitted to subscribers for real-time visualization via SSE.
//
// # Usage
//
//	eng := engine.New(g,
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
//	err := eng.Execute(ctx)
//
// # Event Lifecycle
//
// When a node completes execution:
//  1. An [EventStart] is emitted on each outgoing connection.
//  2. After the specified duration, an [EventEnd] is emitted.
//  3. The downstream node then executes with the received values.
//
// The engine may emit [EventCancel] if execution is interrupted.
package engine
