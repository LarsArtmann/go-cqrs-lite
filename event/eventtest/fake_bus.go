package eventtest

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

type FakeBus struct {
	mu             sync.Mutex
	Published      []event.Event
	PublishErr     error
	subscribeAllFn func(event.Handler) error
}

func NewFakeBus() *FakeBus {
	return &FakeBus{}
}

func (b *FakeBus) Publish(_ context.Context, events ...event.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.PublishErr != nil {
		return b.PublishErr
	}

	b.Published = append(b.Published, events...)

	return nil
}

func (b *FakeBus) Subscribe(_ event.Type, _ event.Handler) error { return nil }

func (b *FakeBus) SubscribeAllFn(fn func(event.Handler) error) *FakeBus {
	b.subscribeAllFn = fn

	return b
}

func (b *FakeBus) SubscribeAll(handler event.Handler) error {
	if b.subscribeAllFn != nil {
		return b.subscribeAllFn(handler)
	}

	return nil
}

func (b *FakeBus) Use(_ ...event.Middleware) error               { return nil }
func (b *FakeBus) UsePublish(_ ...event.PublishMiddleware) error { return nil }
func (b *FakeBus) Close() error                                  { return nil }

var _ event.Bus = (*FakeBus)(nil)
