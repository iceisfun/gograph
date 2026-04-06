package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/iceisfun/gograph/graph"
)

func timeNowMilli() int64 { return time.Now().UnixMilli() }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// handleListGraphs returns all stored graph IDs.
func (s *Server) handleListGraphs(w http.ResponseWriter, r *http.Request) {
	ids, err := s.store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ids)
}

// handleGetGraph returns a single graph by ID.
func (s *Server) handleGetGraph(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	g, err := s.store.Load(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("graph %q not found", id))
		return
	}
	writeJSON(w, http.StatusOK, g)
}

// handleCreateGraph creates a new graph from the request body.
func (s *Server) handleCreateGraph(w http.ResponseWriter, r *http.Request) {
	var g graph.Graph
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if g.ID == "" {
		writeError(w, http.StatusBadRequest, "graph ID is required")
		return
	}
	if g.Nodes == nil {
		g.Nodes = make(map[string]*graph.Node)
	}
	if err := s.store.Save(r.Context(), g.ID, &g); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, &g)
}

// handleUpdateGraph replaces a graph entirely and reconciles the engine.
func (s *Server) handleUpdateGraph(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var g graph.Graph
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	g.ID = id
	if g.Nodes == nil {
		g.Nodes = make(map[string]*graph.Node)
	}
	if err := s.store.Save(r.Context(), id, &g); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Debug: log connection configs from the saved graph.
	for _, c := range g.Connections {
		if len(c.Config) > 0 {
			fmt.Printf("[api] PUT graph conn %s config=%v\n", c.ID, c.Config)
		} else {
			fmt.Printf("[api] PUT graph conn %s config=<empty>\n", c.ID)
		}
	}

	// Reconcile engine state with the new graph.
	if s.engine != nil {
		s.engine.Sync(r.Context())
	}

	writeJSON(w, http.StatusOK, &g)
}

// handleDeleteGraph stops all nodes and deletes the graph.
func (s *Server) handleDeleteGraph(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if s.engine != nil {
		s.engine.Stop()
	}

	if err := s.store.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("graph %q not found", id))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteNode removes a node via the engine (single removal path).
func (s *Server) handleDeleteNode(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("nodeId")

	if s.engine != nil {
		if err := s.engine.RemoveNode(r.Context(), nodeID); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
	} else {
		graphID := r.PathValue("id")
		g, err := s.store.Load(r.Context(), graphID)
		if err != nil {
			writeError(w, http.StatusNotFound, fmt.Sprintf("graph %q not found", graphID))
			return
		}
		if err := g.RemoveNode(nodeID); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		s.store.Save(r.Context(), graphID, g)
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleClickNode sends a click to the node's goroutine.
func (s *Server) handleClickNode(w http.ResponseWriter, r *http.Request) {
	graphID := r.PathValue("id")
	nodeID := r.PathValue("nodeId")

	if s.engine == nil {
		writeError(w, http.StatusNotImplemented, "no engine configured")
		return
	}

	node, err := s.engine.ClickNode(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	s.broker.publish(graphID, graph.TypeNodeUpdate, graph.NodeUpdatePayload{
		Envelope: graph.NewEnvelope(timeNowMilli()),
		Node:     node,
	})

	writeJSON(w, http.StatusOK, node)
}

// handleAddNode adds a node to the graph and starts its goroutine.
func (s *Server) handleAddNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	g, err := s.store.Load(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("graph %q not found", id))
		return
	}

	var n graph.Node
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if n.ID == "" {
		writeError(w, http.StatusBadRequest, "node ID is required")
		return
	}

	// Apply config defaults from the node type schema.
	if nt, ok := s.registry.Lookup(n.Type); ok {
		if n.Config == nil && len(nt.ConfigSchema) > 0 {
			n.Config = make(map[string]string)
		}
		for _, cf := range nt.ConfigSchema {
			if _, exists := n.Config[cf.Key]; !exists {
				n.Config[cf.Key] = cf.Default
			}
		}
	}

	if err := g.AddNode(&n); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err := s.store.Save(r.Context(), id, g); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.broker.publish(id, graph.TypeNodeUpdate, graph.NodeUpdatePayload{
		Envelope: graph.NewEnvelope(timeNowMilli()),
		Node:     &n,
	})

	// Start the node's goroutine.
	if s.engine != nil {
		s.engine.AddNode(r.Context(), n.ID)
	}

	writeJSON(w, http.StatusCreated, &n)
}

// handleAddConnection creates a connection and its wire.
func (s *Server) handleAddConnection(w http.ResponseWriter, r *http.Request) {
	graphID := r.PathValue("id")

	g, err := s.store.Load(r.Context(), graphID)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("graph %q not found", graphID))
		return
	}

	var c graph.Connection
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if c.ID == "" {
		writeError(w, http.StatusBadRequest, "connection ID is required")
		return
	}

	if err := g.Connect(&c); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err := s.store.Save(r.Context(), graphID, g); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.broker.publish(graphID, graph.TypeConnectionUpdate, graph.ConnectionUpdatePayload{
		Envelope:   graph.NewEnvelope(timeNowMilli()),
		Connection: &c,
	})

	if s.engine != nil {
		s.engine.AddConnection(r.Context(), &c)
	}

	writeJSON(w, http.StatusCreated, &c)
}

// handleRemoveConnection removes a connection and its wire.
func (s *Server) handleRemoveConnection(w http.ResponseWriter, r *http.Request) {
	graphID := r.PathValue("id")
	connID := r.PathValue("connId")

	g, err := s.store.Load(r.Context(), graphID)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("graph %q not found", graphID))
		return
	}

	if err := g.Disconnect(connID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err := s.store.Save(r.Context(), graphID, g); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if s.engine != nil {
		s.engine.RemoveConnection(r.Context(), connID)
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleUpdateNode updates a node's config/label.
func (s *Server) handleUpdateNode(w http.ResponseWriter, r *http.Request) {
	graphID := r.PathValue("id")
	nodeID := r.PathValue("nodeId")

	g, err := s.store.Load(r.Context(), graphID)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("graph %q not found", graphID))
		return
	}

	node := g.Node(nodeID)
	if node == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("node %q not found", nodeID))
		return
	}

	var patch struct {
		Label  *string           `json:"label,omitempty"`
		Config map[string]string `json:"config,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if patch.Label != nil {
		node.Label = *patch.Label
	}
	for k, v := range patch.Config {
		if node.Config == nil {
			node.Config = make(map[string]string)
		}
		node.Config[k] = v
	}

	if err := s.store.Save(r.Context(), graphID, g); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.broker.publish(graphID, graph.TypeNodeUpdate, graph.NodeUpdatePayload{
		Envelope: graph.NewEnvelope(timeNowMilli()),
		Node:     node,
	})

	if s.engine != nil && len(patch.Config) > 0 {
		s.engine.UpdateNodeConfig(r.Context(), nodeID, patch.Config)
	}

	writeJSON(w, http.StatusOK, node)
}

// handleExecuteGraph is a placeholder.
func (s *Server) handleExecuteGraph(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleConfig returns frontend configuration.
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	apiBase := s.prefix + "/api"
	writeJSON(w, http.StatusOK, map[string]string{
		"apiBase": apiBase,
		"mode":    "edit",
	})
}

// handleNodeTypes returns all registered node types.
func (s *Server) handleNodeTypes(w http.ResponseWriter, r *http.Request) {
	types := s.registry.Types()
	writeJSON(w, http.StatusOK, types)
}
