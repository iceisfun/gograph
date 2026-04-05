package graph

import (
	"encoding/json"
	"errors"
	"fmt"
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
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Label    string            `json:"label"`
	Position Position          `json:"position"`
	Config   map[string]string `json:"config,omitempty"`
}

// Connection is a directed edge from an output slot to an input slot.
type Connection struct {
	ID       string `json:"id"`
	FromNode string `json:"fromNode"`
	FromSlot string `json:"fromSlot"`
	ToNode   string `json:"toNode"`
	ToSlot   string `json:"toSlot"`
}

// Graph is the top-level container for nodes and connections. It is safe for
// concurrent reads but mutations must be synchronized by the caller or done
// through the provided methods which hold an internal lock.
type Graph struct {
	mu sync.RWMutex

	ID          string            `json:"id"`
	Version     int64             `json:"version"`
	Nodes       map[string]*Node  `json:"nodes"`
	Connections []*Connection     `json:"connections"`
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
		if c.FromNode != id && c.ToNode != id {
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
func (g *Graph) Connect(c *Connection) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if c == nil {
		return errors.New("connection is nil")
	}
	if c.ID == "" {
		return errors.New("connection ID is empty")
	}
	if _, exists := g.Nodes[c.FromNode]; !exists {
		return fmt.Errorf("source node %q not found", c.FromNode)
	}
	if _, exists := g.Nodes[c.ToNode]; !exists {
		return fmt.Errorf("target node %q not found", c.ToNode)
	}

	// Check for duplicate connection ID.
	for _, existing := range g.Connections {
		if existing.ID == c.ID {
			return fmt.Errorf("connection %q already exists", c.ID)
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
		if c.ID == id {
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
func (g *Graph) ConnectionByID(id string) *Connection {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, c := range g.Connections {
		if c.ID == id {
			return c
		}
	}
	return nil
}
