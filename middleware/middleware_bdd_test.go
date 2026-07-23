package middleware_test

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

type bddCommand struct {
	commandID id.CommandID
	streamID  id.StreamID
}

func (c *bddCommand) Type() command.Type    { return "bdd.cmd" }
func (c *bddCommand) StreamID() id.StreamID { return c.streamID }
func (c *bddCommand) ID() id.CommandID      { return c.commandID }

var _ = Describe("Recovery Middleware", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("As a developer protecting my command handlers", func() {
		Context("when my handler panics", func() {
			It("should recover the panic and explain what went wrong", func() {
				mw := middleware.CommandRecovery()
				handler := mw(func(_ context.Context, _ command.Command) error {
					panic("something went terribly wrong")
				})

				err := handler(ctx, &bddCommand{streamID: id.NewStreamID()})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("panic recovered"))
			})
		})

		Context("when my handler succeeds normally", func() {
			It("should let my handler succeed normally without any overhead", func() {
				mw := middleware.CommandRecovery()
				handler := mw(middleware.NoopCommandHandler())

				err := handler(ctx, &bddCommand{streamID: id.NewStreamID()})
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when my handler returns a normal error", func() {
			It("should surface my business error unchanged so I can handle it upstream", func() {
				expectedErr := errors.New("business rule violated")
				mw := middleware.CommandRecovery()
				handler := mw(func(_ context.Context, _ command.Command) error {
					return expectedErr
				})

				err := handler(ctx, &bddCommand{streamID: id.NewStreamID()})
				Expect(err).To(MatchError(expectedErr))
			})
		})
	})
})

var _ = Describe("Retry Middleware", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("As a developer handling transient failures", func() {
		Context("when the handler succeeds on the first attempt", func() {
			It("should not retry", func() {
				var attempts atomic.Int32
				config := middleware.DefaultRetryConfig()
				mw := middleware.CommandRetry(config)
				handler := mw(func(_ context.Context, _ command.Command) error {
					attempts.Add(1)

					return nil
				})

				err := handler(ctx, &bddCommand{streamID: id.NewStreamID()})
				Expect(err).ToNot(HaveOccurred())
				Expect(attempts.Load()).To(Equal(int32(1)))
			})
		})

		Context("when the handler fails with a non-retryable error", func() {
			It("should not retry and return the original error", func() {
				var attempts atomic.Int32
				config := middleware.DefaultRetryConfig()
				mw := middleware.CommandRetry(config)
				handler := mw(func(_ context.Context, _ command.Command) error {
					attempts.Add(1)

					return errorfamily.NewRejection("test.reject", "not retryable")
				})

				err := handler(ctx, &bddCommand{streamID: id.NewStreamID()})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not retryable"))
				Expect(attempts.Load()).To(Equal(int32(1)))
			})
		})

		Context("when the handler fails consistently", func() {
			It("should exhaust retries and report that all attempts failed", func() {
				config := middleware.RetryConfig{
					MaxAttempts:  2,
					InitialDelay: time.Millisecond,
					MaxDelay:     time.Millisecond,
					Multiplier:   1.1,
					IsRetryable:  func(_ error) bool { return true },
				}
				mw := middleware.CommandRetry(config)
				handler := mw(func(_ context.Context, _ command.Command) error {
					return errors.New("transient failure")
				})

				err := handler(ctx, &bddCommand{streamID: id.NewStreamID()})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("all 2 attempts failed"))
			})
		})
	})
})

