package graph

import (
	"fmt"
	"sync"
)

// NodeType defines a category of node with its input and output slots.
// All [Node] instances referencing the same type name share these slot
// definitions. The optional Script field holds Lua source for execution.
type NodeType struct {
	Name       string `json:"name"`
	Label      string `json:"label"`
	Category   string `json:"category,omitempty"`
	Slots      []Slot `json:"slots"`
	Script     string `json:"-"`
	ScriptName string `json:"scriptName,omitempty"`
}

// InputSlots returns only the input slots for this type.
func (nt NodeType) InputSlots() []Slot {
	var out []Slot
	for _, s := range nt.Slots {
		if s.Direction == Input {
			out = append(out, s)
		}
	}
	return out
}

// OutputSlots returns only the output slots for this type.
func (nt NodeType) OutputSlots() []Slot {
	var out []Slot
	for _, s := range nt.Slots {
		if s.Direction == Output {
			out = append(out, s)
		}
	}
	return out
}

// SlotByID returns the slot with the given ID, or false if not found.
func (nt NodeType) SlotByID(id string) (Slot, bool) {
	for _, s := range nt.Slots {
		if s.ID == id {
			return s, true
		}
	}
	return Slot{}, false
}

// Registry manages registered [NodeType] definitions. It is safe for
// concurrent use.
type Registry struct {
	mu    sync.RWMutex
	types map[string]NodeType
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		types: make(map[string]NodeType),
	}
}

// Register adds a node type to the registry. Returns an error if the name
// is empty or already registered.
func (r *Registry) Register(nt NodeType) error {
	if nt.Name == "" {
		return fmt.Errorf("node type name is empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.types[nt.Name]; exists {
		return fmt.Errorf("node type %q already registered", nt.Name)
	}
	r.types[nt.Name] = nt
	return nil
}

// Lookup returns the node type with the given name.
func (r *Registry) Lookup(name string) (NodeType, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	nt, ok := r.types[name]
	return nt, ok
}

// Types returns all registered node types.
func (r *Registry) Types() []NodeType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]NodeType, 0, len(r.types))
	for _, nt := range r.types {
		out = append(out, nt)
	}
	return out
}
