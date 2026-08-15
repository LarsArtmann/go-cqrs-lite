package commandlifecycle_test

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystore "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func newTestCommand(t *testing.T) *command.BasicCommand {
	t.Helper()

	cmd, err := command.New("create_user", id.NewStreamID())
	if err != nil {
		t.Fatalf("failed to create command: %v", err)
	}

	return cmd
}

func newMemoryStore(t *testing.T) *memorystore.MemoryStore {
	t.Helper()

	store := memorystore.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	return store
}

func loadLifecycleEvents(
	t *testing.T,
	store *memorystore.MemoryStore,
	cmd command.Command,
) []event.Event {
	t.Helper()

	ref := commandlifecycle.LifecycleStreamRef(cmd)
	events, err := store.Load(context.Background(), ref)
	if err != nil {
		t.Fatalf("failed to load lifecycle events: %v", err)
	}

	return events
}

func TestLifecycleStreamRef(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmd := newTestCommand(t)
	ref := commandlifecycle.LifecycleStreamRef(cmd)

	g.Expect(ref.Type).To(Equal(commandlifecycle.StreamTypeCommandLifecycle))
	g.Expect(ref.ID.String()).To(Equal(cmd.ID().String()))
}

func TestEventTypeConstants(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(string(commandlifecycle.TypeReceived)).To(Equal("command.received"))
	g.Expect(string(commandlifecycle.TypeFailed)).To(Equal("command.failed"))
	g.Expect(string(commandlifecycle.TypeRetried)).To(Equal("command.retried"))
	g.Expect(string(commandlifecycle.TypeDeadLettered)).To(Equal("command.dead-lettered"))
	g.Expect(string(commandlifecycle.TypeCompleted)).To(Equal("command.completed"))
}

func TestStreamTypeConstant(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(string(commandlifecycle.StreamTypeCommandLifecycle)).To(Equal("CommandLifecycle"))
}

func TestRecorder_RecordReceived(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	err := recorder.RecordReceived(context.Background(), cmd)
	g.Expect(err).NotTo(HaveOccurred())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(1))
	g.Expect(string(events[0].Type())).To(Equal("command.received"))
	g.Expect(events[0].Version()).To(Equal(event.Version(1)))
	g.Expect(events[0].StreamType()).To(Equal(commandlifecycle.StreamTypeCommandLifecycle))
}

func TestRecorder_RecordCompleted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	g.Expect(recorder.RecordReceived(context.Background(), cmd)).To(Succeed())
	g.Expect(recorder.RecordCompleted(context.Background(), cmd)).To(Succeed())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(2))
	g.Expect(string(events[0].Type())).To(Equal("command.received"))
	g.Expect(string(events[1].Type())).To(Equal("command.completed"))
}

func TestRecorder_RecordFailed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)
	testErr := errors.New("transient failure")

	g.Expect(recorder.RecordReceived(context.Background(), cmd)).To(Succeed())
	g.Expect(recorder.RecordFailed(context.Background(), cmd, testErr, 1)).To(Succeed())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(2))
	g.Expect(string(events[1].Type())).To(Equal("command.failed"))
}

func TestRecorder_RecordRetried(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	g.Expect(recorder.RecordReceived(context.Background(), cmd)).To(Succeed())
	g.Expect(recorder.RecordRetried(context.Background(), cmd, 1)).To(Succeed())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(2))
	g.Expect(string(events[1].Type())).To(Equal("command.retried"))
}

func TestRecorder_RecordDeadLettered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)
	testErr := errors.New("permanent failure")

	g.Expect(recorder.RecordReceived(context.Background(), cmd)).To(Succeed())
	g.Expect(recorder.RecordDeadLettered(context.Background(), cmd, testErr, 3)).To(Succeed())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(2))
	g.Expect(string(events[1].Type())).To(Equal("command.dead-lettered"))
}

func TestRecorder_VersionIncrementing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	g.Expect(recorder.RecordReceived(context.Background(), cmd)).To(Succeed())
	g.Expect(recorder.RecordFailed(context.Background(), cmd, errors.New("err"), 1)).To(Succeed())
	g.Expect(recorder.RecordRetried(context.Background(), cmd, 1)).To(Succeed())
	g.Expect(recorder.RecordCompleted(context.Background(), cmd)).To(Succeed())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(4))
	g.Expect(events[0].Version()).To(Equal(event.Version(1)))
	g.Expect(events[1].Version()).To(Equal(event.Version(2)))
	g.Expect(events[2].Version()).To(Equal(event.Version(3)))
	g.Expect(events[3].Version()).To(Equal(event.Version(4)))
}

