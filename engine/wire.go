package engine

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/iceisfun/gograph/graph"
)

// DefaultWireBuffer is the default buffered channel size for wire intake.
const DefaultWireBuffer = 8

// WireMessage is a value traveling through a connection.
type WireMessage struct {
	Value  any
	Source string // from node ID
}

// Arrival is a value that has been delivered to a node's input slot.
type Arrival struct {
	Slot   string
	Value  any
	Source string // from node ID
	ConnID string
	Kind   graph.ConnectionKind // event or state
}

// WireRunner is the interface for connection runtimes. Concrete
// implementations are [EventWire] (dot animation + delay) and
// [StateWire] (change detection + steady state).
type WireRunner interface {
	Run(graphID string, target chan<- Arrival, broker EventBroker)
	Close()
	GetConnID() string
	GetFromNode() string
	GetFromSlot() string
	GetToNode() string
	GetConn() graph.Connection
	GetIntake() chan<- WireMessage
}

// ---------------------------------------------------------------------------
// EventWire — discrete message with optional traversal animation
// ---------------------------------------------------------------------------

// EventWire is a connection runtime for event connections. It accepts
// values from an emitting node, applies a traversal delay, emits SSE
// animation events, and delivers to the downstream node's input channel.
type EventWire struct {
	ConnID   string
	Conn     graph.Connection
	Intake   chan WireMessage
	FromNode string
	FromSlot string
	ToNode   string
	ToSlot   string
	Duration int // ms, 0 = instant

	ctx    context.Context
	cancel context.CancelFunc
}

func (w *EventWire) GetConnID() string              { return w.ConnID }
func (w *EventWire) GetFromNode() string             { return w.FromNode }
func (w *EventWire) GetFromSlot() string             { return w.FromSlot }
func (w *EventWire) GetToNode() string               { return w.ToNode }
func (w *EventWire) GetConn() graph.Connection       { return w.Conn }
func (w *EventWire) GetIntake() chan<- WireMessage    { return w.Intake }
func (w *EventWire) Close()                          { w.cancel() }

