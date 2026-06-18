package event_test

import (
	"context"
	"errors"
	"testing"

	ro "github.com/samber/ro"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

type captureSubscriber struct {
	handler event.Handler
	err     error
}

func (s *captureSubscriber) Subscribe(event.Type, event.Handler) error { return nil }
func (s *captureSubscriber) SubscribeAll(h event.Handler) error {
	s.handler = h

	return s.err
}

func TestSubscriberToObservable_ForwardsEvents(t *testing.T) {
	t.Parallel()

	sub := &captureSubscriber{}
	obs := event.SubscriberToObservable(sub)

	mu, received := subscribeAndCollect(obs)

	evt1 := newTestEvent(t, "user.created")
	evt2 := newTestEvent(t, "user.updated")

	_ = sub.handler(context.Background(), evt1)
	_ = sub.handler(context.Background(), evt2)

	mu.Lock()
	defer mu.Unlock()

	if len(*received) != 2 {
		t.Fatalf("expected 2 events forwarded, got %d", len(*received))
	}

	assertEventType(t, (*received)[0], "user.created")
	assertEventType(t, (*received)[1], "user.updated")
}

func TestSubscriberToObservable_SubscribeError(t *testing.T) {
	t.Parallel()

	sub := &captureSubscriber{err: errors.New("subscribe all failed")}
	obs := event.SubscriberToObservable(sub)

	var capturedErr error

	observer := ro.NewObserverWithContext(
		func(_ context.Context, _ event.Event) {},
		func(_ context.Context, err error) { capturedErr = err },
		func(_ context.Context) {},
	)

	obs.Subscribe(observer)

	if capturedErr == nil {
		t.Fatal("expected error from SubscribeAll to be forwarded to observer")
	}
}

func TestDistinctByEventIDWith_SeedsFromPriorPhase(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	evt1 := newTestEvent(t, "user.created")
	evt2 := newTestEvent(t, "user.updated")
	evt3 := newTestEvent(t, "user.deleted")

	seen := map[id.EventID]struct{}{
		evt1.ID(): {},
		evt2.ID(): {},
	}

	deduped := ro.Pipe1(bus, event.DistinctByEventIDWith(seen))

	mu, received := subscribeAndCollect(deduped)

	bus.Next(evt1) // seeded → suppressed
	bus.Next(evt2) // seeded → suppressed
	bus.Next(evt3) // new → passes
	bus.Next(evt3) // in-stream duplicate → suppressed
	bus.Complete()

	mu.Lock()
	defer mu.Unlock()

	if len(*received) != 1 {
		t.Fatalf("expected 1 event (only evt3), got %d", len(*received))
	}

	if (*received)[0].ID() != evt3.ID() {
		t.Errorf("expected evt3 to pass through, got different event")
	}
}

func TestDistinctByEventIDWith_NilSeed(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()
	deduped := ro.Pipe1(bus, event.DistinctByEventIDWith(nil))

	mu, received := subscribeAndCollect(deduped)

	evt1 := newTestEvent(t, "user.created")
	evt2 := newTestEvent(t, "user.updated")

	bus.Next(evt1)
	bus.Next(evt2)
	bus.Next(evt1) // duplicate
	bus.Complete()

	mu.Lock()
	defer mu.Unlock()

	if len(*received) != 2 {
		t.Fatalf("nil seed: expected 2 deduped events, got %d", len(*received))
	}
}
