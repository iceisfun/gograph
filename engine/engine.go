package engine

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/iceisfun/gograph/graph"
	"github.com/iceisfun/gograph/store"
)

// Engine is a supervisor that manages node goroutines and connection
// wires. It does not execute nodes itself — each node runs in its own
// goroutine with a persistent Lua VM. The engine starts/stops nodes,
// creates/destroys wires, and routes control messages.
type Engine struct {
	mu       sync.RWMutex
	graphID  string
	store    store.GraphStore
	registry *graph.Registry
	broker   EventBroker

	nodes map[string]*nodeRunner // nodeID → runner
	wires map[string]WireRunner      // connID → wire

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	wireBuffer int
}

// EngineOption configures an [Engine].
type EngineOption func(*Engine)

// WithStore sets the graph persistence backend and graph ID.
func WithStore(s store.GraphStore, graphID string) EngineOption {
	return func(e *Engine) { e.store = s; e.graphID = graphID }
}

// WithRegistry sets the node type registry.
func WithRegistry(r *graph.Registry) EngineOption {
	return func(e *Engine) { e.registry = r }
}

// WithBroker sets the SSE event broker.
func WithBroker(b EventBroker) EngineOption {
	return func(e *Engine) { e.broker = b }
}

// WithWireBuffer sets the buffered channel size for wires.
// Defaults to [DefaultWireBuffer].
func WithWireBuffer(n int) EngineOption {
	return func(e *Engine) { e.wireBuffer = n }
}

