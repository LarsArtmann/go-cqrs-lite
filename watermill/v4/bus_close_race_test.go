package watermill

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// subscribeAgainstClose runs Subscribe in a loop while another goroutine
// closes the bus, so the closed guard is read concurrently with Close's
// write (the race detector flags an unsynchronized read).
func subscribeAgainstClose(t *testing.T, subscribe func() error, closeBus func() error) {
	t.Helper()

	var wg sync.WaitGroup

	for range 4 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range 50 {
				_ = subscribe()
			}
		}()
	}

	if err := closeBus(); err != nil {
		t.Errorf("Close(): %v", err)
	}

	wg.Wait()
}

func TestCommandBus_SubscribeConcurrentWithClose(t *testing.T) {
	t.Parallel()

	bus := NewCommandBus()
	defer func() { _ = bus.Close() }()

	subscribeAgainstClose(t,
		func() error {
			return bus.Subscribe("test.cmd", func(context.Context, command.Command) error { return nil })
		},
		bus.Close,
	)
}

func TestEventBus_SubscribeConcurrentWithClose(t *testing.T) {
	t.Parallel()

	bus := NewEventBus()
	defer func() { _ = bus.Close() }()

	subscribeAgainstClose(t,
		func() error {
			return bus.Subscribe("test.evt", func(context.Context, event.Event) error { return nil })
		},
		bus.Close,
	)
}

func TestCommandBus_SubscribeAllConcurrentWithClose(t *testing.T) {
	t.Parallel()

	bus := NewCommandBus()
	defer func() { _ = bus.Close() }()

	subscribeAgainstClose(t,
		func() error {
			return bus.SubscribeAll(func(context.Context, command.Command) error { return nil })
		},
		bus.Close,
	)
}

func TestEventBus_SubscribeAllConcurrentWithClose(t *testing.T) {
	t.Parallel()

	bus := NewEventBus()
	defer func() { _ = bus.Close() }()

	subscribeAgainstClose(t,
		func() error {
			return bus.SubscribeAll(func(context.Context, event.Event) error { return nil })
		},
		bus.Close,
	)
}
