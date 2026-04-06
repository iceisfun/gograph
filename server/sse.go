package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/iceisfun/gograph/graph"
)

// sseBroker manages per-graph SSE client subscriptions and fans out
// published events to all connected clients for a given graph.
type sseBroker struct {
	mu      sync.RWMutex
	clients map[string]map[chan []byte]struct{} // graphID -> set of client channels
}

func newSSEBroker() *sseBroker {
	return &sseBroker{
		clients: make(map[string]map[chan []byte]struct{}),
	}
}

// subscribe registers a new client channel for the given graph ID.
// The returned channel is buffered (size 64) and receives serialized
// SSE event data.
func (b *sseBroker) subscribe(graphID string) chan []byte {
	ch := make(chan []byte, 64)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.clients[graphID] == nil {
		b.clients[graphID] = make(map[chan []byte]struct{})
	}
	b.clients[graphID][ch] = struct{}{}
	return ch
}

// unsubscribe removes a client channel and closes it.
func (b *sseBroker) unsubscribe(graphID string, ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if subs, ok := b.clients[graphID]; ok {
		delete(subs, ch)
		if len(subs) == 0 {
			delete(b.clients, graphID)
		}
	}
	close(ch)
}

// publish sends an event to all clients subscribed to the given graph.
// Slow clients have events dropped via non-blocking send.
func (b *sseBroker) publish(graphID string, eventType string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	msg := fmt.Appendf(nil, "event: %s\ndata: %s\n\n", eventType, data)

	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients[graphID] {
		select {
		case ch <- msg:
		default:
			// slow client, drop event
		}
	}
}

// Publish sends an event to all SSE clients subscribed to the given graph.
// This is the public API for the engine to push events.
func (s *Server) Publish(graphID string, eventType string, payload any) {
	s.broker.publish(graphID, eventType, payload)
}

// handleSSE is the HTTP handler for the SSE event stream endpoint.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	graphID := r.PathValue("id")

	// Verify the graph exists.
	g, err := s.store.Load(r.Context(), graphID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"graph not found: %s"}`, graphID), http.StatusNotFound)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := s.broker.subscribe(graphID)
	defer s.broker.unsubscribe(graphID, ch)

	// Inject current display content into nodes before sending.
	if s.engine != nil {
		s.engine.InjectContent(g)
	}

	// Send the full graph state on connect.
	snapshot := graph.GraphUpdatePayload{
		Envelope: graph.NewEnvelope(time.Now().UnixMilli()),
		Graph:    g,
	}
	initial, _ := json.Marshal(snapshot)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", graph.TypeGraphUpdate, initial)
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			w.Write(msg)
			flusher.Flush()
		}
	}
}
