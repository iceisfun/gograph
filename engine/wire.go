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
}

// Wire is a connection runtime — a goroutine that accepts values from
// an emitting node, applies a traversal delay, emits SSE events, and
// delivers to the downstream node's input channel.
type Wire struct {
	ConnID   string
	Conn     *graph.Connection
	Intake   chan WireMessage // emit() writes here
	FromNode string
	FromSlot string
	ToNode   string
	ToSlot   string
	Duration int // ms, 0 = instant

	ctx    context.Context
	cancel context.CancelFunc
}

// NewWire creates a wire for a connection with a buffered intake channel.
func NewWire(conn *graph.Connection, bufSize int) *Wire {
	ctx, cancel := context.WithCancel(context.Background())
	dur := 0
	if conn.Config != nil {
		if d, ok := conn.Config["duration"]; ok {
			if ms, err := strconv.Atoi(d); err == nil && ms > 0 {
				dur = ms
			}
		}
	}
	return &Wire{
		ConnID:   conn.ID,
		Conn:     conn,
		Intake:   make(chan WireMessage, bufSize),
		FromNode: conn.FromNode,
		FromSlot: conn.FromSlot,
		ToNode:   conn.ToNode,
		ToSlot:   conn.ToSlot,
		Duration: dur,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Run is the wire's goroutine. It reads from Intake, applies the
// traversal delay, emits SSE events, and delivers to the target node.
// It blocks on delivery if the target is busy (visible backpressure).
func (w *Wire) Run(graphID string, target chan<- Arrival, broker EventBroker) {
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

			fmt.Printf("[wire] %s: %s.%s → %s.%s value=%v dur=%dms\n",
				w.ConnID, w.FromNode, w.FromSlot, w.ToNode, w.ToSlot, msg.Value, w.Duration)

			// Event enters the connection — dot appears.
			broker.Publish(graphID, graph.TypeEventStart, graph.EventStartPayload{
				Envelope:     graph.NewEnvelope(now),
				EventID:      eventID,
				ConnectionID: w.ConnID,
				Duration:     w.Duration,
			})

			// Connection applies its traversal delay.
			if w.Duration > 0 {
				timer := time.NewTimer(time.Duration(w.Duration) * time.Millisecond)
				select {
				case <-w.ctx.Done():
					timer.Stop()
					// Wire cancelled mid-flight — kill the dot.
					broker.Publish(graphID, graph.TypeEventCancel, graph.EventCancelPayload{
						Envelope:  graph.NewEnvelope(time.Now().UnixMilli()),
						EventID:   eventID,
						Immediate: true,
					})
					return
				case <-timer.C:
				}
			}

			// Deliver to downstream node. May block if node is busy
			// (backpressure — dot stalls at the end).
			select {
			case <-w.ctx.Done():
				// Wire cancelled while waiting to deliver — kill the dot.
				broker.Publish(graphID, graph.TypeEventCancel, graph.EventCancelPayload{
					Envelope:  graph.NewEnvelope(time.Now().UnixMilli()),
					EventID:   eventID,
					Immediate: true,
				})
				return
			case target <- Arrival{
				Slot:   w.ToSlot,
				Value:  msg.Value,
				Source:  msg.Source,
				ConnID: w.ConnID,
			}:
			}

			// Dot arrives.
			broker.Publish(graphID, graph.TypeEventEnd, graph.EventEndPayload{
				Envelope: graph.NewEnvelope(time.Now().UnixMilli()),
				EventID:  eventID,
			})
		}
	}
}

// Close stops the wire goroutine and closes the intake channel.
func (w *Wire) Close() {
	w.cancel()
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
	Action string            // configure: "connect"/"disconnect"/"update"
	Conn   *graph.Connection // for configure connect/disconnect
	Config map[string]string // for config updates
	Wire   *Wire             // for add/remove wire
	Done   chan struct{}      // closed when processed
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
