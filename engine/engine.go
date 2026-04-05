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
type Executor interface {
	ExecuteNode(ctx context.Context, nt graph.NodeType, inputs map[string]any, config map[string]string) (map[string]any, error)
}

// Engine executes a graph by traversing nodes in topological order and
// running each node's logic via an [Executor]. Events are emitted to
// subscribers for real-time visualization.
//
// The engine supports two execution modes:
//   - Timed traversals: connections with duration > 0 animate dots (EventStart/EventEnd).
//   - Instant propagation: connections with duration = 0 propagate forward immediately
//     when any upstream value changes. No polling — pure event-driven cascade.
//
// Use [Start] to begin periodic execution and [PropagateFrom] to trigger
// immediate forward propagation from a specific node.
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

	// Persistent node outputs — stores the last computed outputs for every
	// node. Downstream nodes read inputs from here so fan-in nodes (AND, etc.)
	// always have the latest value for all inputs.
	nodeOutputs sync.Map // nodeID → map[string]any

	// Change detection for wire state and display content.
	lastWireState sync.Map // connectionID or "nodeID:_display" → last value string

	// Source evaluation interval (replaces wireInterval).
	// Only re-evaluates source nodes and propagates forward.
	sourceInterval time.Duration

	// lifecycle
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// EngineOption configures an [Engine].
type EngineOption func(*Engine)

// WithRegistry sets the node type registry for the engine.
func WithRegistry(r *graph.Registry) EngineOption {
	return func(e *Engine) { e.registry = r }
}

// WithExecutor sets the node executor (typically the Lua bindings).
func WithExecutor(exec Executor) EngineOption {
	return func(e *Engine) { e.executor = exec }
}

// WithStore sets the graph store for live graph reloading.
func WithStore(s store.GraphStore, graphID string) EngineOption {
	return func(e *Engine) { e.store = s; e.graphID = graphID }
}

// WithNodeLogger sets the logger for node lifecycle events.
func WithNodeLogger(l NodeLogger) EngineOption {
	return func(e *Engine) { e.nodeLogger = l }
}

// WithEventLogger sets the logger for event lifecycle events.
func WithEventLogger(l EventLogger) EngineOption {
	return func(e *Engine) { e.eventLogger = l }
}

// WithSourceInterval sets the evaluation interval for source nodes.
// Source nodes (no connected inputs) are re-evaluated on this tick and
// their outputs propagate forward through instant connections.
func WithSourceInterval(d time.Duration) EngineOption {
	return func(e *Engine) { e.sourceInterval = d }
}

// WithWireInterval is a deprecated alias for [WithSourceInterval].
func WithWireInterval(d time.Duration) EngineOption {
	return WithSourceInterval(d)
}

// WithEventDuration sets the default animation duration in milliseconds
// for timed connection traversals. Defaults to 1000ms.
func WithEventDuration(ms int) EngineOption {
	return func(e *Engine) { e.duration = ms }
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

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Start begins periodic execution in the background.
func (e *Engine) Start(ctx context.Context, interval time.Duration) {
	ctx, e.cancel = context.WithCancel(ctx)

	// Main execution loop — full graph with timed traversals.
	e.wg.Go(func() {
		e.fireExecution(ctx)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				e.fireExecution(ctx)
			}
		}
	})

	// Source evaluation loop — re-evaluates source nodes and propagates
	// forward through instant connections.
	if e.sourceInterval > 0 {
		e.wg.Go(func() {
			ticker := time.NewTicker(e.sourceInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					e.evaluateSources(ctx)
				}
			}
		})
	}
}

// Stop cancels periodic execution and waits for all in-flight
// executions to complete.
func (e *Engine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
	e.wg.Wait()
}

func (e *Engine) fireExecution(ctx context.Context) {
	e.wg.Go(func() {
		if err := e.Execute(ctx); err != nil && ctx.Err() == nil {
			e.nodeLogger.NodeSkipped("*", "execution error: "+err.Error())
		}
	})
}

// ---------------------------------------------------------------------------
// Forward propagation — the core of the instant wire model
// ---------------------------------------------------------------------------

// PropagateFrom immediately re-evaluates a node and propagates its outputs
// forward through all instant connections. This is the primary mechanism
// for binary/wire signals — no polling, pure event-driven cascade.
//
// Called by: server click handler, source tick, timed event arrival.
func (e *Engine) PropagateFrom(ctx context.Context, nodeID string) {
	g := e.loadGraph(ctx)
	if g == nil {
		return
	}
	visited := make(map[string]bool)
	e.propagateFrom(ctx, g, nodeID, visited)
}

