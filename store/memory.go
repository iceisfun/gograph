package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/iceisfun/gograph/graph"
)

// MemoryStore is a thread-safe in-memory implementation of [GraphStore].
// Graphs are deep-copied on save and load to prevent aliasing.
type MemoryStore struct {
	mu     sync.RWMutex
	graphs map[string][]byte // id -> JSON-encoded graph
}

// NewMemoryStore creates an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		graphs: make(map[string][]byte),
	}
}

// Save persists a deep copy of the graph.
func (m *MemoryStore) Save(_ context.Context, id string, g *graph.Graph) error {
	data, err := json.Marshal(g)
	if err != nil {
		return fmt.Errorf("memory store: marshal: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.graphs[id] = data
	return nil
}

// Load returns a deep copy of the stored graph.
func (m *MemoryStore) Load(_ context.Context, id string) (*graph.Graph, error) {
	m.mu.RLock()
	data, ok := m.graphs[id]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("memory store: graph %q not found", id)
	}
	var g graph.Graph
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("memory store: unmarshal: %w", err)
	}
	return &g, nil
}

// Delete removes a graph by ID.
func (m *MemoryStore) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.graphs[id]; !ok {
		return fmt.Errorf("memory store: graph %q not found", id)
	}
	delete(m.graphs, id)
	return nil
}

// List returns all stored graph IDs in sorted order.
func (m *MemoryStore) List(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.graphs))
	for id := range m.graphs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}
