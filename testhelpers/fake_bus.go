package testhelpers

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// FakeBus implements event.Bus for testing.
type FakeBus struct {
	mu         sync.Mutex
	Published  []event.Event
	PublishErr error
}

// NewFakeBus creates a FakeBus with no published events.
func NewFakeBus() *FakeBus {
	return &FakeBus{}
}

// Publish appends events or returns PublishErr if set.
func (b *FakeBus) Publish(_ context.Context, events ...event.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.PublishErr != nil {
		return b.PublishErr
	}

	b.Published = append(b.Published, events...)

	return nil
}

// Subscribe is a no-op for testing.
func (b *FakeBus) Subscribe(_ event.Type, _ event.Handler) error { return nil }

// SubscribeAll is a no-op for testing.
func (b *FakeBus) SubscribeAll(_ event.Handler) error { return nil }

// Close is a no-op for testing.
func (b *FakeBus) Close() error { return nil }

var _ event.Bus = (*FakeBus)(nil)
