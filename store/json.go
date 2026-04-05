package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/iceisfun/gograph/graph"
)

// JSONStore is a file-based implementation of [GraphStore] that persists
// each graph as a JSON file in a directory.
type JSONStore struct {
	dir string
}

// NewJSONStore creates a store that writes graphs to the given directory.
// The directory is created if it does not exist.
func NewJSONStore(dir string) (*JSONStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("json store: create dir: %w", err)
	}
	return &JSONStore{dir: dir}, nil
}

func (s *JSONStore) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}

// Save writes the graph as a JSON file.
func (s *JSONStore) Save(_ context.Context, id string, g *graph.Graph) error {
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return fmt.Errorf("json store: marshal: %w", err)
	}
	p := s.path(id)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("json store: create dir: %w", err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return fmt.Errorf("json store: write: %w", err)
	}
	return nil
}

// Load reads and decodes a graph from its JSON file.
func (s *JSONStore) Load(_ context.Context, id string) (*graph.Graph, error) {
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("json store: graph %q not found", id)
		}
		return nil, fmt.Errorf("json store: read: %w", err)
	}
	var g graph.Graph
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("json store: unmarshal: %w", err)
	}
	return &g, nil
}

// Delete removes the JSON file for the given graph ID.
func (s *JSONStore) Delete(_ context.Context, id string) error {
	err := os.Remove(s.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("json store: graph %q not found", id)
		}
		return fmt.Errorf("json store: remove: %w", err)
	}
	return nil
}

// List returns all graph IDs found in the directory.
func (s *JSONStore) List(_ context.Context) ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("json store: list: %w", err)
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if base, ok := strings.CutSuffix(name, ".json"); ok {
			ids = append(ids, base)
		}
	}
	sort.Strings(ids)
	return ids, nil
}
