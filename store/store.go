// Package store defines a persistence interface for graph storage and
// provides in-memory and JSON file implementations.
package store

import (
	"context"

	"github.com/iceisfun/gograph/graph"
)

// GraphStore is the persistence interface for saving, loading, and listing
// graphs. Implementations must be safe for concurrent use.
type GraphStore interface {
	// Save persists a graph under the given ID. If a graph with that ID
	// already exists it is overwritten.
	Save(ctx context.Context, id string, g *graph.Graph) error

	// Load retrieves a graph by ID. Returns an error if not found.
	Load(ctx context.Context, id string) (*graph.Graph, error)

	// Delete removes a graph by ID. Returns an error if not found.
	Delete(ctx context.Context, id string) error

	// List returns the IDs of all stored graphs.
	List(ctx context.Context) ([]string, error)
}
