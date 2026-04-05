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
		graph:    g,
		registry: graph.NewRegistry(),
		duration: 1000,
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
func (e *Engine) Execute(ctx context.Context) error {
	// Reload graph from store to pick up frontend edits.
	if e.store != nil {
		g, err := e.store.Load(ctx, e.graphID)
		if err != nil {
			return fmt.Errorf("load graph: %w", err)
		}
		e.graph = g
	}

	order, err := Order(e.graph)
	if err != nil {
		return err
	}

	// outputs tracks the output values of each node, keyed by node ID then slot ID.
	outputs := make(map[string]map[string]any)

	for _, nodeID := range order {
		if err := ctx.Err(); err != nil {
			e.cancelAll()
			return err
		}

		node := e.graph.Node(nodeID)
		if node == nil {
			continue
		}

		nt, ok := e.registry.Lookup(node.Type)
		if !ok {
			return fmt.Errorf("node %q: unknown type %q", nodeID, node.Type)
		}

		// Gather inputs from upstream connections.
		inputs := e.gatherInputs(nodeID, outputs)

		// Execute the node.
		config := node.Config
		if config == nil {
			config = make(map[string]string)
		}

		var nodeOutputs map[string]any
		if e.executor != nil && nt.Script != "" {
			nodeOutputs, err = e.executor.ExecuteNode(ctx, nt, inputs, config)
			if err != nil {
				return fmt.Errorf("node %q: execution failed: %w", nodeID, err)
			}
		} else {
			// Passthrough: forward inputs as outputs for nodes without scripts.
			nodeOutputs = inputs
		}

		outputs[nodeID] = nodeOutputs

		// If the node has a duration config (e.g. delay nodes), wait before
		// emitting outgoing events. This creates the actual hold time.
		if d, ok := config["duration"]; ok {
			if ms, parseErr := strconv.Atoi(d); parseErr == nil && ms > 0 {
				select {
				case <-ctx.Done():
					e.cancelAll()
					return ctx.Err()
				case <-time.After(time.Duration(ms) * time.Millisecond):
				}
			}
		}

		// Emit events on outgoing connections.
		// Per-node duration override via config["duration"].
		e.emitNodeOutputEvents(nodeID, config)
	}

	return nil
}

// gatherInputs collects values from upstream connections for a given node.
func (e *Engine) gatherInputs(nodeID string, outputs map[string]map[string]any) map[string]any {
	e.graph.RLock()
	defer e.graph.RUnlock()

	inputs := make(map[string]any)
	for _, c := range e.graph.Connections {
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

// emitNodeOutputEvents emits EventStart for each outgoing connection from the node.
// The config map is checked for a "duration" key to override the default event duration.
func (e *Engine) emitNodeOutputEvents(nodeID string, config map[string]string) {
	e.graph.RLock()
	connections := make([]*graph.Connection, 0)
	for _, c := range e.graph.Connections {
		if c.FromNode == nodeID {
			connections = append(connections, c)
		}
	}
	e.graph.RUnlock()

	// Per-node duration override.
	duration := e.duration
	if d, ok := config["duration"]; ok {
		if ms, err := strconv.Atoi(d); err == nil && ms > 0 {
			duration = ms
		}
	}

	now := time.Now().UnixMilli()
	for _, c := range connections {
		eventID := generateEventID()
		e.emit(Event{
			Type: graph.TypeEventStart,
			Payload: graph.EventStartPayload{
				Envelope:     graph.NewEnvelope(now),
				EventID:      eventID,
				ConnectionID: c.ID,
				Duration:     duration,
			},
		})

		// Schedule the end event after the duration.
		go func(eid string, dur int) {
			time.Sleep(time.Duration(dur) * time.Millisecond)
			e.emit(Event{
				Type: graph.TypeEventEnd,
				Payload: graph.EventEndPayload{
					Envelope: graph.NewEnvelope(time.Now().UnixMilli()),
					EventID:  eid,
				},
			})
		}(eventID, duration)
	}
}

// cancelAll emits cancel events for all in-flight events. Called when
// execution is interrupted by context cancellation.
func (e *Engine) cancelAll() {
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