func (e *Engine) propagateFrom(ctx context.Context, g *graph.Graph, nodeID string, visited map[string]bool) {
	if ctx.Err() != nil || visited[nodeID] {
		return
	}
	visited[nodeID] = true

	node := g.Node(nodeID)
	if node == nil {
		return
	}
	nt, ok := e.registry.Lookup(node.Type)
	if !ok {
		return
	}

	// Gather inputs from persistent state.
	inputs := e.gatherPersistentInputs(g, nodeID)
	if len(inputs) == 0 && len(nt.InputSlots()) > 0 {
		return
	}

	config := node.Config
	if config == nil {
		config = make(map[string]string)
	}

	// Execute the node.
	var outputs map[string]any
	var err error
	if e.executor != nil && nt.Script != "" {
		outputs, err = e.executor.ExecuteNode(ctx, nt, inputs, config)
		if err != nil {
			return
		}
	} else {
		outputs = inputs
	}

	// Handle _display with change detection.
	e.emitDisplayIfChanged(nodeID, outputs)
	delete(outputs, "_display")

	// Store persistently.
	e.nodeOutputs.Store(nodeID, outputs)

	// Follow instant outgoing connections.
	g.RLock()
	var downstream []string
	for _, c := range g.Connections {
		if c.FromNode != nodeID {
			continue
		}
		if isTimedConnection(c) {
			continue
		}
		e.emitConnectionStateIfChanged(c, outputs)
		downstream = append(downstream, c.ToNode)
	}
	g.RUnlock()

	// Recurse into instant downstream nodes.
	for _, nextID := range downstream {
		e.propagateFrom(ctx, g, nextID, visited)
	}
}

// ---------------------------------------------------------------------------
// Source evaluation — replaces evaluateWires
// ---------------------------------------------------------------------------

// evaluateSources re-evaluates only source nodes (no connected inputs)
// and propagates their outputs forward through instant connections.
func (e *Engine) evaluateSources(ctx context.Context) {
	g := e.loadGraph(ctx)
	if g == nil {
		return
	}

	order, err := Order(g)
	if err != nil {
		return
	}

	visited := make(map[string]bool)

	for _, nodeID := range order {
		if ctx.Err() != nil {
			return
		}

		nt, ok := e.registry.Lookup(func() string {
			n := g.Node(nodeID)
			if n == nil {
				return ""
			}
			return n.Type
		}())
		if !ok {
			continue
		}

		// Only evaluate source nodes: those with no input slots or no
		// connected inputs.
		if len(nt.InputSlots()) > 0 && e.hasConnectedInputs(g, nodeID) {
			continue
		}

		// Execute and propagate forward.
		e.propagateFrom(ctx, g, nodeID, visited)
	}
}

// ---------------------------------------------------------------------------
// Full timed execution
// ---------------------------------------------------------------------------

