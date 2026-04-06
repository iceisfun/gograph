package server

import "net/http"

// registerRoutes sets up all HTTP routes on the server's mux.
func (s *Server) registerRoutes() {
	p := s.prefix

	// Graph CRUD
	s.mux.HandleFunc("GET "+p+"/api/graphs", s.handleListGraphs)
	s.mux.HandleFunc("GET "+p+"/api/graphs/{id}", s.handleGetGraph)
	s.mux.HandleFunc("POST "+p+"/api/graphs", s.handleCreateGraph)
	s.mux.HandleFunc("PUT "+p+"/api/graphs/{id}", s.handleUpdateGraph)
	s.mux.HandleFunc("DELETE "+p+"/api/graphs/{id}", s.handleDeleteGraph)

	// Node operations
	s.mux.HandleFunc("POST "+p+"/api/graphs/{id}/nodes", s.handleAddNode)
	s.mux.HandleFunc("PATCH "+p+"/api/graphs/{id}/nodes/{nodeId}", s.handleUpdateNode)
	s.mux.HandleFunc("DELETE "+p+"/api/graphs/{id}/nodes/{nodeId}", s.handleDeleteNode)
	s.mux.HandleFunc("POST "+p+"/api/graphs/{id}/nodes/{nodeId}/click", s.handleClickNode)

	// Connection operations
	s.mux.HandleFunc("POST "+p+"/api/graphs/{id}/connections", s.handleAddConnection)
	s.mux.HandleFunc("DELETE "+p+"/api/graphs/{id}/connections/{connId}", s.handleRemoveConnection)

	// Execution placeholder
	s.mux.HandleFunc("POST "+p+"/api/graphs/{id}/execute", s.handleExecuteGraph)

	// SSE event stream
	s.mux.HandleFunc("GET "+p+"/api/graphs/{id}/events", s.handleSSE)

	// Configuration and node types
	s.mux.HandleFunc("GET "+p+"/api/config", s.handleConfig)
	s.mux.HandleFunc("GET "+p+"/api/node-types", s.handleNodeTypes)

	// Static files as catch-all
	if s.prefix != "" {
		s.mux.Handle(p+"/", http.StripPrefix(p, staticHandler(s.staticFS)))
	} else {
		s.mux.Handle("/", staticHandler(s.staticFS))
	}
}
