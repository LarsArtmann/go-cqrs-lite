package middleware_test

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/middleware/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

func mustNewQuery(queryType query.Type) *query.BasicQuery {
	q, err := query.New(queryType)
	if err != nil {
		panic(err)
	}
	return q
}


type bddCommand struct {
	aggregateID id.AggregateID
}

func (c *bddCommand) Type() command.Type          { return "bdd.cmd" }
func (c *bddCommand) AggregateID() id.AggregateID { return c.aggregateID }

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

				err := handler(ctx, &bddCommand{aggregateID: id.NewAggregateID()})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("panic recovered"))
			})
		})

		Context("when my handler succeeds normally", func() {
			It("should let my handler succeed normally without any overhead", func() {
				mw := middleware.CommandRecovery()
				handler := mw(middleware.NoopCommandHandler())

				err := handler(ctx, &bddCommand{aggregateID: id.NewAggregateID()})
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

				err := handler(ctx, &bddCommand{aggregateID: id.NewAggregateID()})
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

				err := handler(ctx, &bddCommand{aggregateID: id.NewAggregateID()})
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

					return event.NewRejection("test.reject", "not retryable")
				})

				err := handler(ctx, &bddCommand{aggregateID: id.NewAggregateID()})
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

				err := handler(ctx, &bddCommand{aggregateID: id.NewAggregateID()})
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
					err := handler(ctx, &bddCommand{aggregateID: id.NewAggregateID()})
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

				_ = wrapped(ctx, &bddCommand{aggregateID: id.NewAggregateID()})
				_ = wrapped(ctx, &bddCommand{aggregateID: id.NewAggregateID()})

				err := wrapped(ctx, &bddCommand{aggregateID: id.NewAggregateID()})
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

				aggID := id.NewAggregateID()
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

				q := mustNewQuery("test.query")
				result, err := handler(ctx, q)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("panic recovered"))
				Expect(result).To(BeNil())
			})
		})
	})
})
