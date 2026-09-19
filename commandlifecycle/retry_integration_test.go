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
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
)

func fastRetryConfig(maxAttempts int) middleware.RetryConfig {
	config := middleware.DefaultRetryConfig()
	config.MaxAttempts = maxAttempts
	config.InitialDelay = time.Millisecond
	config.IsRetryable = func(error) bool { return true }

	return config
}

func TestIntegration_RealRetryMiddleware_SucceedsOnThirdAttempt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	outer, attempt := commandlifecycle.New(recorder)

	callCount := 0
	handler := attempt(func(_ context.Context, _ command.Command) error {
		callCount++
		if callCount < 3 {
			return errors.New("transient")
		}

		return nil
	})

	retryMW := middleware.CommandRetry(fastRetryConfig(3))
	composed := outer(retryMW(handler))

	g.Expect(composed(context.Background(), cmd)).To(Succeed())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(6))
	g.Expect(string(events[0].Type())).To(Equal("command.received"))
	g.Expect(string(events[1].Type())).To(Equal("command.failed"))
	g.Expect(string(events[2].Type())).To(Equal("command.retried"))
	g.Expect(string(events[3].Type())).To(Equal("command.failed"))
	g.Expect(string(events[4].Type())).To(Equal("command.retried"))
	g.Expect(string(events[5].Type())).To(Equal("command.completed"))

	failedPayload1, err := event.DecodePayloadAuto[commandlifecycle.FailedPayload](events[1])
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(failedPayload1.Attempt).To(Equal(1))

	failedPayload2, err := event.DecodePayloadAuto[commandlifecycle.FailedPayload](events[3])
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(failedPayload2.Attempt).To(Equal(2))
}

func TestIntegration_RealRetryMiddleware_ExhaustedAllAttempts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	outer, attempt := commandlifecycle.New(recorder)

	handler := attempt(func(_ context.Context, _ command.Command) error {
		return errors.New("always fails")
	})

	retryMW := middleware.CommandRetry(fastRetryConfig(3))
	composed := outer(retryMW(handler))

	err := composed(context.Background(), cmd)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("always fails"))

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(7))
	g.Expect(string(events[0].Type())).To(Equal("command.received"))
	g.Expect(string(events[1].Type())).To(Equal("command.failed"))
	g.Expect(string(events[2].Type())).To(Equal("command.retried"))
	g.Expect(string(events[3].Type())).To(Equal("command.failed"))
	g.Expect(string(events[4].Type())).To(Equal("command.retried"))
	g.Expect(string(events[5].Type())).To(Equal("command.failed"))
	g.Expect(string(events[6].Type())).To(Equal("command.dead-lettered"))

	dlPayload, err := event.DecodePayloadAuto[commandlifecycle.DeadLetteredPayload](events[6])
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(dlPayload.Attempts).To(Equal(3))
	g.Expect(dlPayload.Error).To(ContainSubstring("always fails"))
}

func TestIntegration_RealRetryMiddleware_SucceedsFirstTry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := newMemoryStore(t)
	recorder := commandlifecycle.NewRecorder(store)
	cmd := newTestCommand(t)

	outer, attempt := commandlifecycle.New(recorder)

	handler := attempt(func(_ context.Context, _ command.Command) error {
		return nil
	})

	retryMW := middleware.CommandRetry(fastRetryConfig(3))
	composed := outer(retryMW(handler))

	g.Expect(composed(context.Background(), cmd)).To(Succeed())

	events := loadLifecycleEvents(t, store, cmd)
	g.Expect(events).To(HaveLen(2))
	g.Expect(string(events[0].Type())).To(Equal("command.received"))
	g.Expect(string(events[1].Type())).To(Equal("command.completed"))
}
