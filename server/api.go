package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/iceisfun/gograph/graph"
)

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

// handleUpdateGraph replaces a graph entirely.
func (s *Server) handleUpdateGraph(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var g graph.Graph
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	// Ensure the graph ID matches the URL.
	g.ID = id
	if g.Nodes == nil {
		g.Nodes = make(map[string]*graph.Node)
	}
	if err := s.store.Save(r.Context(), id, &g); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, &g)
}

// handleDeleteGraph removes a graph by ID.
func (s *Server) handleDeleteGraph(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("graph %q not found", id))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleExecuteGraph is a placeholder for triggering graph execution.
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
