package query_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
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
				Expect(
					registerHandler(dispatcher, "query.user.name", "Alice"),
				).ToNot(HaveOccurred())

				result, err := dispatcher.Dispatch(ctx, &getUserName{})
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("Alice"))
			})
		})

		Context("when I use DispatchTyped with the correct result type", func() {
			It("should return the strongly typed result", func() {
				Expect(registerHandler(dispatcher, "query.active.count", 42)).ToNot(HaveOccurred())

				result, err := query.DispatchTyped[int](ctx, dispatcher, &getActiveCount{})
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(42))
			})
		})

		Context("when I use DispatchTyped with the wrong result type", func() {
			It("should return a type mismatch error", func() {
				Expect(registerHandler(dispatcher, "query.active.count", 42)).ToNot(HaveOccurred())

				_, err := query.DispatchTyped[string](ctx, dispatcher, &getActiveCount{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unexpected result type"))
			})
		})

		Context("when I dispatch an unregistered query", func() {
			It("should return a handler not found error", func() {
				_, err := dispatcher.Dispatch(ctx, &getUserName{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no handler registered"))
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
					queryMiddleware(&callOrder, "log"),
				)

				Expect(
					registerCallOrderHandler(dispatcher, "query.user.name", &callOrder, "Bob"),
				).ToNot(HaveOccurred())

				result, err := dispatcher.Dispatch(ctx, &getUserName{})
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("Bob"))
				Expect(callOrder).To(Equal([]string{"log", "handler"}))
			})
		})
	})
})
