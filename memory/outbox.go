package memory

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher"
)

// MemoryOutboxStore is an in-memory implementation of event.Outbox for testing.
type MemoryOutboxStore struct {
	dispatcher.Lifecycle

	mu           sync.RWMutex
	entries      []outboxEntry
	entryCounter int
}

var _ event.Outbox = (*MemoryOutboxStore)(nil)

// NewMemoryOutboxStore creates a new in-memory outbox store.
func NewMemoryOutboxStore() *MemoryOutboxStore {
	return &MemoryOutboxStore{
		Lifecycle:    dispatcher.Lifecycle{},
		mu:           sync.RWMutex{},
		entries:      make([]outboxEntry, 0),
		entryCounter: 0,
	}
}

type outboxEntry struct {
	id        event.OutboxID
	events    []event.Event
	createdAt time.Time
}

// Append writes events to the outbox.
func (o *MemoryOutboxStore) Append(_ context.Context, events []event.Event) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.entryCounter++

	entry := outboxEntry{
		id:        event.NewOutboxID(fmt.Sprintf("outbox-%d", o.entryCounter)),
		events:    slices.Clone(events),
		createdAt: time.Now(),
	}

	o.entries = append(o.entries, entry)

	return nil
}

// PollPending returns unacknowledged outbox entries, oldest first.
func (o *MemoryOutboxStore) PollPending(_ context.Context, limit int) ([]event.OutboxEntry, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make([]event.OutboxEntry, 0, limit)

	for _, entry := range o.entries {
		result = append(result, event.OutboxEntry{
			ID:        entry.id,
			Events:    slices.Clone(entry.events),
			CreatedAt: entry.createdAt,
		})

		if len(result) >= limit {
			break
		}
	}

	return result, nil
}

// Ack marks entries as successfully published.
func (o *MemoryOutboxStore) Ack(_ context.Context, ids []event.OutboxID) error {
	if len(ids) == 0 {
		return nil
	}

	idSet := make(map[event.OutboxID]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	o.entries = slices.DeleteFunc(o.entries, func(e outboxEntry) bool {
		_, ok := idSet[e.id]

		return ok
	})

	return nil
}

// Close marks the store as closed.
func (o *MemoryOutboxStore) Close() error {
	return o.Lifecycle.Close() //nolint:wrapcheck
}