// Execute runs the full graph in topological order, handling timed
// connection traversals with sleep. After each node executes, instant
// downstream connections propagate forward immediately.
func (e *Engine) Execute(ctx context.Context) error {
	g := e.loadGraph(ctx)
	if g == nil {
		return fmt.Errorf("failed to load graph")
	}

	order, err := Order(g)
	if err != nil {
		return err
	}

	// Local outputs for timed traversal flow within this execution.
	outputs := make(map[string]map[string]any)
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

		// Gather inputs from the local execution-scoped map.
		inputs := e.gatherInputs(g, nodeID, outputs)
		if len(inputs) == 0 && len(nt.InputSlots()) > 0 {
			e.nodeLogger.NodeSkipped(nodeID, "has input slots but no incoming connections")
			continue
		}

		// Wait for incoming timed traversals.
		if waitErr := e.waitForArrivals(ctx, g, nodeID, emitTimes); waitErr != nil {
			return waitErr
		}

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

		// Handle _display.
		if display, ok := nodeOutputs["_display"]; ok {
			if text, ok := display.(string); ok {
				e.emit(Event{
					Type: graph.TypeNodeContent,
					Payload: graph.NodeContentPayload{
						Envelope: graph.NewEnvelope(time.Now().UnixMilli()),
						NodeID:   nodeID,
						Text:     text,
					},
				})
			}
			delete(nodeOutputs, "_display")
		}

		outputs[nodeID] = nodeOutputs
		e.nodeOutputs.Store(nodeID, nodeOutputs) // persist for propagation
		e.nodeLogger.NodeExecuted(nodeID, node.Type, len(nodeOutputs))

		// Node hold (delay nodes).
		if d, ok := config["duration"]; ok {
			if ms, parseErr := strconv.Atoi(d); parseErr == nil && ms > 0 {
				e.nodeLogger.NodeHolding(nodeID, ms)
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

		e.logDisconnectedOutputs(g, nodeID, nt)

		// Emit timed events and propagate instant connections forward.
		e.emitTimedAndPropagateInstant(ctx, g, nodeID, nodeOutputs)
		emitTimes[nodeID] = time.Now()
	}

	return nil
}

// emitTimedAndPropagateInstant handles outgoing connections after a node
// executes in the full Execute cycle. Timed connections get EventStart/EventEnd.
// Instant connections propagate forward immediately via propagateFrom.
func (e *Engine) emitTimedAndPropagateInstant(ctx context.Context, g *graph.Graph, nodeID string, outputs map[string]any) {
	g.RLock()
	var timedConns []*graph.Connection
	var instantDownstream []string
	for _, c := range g.Connections {
		if c.FromNode != nodeID {
			continue
		}
		if isTimedConnection(c) {
			timedConns = append(timedConns, c)
		} else {
			e.emitConnectionStateIfChanged(c, outputs)
			instantDownstream = append(instantDownstream, c.ToNode)
		}
	}
	g.RUnlock()

	// Emit timed events.
	now := time.Now().UnixMilli()
	for _, c := range timedConns {
		eventID := generateEventID()
		e.eventLogger.EventEmitted(eventID, c.ID, nodeID, c.ToNode, connDuration(c))
		e.emit(Event{
			Type: graph.TypeEventStart,
			Payload: graph.EventStartPayload{
				Envelope:     graph.NewEnvelope(now),
				EventID:      eventID,
				ConnectionID: c.ID,
				Duration:     connDuration(c),
			},
		})
		go func(eid, connID, toNode string, dur int) {
			time.Sleep(time.Duration(dur) * time.Millisecond)
			e.eventLogger.EventArrived(eid, connID, toNode)
			e.emit(Event{
				Type: graph.TypeEventEnd,
				Payload: graph.EventEndPayload{
					Envelope: graph.NewEnvelope(time.Now().UnixMilli()),
					EventID:  eid,
				},
			})
		}(eventID, c.ID, c.ToNode, connDuration(c))
	}

	// Propagate instant downstream.
	visited := make(map[string]bool)
	visited[nodeID] = true
	for _, nextID := range instantDownstream {
		e.propagateFrom(ctx, g, nextID, visited)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (e *Engine) emit(evt Event) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, sub := range e.subscribers {
		sub.send(evt)
	}
}

func (e *Engine) loadGraph(ctx context.Context) *graph.Graph {
	if e.store != nil {
		g, err := e.store.Load(ctx, e.graphID)
		if err != nil {
			return nil
		}
		return g
	}
	return e.graph
}

// gatherInputs reads from a local execution-scoped outputs map (for Execute).
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

// gatherPersistentInputs reads from the persistent nodeOutputs map (for propagation).
func (e *Engine) gatherPersistentInputs(g *graph.Graph, nodeID string) map[string]any {
	g.RLock()
	defer g.RUnlock()
	inputs := make(map[string]any)
	for _, c := range g.Connections {
		if c.ToNode == nodeID {
			if stored, ok := e.nodeOutputs.Load(c.FromNode); ok {
				nodeOut := stored.(map[string]any)
				if val, ok := nodeOut[c.FromSlot]; ok {
					inputs[c.ToSlot] = val
				}
			}
		}
	}
	return inputs
}

func (e *Engine) hasConnectedInputs(g *graph.Graph, nodeID string) bool {
	g.RLock()
	defer g.RUnlock()
	for _, c := range g.Connections {
		if c.ToNode == nodeID {
			return true
		}
	}
	return false
}

func (e *Engine) emitDisplayIfChanged(nodeID string, outputs map[string]any) {
	display, ok := outputs["_display"]
	if !ok {
		return
	}
	text, ok := display.(string)
	if !ok {
		return
	}
	key := nodeID + ":_display"
	if prev, loaded := e.lastWireState.Load(key); loaded && prev.(string) == text {
		return
	}
	e.lastWireState.Store(key, text)
	e.emit(Event{
		Type: graph.TypeNodeContent,
		Payload: graph.NodeContentPayload{
			Envelope: graph.NewEnvelope(time.Now().UnixMilli()),
			NodeID:   nodeID,
			Text:     text,
		},
	})
}

func (e *Engine) emitConnectionStateIfChanged(c *graph.Connection, outputs map[string]any) {
	val := ""
	if v, ok := outputs[c.FromSlot]; ok {
		val = fmt.Sprintf("%v", v)
	}
	if prev, loaded := e.lastWireState.Load(c.ID); loaded && prev.(string) == val {
		return
	}
	e.lastWireState.Store(c.ID, val)
	active := val != "" && val != "0" && val != "false" && val != "off" && val != "<nil>"
	e.emit(Event{
		Type: graph.TypeConnectionState,
		Payload: graph.ConnectionStatePayload{
			Envelope:     graph.NewEnvelope(time.Now().UnixMilli()),
			ConnectionID: c.ID,
			Active:       active,
			Value:        val,
		},
	})
}

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
		arrival := emitTime.Add(time.Duration(connDuration(c)) * time.Millisecond)
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

// ---------------------------------------------------------------------------
// Utilities
// ---------------------------------------------------------------------------

func isTimedConnection(c *graph.Connection) bool {
	return connDuration(c) > 0
}

func connDuration(c *graph.Connection) int {
	if c.Config == nil {
		return 0
	}
	d, ok := c.Config["duration"]
	if !ok {
		return 0
	}
	ms, err := strconv.Atoi(d)
	if err != nil || ms <= 0 {
		return 0
	}
	return ms
}

func generateEventID() string {
	var buf [8]byte
	rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}
