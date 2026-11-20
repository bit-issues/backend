package events

import (
	"sync"

	"go.uber.org/zap"
)

// subscriberBuffer is the per-subscriber channel capacity. A consumer that
// falls this far behind has events dropped for it rather than blocking
// publishers.
const subscriberBuffer = 64

// Hub is an in-memory fan-out bus broadcasting events to all current
// subscribers. It is safe for concurrent use and is designed for a single
// application instance.
type Hub struct {
	mu     sync.RWMutex
	subs   map[chan Event]struct{}
	closed bool

	logger *zap.Logger
}

// NewHub creates a Hub with the given logger.
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		mu:     sync.RWMutex{},
		subs:   make(map[chan Event]struct{}),
		closed: false,

		logger: logger,
	}
}

// Subscribe registers a new subscriber and returns its event channel and a
// cancel function. The cancel function is idempotent; after it returns (or
// after Close), the channel is closed. A subscriber registered on a closed
// hub receives an already-closed channel.
func (h *Hub) Subscribe() (<-chan Event, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan Event, subscriberBuffer)
	if h.closed {
		close(ch)
		return ch, func() {}
	}

	h.subs[ch] = struct{}{}
	return ch, func() { h.unsubscribe(ch) }
}

// Publish broadcasts an event to all current subscribers without blocking;
// events that do not fit into a subscriber's buffer are dropped for that
// subscriber.
func (h *Hub) Publish(e Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.closed {
		return
	}

	for ch := range h.subs {
		select {
		case ch <- e:
		default:
			h.logger.Debug("event dropped, subscriber buffer full",
				zap.String("type", e.Type),
				zap.Int64("task_id", e.TaskID),
			)
		}
	}
}

// Close stops the hub and closes all subscriber channels. Further Publish
// calls are no-ops. Safe to call multiple times.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return
	}
	h.closed = true

	for ch := range h.subs {
		close(ch)
		delete(h.subs, ch)
	}
}

func (h *Hub) unsubscribe(ch chan Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.subs[ch]; ok {
		delete(h.subs, ch)
		close(ch)
	}
}
