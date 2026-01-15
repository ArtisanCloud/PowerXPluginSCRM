package stream

import (
	"sync"
	"time"
)

// Event represents a message emitted to MCP stream subscribers.
type Event struct {
	SessionID string      `json:"session_id"`
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// Broker fan-outs events to subscribers keyed by session id.
type Broker struct {
	mu   sync.RWMutex
	subs map[string]map[chan Event]struct{}
}

// NewBroker constructs a broker instance.
func NewBroker() *Broker {
	return &Broker{subs: make(map[string]map[chan Event]struct{})}
}

var (
	defaultBroker *Broker
	once          sync.Once
)

// DefaultBroker returns a process-wide broker instance.
func DefaultBroker() *Broker {
	once.Do(func() {
		defaultBroker = NewBroker()
	})
	return defaultBroker
}

// Subscribe registers a stream for the given session and returns a channel and cleanup function.
func (b *Broker) Subscribe(sessionID string) (<-chan Event, func()) {
	if b == nil {
		ch := make(chan Event)
		close(ch)
		return ch, func() {}
	}

	ch := make(chan Event, 16)
	b.mu.Lock()
	if b.subs[sessionID] == nil {
		b.subs[sessionID] = make(map[chan Event]struct{})
	}
	b.subs[sessionID][ch] = struct{}{}
	b.mu.Unlock()

	cleanup := func() {
		b.mu.Lock()
		if subs, ok := b.subs[sessionID]; ok {
			if _, exists := subs[ch]; exists {
				delete(subs, ch)
				close(ch)
			}
			if len(subs) == 0 {
				delete(b.subs, sessionID)
			}
		}
		b.mu.Unlock()
	}

	return ch, cleanup
}

// Publish delivers an event to all subscribers of the session.
func (b *Broker) Publish(event Event) {
	if b == nil || event.SessionID == "" {
		return
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	b.mu.RLock()
	subs := b.subs[event.SessionID]
	for ch := range subs {
		select {
		case ch <- event:
		default:
			// drop if subscriber is slow
		}
	}
	b.mu.RUnlock()
}
