package query_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestQueryBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Query BDD Suite")
}

type getUserName struct{}

func (q *getUserName) Type() query.Type { return "query.user.name" }

type getActiveCount struct{}

func (q *getActiveCount) Type() query.Type { return "query.active.count" }

var _ = Describe("Query Dispatcher", func() {
	var (
		ctx        context.Context
		dispatcher *query.Dispatcher
	)

	BeforeEach(func() {
		ctx = context.Background()
		dispatcher = query.NewDispatcher()
	})

	Describe("As a developer building read-side queries", func() {
		Context("when I register a handler and dispatch the matching query", func() {
			It("should return the typed result", func() {
				registerHandler(dispatcher, "query.user.name", "Alice")

				result, err := dispatcher.Dispatch(ctx, &getUserName{})
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("Alice"))
			})
		})

		Context("when I use DispatchTyped with the correct result type", func() {
			It("should return the strongly typed result", func() {
				registerHandler(dispatcher, "query.active.count", 42)

				result, err := query.DispatchTyped[int](ctx, dispatcher, &getActiveCount{})
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(42))
			})
		})

		Context("when I use DispatchTyped with the wrong result type", func() {
			It("should return a type mismatch error", func() {
				registerHandler(dispatcher, "query.active.count", 42)

				_, err := query.DispatchTyped[string](ctx, dispatcher, &getActiveCount{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unexpected result type"))
			})
		})

		Context("when I dispatch an unregistered query", func() {
			It("should return a query not supported error", func() {
				_, err := dispatcher.Dispatch(ctx, &getUserName{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not supported"))
			})
		})

		Context("when I close the dispatcher", func() {
			It("should reject further dispatch and registration", func() {
				Expect(dispatcher.Close()).To(Succeed())

				err := dispatcher.Register(
					"query.user.name",
					func(_ context.Context, _ query.Query) (any, error) {
						return "", nil
					},
				)
				Expect(err).To(HaveOccurred())

				_, err = dispatcher.Dispatch(ctx, &getUserName{})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when I apply query middleware", func() {
			It("should wrap the handler in order", func() {
				var callOrder []string

				dispatcher.Use(
					func(next func(context.Context, query.Query) (any, error)) func(context.Context, query.Query) (any, error) {
						return func(_ context.Context, q query.Query) (any, error) {
							callOrder = append(callOrder, "log")

							return next(context.Background(), q)
						}
					},
				)

				registerCallOrderHandler(dispatcher, "query.user.name", &callOrder, "Bob")

				result, err := dispatcher.Dispatch(ctx, &getUserName{})
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("Bob"))
				Expect(callOrder).To(Equal([]string{"log", "handler"}))
			})
		})
	})
})
