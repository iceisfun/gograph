package graph

import "encoding/json"

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
	TypeNodeContent      = "node.content"
	TypeConnectionState  = "connection.state"
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

// ContentSlot is a named visual region inside a node's content area.
// Concrete implementations are [TextSlot], [ProgressSlot], [LedSlot],
// [SpinnerSlot], [BadgeSlot], [SparklineSlot], and [ImageSlot].
type ContentSlot interface {
	SlotType() string // "text", "progress", "led", "spinner", "badge", "sparkline", "image"
}

// BaseSlot holds fields shared by slot types that support color and animation.
type BaseSlot struct {
	Type     string `json:"type"`
	Color    string `json:"color,omitempty"`
	Animate  string `json:"animate,omitempty"`  // flash|pulse|none
	Duration int    `json:"duration,omitempty"` // animation ms
}

// TextSlot renders styled text.
type TextSlot struct {
	BaseSlot
	Text  string `json:"text,omitempty"`
	Size  int    `json:"size,omitempty"`  // font size px
	Align string `json:"align,omitempty"` // left|center|right
	Font  string `json:"font,omitempty"`  // monospace|sans-serif
}

func (s *TextSlot) SlotType() string { return "text" }

// MarshalJSON ensures the type field is always "text".
func (s *TextSlot) MarshalJSON() ([]byte, error) {
	type Alias TextSlot
	tmp := (*Alias)(s)
	tmp.Type = "text"
	return json.Marshal(tmp)
}

// ProgressSlot renders a progress bar.
type ProgressSlot struct {
	BaseSlot
	Value float64 `json:"value"` // 0.0..1.0
}

func (s *ProgressSlot) SlotType() string { return "progress" }

func (s *ProgressSlot) MarshalJSON() ([]byte, error) {
	type Alias ProgressSlot
	tmp := (*Alias)(s)
	tmp.Type = "progress"
	return json.Marshal(tmp)
}

// LedSlot renders a row of LED indicators.
type LedSlot struct {
	BaseSlot
	States []bool `json:"states"` // per-LED on/off
}

func (s *LedSlot) SlotType() string { return "led" }

func (s *LedSlot) MarshalJSON() ([]byte, error) {
	type Alias LedSlot
	tmp := (*Alias)(s)
	tmp.Type = "led"
	return json.Marshal(tmp)
}

// SpinnerSlot renders a rotating spinner.
type SpinnerSlot struct {
	BaseSlot
	Visible bool `json:"visible"`
}

func (s *SpinnerSlot) SlotType() string { return "spinner" }

func (s *SpinnerSlot) MarshalJSON() ([]byte, error) {
	type Alias SpinnerSlot
	tmp := (*Alias)(s)
	tmp.Type = "spinner"
	return json.Marshal(tmp)
}

// BadgeSlot renders a colored pill with text.
type BadgeSlot struct {
	BaseSlot
	Text       string `json:"text,omitempty"`
	Background string `json:"background,omitempty"` // pill fill color
}

func (s *BadgeSlot) SlotType() string { return "badge" }

func (s *BadgeSlot) MarshalJSON() ([]byte, error) {
	type Alias BadgeSlot
	tmp := (*Alias)(s)
	tmp.Type = "badge"
	return json.Marshal(tmp)
}

// SparklineSlot renders a tiny inline chart.
type SparklineSlot struct {
	BaseSlot
	Values []float64 `json:"values"`          // data points
	Min    *float64  `json:"min,omitempty"`    // scale minimum (auto if nil)
	Max    *float64  `json:"max,omitempty"`    // scale maximum (auto if nil)
}

func (s *SparklineSlot) SlotType() string { return "sparkline" }

func (s *SparklineSlot) MarshalJSON() ([]byte, error) {
	type Alias SparklineSlot
	tmp := (*Alias)(s)
	tmp.Type = "sparkline"
	return json.Marshal(tmp)
}

// ImageSlot renders an inline image.
type ImageSlot struct {
	BaseSlot
	Src    string `json:"src"`              // data URI or URL
	Width  int    `json:"width,omitempty"`  // display width px
	Height int    `json:"height,omitempty"` // display height px
}

func (s *ImageSlot) SlotType() string { return "image" }

