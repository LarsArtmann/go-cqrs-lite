package testhelpers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// FakeOutbox implements event.Outbox for testing.
type FakeOutbox struct {
	mu       sync.RWMutex
	Entries  []event.OutboxEntry
	nextID   int
	appendFn func(events []event.Event) error
}

// NewFakeOutbox creates a FakeOutbox with no entries.
func NewFakeOutbox() *FakeOutbox {
	return &FakeOutbox{}
}

// AppendFn sets an optional override for Append calls.
func (o *FakeOutbox) AppendFn(fn func(events []event.Event) error) *FakeOutbox {
	o.appendFn = fn

	return o
}

// Append writes events to the outbox.
func (o *FakeOutbox) Append(_ context.Context, events []event.Event) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.appendFn != nil {
		return o.appendFn(events)
	}

	o.Entries = append(o.Entries, event.OutboxEntry{
		ID:        event.NewOutboxID(fmt.Sprintf("outbox-%d", o.nextID)),
		Events:    events,
		CreatedAt: time.Now(),
	})
	o.nextID++

	return nil
}

// PollPending returns all entries.
func (o *FakeOutbox) PollPending(_ context.Context, _ int) ([]event.OutboxEntry, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.Entries, nil
}

// Ack removes entries matching the given IDs.
func (o *FakeOutbox) Ack(_ context.Context, ids []event.OutboxID) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	ackSet := make(map[event.OutboxID]struct{}, len(ids))
	for _, id := range ids {
		ackSet[id] = struct{}{}
	}

	var remaining []event.OutboxEntry

	for _, entry := range o.Entries {
		if _, ok := ackSet[entry.ID]; !ok {
			remaining = append(remaining, entry)
		}
	}

	o.Entries = remaining

	return nil
}

// Close is a no-op for testing.
func (o *FakeOutbox) Close() error { return nil }

var _ event.Outbox = (*FakeOutbox)(nil)
