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
	// Check connection kind compatibility (unless either is "any").
	if from.DataType != "any" && to.DataType != "any" {
		if SlotConnectionKind(from.DataType) != SlotConnectionKind(to.DataType) {
			return fmt.Errorf("incompatible connection kinds: %q (%s) -> %q (%s)",
				from.DataType, SlotConnectionKind(from.DataType),
				to.DataType, SlotConnectionKind(to.DataType))
		}
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
		fromNode, ok := g.Nodes[c.GetFromNode()]
		if !ok {
			return fmt.Errorf("connection %q references unknown source node %q", c.GetID(), c.GetFromNode())
		}
		toNode, ok := g.Nodes[c.GetToNode()]
		if !ok {
			return fmt.Errorf("connection %q references unknown target node %q", c.GetID(), c.GetToNode())
		}

		fromType, _ := r.Lookup(fromNode.Type)
		toType, _ := r.Lookup(toNode.Type)

		fromSlot, ok := fromType.SlotByID(c.GetFromSlot())
		if !ok {
			return fmt.Errorf("connection %q references unknown slot %q on node %q (type %q)",
				c.GetID(), c.GetFromSlot(), c.GetFromNode(), fromNode.Type)
		}
		toSlot, ok := toType.SlotByID(c.GetToSlot())
		if !ok {
			return fmt.Errorf("connection %q references unknown slot %q on node %q (type %q)",
				c.GetID(), c.GetToSlot(), c.GetToNode(), toNode.Type)
		}

		if err := validator(fromSlot, toSlot); err != nil {
			return fmt.Errorf("connection %q: %w", c.GetID(), err)
		}
	}

	return nil
}
