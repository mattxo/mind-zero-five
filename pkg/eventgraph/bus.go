package eventgraph

import (
	"context"
	"sync"
)

// Bus wraps an EventStore with in-process fan-out notification.
// When Append is called, all subscribers receive the new event.
type Bus struct {
	EventStore
	mu   sync.RWMutex
	subs map[chan *Event]struct{}
}

// NewBus creates a Bus wrapping the given store.
func NewBus(store EventStore) *Bus {
	return &Bus{
		EventStore: store,
		subs:       make(map[chan *Event]struct{}),
	}
}

// Append delegates to the underlying store, then fans out to all subscribers.
func (b *Bus) Append(ctx context.Context, eventType, source string, content map[string]any, causes []string, conversationID string) (*Event, error) {
	e, err := b.EventStore.Append(ctx, eventType, source, content, causes, conversationID)
	if err != nil {
		return nil, err
	}

	b.mu.RLock()
	for ch := range b.subs {
		select {
		case ch <- e:
		default:
			// subscriber is behind; drop to avoid blocking Append
		}
	}
	b.mu.RUnlock()

	return e, nil
}

// Subscribe returns a buffered channel that receives all new events.
func (b *Bus) Subscribe() chan *Event {
	ch := make(chan *Event, 64)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber and closes its channel.
func (b *Bus) Unsubscribe(ch chan *Event) {
	b.mu.Lock()
	delete(b.subs, ch)
	b.mu.Unlock()
	close(ch)
}
