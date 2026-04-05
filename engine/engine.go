package engine

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/iceisfun/gograph/graph"
	"github.com/iceisfun/gograph/store"
)

// Executor runs a node's logic with the given inputs and returns outputs.
// The config parameter carries the node's instance configuration (e.g.
// "duration" for delay nodes). The lua package provides the primary
// implementation.
type Executor interface {
	ExecuteNode(ctx context.Context, nt graph.NodeType, inputs map[string]any, config map[string]string) (map[string]any, error)
}

// Engine executes a graph by traversing nodes in topological order and
// running each node's logic via an [Executor]. Events are emitted to
// subscribers for real-time visualization.
type Engine struct {
	mu          sync.RWMutex
	graph       *graph.Graph
	graphID     string
	store       store.GraphStore
	registry    *graph.Registry
	executor    Executor
	subscribers []*Subscriber
	duration    int // default event duration in ms
	nodeLogger  NodeLogger
	eventLogger EventLogger
}

// EngineOption configures an [Engine].
type EngineOption func(*Engine)

// WithRegistry sets the node type registry for the engine.
func WithRegistry(r *graph.Registry) EngineOption {
	return func(e *Engine) {
		e.registry = r
	}
}

// WithExecutor sets the node executor (typically the Lua bindings).
func WithExecutor(exec Executor) EngineOption {
	return func(e *Engine) {
		e.executor = exec
	}
}

// WithStore sets the graph store. When set, the engine reloads the graph
// from the store before each execution to pick up changes made through
// the REST API (e.g. node/connection edits from the frontend).
func WithStore(s store.GraphStore, graphID string) EngineOption {
	return func(e *Engine) {
		e.store = s
		e.graphID = graphID
	}
}

// WithNodeLogger sets the logger for node lifecycle events.
// Defaults to [NopNodeLogger].
func WithNodeLogger(l NodeLogger) EngineOption {
	return func(e *Engine) {
		e.nodeLogger = l
	}
}

// WithEventLogger sets the logger for event lifecycle events.
// Defaults to [NopEventLogger].
func WithEventLogger(l EventLogger) EngineOption {
	return func(e *Engine) {
		e.eventLogger = l
	}
}

// WithEventDuration sets the default animation duration in milliseconds
// for events traversing connections. Defaults to 1000ms.
func WithEventDuration(ms int) EngineOption {
	return func(e *Engine) {
		e.duration = ms
	}
}

