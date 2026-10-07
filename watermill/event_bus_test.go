package watermill_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

func TestEventBusPublishSubscribe(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewEventBus()
	defer bus.Close()

	var received atomic.Int32

	err := bus.Subscribe("user.created", func(_ context.Context, _ event.Event) error {
		received.Add(1)

		return nil
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	streamID := id.NewStreamID()
	evt, err := event.NewEvent("user.created", streamID, "User", event.Version(1),
		[]byte(`{"name":"alice"}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = bus.Publish(context.Background(), evt)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	waitFor(t, func() bool { return received.Load() > 0 }, 2*time.Second)
	if received.Load() != 1 {
		t.Fatalf("expected 1 event received, got %d", received.Load())
	}
}

func TestEventBusSubscribeAll(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewEventBus()
	defer bus.Close()

	var received atomic.Int32

	err := bus.SubscribeAll(func(_ context.Context, _ event.Event) error {
		received.Add(1)

		return nil
	})
	if err != nil {
		t.Fatalf("subscribeAll: %v", err)
	}

	streamID := id.NewStreamID()
	for _, et := range []event.Type{"a.b", "c.d"} {
		evt, _ := event.NewEvent(et, streamID, "T", event.Version(1), nil)
		_ = bus.Publish(context.Background(), evt)
	}

	waitFor(t, func() bool { return received.Load() >= 2 }, 2*time.Second)
	if received.Load() != 2 {
		t.Fatalf("expected 2 events, got %d", received.Load())
	}
}

func TestEventBusPublishEmpty(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewEventBus()
	defer bus.Close()

	err := bus.Publish(context.Background())
	if err != nil {
		t.Fatalf("publish empty: %v", err)
	}
}

func TestEventBusCloseIdempotent(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewEventBus()

	if err := bus.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}

	if err := bus.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestEventBusPublishAfterClose(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewEventBus()
	_ = bus.Close()

	streamID := id.NewStreamID()
	evt, _ := event.NewEvent("x.y", streamID, "T", event.Version(1), nil)
	err := bus.Publish(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error publishing after close")
	}
}

func TestEventBusMiddleware(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewEventBus()
	defer bus.Close()

	var order []string
	var mu sync.Mutex

	record := func(s string) event.Middleware {
		return func(next event.Handler) event.Handler {
			return func(ctx context.Context, evt event.Event) error {
				mu.Lock()
				order = append(order, s)
				mu.Unlock()

				return next(ctx, evt)
			}
		}
	}

	_ = bus.Use(record("outer"), record("inner"))

	var received atomic.Int32
	_ = bus.Subscribe("test.event", func(_ context.Context, _ event.Event) error {
		received.Add(1)

		return nil
	})

	streamID := id.NewStreamID()
	evt, _ := event.NewEvent("test.event", streamID, "T", event.Version(1), nil)
	_ = bus.Publish(context.Background(), evt)

	waitFor(t, func() bool { return received.Load() > 0 }, 2*time.Second)

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 || order[0] != "outer" || order[1] != "inner" {
		t.Fatalf("middleware order wrong: %v", order)
	}
}

func waitFor(t *testing.T, cond func() bool, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestEventBusPublishRacingCloseNeverLeaksRawTransportError is a contract
// pin for the publish/close drain; the leak itself was proven at the consumer
// level (nsfw-classifier's rooms stress repro, 2026-10-07). Concurrent
// Publish calls hammering the bus while Close runs must never observe the
// backend's RAW closed error (watermill's "Pub/Sub closed") — every publish
// either succeeds or fails with the typed event.ErrBusClosed the dispatch
// tolerance in consumers matches on. This single-shot shape pins the
// contract; it cannot deterministically reproduce the guard straddle.
func TestEventBusPublishRacingCloseNeverLeaksRawTransportError(t *testing.T) {
	t.Parallel()

	for i := range 300 {
		bus := cqrswatermill.NewEventBus()

		err := bus.Subscribe("race.evt", func(_ context.Context, _ event.Event) error {
			return nil
		})
		if err != nil {
			t.Fatalf("iteration %d: subscribe: %v", i, err)
		}

		evt, err := event.NewEvent("race.evt", id.NewStreamID(), "Race", event.Version(1),
			[]byte(`{"n":1}`))
		if err != nil {
			t.Fatalf("iteration %d: NewEvent: %v", i, err)
		}

		var wg sync.WaitGroup
		rawErr := make(chan error, 1)
		stop := make(chan struct{})

		for range 6 {
			wg.Add(1)

			go func() {
				defer wg.Done()

				for {
					select {
					case <-stop:
						return
					default:
					}

					if pubErr := bus.Publish(context.Background(), evt); pubErr != nil {
						if !errors.Is(pubErr, event.ErrBusClosed) {
							select {
							case rawErr <- pubErr:
							default:
							}
						}

						return
					}
				}
			}()
		}

		time.Sleep(200 * time.Microsecond)
		closeErr := bus.Close()
		close(stop)
		wg.Wait()

		if closeErr != nil {
			t.Fatalf("iteration %d: close: %v", i, closeErr)
		}

		select {
		case err := <-rawErr:
			t.Fatalf("iteration %d: publish leaked a raw transport error (want nil or typed ErrBusClosed): %v", i, err)
		default:
		}
	}
}
