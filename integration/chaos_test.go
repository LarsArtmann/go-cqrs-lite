package integration_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/middleware"
)

type chaosCmd struct {
	aggregateID id.AggregateID
	fail        bool
	panicMsg    string
}

func (c *chaosCmd) Type() command.Type          { return "chaos.command" }
func (c *chaosCmd) AggregateID() id.AggregateID { return c.aggregateID }

func TestChaos_CommandHandler_Error(t *testing.T) {
	t.Parallel()

	disp := command.NewDispatcher()
	disp.Use(middleware.CommandRecovery())

	handlerErr := errors.New("chaos: handler failed")
	disp.Register("chaos.command", func(_ context.Context, cmd command.Command) error {
		return handlerErr
	})

	err := disp.Dispatch(context.Background(), &chaosCmd{aggregateID: id.NewAggregateID()})
	require.ErrorIs(t, err, handlerErr)
}

func TestChaos_CommandHandler_Panic_Recovered(t *testing.T) {
	t.Parallel()

	disp := command.NewDispatcher()
	disp.Use(middleware.CommandRecovery())

	disp.Register("chaos.command", func(_ context.Context, cmd command.Command) error {
		panic("chaos: unexpected panic")
	})

	err := disp.Dispatch(context.Background(), &chaosCmd{aggregateID: id.NewAggregateID()})
	require.Error(t, err)
	require.Contains(t, err.Error(), "panic recovered")
}

func TestChaos_CommandHandler_Panic_NoRecovery(t *testing.T) {
	defer func() {
		r := recover()
		require.NotNil(t, r, "expected panic to propagate")
		require.Contains(t, fmt.Sprintf("%v", r), "chaos: propagated")
	}()

	disp := command.NewDispatcher()

	disp.Register("chaos.command", func(_ context.Context, cmd command.Command) error {
		panic("chaos: propagated")
	})

	_ = disp.Dispatch(context.Background(), &chaosCmd{aggregateID: id.NewAggregateID()})
}

func newRetryDispatcher(maxAttempts int, attempts *int, failUntil int, permanent bool) *command.Dispatcher {
	disp := command.NewDispatcher()
	disp.Use(middleware.CommandRetry(middleware.RetryConfig{
		MaxAttempts:  maxAttempts,
		InitialDelay: time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   1.5,
		IsRetryable:  func(err error) bool { return true },
	}))

	*attempts = 0
	disp.Register("chaos.command", func(_ context.Context, cmd command.Command) error {
		*attempts++
		if permanent {
			return errors.New("chaos: permanent failure")
		}
		if *attempts < failUntil {
			return errors.New("chaos: transient failure")
		}
		return nil
	})
	return disp
}

func TestChaos_CommandRetry_SucceedsAfterFailures(t *testing.T) {
	t.Parallel()

	var attempts int
	disp := newRetryDispatcher(5, &attempts, 3, false)

	err := disp.Dispatch(context.Background(), &chaosCmd{aggregateID: id.NewAggregateID()})
	require.NoError(t, err)
	require.Equal(t, 3, attempts)
}

func TestChaos_EventHandler_Panic_RecoveryMiddleware(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	t.Cleanup(func() { _ = bus.Close() })

	err := bus.Use(middleware.EventRecovery())
	require.NoError(t, err)

	received := 0
	err = bus.Subscribe("chaos.event", func(_ context.Context, evt event.Event) error {
		received++
		if received == 1 {
			panic("chaos: first event panics")
		}

		return nil
	})
	require.NoError(t, err)

	aggID := id.NewAggregateID()
	evt1, err := event.NewEvent("chaos.event", aggID, "Chaos", event.Version(1), nil)
	require.NoError(t, err)

	_ = bus.Publish(context.Background(), evt1)
}

func TestChaos_Context_Cancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	disp := command.NewDispatcher()
	disp.Register("chaos.command", func(ctx context.Context, cmd command.Command) error {
		return ctx.Err()
	})

	err := disp.Dispatch(ctx, &chaosCmd{aggregateID: id.NewAggregateID()})
	require.ErrorIs(t, err, context.Canceled)
}

func TestChaos_CommandRetry_ExhaustsAllAttempts(t *testing.T) {
	t.Parallel()

	var attempts int
	disp := newRetryDispatcher(3, &attempts, 0, true)

	err := disp.Dispatch(context.Background(), &chaosCmd{aggregateID: id.NewAggregateID()})
	require.Error(t, err)
	require.Equal(t, 3, attempts)
}

func TestChaos_EventPublish_Fails(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	t.Cleanup(func() { _ = bus.Close() })

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent("chaos.publish", aggID, "Chaos", event.Version(1), nil)
	require.NoError(t, err)

	_ = bus.Publish(context.Background(), evt)
}
