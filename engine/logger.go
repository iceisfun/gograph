package engine

import (
	"log"
)

// NodeLogger receives lifecycle events for node execution.
type NodeLogger interface {
	// NodeSkipped is called when a node is skipped (disconnected inputs).
	NodeSkipped(nodeID string, reason string)
	// NodeWaiting is called when a node is waiting for incoming traversals.
	NodeWaiting(nodeID string, waitMs int64)
	// NodeExecuting is called when a node begins script execution.
	NodeExecuting(nodeID string, nodeType string, inputCount int)
	// NodeExecuted is called when a node finishes script execution.
	NodeExecuted(nodeID string, nodeType string, outputCount int)
	// NodeHolding is called when a node is holding (delay/processing wait).
	NodeHolding(nodeID string, durationMs int)
	// NodeDisconnected is called when a node has an unconnected output slot.
	NodeDisconnected(nodeID string, slotID string)
}

// EventLogger receives lifecycle events for connection traversal events.
type EventLogger interface {
	// EventEmitted is called when an event is created on a connection.
	EventEmitted(eventID string, connectionID string, fromNode string, toNode string, durationMs int)
	// EventArrived is called when an event completes traversal.
	EventArrived(eventID string, connectionID string, toNode string)
	// EventCancelled is called when all events are cancelled.
	EventCancelled(reason string)
}

// NopNodeLogger is a no-op implementation of [NodeLogger].
type NopNodeLogger struct{}

func (NopNodeLogger) NodeSkipped(string, string)        {}
func (NopNodeLogger) NodeWaiting(string, int64)         {}
func (NopNodeLogger) NodeExecuting(string, string, int) {}
func (NopNodeLogger) NodeExecuted(string, string, int)  {}
func (NopNodeLogger) NodeHolding(string, int)           {}
func (NopNodeLogger) NodeDisconnected(string, string)   {}

// NopEventLogger is a no-op implementation of [EventLogger].
type NopEventLogger struct{}

func (NopEventLogger) EventEmitted(string, string, string, string, int) {}
func (NopEventLogger) EventArrived(string, string, string)              {}
func (NopEventLogger) EventCancelled(string)                            {}

// DebugNodeLogger logs node lifecycle events to the standard logger.
type DebugNodeLogger struct{}

func (DebugNodeLogger) NodeSkipped(nodeID, reason string) {
	log.Printf("[node] %s: skipped (%s)", nodeID, reason)
}

func (DebugNodeLogger) NodeWaiting(nodeID string, waitMs int64) {
	log.Printf("[node] %s: waiting %dms for incoming traversals", nodeID, waitMs)
}

func (DebugNodeLogger) NodeExecuting(nodeID, nodeType string, inputCount int) {
	log.Printf("[node] %s (%s): executing with %d inputs", nodeID, nodeType, inputCount)
}

func (DebugNodeLogger) NodeExecuted(nodeID, nodeType string, outputCount int) {
	log.Printf("[node] %s (%s): done, %d outputs", nodeID, nodeType, outputCount)
}

func (DebugNodeLogger) NodeHolding(nodeID string, durationMs int) {
	log.Printf("[node] %s: holding %dms", nodeID, durationMs)
}

func (DebugNodeLogger) NodeDisconnected(nodeID, slotID string) {
	log.Printf("[node] %s: output %q not connected", nodeID, slotID)
}

// DebugEventLogger logs event lifecycle events to the standard logger.
type DebugEventLogger struct{}

func (DebugEventLogger) EventEmitted(eventID, connectionID, fromNode, toNode string, durationMs int) {
	if durationMs > 0 {
		log.Printf("[event] %s: emitted on %s (%s → %s) traversal %dms", eventID, connectionID, fromNode, toNode, durationMs)
	} else {
		log.Printf("[event] %s: emitted on %s (%s → %s) instant", eventID, connectionID, fromNode, toNode)
	}
}

func (DebugEventLogger) EventArrived(eventID, connectionID, toNode string) {
	log.Printf("[event] %s: arrived at %s via %s", eventID, toNode, connectionID)
}

func (DebugEventLogger) EventCancelled(reason string) {
	log.Printf("[event] all events cancelled: %s", reason)
}
