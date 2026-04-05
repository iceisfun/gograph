package engine

import (
	"fmt"

	"github.com/iceisfun/gograph/graph"
)

// Order returns the node IDs of a graph in topological execution order
// using Kahn's algorithm. Returns an error if the graph contains cycles.
func Order(g *graph.Graph) ([]string, error) {
	g.RLock()
	defer g.RUnlock()

	// Build adjacency list and in-degree count.
	inDegree := make(map[string]int)
	adj := make(map[string][]string)

	for id := range g.Nodes {
		inDegree[id] = 0
	}

	for _, c := range g.Connections {
		adj[c.FromNode] = append(adj[c.FromNode], c.ToNode)
		inDegree[c.ToNode]++
	}

	// Start with nodes that have no incoming connections.
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var order []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)

		for _, next := range adj[node] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(order) != len(g.Nodes) {
		return nil, fmt.Errorf("graph contains a cycle: ordered %d of %d nodes", len(order), len(g.Nodes))
	}

	return order, nil
}
