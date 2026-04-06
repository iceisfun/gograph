package graph

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
)

// Direction indicates whether a slot accepts or produces data.
type Direction int

const (
	// Input is a slot that receives data from an upstream connection.
	Input Direction = iota
	// Output is a slot that sends data to a downstream connection.
	Output
)

// String returns "input" or "output".
func (d Direction) String() string {
	switch d {
	case Input:
		return "input"
	case Output:
		return "output"
	default:
		return fmt.Sprintf("Direction(%d)", int(d))
	}
}

// MarshalJSON encodes the direction as a JSON string.
func (d Direction) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON decodes a direction from a JSON string.
func (d *Direction) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case "input":
		*d = Input
	case "output":
		*d = Output
	default:
		return fmt.Errorf("unknown direction %q", s)
	}
	return nil
}

// Position is a 2D coordinate on the canvas.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Slot is a typed port on a node. Slots are defined on [NodeType], not on
// individual [Node] instances. All nodes of the same type share identical
// slot definitions.
type Slot struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Direction Direction `json:"direction"`
	DataType  string    `json:"dataType"`
}

// Node is a positioned vertex in the graph. It references a [NodeType] by
// name and stores canvas coordinates for rendering.
type Node struct {
	ID       string                    `json:"id"`
	Type     string                    `json:"type"`
	Label    string                    `json:"label"`
	Position Position                  `json:"position"`
	Config   map[string]string         `json:"config,omitempty"`
	Content  map[string]ContentSlot    `json:"-"` // runtime display state, injected at SSE time
}

// MarshalJSON serializes the node, including Content when populated.
func (n *Node) MarshalJSON() ([]byte, error) {
	type Alias struct {
		ID       string                       `json:"id"`
		Type     string                       `json:"type"`
		Label    string                       `json:"label"`
		Position Position                     `json:"position"`
		Config   map[string]string            `json:"config,omitempty"`
		Content  map[string]json.RawMessage   `json:"content,omitempty"`
	}
	a := Alias{ID: n.ID, Type: n.Type, Label: n.Label, Position: n.Position, Config: n.Config}
	if len(n.Content) > 0 {
		a.Content = make(map[string]json.RawMessage, len(n.Content))
		for k, v := range n.Content {
			raw, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			a.Content[k] = raw
		}
	}
	return json.Marshal(a)
}

// UnmarshalJSON deserializes the node. Content slots are decoded
// polymorphically if present.
func (n *Node) UnmarshalJSON(data []byte) error {
	type Alias struct {
		ID       string                       `json:"id"`
		Type     string                       `json:"type"`
		Label    string                       `json:"label"`
		Position Position                     `json:"position"`
		Config   map[string]string            `json:"config,omitempty"`
		Content  map[string]json.RawMessage   `json:"content,omitempty"`
	}
	var a Alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	n.ID = a.ID
	n.Type = a.Type
	n.Label = a.Label
	n.Position = a.Position
	n.Config = a.Config
	if len(a.Content) > 0 {
		n.Content = make(map[string]ContentSlot, len(a.Content))
		for k, raw := range a.Content {
			s, err := unmarshalContentSlot(raw)
			if err != nil {
				return err
			}
			n.Content[k] = s
		}
	}
	return nil
}

// ConnectionKind distinguishes event-driven connections (discrete messages
// with optional traversal animation) from state connections (continuous
// values like coils, registers, and discrete I/O).
type ConnectionKind string

const (
	EventKind ConnectionKind = "event"
	StateKind ConnectionKind = "state"
)

// Connection is a directed edge from an output slot to an input slot.
// Concrete implementations are [EventConnection] and [StateConnection].
type Connection interface {
	GetID() string
	GetFromNode() string
	GetFromSlot() string
	GetToNode() string
	GetToSlot() string
	GetConfig() map[string]string
	Kind() ConnectionKind
}

// BaseConnection holds fields common to all connection types.
type BaseConnection struct {
	ID       string            `json:"id"`
	FromNode string            `json:"fromNode"`
	FromSlot string            `json:"fromSlot"`
	ToNode   string            `json:"toNode"`
	ToSlot   string            `json:"toSlot"`
	Config   map[string]string `json:"config,omitempty"`
}

