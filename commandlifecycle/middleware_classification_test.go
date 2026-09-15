package commandlifecycle_test

import (
	"context"
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
)

// defaultRetryConfig keeps the errorfamily-based IsRetryable predicate so
// rejection-family errors are never retried.
func defaultRetryConfig() middleware.RetryConfig {
	config := middleware.DefaultRetryConfig()
	config.MaxAttempts = 3

	return config
}

func eventTypes(events []event.Event) []string {
	types := make([]string, 0, len(events))
	for _, evt := range events {
		types = append(types, string(evt.Type()))
	}

	return types
}

func TestIntegration_RejectionWithoutRetryMiddleware(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	outer, attempt := commandlifecycle.New(recorder)
	handler := attempt(func(_ context.Context, _ command.Command) error {
		return errorfamily.NewRejection("INSUFFICIENT_FUNDS", "balance too low")
	})

	err := outer(handler)(context.Background(), cmd)
	g.Expect(err).To(HaveOccurred())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(eventTypes(events)).To(Equal([]string{"command.received", "command.rejected"}))

	payload, err := event.DecodePayloadAuto[commandlifecycle.RejectedPayload](events[1])
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(payload.Family).To(Equal("rejection"))
	g.Expect(payload.ErrorCode).To(Equal("INSUFFICIENT_FUNDS"))
	g.Expect(payload.Attempt).To(Equal(1))
}

func TestIntegration_RejectionIsNeverRetriedOrDeadLettered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	outer, attempt := commandlifecycle.New(recorder)
	retryMW := middleware.CommandRetry(defaultRetryConfig())

	calls := 0
	handler := attempt(func(_ context.Context, _ command.Command) error {
		calls++

		return errorfamily.NewRejection("INVALID_STATE", "cannot close paid invoice")
	})

	err := outer(retryMW(handler))(context.Background(), cmd)
	g.Expect(err).To(HaveOccurred())
	g.Expect(calls).To(Equal(1))

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(eventTypes(events)).To(Equal([]string{"command.received", "command.rejected"}))
}

func TestIntegration_ConflictIsARejection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	outer, attempt := commandlifecycle.New(recorder)
	handler := attempt(func(_ context.Context, _ command.Command) error {
		return errorfamily.NewConflict("VERSION_MISMATCH", "concurrent update")
	})

	g.Expect(outer(handler)(context.Background(), cmd)).To(HaveOccurred())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(eventTypes(events)).To(Equal([]string{"command.received", "command.rejected"}))

	payload, err := event.DecodePayloadAuto[commandlifecycle.RejectedPayload](events[1])
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(payload.Family).To(Equal("conflict"))
}

func TestIntegration_NonRejectionStillFailsAndDeadLetters(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	outer, attempt := commandlifecycle.New(recorder)
	handler := attempt(func(_ context.Context, _ command.Command) error {
		return errors.New("boom")
	})

	g.Expect(outer(handler)(context.Background(), cmd)).To(HaveOccurred())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(eventTypes(events)).To(Equal([]string{
		"command.received", "command.failed", "command.dead-lettered",
	}))

	failed, err := event.DecodePayloadAuto[commandlifecycle.FailedPayload](events[1])
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(failed.CommandID).To(Equal(commandlifecycle.CommandKey(cmd.ID().String())))
}

func TestStandaloneOuter_RejectionEmitsRejectedNotDeadLetter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	handler := commandlifecycle.Middleware(
		recorder,
	)(
		func(_ context.Context, _ command.Command) error {
			return errorfamily.NewRejection("FORBIDDEN", "actor lacks permission")
		},
	)

	g.Expect(handler(context.Background(), cmd)).To(HaveOccurred())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(eventTypes(events)).To(Equal([]string{"command.received", "command.rejected"}))
}

func TestStandaloneAttempt_RejectionCarriesAttempt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	handler := commandlifecycle.AttemptMiddleware(
		recorder,
	)(
		func(_ context.Context, _ command.Command) error {
			return errorfamily.NewRejection("QUOTA_EXCEEDED", "daily quota used")
		},
	)

	g.Expect(handler(context.Background(), cmd)).To(HaveOccurred())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(eventTypes(events)).To(Equal([]string{"command.rejected"}))

	payload, err := event.DecodePayloadAuto[commandlifecycle.RejectedPayload](events[0])
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(payload.Attempt).To(Equal(1))
}
