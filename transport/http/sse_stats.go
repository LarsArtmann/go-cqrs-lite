package http

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// ClientStats describes a single connected SSE client.
type ClientStats struct {
	ID             SSEClientID
	BufferedEvents int // events waiting in the client's channel
}

// Stats returns per-client statistics for observability and debugging.
// The slice is empty when no clients are connected.
func (b *SSEBroker) Stats() []ClientStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	stats := make([]ClientStats, 0, len(b.clients))

	for id, ch := range b.clients {
		stats = append(stats, ClientStats{
			ID:             id,
			BufferedEvents: len(ch),
		})
	}

	return stats
}

// CloseWithGrace shuts down the broker, allowing up to grace for in-flight
// events to be consumed by clients before channels are closed. A grace of 0
// is equivalent to Close().
//
// During the grace period, the broker stops accepting new events from the bus
// but existing buffered events remain in client channels for handlers to drain.
// After grace elapses, all client channels are closed and the broker is shut down.
func (b *SSEBroker) CloseWithGrace(grace time.Duration) {
	b.cancel()

	if grace > 0 {
		time.Sleep(grace)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ch := range b.clients {
		close(ch)
	}

	b.clients = make(map[SSEClientID]chan event.Event)
}