func (c *BaseConnection) GetID() string                { return c.ID }
func (c *BaseConnection) GetFromNode() string           { return c.FromNode }
func (c *BaseConnection) GetFromSlot() string           { return c.FromSlot }
func (c *BaseConnection) GetToNode() string             { return c.ToNode }
func (c *BaseConnection) GetToSlot() string             { return c.ToSlot }
func (c *BaseConnection) GetConfig() map[string]string  { return c.Config }

// EventConnection carries discrete messages with optional traversal animation.
type EventConnection struct {
	BaseConnection
	ConnectionKind ConnectionKind `json:"kind"`
	Duration       int            `json:"duration,omitempty"`
}

func (c *EventConnection) Kind() ConnectionKind { return EventKind }

// MarshalJSON ensures the kind field is always "event".
func (c *EventConnection) MarshalJSON() ([]byte, error) {
	type Alias EventConnection
	tmp := (*Alias)(c)
	tmp.ConnectionKind = EventKind
	return json.Marshal(tmp)
}

// StateConnection carries continuous state — a value that "is", not a
// message that "travels". The wire publishes connection.state SSE events
// on value change rather than animating a traversal dot.
type StateConnection struct {
	BaseConnection
	ConnectionKind ConnectionKind `json:"kind"`
	StateDataType  string         `json:"stateDataType,omitempty"` // "bool", "numeric", "string"
}

func (c *StateConnection) Kind() ConnectionKind { return StateKind }

// MarshalJSON ensures the kind field is always "state".
func (c *StateConnection) MarshalJSON() ([]byte, error) {
	type Alias StateConnection
	tmp := (*Alias)(c)
	tmp.ConnectionKind = StateKind
	return json.Marshal(tmp)
}

// NewEventConnection creates an event connection with the given fields.
// Duration is extracted from Config["duration"] if present.
func NewEventConnection(id, fromNode, fromSlot, toNode, toSlot string, config map[string]string) *EventConnection {
	dur := 0
	if config != nil {
		if d, ok := config["duration"]; ok {
			if ms, err := strconv.Atoi(d); err == nil && ms > 0 {
				dur = ms
			}
		}
	}
	return &EventConnection{
		BaseConnection: BaseConnection{
			ID: id, FromNode: fromNode, FromSlot: fromSlot,
			ToNode: toNode, ToSlot: toSlot, Config: config,
		},
		Duration: dur,
	}
}

// NewStateConnection creates a state connection with the given fields.
func NewStateConnection(id, fromNode, fromSlot, toNode, toSlot, dataType string, config map[string]string) *StateConnection {
	return &StateConnection{
		BaseConnection: BaseConnection{
			ID: id, FromNode: fromNode, FromSlot: fromSlot,
			ToNode: toNode, ToSlot: toSlot, Config: config,
		},
		StateDataType: dataType,
	}
}

