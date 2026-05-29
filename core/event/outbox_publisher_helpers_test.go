package event

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type stubOutbox struct {
	mu      sync.Mutex
	entries []OutboxEntry
	acked   []OutboxID
	pollErr error
	ackErr  error
}

func (o *stubOutbox) Append(_ context.Context, _ []Event) error { return nil }

func (o *stubOutbox) PollPending(ctx context.Context, limit int) ([]OutboxEntry, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if o.pollErr != nil {
		return nil, o.pollErr
	}

	if len(o.entries) > limit {
		return o.entries[:limit], nil
	}

	return o.entries, nil
}

func (o *stubOutbox) Ack(_ context.Context, ids []OutboxID) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.ackErr != nil {
		return o.ackErr
	}

	o.acked = append(o.acked, ids...)

	return nil
}

func (o *stubOutbox) Close() error { return nil }

type stubPublisher struct {
	mu           sync.Mutex
	published    []Event
	publishErr   error
	publishErrFn func() error
}

func (b *stubPublisher) Publish(_ context.Context, events ...Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.publishErrFn != nil {
		err := b.publishErrFn()
		if err != nil {
			return err
		}
	}

	if b.publishErr != nil {
		return b.publishErr
	}

	b.published = append(b.published, events...)

	return nil
}

func (b *stubPublisher) Subscribe(_ Type, _ Handler) error { return nil }
func (b *stubPublisher) SubscribeAll(_ Handler) error      { return nil }
func (b *stubPublisher) Use(_ ...Middleware) error         { return nil }
func (b *stubPublisher) Close() error                      { return nil }

func mustNewTestEvent(eventType Type) Event {
	evt, err := NewEvent(
		eventType,
		id.MustParseAggregateID("01HGW5FPJPYK5RE8ACZDesWMY2"),
		"TestAggregate",
		1,
		[]byte(`{"key":"value"}`),
	)
	if err != nil {
		panic("mustNewTestEvent: " + err.Error())
	}

	return evt
}
