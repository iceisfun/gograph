package graph

import (
	"fmt"
)

// ConnectionValidator determines whether two slots may be connected.
// The default validator checks direction and data type compatibility.
type ConnectionValidator func(from, to Slot) error

// CanConnect checks whether a connection from an output slot to an input
// slot is valid. It verifies directions and data type compatibility.
// Two slots are type-compatible if either has DataType "any" or both
// share the same DataType.
func CanConnect(from, to Slot) error {
	if from.Direction != Output {
		return fmt.Errorf("slot %q is not an output", from.ID)
	}
	if to.Direction != Input {
		return fmt.Errorf("slot %q is not an input", to.ID)
	}
	if from.DataType != "any" && to.DataType != "any" && from.DataType != to.DataType {
		return fmt.Errorf("incompatible data types: %q -> %q", from.DataType, to.DataType)
	}
	return nil
}

// Validate checks the structural integrity of a graph against a registry.
// It verifies that every node references a registered type and that every
// connection references valid nodes and compatible slots.
func (g *Graph) Validate(r *Registry) error {
	return g.ValidateWith(r, CanConnect)
}

// ValidateWith checks the graph using a custom connection validator.
func (g *Graph) ValidateWith(r *Registry, validator ConnectionValidator) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Validate all nodes reference registered types.
	for _, n := range g.Nodes {
		if _, ok := r.Lookup(n.Type); !ok {
			return fmt.Errorf("node %q references unknown type %q", n.ID, n.Type)
		}
	}

	// Validate all connections.
	for _, c := range g.Connections {
		fromNode, ok := g.Nodes[c.FromNode]
		if !ok {
			return fmt.Errorf("connection %q references unknown source node %q", c.ID, c.FromNode)
		}
		toNode, ok := g.Nodes[c.ToNode]
		if !ok {
			return fmt.Errorf("connection %q references unknown target node %q", c.ID, c.ToNode)
		}

		fromType, _ := r.Lookup(fromNode.Type)
		toType, _ := r.Lookup(toNode.Type)

		fromSlot, ok := fromType.SlotByID(c.FromSlot)
		if !ok {
			return fmt.Errorf("connection %q references unknown slot %q on node %q (type %q)",
				c.ID, c.FromSlot, c.FromNode, fromNode.Type)
		}
		toSlot, ok := toType.SlotByID(c.ToSlot)
		if !ok {
			return fmt.Errorf("connection %q references unknown slot %q on node %q (type %q)",
				c.ID, c.ToSlot, c.ToNode, toNode.Type)
		}

		if err := validator(fromSlot, toSlot); err != nil {
			return fmt.Errorf("connection %q: %w", c.ID, err)
		}
	}

	return nil
}