// Graph is the top-level container for nodes and connections. It is safe for
// concurrent reads but mutations must be synchronized by the caller or done
// through the provided methods which hold an internal lock.
type Graph struct {
	mu sync.RWMutex

	ID          string            `json:"id"`
	Version     int64             `json:"version"`
	Nodes       map[string]*Node  `json:"nodes"`
	Connections []Connection      `json:"-"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NewGraph creates an empty graph with the given ID.
func NewGraph(id string) *Graph {
	return &Graph{
		ID:    id,
		Nodes: make(map[string]*Node),
	}
}

// AddNode adds a node to the graph. Returns an error if a node with the
// same ID already exists.
func (g *Graph) AddNode(n *Node) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if n == nil {
		return errors.New("node is nil")
	}
	if n.ID == "" {
		return errors.New("node ID is empty")
	}
	if _, exists := g.Nodes[n.ID]; exists {
		return fmt.Errorf("node %q already exists", n.ID)
	}
	g.Nodes[n.ID] = n
	g.Version++
	return nil
}

// RemoveNode removes a node and all connections that reference it.
func (g *Graph) RemoveNode(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.Nodes[id]; !exists {
		return fmt.Errorf("node %q not found", id)
	}
	delete(g.Nodes, id)

	// Remove connections referencing this node.
	filtered := g.Connections[:0]
	for _, c := range g.Connections {
		if c.GetFromNode() != id && c.GetToNode() != id {
			filtered = append(filtered, c)
		}
	}
	g.Connections = filtered
	g.Version++
	return nil
}

// Node returns a node by ID, or nil if not found.
func (g *Graph) Node(id string) *Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Nodes[id]
}

// Connect adds a connection to the graph. It validates that the referenced
// nodes exist but does not validate slot compatibility; use [Validate] with
// a [Registry] for full validation.
func (g *Graph) Connect(c Connection) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if c == nil {
		return errors.New("connection is nil")
	}
	if c.GetID() == "" {
		return errors.New("connection ID is empty")
	}
	if _, exists := g.Nodes[c.GetFromNode()]; !exists {
		return fmt.Errorf("source node %q not found", c.GetFromNode())
	}
	if _, exists := g.Nodes[c.GetToNode()]; !exists {
		return fmt.Errorf("target node %q not found", c.GetToNode())
	}

	// Check for duplicate connection ID.
	for _, existing := range g.Connections {
		if existing.GetID() == c.GetID() {
			return fmt.Errorf("connection %q already exists", c.GetID())
		}
	}

	g.Connections = append(g.Connections, c)
	g.Version++
	return nil
}

// Disconnect removes a connection by ID.
func (g *Graph) Disconnect(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i, c := range g.Connections {
		if c.GetID() == id {
			g.Connections = append(g.Connections[:i], g.Connections[i+1:]...)
			g.Version++
			return nil
		}
	}
	return fmt.Errorf("connection %q not found", id)
}

// RLock acquires a read lock on the graph. This is useful for external
// code that needs consistent reads across multiple fields.
func (g *Graph) RLock() { g.mu.RLock() }

// RUnlock releases the read lock.
func (g *Graph) RUnlock() { g.mu.RUnlock() }

// ConnectionByID returns a connection by ID, or nil if not found.
func (g *Graph) ConnectionByID(id string) Connection {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, c := range g.Connections {
		if c.GetID() == id {
			return c
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Polymorphic JSON serialization for Graph
// ---------------------------------------------------------------------------

// graphJSON is the JSON representation of Graph with raw connection blobs.
type graphJSON struct {
	ID          string             `json:"id"`
	Version     int64              `json:"version"`
	Nodes       map[string]*Node   `json:"nodes"`
	Connections []json.RawMessage  `json:"connections"`
	Metadata    map[string]string  `json:"metadata,omitempty"`
}

// MarshalJSON serializes the graph, encoding each Connection via its
// concrete type (which includes the "kind" discriminator).
func (g *Graph) MarshalJSON() ([]byte, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	conns := make([]json.RawMessage, len(g.Connections))
	for i, c := range g.Connections {
		raw, err := json.Marshal(c)
		if err != nil {
			return nil, fmt.Errorf("marshal connection %d: %w", i, err)
		}
		conns[i] = raw
	}
	return json.Marshal(graphJSON{
		ID:          g.ID,
		Version:     g.Version,
		Nodes:       g.Nodes,
		Connections: conns,
		Metadata:    g.Metadata,
	})
}

// UnmarshalJSON deserializes the graph, dispatching each connection to
// the correct concrete type based on the "kind" field. Connections
// without a "kind" field default to [EventConnection].
func (g *Graph) UnmarshalJSON(data []byte) error {
	var raw graphJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	g.ID = raw.ID
	g.Version = raw.Version
	g.Nodes = raw.Nodes
	g.Metadata = raw.Metadata

	g.Connections = make([]Connection, 0, len(raw.Connections))
	for i, blob := range raw.Connections {
		c, err := unmarshalConnection(blob)
		if err != nil {
			return fmt.Errorf("connection %d: %w", i, err)
		}
		g.Connections = append(g.Connections, c)
	}
	return nil
}

// unmarshalConnection decodes a single connection from JSON, using the
// "kind" discriminator to pick the concrete type.
func unmarshalConnection(data json.RawMessage) (Connection, error) {
	var probe struct {
		Kind ConnectionKind `json:"kind"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, err
	}
	switch probe.Kind {
	case StateKind:
		var c StateConnection
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return &c, nil
	default: // "" or "event"
		var c EventConnection
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return &c, nil
	}
}
