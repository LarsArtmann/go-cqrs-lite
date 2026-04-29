package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// MemoryOutboxStore is an in-memory implementation of event.Outbox for testing.
type MemoryOutboxStore struct {
	mu      sync.RWMutex
	entries []outboxEntry
	nextID  int
}

var _ event.Outbox = (*MemoryOutboxStore)(nil)

// NewMemoryOutboxStore creates a new in-memory outbox store.
func NewMemoryOutboxStore() *MemoryOutboxStore {
	return &MemoryOutboxStore{
		mu:      sync.RWMutex{},
		entries: make([]outboxEntry, 0),
		nextID:  0,
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

	o.nextID++

	entry := outboxEntry{
		id:        event.OutboxID(fmt.Sprintf("outbox-%d", o.nextID)),
		events:    append([]event.Event(nil), events...),
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
			ID:     entry.id,
			Events: append([]event.Event(nil), entry.events...),
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

	filtered := make([]outboxEntry, 0, len(o.entries))

	for _, entry := range o.entries {
		if _, ok := idSet[entry.id]; !ok {
			filtered = append(filtered, entry)
		}
	}

	o.entries = filtered

	return nil
}
