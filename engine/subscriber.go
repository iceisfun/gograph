package engine

// Event is an engine-produced event that should be forwarded to SSE clients.
type Event struct {
	Type    string // one of graph.Type* constants
	Payload any    // one of graph.*Payload types
}

// Subscriber receives engine events through a buffered channel.
// Call [Done] when finished to release resources.
type Subscriber struct {
	ch   chan Event
	done chan struct{}
}

func newSubscriber(bufferSize int) *Subscriber {
	return &Subscriber{
		ch:   make(chan Event, bufferSize),
		done: make(chan struct{}),
	}
}

// Events returns the channel that delivers engine events.
func (s *Subscriber) Events() <-chan Event {
	return s.ch
}

// Done signals that this subscriber is no longer interested in events.
// The engine will stop sending and close the channel.
func (s *Subscriber) Done() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
}

// send attempts a non-blocking send. Returns false if the subscriber's
// buffer is full or it has been closed.
func (s *Subscriber) send(evt Event) bool {
	select {
	case <-s.done:
		return false
	default:
	}

	select {
	case s.ch <- evt:
		return true
	default:
		return false // buffer full, drop event
	}
}

// close closes the event channel. Called by the engine when cleaning up.
func (s *Subscriber) close() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	close(s.ch)
}