func TestRecorder_SurvivesRestart_VersionContinuity(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	cmd := newTestCommand(t)

	// First "process" records two events.
	rec1 := commandlifecycle.NewRecorder(store)
	g.Expect(rec1.RecordReceived(context.Background(), cmd)).To(Succeed())
	g.Expect(rec1.RecordFailed(context.Background(), cmd, errors.New("boom"), 1)).To(Succeed())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(2))

	// Simulate restart: a brand-new Recorder with no in-memory cache.
	rec2 := commandlifecycle.NewRecorder(store)
	g.Expect(rec2.RecordRetried(context.Background(), cmd, 1)).To(Succeed())
	g.Expect(rec2.RecordCompleted(context.Background(), cmd)).To(Succeed())

	events = loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(4))
	// Versions continue from where the store left off — no reset to 1.
	g.Expect(events[0].Version()).To(Equal(event.Version(1)))
	g.Expect(events[1].Version()).To(Equal(event.Version(2)))
	g.Expect(events[2].Version()).To(Equal(event.Version(3)))
	g.Expect(events[3].Version()).To(Equal(event.Version(4)))
}

func TestRecorder_CausationLinksToCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	g.Expect(recorder.RecordReceived(context.Background(), cmd)).To(Succeed())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(1))

	md := events[0].Metadata()
	g.Expect(md.Causation).NotTo(BeNil())
	g.Expect(md.Causation.CommandID.Equal(cmd.ID())).To(BeTrue())
	g.Expect(md.Causation.CommandType).To(Equal("create_user"))
}

func TestRecorder_PropagatesCommandTracing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)

	actor := id.NewUserActor(id.NewUserID())
	correlationID := id.NewCorrelationID()
	userID := id.NewUserID()

	cmd, err := command.New(
		"create_user", id.NewStreamID(),
		command.WithActor(actor),
		command.WithCorrelationID(correlationID),
		command.WithUserID(userID),
	)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(recorder.RecordFailed(context.Background(), cmd, errors.New("boom"), 1)).To(Succeed())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(1))

	md := events[0].Metadata()
	g.Expect(md.ActorID.Equal(actor)).To(BeTrue(),
		"lifecycle events must answer 'who triggered the command that failed?'")
	g.Expect(md.ActorID.PrefixedString()).To(HavePrefix("user:"))
	g.Expect(md.CorrelationID).To(Equal(correlationID))
	g.Expect(md.UserID).To(Equal(userID))
}

func TestRecorder_BestEffort_DoesNotFail(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	recorder := commandlifecycle.NewRecorder(closedSink{})
	cmd := newTestCommand(t)

	err := recorder.RecordReceived(context.Background(), cmd)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRecorder_StrictMode_FailsOnSinkError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	recorder := commandlifecycle.NewRecorder(closedSink{}, commandlifecycle.WithStrict())
	cmd := newTestCommand(t)

	err := recorder.RecordReceived(context.Background(), cmd)
	g.Expect(err).To(HaveOccurred())
}

func TestMiddleware_Outer_SuccessEmitsReceivedAndCompleted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	mw := commandlifecycle.Middleware(recorder)
	handler := mw(func(_ context.Context, _ command.Command) error { return nil })

	g.Expect(handler(context.Background(), cmd)).To(Succeed())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(2))
	g.Expect(string(events[0].Type())).To(Equal("command.received"))
	g.Expect(string(events[1].Type())).To(Equal("command.completed"))
}

func TestMiddleware_Outer_FailureEmitsReceivedAndDeadLettered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	mw := commandlifecycle.Middleware(recorder)
	handler := mw(func(_ context.Context, _ command.Command) error { return errors.New("fail") })

	err := handler(context.Background(), cmd)
	g.Expect(err).To(MatchError("fail"))

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(2))
	g.Expect(string(events[0].Type())).To(Equal("command.received"))
	g.Expect(string(events[1].Type())).To(Equal("command.dead-lettered"))
}

func TestNew_OuterAndAttempt_SuccessPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	outer, attempt := commandlifecycle.New(recorder)

	handler := outer(attempt(func(_ context.Context, _ command.Command) error { return nil }))

	g.Expect(handler(context.Background(), cmd)).To(Succeed())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(2))
	g.Expect(string(events[0].Type())).To(Equal("command.received"))
	g.Expect(string(events[1].Type())).To(Equal("command.completed"))
}

func TestNew_OuterAndAttempt_FailurePath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	outer, attempt := commandlifecycle.New(recorder)
	testErr := errors.New("handler failed")

	handler := outer(attempt(func(_ context.Context, _ command.Command) error { return testErr }))

	err := handler(context.Background(), cmd)
	g.Expect(err).To(MatchError(testErr))

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(3))
	g.Expect(string(events[0].Type())).To(Equal("command.received"))
	g.Expect(string(events[1].Type())).To(Equal("command.failed"))
	g.Expect(string(events[2].Type())).To(Equal("command.dead-lettered"))
}

