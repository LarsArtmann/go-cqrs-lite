package commandlifecycle_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	. "github.com/onsi/gomega"
)

func TestAttemptMiddleware_Standalone_ClearsAfterSuccess(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	attempt := commandlifecycle.AttemptMiddleware(recorder)

	handler := attempt(func(_ context.Context, cmd command.Command) error {
		return nil
	})

	ctx := context.Background()
	cmd := newTestCommand(t)

	g.Expect(handler(ctx, cmd)).To(Succeed())
	g.Expect(handler(ctx, cmd)).To(Succeed())

	// Standalone attempt middleware records nothing on success, so the
	// lifecycle stream must not exist. Before the clear-on-success fix, the
	// second dispatch saw attemptNum=2 and emitted a spurious retried event,
	// creating the stream.
	_, err := store.Load(ctx, commandlifecycle.LifecycleStreamRef(cmd))
	g.Expect(err).To(MatchError(event.ErrStreamNotFound))
}

func TestAttemptMiddleware_Standalone_FailureThenRetryIncrements(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	attempt := commandlifecycle.AttemptMiddleware(recorder)

	fail := true
	handler := attempt(func(_ context.Context, cmd command.Command) error {
		if fail {
			return errors.New("boom")
		}

		return nil
	})

	ctx := context.Background()
	cmd := newTestCommand(t)

	g.Expect(handler(ctx, cmd)).To(MatchError("boom"))

	fail = false
	g.Expect(handler(ctx, cmd)).To(Succeed())

	events := loadLifecycleEvents(t, store, cmd)

	retried := 0
	for _, evt := range events {
		if evt.Type() == commandlifecycle.TypeRetried {
			retried++
		}
	}

	g.Expect(retried).To(Equal(1),
		"second dispatch after a failure must emit exactly one retried event")
}

func TestMiddleware_SharedTracker_StillReportsAccurateAttempts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	outer, attempt := commandlifecycle.New(recorder)

	inner := func(_ context.Context, _ command.Command) error {
		return errors.New("always fails")
	}

	pipeline := outer(attempt(inner))

	ctx := context.Background()
	cmd, err := command.New("create_user", id.NewStreamID())
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(pipeline(ctx, cmd)).To(MatchError("always fails"))

	events := loadLifecycleEvents(t, store, cmd)

	var deadLettered event.Event
	for _, evt := range events {
		if evt.Type() == commandlifecycle.TypeDeadLettered {
			deadLettered = evt
		}
	}

	g.Expect(deadLettered).NotTo(BeNil())

	payload, decodeErr := event.DecodePayloadAuto[commandlifecycle.DeadLetteredPayload](
		deadLettered,
	)
	g.Expect(decodeErr).NotTo(HaveOccurred())
	g.Expect(payload.Attempts).To(Equal(1),
		"single dispatch must report attempts=1")
}