var _ = Describe("Circuit Breaker Middleware", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("As a developer protecting downstream services", func() {
		Context("when the circuit is closed and handlers succeed", func() {
			It("should allow all requests through", func() {
				config := middleware.DefaultCircuitBreakerConfig()
				mw := middleware.CommandCircuitBreaker(config)
				handler := mw(middleware.NoopCommandHandler())

				for range 10 {
					err := handler(ctx, &bddCommand{streamID: id.NewStreamID()})
					Expect(err).ToNot(HaveOccurred())
				}
			})
		})

		Context("when failures exceed the threshold", func() {
			It("should open the circuit and reject subsequent calls", func() {
				config := middleware.CircuitBreakerConfig{
					FailureThreshold: 2,
					SuccessThreshold: 1,
					Timeout:          time.Hour,
					IsFailure:        func(_ error) bool { return true },
				}
				failHandler := func(_ context.Context, _ command.Command) error {
					return errors.New("service down")
				}
				mw := middleware.CommandCircuitBreaker(config)
				wrapped := mw(failHandler)

				_ = wrapped(ctx, &bddCommand{streamID: id.NewStreamID()})
				_ = wrapped(ctx, &bddCommand{streamID: id.NewStreamID()})

				err := wrapped(ctx, &bddCommand{streamID: id.NewStreamID()})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("circuit breaker open"))
			})
		})
	})
})

var _ = Describe("Event and Query Middleware Variants", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("As a developer using event middleware", func() {
		Context("when I apply recovery to an event handler", func() {
			It("should recover panics in event handlers", func() {
				mw := middleware.EventRecovery()
				handler := mw(func(_ context.Context, _ event.Event) error {
					panic("event handler panic")
				})

				aggID := id.NewStreamID()
				evt, err := event.NewEvent("TestEvent", aggID, "Test", 1, nil)
				Expect(err).ToNot(HaveOccurred())

				err = handler(ctx, evt)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("panic recovered"))
			})
		})
	})

	Describe("As a developer using query middleware", func() {
		Context("when I apply recovery to a query handler", func() {
			It("should recover panics in query handlers", func() {
				mw := middleware.QueryRecovery()
				handler := mw(func(_ context.Context, _ query.Query) (any, error) {
					panic("query handler panic")
				})

				q, err := query.New("test.query")
				Expect(err).ToNot(HaveOccurred())
				result, err := handler(ctx, q)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("panic recovered"))
				Expect(result).To(BeNil())
			})
		})
	})
})

var _ = Describe("Command Idempotency Middleware", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("As a developer protecting against duplicate commands", func() {
		Context("when the same command is dispatched twice", func() {
			It("should execute the handler once and reject the duplicate", func() {
				store := idempotency.NewMemoryStore(0)
				defer store.Close()

				var callCount int32
				mw := middleware.CommandIdempotency(store, time.Minute, nil)
				handler := mw(func(_ context.Context, _ command.Command) error {
					callCount++

					return nil
				})

				cmd := &bddCommand{
					commandID: id.NewCommandID(),
					streamID:  id.NewStreamID(),
				}

				Expect(handler(ctx, cmd)).To(Succeed())
				err := handler(ctx, cmd)
				Expect(err).To(MatchError(idempotency.ErrDuplicate))
				Expect(callCount).To(Equal(int32(1)))
			})
		})

		Context("when two different commands arrive", func() {
			It("should execute the handler for both", func() {
				store := idempotency.NewMemoryStore(0)
				defer store.Close()

				var callCount int32
				mw := middleware.CommandIdempotency(store, time.Minute, nil)
				handler := mw(func(_ context.Context, _ command.Command) error {
					callCount++

					return nil
				})

				Expect(
					handler(
						ctx,
						&bddCommand{streamID: id.NewStreamID(), commandID: id.NewCommandID()},
					),
				).To(Succeed())
				Expect(
					handler(
						ctx,
						&bddCommand{streamID: id.NewStreamID(), commandID: id.NewCommandID()},
					),
				).To(Succeed())
				Expect(callCount).To(Equal(int32(2)))
			})
		})

		Context("when a custom key extractor returns empty", func() {
			It("should skip dedup and always pass through", func() {
				store := idempotency.NewMemoryStore(0)
				defer store.Close()

				var callCount int32
				mw := middleware.CommandIdempotency(
					store,
					time.Minute,
					func(_ command.Command) string { return "" },
				)
				handler := mw(func(_ context.Context, _ command.Command) error {
					callCount++

					return nil
				})

				Expect(
					handler(
						ctx,
						&bddCommand{streamID: id.NewStreamID(), commandID: id.NewCommandID()},
					),
				).To(Succeed())
				Expect(
					handler(
						ctx,
						&bddCommand{streamID: id.NewStreamID(), commandID: id.NewCommandID()},
					),
				).To(Succeed())
				Expect(callCount).To(Equal(int32(2)))
			})
		})
	})
})