// New creates a new engine supervisor.
func New(opts ...EngineOption) *Engine {
	e := &Engine{
		nodes:      make(map[string]*nodeRunner),
		wires:      make(map[string]WireRunner),
		wireBuffer: DefaultWireBuffer,
		registry:   graph.NewRegistry(),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Start loads the graph, creates wires and node runners, and starts
// everything. Each node runs in its own goroutine.
func (e *Engine) Start(ctx context.Context) error {
	e.ctx, e.cancel = context.WithCancel(ctx)

	g, err := e.store.Load(ctx, e.graphID)
	if err != nil {
		return fmt.Errorf("load graph: %w", err)
	}

	// Create all node runners first (but don't start them yet).
	g.RLock()
	for _, node := range g.Nodes {
		if err := e.createNodeRunner(node); err != nil {
			g.RUnlock()
			return fmt.Errorf("node %q: %w", node.ID, err)
		}
	}

	// Create wires for all connections and register with source nodes.
	// Safe to mutate outputWires directly here — goroutines haven't started.
	for _, conn := range g.Connections {
		w := e.createWire(conn)
		if nr, ok := e.nodes[conn.GetFromNode()]; ok {
			nr.outputWires[conn.GetFromSlot()] = append(nr.outputWires[conn.GetFromSlot()], w)
		}
	}
	g.RUnlock()

	// Start all node runners.
	for _, nr := range e.nodes {
		nr := nr // capture
		e.wg.Go(func() { nr.run() })
	}

	return nil
}

// Stop cancels all goroutines and waits for them to finish.
// Each node's on_shutdown handler fires via ctx.Done in the select loop.
func (e *Engine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
	e.wg.Wait()

	// Close all wire intake channels.
	e.mu.Lock()
	for _, w := range e.wires {
		w.Close()
	}
	e.wires = make(map[string]WireRunner)
	e.nodes = make(map[string]*nodeRunner)
	e.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Node management
// ---------------------------------------------------------------------------

// AddNode creates and starts a node runner. Does NOT create wires —
// use AddConnection for that (after both endpoints exist).
func (e *Engine) AddNode(ctx context.Context, nodeID string) error {
	g, err := e.store.Load(ctx, e.graphID)
	if err != nil {
		return err
	}

	node := g.Node(nodeID)
	if node == nil {
		return fmt.Errorf("node %q not found", nodeID)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.nodes[nodeID]; exists {
		return nil // already running
	}

	if err := e.createNodeRunner(node); err != nil {
		return err
	}

	nr := e.nodes[nodeID]
	e.wg.Go(func() { nr.run() })

	return nil
}

// RemoveNode shuts down a node, closes its wires, and notifies peers.
func (e *Engine) RemoveNode(ctx context.Context, nodeID string) error {
	e.mu.Lock()

	nr, ok := e.nodes[nodeID]
	if !ok {
		e.mu.Unlock()
		return fmt.Errorf("node %q not running", nodeID)
	}

	// Collect wires connected to this node.
	var affectedWires []WireRunner
	for _, w := range e.wires {
		if w.GetFromNode() == nodeID || w.GetToNode() == nodeID {
			affectedWires = append(affectedWires, w)
		}
	}

	// Close all wires first — unblocks wire goroutines and any stuck emit().
	for _, w := range affectedWires {
		w.Close()
		delete(e.wires, w.GetConnID())
	}

	delete(e.nodes, nodeID)
	e.mu.Unlock()

	// Cancel the node's context — unblocks emit(), select loop, everything.
	// The run() goroutine will exit via <-ctx.Done() and fire on_shutdown.
	nr.cancel()

	// Notify peers about disconnect (non-blocking).
	for _, w := range affectedWires {
		peerID := w.GetToNode()
		if peerID == nodeID {
			peerID = w.GetFromNode()
		}
		e.sendControl(peerID, ControlMsg{
			Kind: CtrlDisconnect,
			Conn: w.GetConn(),
		})
		// Remove wire from peer's output list.
		if w.GetFromNode() != nodeID {
			e.sendControl(w.GetFromNode(), ControlMsg{Kind: CtrlRemoveWire, Wire: w})
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Connection management
// ---------------------------------------------------------------------------

// AddConnection creates a wire for a connection and notifies both nodes.
func (e *Engine) AddConnection(ctx context.Context, conn graph.Connection) error {
	e.mu.Lock()
	e.createWire(conn)
	e.mu.Unlock()

	// Notify both endpoints.
	e.sendControl(conn.GetFromNode(), ControlMsg{
		Kind: CtrlAddWire, Wire: e.wires[conn.GetID()],
	})
	e.sendControl(conn.GetFromNode(), ControlMsg{
		Kind: CtrlConnect, Conn: conn,
	})
	e.sendControl(conn.GetToNode(), ControlMsg{
		Kind: CtrlConnect, Conn: conn,
	})

	return nil
}

// RemoveConnection closes a wire and notifies both nodes.
func (e *Engine) RemoveConnection(ctx context.Context, connID string) error {
	e.mu.Lock()
	w, ok := e.wires[connID]
	if !ok {
		e.mu.Unlock()
		return fmt.Errorf("wire %q not found", connID)
	}
	w.Close()
	delete(e.wires, connID)

	// Remove from source node's output list.
	e.sendControl(w.GetFromNode(), ControlMsg{Kind: CtrlRemoveWire, Wire: w})
	e.mu.Unlock()

	// Notify both endpoints.
	e.sendControl(w.GetFromNode(), ControlMsg{
		Kind: CtrlDisconnect, Conn: w.GetConn(),
	})
	e.sendControl(w.GetToNode(), ControlMsg{
		Kind: CtrlDisconnect, Conn: w.GetConn(),
	})

	return nil
}

// ---------------------------------------------------------------------------
// Node interaction
// ---------------------------------------------------------------------------

// ClickNode sends a click to the node and waits for it to process.
// Returns the node (with any config updates applied).
func (e *Engine) ClickNode(ctx context.Context, nodeID string) (*graph.Node, error) {
	done := make(chan struct{})
	e.sendControl(nodeID, ControlMsg{Kind: CtrlClick, Done: done})
	<-done

	// Reload node from graph with updated config.
	nr := e.getRunner(nodeID)
	if nr == nil {
		return nil, fmt.Errorf("node %q not running", nodeID)
	}

	// Save config updates to the graph.
	g, err := e.store.Load(ctx, e.graphID)
	if err != nil {
		return nil, err
	}
	node := g.Node(nodeID)
	if node == nil {
		return nil, fmt.Errorf("node %q not found", nodeID)
	}
	if node.Config == nil {
		node.Config = make(map[string]string)
	}
	for k, v := range nr.config {
		node.Config[k] = v
	}
	e.store.Save(ctx, e.graphID, g)

	return node, nil
}

// ConnectNode notifies a node that a connection was made.
func (e *Engine) ConnectNode(ctx context.Context, nodeID string, conn graph.Connection) {
	e.sendControl(nodeID, ControlMsg{Kind: CtrlConnect, Conn: conn})
}

// DisconnectNode notifies a node that a connection was removed.
func (e *Engine) DisconnectNode(ctx context.Context, nodeID string, conn graph.Connection) {
	e.sendControl(nodeID, ControlMsg{Kind: CtrlDisconnect, Conn: conn})
}

// UpdateNodeConfig pushes config changes to a running node's VM.
func (e *Engine) UpdateNodeConfig(ctx context.Context, nodeID string, config map[string]string) {
	e.sendControl(nodeID, ControlMsg{Kind: CtrlUpdateConfig, Config: config})
}

// updateNodeLabel persists a label change and publishes a node.update event.
// Called from node runners via callback — safe to call from any goroutine.
func (e *Engine) updateNodeLabel(nodeID, label string) {
	ctx := e.ctx
	g, err := e.store.Load(ctx, e.graphID)
	if err != nil {
		return
	}
	node := g.Node(nodeID)
	if node == nil {
		return
	}
	node.Label = label
	_ = e.store.Save(ctx, e.graphID, g)
	e.broker.Publish(e.graphID, graph.TypeNodeUpdate, graph.NodeUpdatePayload{
		Envelope: graph.NewEnvelope(time.Now().UnixMilli()),
		Node:     node,
	})
}

// InjectContent populates each node's Content map with the current
// display state from running node runners. Call this on a graph snapshot
// before sending it to a new SSE client.
func (e *Engine) InjectContent(g *graph.Graph) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for nodeID, nr := range e.nodes {
		nr.contentMu.RLock()
		if len(nr.contentSlots) == 0 {
			nr.contentMu.RUnlock()
			continue
		}
		node := g.Node(nodeID)
		if node == nil {
			nr.contentMu.RUnlock()
			continue
		}
		node.Content = make(map[string]graph.ContentSlot, len(nr.contentSlots))
		for k, v := range nr.contentSlots {
			node.Content[k] = v
		}
		nr.contentMu.RUnlock()
	}
}

// Sync reconciles the running engine state with the stored graph.
// Nodes/connections that were removed from the graph are shut down.
// Nodes/connections that were added are started. Called after the graph
// is replaced via PUT.
func (e *Engine) Sync(ctx context.Context) {
	g, err := e.store.Load(ctx, e.graphID)
	if err != nil {
		return
	}

	e.mu.Lock()

	// --- Phase 1: Identify what's stale or changed ---

	var staleNodes []*nodeRunner
	var staleNodeIDs []string
	type configUpdate struct {
		nodeID string
		config map[string]string
	}
	var configUpdates []configUpdate

	for nodeID, nr := range e.nodes {
		node := g.Node(nodeID)
		if node == nil {
			staleNodes = append(staleNodes, nr)
			staleNodeIDs = append(staleNodeIDs, nodeID)
		} else if nodeConfigChanged(nr.config, node.Config) {
			configUpdates = append(configUpdates, configUpdate{
				nodeID: nodeID,
				config: node.Config,
			})
		}
	}

	var staleWires []WireRunner
	for connID, w := range e.wires {
		gConn := g.ConnectionByID(connID)
		if gConn == nil {
			staleWires = append(staleWires, w)
		} else if wireConfigChanged(w, gConn) {
			// Config changed (e.g. duration updated) — recreate.
			// Removing from e.wires makes it look "new" in Phase 4.
			staleWires = append(staleWires, w)
		}
	}

	if len(staleNodes) > 0 {
		fmt.Printf("[engine] sync: removing %d nodes, %d wires\n", len(staleNodes), len(staleWires))
	}

	// --- Phase 2: Cancel everything simultaneously ---

	// Cancel all stale node contexts (stops handlers, emit, tick timers).
	for _, nr := range staleNodes {
		nr.cancel()
	}

	// Close stale wires and remove from source nodes' output lists.
	// We send CtrlRemoveWire BEFORE closing so the node goroutine
	// stops writing to the wire before we shut it down.
	e.mu.Unlock()
	for _, w := range staleWires {
		// Remove wire from source node's output list (via control chan).
		e.sendControl(w.GetFromNode(), ControlMsg{Kind: CtrlRemoveWire, Wire: w})
	}
	e.mu.Lock()

	for _, w := range staleWires {
		e.broker.Publish(e.graphID, graph.TypeEventCancel, graph.EventCancelPayload{
			Envelope:  graph.NewEnvelope(0),
			EventID:   "*:" + w.GetConnID(),
			Immediate: true,
		})
		w.Close()
		delete(e.wires, w.GetConnID())
	}

	// --- Phase 3: Clean up remaining wires attached to stale nodes ---

	for _, nodeID := range staleNodeIDs {
		for connID, w := range e.wires {
			if w.GetFromNode() == nodeID || w.GetToNode() == nodeID {
				w.Close()
				delete(e.wires, connID)
			}
		}
		delete(e.nodes, nodeID)
	}

	e.mu.Unlock()

	// --- Phase 3b: Push config updates to running nodes ---
	for _, cu := range configUpdates {
		e.sendControl(cu.nodeID, ControlMsg{Kind: CtrlUpdateConfig, Config: cu.config})
	}

	// --- Phase 4: Add new nodes/connections ---

	g.RLock()
	var addNodes []string
	for nodeID := range g.Nodes {
		if e.getRunner(nodeID) == nil {
			addNodes = append(addNodes, nodeID)
		}
	}

	var addConns []graph.Connection
	for _, conn := range g.Connections {
		e.mu.RLock()
		_, exists := e.wires[conn.GetID()]
		e.mu.RUnlock()
		if !exists {
			addConns = append(addConns, conn)
		}
	}
	g.RUnlock()

	for _, nodeID := range addNodes {
		e.AddNode(ctx, nodeID)
	}
	for _, conn := range addConns {
		e.AddConnection(ctx, conn)
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (e *Engine) createNodeRunner(node *graph.Node) error {
	nt, ok := e.registry.Lookup(node.Type)
	if !ok {
		return fmt.Errorf("unknown type %q", node.Type)
	}
	if nt.Script == "" {
		return nil // no script — nothing to run
	}

	config := node.Config
	if config == nil {
		config = make(map[string]string)
	}

	nr, err := newNodeRunner(e.ctx, node.ID, e.graphID, nt, config, e.broker, e.updateNodeLabel)
	if err != nil {
		return err
	}
	e.nodes[node.ID] = nr
	return nil
}

// createWire creates a wire and starts its goroutine. Does NOT register
// with the source node's outputWires — that must be done separately
// (directly for initial setup, via CtrlAddWire for runtime changes).
func (e *Engine) createWire(conn graph.Connection) WireRunner {
	w := NewWire(conn, e.wireBuffer, e.ctx)

	e.wires[conn.GetID()] = w

	// Start the wire goroutine — delivers to target node's input.
	target := e.getInputChan(conn.GetToNode())
	if target != nil {
		e.wg.Go(func() {
			w.Run(e.graphID, target, e.broker)
		})
	}

	return w
}

func nodeConfigChanged(running map[string]string, stored map[string]string) bool {
	if len(running) != len(stored) {
		return true
	}
	for k, v := range stored {
		if running[k] != v {
			return true
		}
	}
	return false
}

func (e *Engine) getInputChan(nodeID string) chan<- Arrival {
	if nr, ok := e.nodes[nodeID]; ok {
		return nr.inputChan
	}
	return nil
}

func (e *Engine) getRunner(nodeID string) *nodeRunner {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.nodes[nodeID]
}

// wireConfigChanged returns true if the connection's config differs from
// the running wire's config (e.g. duration was changed, or kind changed).
func wireConfigChanged(w WireRunner, conn graph.Connection) bool {
	// Kind changed — must recreate.
	if w.GetConn().Kind() != conn.Kind() {
		return true
	}
	// For event wires, check duration.
	if ew, ok := w.(*EventWire); ok {
		newDur := 0
		if cfg := conn.GetConfig(); cfg != nil {
			if d, ok := cfg["duration"]; ok {
				if ms, err := strconv.Atoi(d); err == nil && ms > 0 {
					newDur = ms
				}
			}
		}
		return ew.Duration != newDur
	}
	return false
}

func (e *Engine) sendControl(nodeID string, msg ControlMsg) {
	e.mu.RLock()
	nr, ok := e.nodes[nodeID]
	e.mu.RUnlock()
	if !ok {
		return
	}
	select {
	case nr.controlChan <- msg:
	case <-e.ctx.Done():
	}
}
