package graph

// ProtocolVersion is the current version of the SSE wire protocol.
const ProtocolVersion = 1

// SSE event type constants. These are used as the "event:" field in SSE
// messages and must match the TypeScript constants in core/protocol.ts.
const (
	TypeEventStart       = "event.start"
	TypeEventUpdate      = "event.update"
	TypeEventEnd         = "event.end"
	TypeEventCancel      = "event.cancel"
	TypeGraphUpdate      = "graph.update"
	TypeNodeUpdate       = "node.update"
	TypeNodeActive       = "node.active"
	TypeConnectionUpdate = "connection.update"
)

// Envelope is the common header embedded in every SSE event payload.
type Envelope struct {
	Version   int   `json:"v"`
	Timestamp int64 `json:"ts"`
}

// NewEnvelope creates an envelope with the current protocol version and
// the given timestamp in Unix milliseconds.
func NewEnvelope(ts int64) Envelope {
	return Envelope{
		Version:   ProtocolVersion,
		Timestamp: ts,
	}
}

// EventStartPayload is sent when a new event spawns at an output slot
// and begins traversing a connection.
type EventStartPayload struct {
	Envelope
	EventID      string         `json:"eventID"`
	ConnectionID string         `json:"connectionID"`
	Color        string         `json:"color,omitempty"`
	Duration     int            `json:"duration"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// EventUpdatePayload is sent for mid-flight mutations to an active event
// (color change, intensity change, metadata update).
type EventUpdatePayload struct {
	Envelope
	EventID   string         `json:"eventID"`
	Color     string         `json:"color,omitempty"`
	Intensity float64        `json:"intensity,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// EventEndPayload is sent when an event completes its traversal and
// arrives at the target input slot.
type EventEndPayload struct {
	Envelope
	EventID string `json:"eventID"`
}

// EventCancelPayload is sent to cancel an in-flight event. If Immediate
// is true the client removes it instantly; otherwise it plays a fade-out.
type EventCancelPayload struct {
	Envelope
	EventID   string `json:"eventID"`
	Immediate bool   `json:"immediate"`
}

// GraphUpdatePayload is sent when the full graph state changes. The
// client replaces its local graph model with the provided state.
type GraphUpdatePayload struct {
	Envelope
	Graph *Graph `json:"graph"`
}

// NodeUpdatePayload is sent when a single node is added or modified.
type NodeUpdatePayload struct {
	Envelope
	Node *Node `json:"node"`
}

// NodeActivePayload is sent when a node becomes active (e.g. a delay node
// starts holding). The frontend renders a glow effect for the duration.
type NodeActivePayload struct {
	Envelope
	NodeID   string `json:"nodeID"`
	Duration int    `json:"duration"`
}

// ConnectionUpdatePayload is sent when a single connection is added or modified.
type ConnectionUpdatePayload struct {
	Envelope
	Connection *Connection `json:"connection"`
}
