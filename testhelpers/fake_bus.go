package testhelpers

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// FakeBus implements event.Bus for testing.
type FakeBus struct {
	mu             sync.Mutex
	Published      []event.Event
	PublishErr     error
	subscribeAllFn func(event.Handler) error
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

// SubscribeAllFn sets an optional override for SubscribeAll calls.
func (b *FakeBus) SubscribeAllFn(fn func(event.Handler) error) *FakeBus {
	b.subscribeAllFn = fn

	return b
}

// SubscribeAll is a no-op for testing.
func (b *FakeBus) SubscribeAll(handler event.Handler) error {
	if b.subscribeAllFn != nil {
		return b.subscribeAllFn(handler)
	}

	return nil
}

// Use is a no-op for testing.
func (b *FakeBus) Use(_ ...event.Middleware) error { return nil }

// UsePublish is a no-op for testing.
func (b *FakeBus) UsePublish(_ ...event.PublishMiddleware) error { return nil }

// Close is a no-op for testing.
func (b *FakeBus) Close() error { return nil }

var _ event.Bus = (*FakeBus)(nil)
