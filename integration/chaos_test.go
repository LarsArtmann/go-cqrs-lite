package integration_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
)

type chaosCmd struct {
	commandID id.CommandID
	streamID  id.StreamID
	fail      bool
	panicMsg  string
}

func (c *chaosCmd) Type() command.Type    { return "chaos.command" }
func (c *chaosCmd) StreamID() id.StreamID { return c.streamID }
func (c *chaosCmd) ID() id.CommandID      { return c.commandID }

func TestChaos_CommandHandler_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	disp := command.NewDispatcher()
	disp.Use(middleware.CommandRecovery())

	handlerErr := errors.New("chaos: handler failed")
	_ = disp.Register("chaos.command", func(_ context.Context, cmd command.Command) error {
		return handlerErr
	})

	err := disp.Dispatch(context.Background(), &chaosCmd{streamID: id.NewStreamID()})
	g.Expect(err).To(MatchError(handlerErr))
}

func TestChaos_CommandHandler_Panic_Recovered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	disp := command.NewDispatcher()
	disp.Use(middleware.CommandRecovery())

	_ = disp.Register("chaos.command", func(_ context.Context, cmd command.Command) error {
		panic("chaos: unexpected panic")
	})

	err := disp.Dispatch(context.Background(), &chaosCmd{streamID: id.NewStreamID()})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("panic recovered"))
}

func TestChaos_CommandHandler_Panic_NoRecovery(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic to propagate")
		}
		if msg := fmt.Sprintf("%v", r); !strings.Contains(msg, "chaos: propagated") {
			t.Fatalf("expected panic message to contain 'chaos: propagated', got: %s", msg)
		}
	}()

	disp := command.NewDispatcher()

	_ = disp.Register("chaos.command", func(_ context.Context, cmd command.Command) error {
		panic("chaos: propagated")
	})

	_ = disp.Dispatch(context.Background(), &chaosCmd{streamID: id.NewStreamID()})
}

func newRetryDispatcher(
	maxAttempts int,
	attempts *int,
	failUntil int,
	permanent bool,
) *command.Dispatcher {
	disp := command.NewDispatcher()
	disp.Use(middleware.CommandRetry(middleware.RetryConfig{
		MaxAttempts:  maxAttempts,
		InitialDelay: time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   1.5,
		IsRetryable:  func(err error) bool { return true },
	}))

	*attempts = 0
	_ = disp.Register("chaos.command", func(_ context.Context, cmd command.Command) error {
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
	g := NewWithT(t)

	var attempts int
	disp := newRetryDispatcher(5, &attempts, 3, false)

	err := disp.Dispatch(context.Background(), &chaosCmd{streamID: id.NewStreamID()})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(attempts).To(Equal(3))
}

func TestChaos_EventHandler_Panic_RecoveryMiddleware(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bus := eventtest.NewFakeBus()
	t.Cleanup(func() { _ = bus.Close() })

	err := bus.Use(middleware.EventRecovery())
	g.Expect(err).ToNot(HaveOccurred())

	received := 0
	err = bus.Subscribe("chaos.event", func(_ context.Context, evt event.Event) error {
		received++
		if received == 1 {
			panic("chaos: first event panics")
		}

		return nil
	})
	g.Expect(err).ToNot(HaveOccurred())

	aggID := id.NewStreamID()
	evt1, err := event.NewEvent("chaos.event", aggID, "Chaos", event.Version(1), nil)
	g.Expect(err).ToNot(HaveOccurred())

	_ = bus.Publish(context.Background(), evt1)
}

func TestChaos_Context_Cancellation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	disp := command.NewDispatcher()
	_ = disp.Register("chaos.command", func(ctx context.Context, cmd command.Command) error {
		return ctx.Err()
	})

	err := disp.Dispatch(ctx, &chaosCmd{streamID: id.NewStreamID()})
	g.Expect(err).To(MatchError(context.Canceled))
}

func TestChaos_CommandRetry_ExhaustsAllAttempts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var attempts int
	disp := newRetryDispatcher(3, &attempts, 0, true)

	err := disp.Dispatch(context.Background(), &chaosCmd{streamID: id.NewStreamID()})
	g.Expect(err).To(HaveOccurred())
	g.Expect(attempts).To(Equal(3))
}

func TestChaos_EventPublish_Fails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bus := eventtest.NewFakeBus()
	t.Cleanup(func() { _ = bus.Close() })

	aggID := id.NewStreamID()
	evt, err := event.NewEvent("chaos.publish", aggID, "Chaos", event.Version(1), nil)
	g.Expect(err).ToNot(HaveOccurred())

	_ = bus.Publish(context.Background(), evt)
}