// New creates a new engine for the given graph.
func New(g *graph.Graph, opts ...EngineOption) *Engine {
	e := &Engine{
		graph:       g,
		registry:    graph.NewRegistry(),
		duration:    1000,
		nodeLogger:  NopNodeLogger{},
		eventLogger: NopEventLogger{},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Subscribe creates a new event subscriber with the given buffer size.
func (e *Engine) Subscribe(bufferSize int) *Subscriber {
	sub := newSubscriber(bufferSize)
	e.mu.Lock()
	e.subscribers = append(e.subscribers, sub)
	e.mu.Unlock()
	return sub
}

// emit sends an event to all active subscribers.
func (e *Engine) emit(evt Event) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, sub := range e.subscribers {
		sub.send(evt)
	}
}

// Execute runs the graph in topological order. For each node:
// 1. Gather input values from upstream connections
// 2. Execute the node's logic via the Executor
// 3. Emit EventStart on each outgoing connection
// 4. Store outputs for downstream nodes
//
// Returns an error if the graph has cycles, a node type is missing, or
// execution fails.
// Execute runs the graph in topological order. It is safe to call
// concurrently — each invocation works with its own graph snapshot
// and local state, so multiple executions can overlap.
func (e *Engine) Execute(ctx context.Context) error {
	// Load a snapshot of the graph for this execution.
	var g *graph.Graph
	if e.store != nil {
		var err error
		g, err = e.store.Load(ctx, e.graphID)
		if err != nil {
			return fmt.Errorf("load graph: %w", err)
		}
	} else {
		g = e.graph
	}

	order, err := Order(g)
	if err != nil {
		return err
	}

	// outputs tracks the output values of each node, keyed by node ID then slot ID.
	outputs := make(map[string]map[string]any)
	// emitTimes tracks when each node emitted its outgoing events, so
	// downstream nodes can wait for traversal to complete.
	emitTimes := make(map[string]time.Time)

	for _, nodeID := range order {
		if err := ctx.Err(); err != nil {
			e.cancelAll()
			return err
		}

		node := g.Node(nodeID)
		if node == nil {
			continue
		}

		nt, ok := e.registry.Lookup(node.Type)
		if !ok {
			return fmt.Errorf("node %q: unknown type %q", nodeID, node.Type)
		}

		// Gather inputs from upstream connections.
		inputs := e.gatherInputs(g, nodeID, outputs)

		// Skip nodes that expect inputs but have none connected.
		if len(inputs) == 0 && len(nt.InputSlots()) > 0 {
			e.nodeLogger.NodeSkipped(nodeID, "has input slots but no incoming connections")
			continue
		}

		// Wait for incoming traversals to complete.
		if waitErr := e.waitForArrivals(ctx, g, nodeID, emitTimes); waitErr != nil {
			return waitErr
		}

		// Execute the node.
		config := node.Config
		if config == nil {
			config = make(map[string]string)
		}

		e.nodeLogger.NodeExecuting(nodeID, node.Type, len(inputs))

		var nodeOutputs map[string]any
		if e.executor != nil && nt.Script != "" {
			nodeOutputs, err = e.executor.ExecuteNode(ctx, nt, inputs, config)
			if err != nil {
				return fmt.Errorf("node %q: execution failed: %w", nodeID, err)
			}
		} else {
			nodeOutputs = inputs
		}

		outputs[nodeID] = nodeOutputs
		e.nodeLogger.NodeExecuted(nodeID, node.Type, len(nodeOutputs))

		// If the node has a duration config (e.g. delay nodes), wait before
		// emitting outgoing events. This creates the actual hold time.
		if d, ok := config["duration"]; ok {
			if ms, parseErr := strconv.Atoi(d); parseErr == nil && ms > 0 {
				e.nodeLogger.NodeHolding(nodeID, ms)
				// Notify frontend so the node glows during the hold.
				e.emit(Event{
					Type: graph.TypeNodeActive,
					Payload: graph.NodeActivePayload{
						Envelope: graph.NewEnvelope(time.Now().UnixMilli()),
						NodeID:   nodeID,
						Duration: ms,
					},
				})
				select {
				case <-ctx.Done():
					e.cancelAll()
					return ctx.Err()
				case <-time.After(time.Duration(ms) * time.Millisecond):
				}
			}
		}

		// Log disconnected output slots.
		e.logDisconnectedOutputs(g, nodeID, nt)

		// Emit events on outgoing connections and record the emit time.
		e.emitNodeOutputEvents(g, nodeID)
		emitTimes[nodeID] = time.Now()
	}

	return nil
}

// waitForArrivals waits until all incoming connection traversals have
// completed. For each incoming connection, we compute when the upstream
// node emitted plus the connection's traversal duration, and sleep until
// the latest arrival time.
func (e *Engine) waitForArrivals(ctx context.Context, g *graph.Graph, nodeID string, emitTimes map[string]time.Time) error {
	g.RLock()
	var latestArrival time.Time
	for _, c := range g.Connections {
		if c.ToNode != nodeID {
			continue
		}
		emitTime, ok := emitTimes[c.FromNode]
		if !ok {
			continue
		}
		duration := 0
		if c.Config != nil {
			if d, ok := c.Config["duration"]; ok {
				if ms, err := strconv.Atoi(d); err == nil && ms > 0 {
					duration = ms
				}
			}
		}
		arrival := emitTime.Add(time.Duration(duration) * time.Millisecond)
		if arrival.After(latestArrival) {
			latestArrival = arrival
		}
	}
	g.RUnlock()

	if !latestArrival.IsZero() {
		wait := time.Until(latestArrival)
		if wait > 0 {
			e.nodeLogger.NodeWaiting(nodeID, wait.Milliseconds())
			select {
			case <-ctx.Done():
				e.cancelAll()
				return ctx.Err()
			case <-time.After(wait):
			}
		}
	}
	return nil
}

// gatherInputs collects values from upstream connections for a given node.
func (e *Engine) gatherInputs(g *graph.Graph, nodeID string, outputs map[string]map[string]any) map[string]any {
	g.RLock()
	defer g.RUnlock()

	inputs := make(map[string]any)
	for _, c := range g.Connections {
		if c.ToNode == nodeID {
			if nodeOut, ok := outputs[c.FromNode]; ok {
				if val, ok := nodeOut[c.FromSlot]; ok {
					inputs[c.ToSlot] = val
				}
			}
		}
	}
	return inputs
}

// logDisconnectedOutputs logs output slots that have no outgoing connections.
func (e *Engine) logDisconnectedOutputs(g *graph.Graph, nodeID string, nt graph.NodeType) {
	g.RLock()
	defer g.RUnlock()

	for _, slot := range nt.OutputSlots() {
		connected := false
		for _, c := range g.Connections {
			if c.FromNode == nodeID && c.FromSlot == slot.ID {
				connected = true
				break
			}
		}
		if !connected {
			e.nodeLogger.NodeDisconnected(nodeID, slot.ID)
		}
	}
}

// emitNodeOutputEvents emits EventStart for each outgoing connection from
// the node. Traversal duration is read from each connection's Config["duration"].
// Connections without a duration are treated as instant (duration=0).
func (e *Engine) emitNodeOutputEvents(g *graph.Graph, nodeID string) {
	g.RLock()
	connections := make([]*graph.Connection, 0)
	for _, c := range g.Connections {
		if c.FromNode == nodeID {
			connections = append(connections, c)
		}
	}
	g.RUnlock()

	now := time.Now().UnixMilli()
	for _, c := range connections {
		// Per-connection traversal duration.
		duration := 0
		if c.Config != nil {
			if d, ok := c.Config["duration"]; ok {
				if ms, err := strconv.Atoi(d); err == nil && ms > 0 {
					duration = ms
				}
			}
		}

		eventID := generateEventID()
		e.eventLogger.EventEmitted(eventID, c.ID, nodeID, c.ToNode, duration)
		e.emit(Event{
			Type: graph.TypeEventStart,
			Payload: graph.EventStartPayload{
				Envelope:     graph.NewEnvelope(now),
				EventID:      eventID,
				ConnectionID: c.ID,
				Duration:     duration,
			},
		})

		if duration > 0 {
			// Timed: schedule end event after traversal.
			go func(eid string, connID string, toNode string, dur int) {
				time.Sleep(time.Duration(dur) * time.Millisecond)
				e.eventLogger.EventArrived(eid, connID, toNode)
				e.emit(Event{
					Type: graph.TypeEventEnd,
					Payload: graph.EventEndPayload{
						Envelope: graph.NewEnvelope(time.Now().UnixMilli()),
						EventID:  eid,
					},
				})
			}(eventID, c.ID, c.ToNode, duration)
		} else {
			// Instant: emit end immediately.
			e.eventLogger.EventArrived(eventID, c.ID, c.ToNode)
			e.emit(Event{
				Type: graph.TypeEventEnd,
				Payload: graph.EventEndPayload{
					Envelope: graph.NewEnvelope(now),
					EventID:  eventID,
				},
			})
		}
	}
}

// cancelAll emits cancel events for all in-flight events. Called when
// execution is interrupted by context cancellation.
func (e *Engine) cancelAll() {
	e.eventLogger.EventCancelled("context cancelled")
	e.emit(Event{
		Type: graph.TypeEventCancel,
		Payload: graph.EventCancelPayload{
			Envelope:  graph.NewEnvelope(time.Now().UnixMilli()),
			EventID:   "*",
			Immediate: true,
		},
	})
}

// generateEventID creates a short random event identifier.
func generateEventID() string {
	var buf [8]byte
	rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}