var _ = Describe("Query Idempotency Middleware", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("As a developer protecting against duplicate queries", func() {
		Context("when the same query key is seen twice", func() {
			It("should return a result once and reject the duplicate", func() {
				store := idempotency.NewMemoryStore(0)
				defer store.Close()

				var callCount int32
				mw := middleware.QueryIdempotency(
					store,
					time.Minute,
					func(_ query.Query) string { return "q-key-123" },
				)
				handler := mw(func(_ context.Context, _ query.Query) (any, error) {
					callCount++

					return "result-data", nil
				})

				q, err := query.New("test.query")
				Expect(err).ToNot(HaveOccurred())

				result, err := handler(ctx, q)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("result-data"))

				result2, err := handler(ctx, q)
				Expect(err).To(MatchError(idempotency.ErrDuplicate))
				Expect(result2).To(BeNil())
				Expect(callCount).To(Equal(int32(1)))
			})
		})

		Context("when the nil keyExtractor is provided", func() {
			It("should panic at construction with a clear message", func() {
				store := idempotency.NewMemoryStore(0)
				defer store.Close()

				Expect(func() {
					middleware.QueryIdempotency(store, time.Minute, nil)
				}).To(PanicWith(MatchRegexp("keyExtractor must not be nil")))
			})
		})
	})
})

var _ = Describe("Event Idempotency Middleware", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("As a developer protecting against duplicate events", func() {
		Context("when the same event is seen twice", func() {
			It("should process the handler once and reject the duplicate", func() {
				store := idempotency.NewMemoryStore(0)
				defer store.Close()

				var callCount int32
				mw := middleware.EventIdempotency(store, time.Minute, nil)
				handler := mw(func(_ context.Context, evt event.Event) error {
					atomic.AddInt32(&callCount, 1)

					return nil
				})

				aggID := id.NewStreamID()
				evt, err := event.NewEvent("test.event", aggID, "Test", 1, nil)
				Expect(err).To(Succeed())

				Expect(handler(ctx, evt)).To(Succeed())
				err = handler(ctx, evt)
				Expect(err).To(MatchError(idempotency.ErrDuplicate))
				Expect(callCount).To(Equal(int32(1)))
			})
		})

		Context("when two different events arrive", func() {
			It("should process both events", func() {
				store := idempotency.NewMemoryStore(0)
				defer store.Close()

				var callCount int32
				mw := middleware.EventIdempotency(store, time.Minute, nil)
				handler := mw(func(_ context.Context, evt event.Event) error {
					atomic.AddInt32(&callCount, 1)

					return nil
				})

				aggID := id.NewStreamID()
				evt1, err := event.NewEvent("test.event", aggID, "Test", 1, nil)
				Expect(err).To(Succeed())
				evt2, err := event.NewEvent("test.event2", aggID, "Test", 1, nil)
				Expect(err).To(Succeed())

				Expect(handler(ctx, evt1)).To(Succeed())
				Expect(handler(ctx, evt2)).To(Succeed())
				Expect(callCount).To(Equal(int32(2)))
			})
		})

		Context("when a custom key extractor returns empty", func() {
			It("should skip dedup and process all events", func() {
				store := idempotency.NewMemoryStore(0)
				defer store.Close()

				var callCount int32
				mw := middleware.EventIdempotency(store, time.Minute, func(_ event.Event) string {
					return ""
				})
				handler := mw(func(_ context.Context, evt event.Event) error {
					atomic.AddInt32(&callCount, 1)

					return nil
				})

				aggID := id.NewStreamID()
				evt, err := event.NewEvent("test.event", aggID, "Test", 1, nil)
				Expect(err).To(Succeed())

				Expect(handler(ctx, evt)).To(Succeed())
				Expect(handler(ctx, evt)).To(Succeed())
				Expect(callCount).To(Equal(int32(2)))
			})
		})
	})
})