func TestAttemptMiddleware_DetectsRetries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	attempt := commandlifecycle.AttemptMiddleware(recorder)

	callCount := 0
	handler := attempt(func(_ context.Context, _ command.Command) error {
		callCount++
		if callCount < 3 {
			return errors.New("transient")
		}

		return nil
	})

	// Simulate retry: call handler 3 times for same command
	g.Expect(handler(context.Background(), cmd)).To(MatchError("transient"))
	g.Expect(handler(context.Background(), cmd)).To(MatchError("transient"))
	g.Expect(handler(context.Background(), cmd)).To(Succeed())

	events := loadLifecycleEvents(t, store, cmd)
	// Attempt 1: failed (no retried)
	// Attempt 2: retried(1) + failed
	// Attempt 3: retried(2) + success (no event from attempt middleware)
	g.Expect(events).To(HaveLen(4))
	g.Expect(string(events[0].Type())).To(Equal("command.failed"))
	g.Expect(string(events[1].Type())).To(Equal("command.retried"))
	g.Expect(string(events[2].Type())).To(Equal("command.failed"))
	g.Expect(string(events[3].Type())).To(Equal("command.retried"))
}

func TestNew_FullRetryScenario_SucceedsOnThirdAttempt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	outer, attempt := commandlifecycle.New(recorder)

	callCount := 0
	innerHandler := attempt(func(_ context.Context, _ command.Command) error {
		callCount++
		if callCount < 3 {
			return errors.New("transient")
		}

		return nil
	})

	// Simulate retry middleware wrapping the attempt handler.
	maxAttempts := 3
	composed := outer(func(ctx context.Context, c command.Command) error {
		var lastErr error

		for a := 0; a < maxAttempts; a++ {
			lastErr = innerHandler(ctx, c)
			if lastErr == nil {
				return nil
			}
		}

		return lastErr
	})

	g.Expect(composed(context.Background(), cmd)).To(Succeed())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(6))
	g.Expect(string(events[0].Type())).To(Equal("command.received"))
	g.Expect(string(events[1].Type())).To(Equal("command.failed"))
	g.Expect(string(events[2].Type())).To(Equal("command.retried"))
	g.Expect(string(events[3].Type())).To(Equal("command.failed"))
	g.Expect(string(events[4].Type())).To(Equal("command.retried"))
	g.Expect(string(events[5].Type())).To(Equal("command.completed"))
}

func TestNew_FullRetryScenario_ExhaustedAllAttempts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)
	testErr := errors.New("always fails")

	outer, attempt := commandlifecycle.New(recorder)

	innerHandler := attempt(func(_ context.Context, _ command.Command) error {
		return testErr
	})

	maxAttempts := 3
	composed := outer(func(ctx context.Context, c command.Command) error {
		var lastErr error

		for a := 0; a < maxAttempts; a++ {
			lastErr = innerHandler(ctx, c)
			if lastErr == nil {
				return nil
			}
		}

		return lastErr
	})

	err := composed(context.Background(), cmd)
	g.Expect(err).To(MatchError(testErr))

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(7))
	g.Expect(string(events[0].Type())).To(Equal("command.received"))
	g.Expect(string(events[1].Type())).To(Equal("command.failed"))
	g.Expect(string(events[2].Type())).To(Equal("command.retried"))
	g.Expect(string(events[3].Type())).To(Equal("command.failed"))
	g.Expect(string(events[4].Type())).To(Equal("command.retried"))
	g.Expect(string(events[5].Type())).To(Equal("command.failed"))
	g.Expect(string(events[6].Type())).To(Equal("command.dead-lettered"))

	payload, decodeErr := event.DecodePayloadAuto[commandlifecycle.DeadLetteredPayload](
		events[6],
	)
	g.Expect(decodeErr).NotTo(HaveOccurred())
	g.Expect(payload.Attempts).To(Equal(3))
	g.Expect(payload.Error).To(Equal("always fails"))
}

// closedSink is an event.Store that always errors, simulating a broken store.
type closedSink struct{}

func (closedSink) Save(_ context.Context, _ id.StreamRef, _ []event.Event, _ event.Version) error {
	return errors.New("sink closed")
}

func (closedSink) AppendBatch(_ context.Context, _ id.StreamRef, _ []event.Event) error {
	return errors.New("sink closed")
}

func (closedSink) Load(_ context.Context, _ id.StreamRef) ([]event.Event, error) {
	return nil, errors.New("sink closed")
}

func (closedSink) LoadFromVersion(
	_ context.Context,
	_ id.StreamRef,
	_ event.Version,
) ([]event.Event, error) {
	return nil, errors.New("sink closed")
}

func (closedSink) LoadToVersion(
	_ context.Context,
	_ id.StreamRef,
	_ event.Version,
) ([]event.Event, error) {
	return nil, errors.New("sink closed")
}

func (closedSink) LoadToTimestamp(
	_ context.Context,
	_ id.StreamRef,
	_ time.Time,
) ([]event.Event, error) {
	return nil, errors.New("sink closed")
}