func (s *ImageSlot) MarshalJSON() ([]byte, error) {
	type Alias ImageSlot
	tmp := (*Alias)(s)
	tmp.Type = "image"
	return json.Marshal(tmp)
}

// SvgSlot renders inline SVG markup.
type SvgSlot struct {
	BaseSlot
	Markup string `json:"markup"`           // raw SVG string
	Width  int    `json:"width,omitempty"`  // display width px
	Height int    `json:"height,omitempty"` // display height px
}

func (s *SvgSlot) SlotType() string { return "svg" }

func (s *SvgSlot) MarshalJSON() ([]byte, error) {
	type Alias SvgSlot
	tmp := (*Alias)(s)
	tmp.Type = "svg"
	return json.Marshal(tmp)
}

// unmarshalContentSlot decodes a single content slot from JSON using the
// "type" discriminator.
func unmarshalContentSlot(data json.RawMessage) (ContentSlot, error) {
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, err
	}
	switch probe.Type {
	case "progress":
		var s ProgressSlot
		return &s, json.Unmarshal(data, &s)
	case "led":
		var s LedSlot
		return &s, json.Unmarshal(data, &s)
	case "spinner":
		var s SpinnerSlot
		return &s, json.Unmarshal(data, &s)
	case "badge":
		var s BadgeSlot
		return &s, json.Unmarshal(data, &s)
	case "sparkline":
		var s SparklineSlot
		return &s, json.Unmarshal(data, &s)
	case "image":
		var s ImageSlot
		return &s, json.Unmarshal(data, &s)
	case "svg":
		var s SvgSlot
		return &s, json.Unmarshal(data, &s)
	default: // "" or "text"
		var s TextSlot
		return &s, json.Unmarshal(data, &s)
	}
}

// NodeContentPayload is sent when a node's display content changes.
type NodeContentPayload struct {
	Envelope
	NodeID string                 `json:"nodeID"`
	Text   string                 `json:"text,omitempty"` // backward compat
	Image  string                 `json:"image,omitempty"`
	Slots  map[string]ContentSlot `json:"-"`
}

// MarshalJSON serializes NodeContentPayload with polymorphic slots.
func (p NodeContentPayload) MarshalJSON() ([]byte, error) {
	type Alias struct {
		Envelope
		NodeID string                       `json:"nodeID"`
		Text   string                       `json:"text,omitempty"`
		Image  string                       `json:"image,omitempty"`
		Slots  map[string]json.RawMessage   `json:"slots,omitempty"`
	}
	a := Alias{Envelope: p.Envelope, NodeID: p.NodeID, Text: p.Text, Image: p.Image}
	if len(p.Slots) > 0 {
		a.Slots = make(map[string]json.RawMessage, len(p.Slots))
		for k, v := range p.Slots {
			raw, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			a.Slots[k] = raw
		}
	}
	return json.Marshal(a)
}

// UnmarshalJSON deserializes NodeContentPayload with polymorphic slots.
func (p *NodeContentPayload) UnmarshalJSON(data []byte) error {
	type Alias struct {
		Envelope
		NodeID string                       `json:"nodeID"`
		Text   string                       `json:"text,omitempty"`
		Image  string                       `json:"image,omitempty"`
		Slots  map[string]json.RawMessage   `json:"slots,omitempty"`
	}
	var a Alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	p.Envelope = a.Envelope
	p.NodeID = a.NodeID
	p.Text = a.Text
	p.Image = a.Image
	if len(a.Slots) > 0 {
		p.Slots = make(map[string]ContentSlot, len(a.Slots))
		for k, raw := range a.Slots {
			s, err := unmarshalContentSlot(raw)
			if err != nil {
				return err
			}
			p.Slots[k] = s
		}
	}
	return nil
}

// ConnectionStatePayload is sent for instant connections to convey the
// continuous state of the wire. Unlike timed events (which animate a dot),
// stateful connections show a steady visual based on whether the value is
// active (truthy).
type ConnectionStatePayload struct {
	Envelope
	ConnectionID string `json:"connectionID"`
	Active       bool   `json:"active"`
	Value        string `json:"value,omitempty"`
}

// ConnectionUpdatePayload is sent when a single connection is added or modified.
type ConnectionUpdatePayload struct {
	Envelope
	Connection Connection `json:"connection"`
}