// NewEventWire creates an event wire from a connection.
func NewEventWire(conn graph.Connection, bufSize int, parent ...context.Context) *EventWire {
	p := context.Background()
	if len(parent) > 0 && parent[0] != nil {
		p = parent[0]
	}
	ctx, cancel := context.WithCancel(p)
	dur := 0
	if cfg := conn.GetConfig(); cfg != nil {
		if d, ok := cfg["duration"]; ok {
			if ms, err := strconv.Atoi(d); err == nil && ms > 0 {
				dur = ms
			}
		}
	}
	return &EventWire{
		ConnID:   conn.GetID(),
		Conn:     conn,
		Intake:   make(chan WireMessage, bufSize),
		FromNode: conn.GetFromNode(),
		FromSlot: conn.GetFromSlot(),
		ToNode:   conn.GetToNode(),
		ToSlot:   conn.GetToSlot(),
		Duration: dur,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Run is the event wire's goroutine.
func (w *EventWire) Run(graphID string, target chan<- Arrival, broker EventBroker) {
	for {
		select {
		case <-w.ctx.Done():
			return
		case msg, ok := <-w.Intake:
			if !ok {
				return
			}

			now := time.Now().UnixMilli()
			eventID := generateID()

			fmt.Printf("[wire:event] %s: %s.%s → %s.%s value=%v dur=%dms\n",
				w.ConnID, w.FromNode, w.FromSlot, w.ToNode, w.ToSlot, msg.Value, w.Duration)

			broker.Publish(graphID, graph.TypeEventStart, graph.EventStartPayload{
				Envelope:     graph.NewEnvelope(now),
				EventID:      eventID,
				ConnectionID: w.ConnID,
				Duration:     w.Duration,
			})

			if w.Duration > 0 {
				timer := time.NewTimer(time.Duration(w.Duration) * time.Millisecond)
				select {
				case <-w.ctx.Done():
					timer.Stop()
					broker.Publish(graphID, graph.TypeEventCancel, graph.EventCancelPayload{
						Envelope:  graph.NewEnvelope(time.Now().UnixMilli()),
						EventID:   eventID,
						Immediate: true,
					})
					return
				case <-timer.C:
				}
			}

			select {
			case <-w.ctx.Done():
				broker.Publish(graphID, graph.TypeEventCancel, graph.EventCancelPayload{
					Envelope:  graph.NewEnvelope(time.Now().UnixMilli()),
					EventID:   eventID,
					Immediate: true,
				})
				return
			case target <- Arrival{
				Slot:   w.ToSlot,
				Value:  msg.Value,
				Source: msg.Source,
				ConnID: w.ConnID,
				Kind:   graph.EventKind,
			}:
			}

			broker.Publish(graphID, graph.TypeEventEnd, graph.EventEndPayload{
				Envelope: graph.NewEnvelope(time.Now().UnixMilli()),
				EventID:  eventID,
			})
		}
	}
}

// ---------------------------------------------------------------------------
// StateWire — continuous state with change detection
// ---------------------------------------------------------------------------

// StateWire is a connection runtime for state connections. It tracks the
// previous value and only propagates + publishes when the value changes.
// No dot animation — the frontend renders a steady glow based on the
// connection.state SSE event.
type StateWire struct {
	ConnID   string
	Conn     graph.Connection
	Intake   chan WireMessage
	FromNode string
	FromSlot string
	ToNode   string
	ToSlot   string

	ctx    context.Context
	cancel context.CancelFunc
}

func (w *StateWire) GetConnID() string              { return w.ConnID }
func (w *StateWire) GetFromNode() string             { return w.FromNode }
func (w *StateWire) GetFromSlot() string             { return w.FromSlot }
func (w *StateWire) GetToNode() string               { return w.ToNode }
func (w *StateWire) GetConn() graph.Connection       { return w.Conn }
func (w *StateWire) GetIntake() chan<- WireMessage    { return w.Intake }
func (w *StateWire) Close()                          { w.cancel() }

// NewStateWire creates a state wire from a connection.
func NewStateWire(conn graph.Connection, bufSize int, parent ...context.Context) *StateWire {
	p := context.Background()
	if len(parent) > 0 && parent[0] != nil {
		p = parent[0]
	}
	ctx, cancel := context.WithCancel(p)
	return &StateWire{
		ConnID:   conn.GetID(),
		Conn:     conn,
		Intake:   make(chan WireMessage, bufSize),
		FromNode: conn.GetFromNode(),
		FromSlot: conn.GetFromSlot(),
		ToNode:   conn.GetToNode(),
		ToSlot:   conn.GetToSlot(),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Run is the state wire's goroutine. It performs change detection and
// publishes connection.state SSE events on value change.
func (w *StateWire) Run(graphID string, target chan<- Arrival, broker EventBroker) {
	var prev any
	first := true

	for {
		select {
		case <-w.ctx.Done():
			return
		case msg, ok := <-w.Intake:
			if !ok {
				return
			}

			// Change detection: skip if value hasn't changed.
			valStr := fmt.Sprintf("%v", msg.Value)
			prevStr := fmt.Sprintf("%v", prev)
			if !first && valStr == prevStr {
				continue
			}
			prev = msg.Value
			first = false

			active := isTruthy(msg.Value)

			fmt.Printf("[wire:state] %s: %s.%s → %s.%s value=%v active=%v\n",
				w.ConnID, w.FromNode, w.FromSlot, w.ToNode, w.ToSlot, msg.Value, active)

			// Publish steady-state to frontend.
			broker.Publish(graphID, graph.TypeConnectionState, graph.ConnectionStatePayload{
				Envelope:     graph.NewEnvelope(time.Now().UnixMilli()),
				ConnectionID: w.ConnID,
				Active:       active,
				Value:        valStr,
			})

			// Deliver to downstream node immediately (no delay).
			select {
			case <-w.ctx.Done():
				return
			case target <- Arrival{
				Slot:   w.ToSlot,
				Value:  msg.Value,
				Source: msg.Source,
				ConnID: w.ConnID,
				Kind:   graph.StateKind,
			}:
			}
		}
	}
}

// isTruthy returns true for values that represent an "on" state.
func isTruthy(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != "" && val != "0" && val != "false" && val != "off"
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	default:
		return v != nil
	}
}

// ---------------------------------------------------------------------------
// Wire constructor — picks the right type based on connection kind
// ---------------------------------------------------------------------------

// NewWire creates the appropriate WireRunner for a connection.
func NewWire(conn graph.Connection, bufSize int, parent ...context.Context) WireRunner {
	if conn.Kind() == graph.StateKind {
		return NewStateWire(conn, bufSize, parent...)
	}
	return NewEventWire(conn, bufSize, parent...)
}

// ---------------------------------------------------------------------------
// Control messages — engine to nodeRunner communication
// ---------------------------------------------------------------------------

// ControlKind identifies what kind of control message this is.
type ControlKind int

const (
	CtrlClick        ControlKind = iota // user clicked
	CtrlConnect                         // connection made to/from this node
	CtrlDisconnect                      // connection removed
	CtrlUpdateConfig                    // node config changed
	CtrlAddWire                         // new output wire registered
	CtrlRemoveWire                      // output wire unregistered
	CtrlShutdown                        // stop the node
)

// ControlMsg is a message sent from the engine to a nodeRunner.
type ControlMsg struct {
	Kind   ControlKind
	Action string             // configure: "connect"/"disconnect"/"update"
	Conn   graph.Connection   // for configure connect/disconnect
	Config map[string]string  // for config updates
	Wire   WireRunner         // for add/remove wire
	Done   chan struct{}       // closed when processed
}

// EventBroker publishes SSE events. Implemented by the server's SSE broker.
type EventBroker interface {
	Publish(graphID string, eventType string, payload any)
}

func generateID() string {
	var buf [8]byte
	rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}
