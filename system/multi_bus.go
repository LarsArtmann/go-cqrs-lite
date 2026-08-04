package system

import (
	"context"
	"fmt"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// MultiBus fans out event publishing to multiple [event.Publisher] instances.
// This implements D9 (multi-bus support): events from a source-of-truth instance
// can be published to multiple buses simultaneously (e.g., GoChannel for local
// projections + NATS for cross-service distribution).
//
// Publish is synchronous: all publishers receive the events before Publish
// returns. If any publisher fails, the first error is returned (remaining
// publishers may or may not have received the events).
type MultiBus struct {
	mu         sync.RWMutex
	publishers []event.Publisher
}

// NewMultiBus creates a MultiBus from the given publishers.
func NewMultiBus(publishers ...event.Publisher) *MultiBus {
	return &MultiBus{publishers: publishers}
}

// AddPublisher appends a publisher to the fan-out list.
func (m *MultiBus) AddPublisher(p event.Publisher) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.publishers = append(m.publishers, p)
}

// Publish sends events to all registered publishers sequentially.
// Returns the first error encountered.
func (m *MultiBus) Publish(ctx context.Context, events ...event.Event) error {
	m.mu.RLock()
	publishers := make([]event.Publisher, len(m.publishers))
	copy(publishers, m.publishers)
	m.mu.RUnlock()

	for i, pub := range publishers {
		if err := pub.Publish(ctx, events...); err != nil {
			return fmt.Errorf("system: multi-bus publisher %d: %w", i, err)
		}
	}

	return nil
}

// Compile-time assertion.
var _ event.Publisher = (*MultiBus)(nil)
