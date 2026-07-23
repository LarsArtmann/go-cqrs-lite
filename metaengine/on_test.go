package metaengine_test

import (
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("On constructor", func() {
	// Local domain types for On classification tests.
	type event struct {
		ID   string
		Name string
		Prev string
	}
	type result struct{ Name string }

	var sample event

	BeforeEach(func() {
		sample = event{ID: "e1", Name: "hello"}
	})

	DescribeTable(
		"classifying handler signatures into fold kinds",
		func(handler any, expectedKind metaengine.FoldKind) {
			fold := metaengine.On(sample, handler)
			Expect(fold.Kind).To(Equal(expectedKind))
		},
		Entry("insert: func(e) (K, V)",
			func(e event) (string, result) { return e.ID, result{Name: e.Name} },
			metaengine.FoldInsert),
		Entry("update: func(e, prev V) V",
			func(e event, prev result) result {
				prev.Name = e.Name
				return prev
			},
			metaengine.FoldUpdate),
		Entry("set: func(e) K",
			func(e event) string { return e.ID },
			metaengine.FoldSet),
		Entry("count: func(e) Delta",
			func(e event) metaengine.Delta { return metaengine.Delta{"count": +1} },
			metaengine.FoldCount),
		Entry("edge: func(e) Edge",
			func(e event) metaengine.Edge { return metaengine.Edge{From: e.ID, To: e.Name} },
			metaengine.FoldEdge),
		Entry("remove: Remove[V]()",
			metaengine.Remove[result](),
			metaengine.FoldRemove),
		Entry("skip: func(e) Skip",
			func(e event) metaengine.Skip { return metaengine.Skip{} },
			metaengine.FoldSkip),
	)

	Describe("recording the event type", func() {
		It("uses the Go struct name as the event type", func() {
			fold := metaengine.On(sample, func(e event) (string, result) {
				return e.ID, result{Name: e.Name}
			})
			Expect(fold.EventType).To(Equal("event"))
		})

		It("unwraps pointer samples to get the struct name", func() {
			fold := metaengine.On(&sample, func(e event) (string, result) {
				return e.ID, result{Name: e.Name}
			})
			Expect(fold.EventType).To(Equal("event"))
		})
	})

	When("the handler is not a function", func() {
		It("panics with a clear message", func() {
			Expect(func() {
				metaengine.On(sample, 42)
			}).To(PanicWith(MatchRegexp("handler must be a function")))
		})
	})

	When("the handler has the wrong first parameter type", func() {
		It("panics with a clear message naming the expected and actual types", func() {
			Expect(func() {
				metaengine.On(sample, func(e string) (string, result) {
					return e, result{}
				})
			}).To(PanicWith(MatchRegexp("handler first param must be")))
		})
	})

	When("the handler has too many parameters", func() {
		It("panics", func() {
			Expect(func() {
				metaengine.On(sample, func(e event, x string, y int) result {
					return result{}
				})
			}).To(PanicWith(MatchRegexp("handler must have 1-2 params")))
		})
	})
})

var _ = Describe("Remove sentinel", func() {
	type value struct{ Data string }

	It("records the value type for projection matching", func() {
		fold := metaengine.On(struct{ ID string }{}, metaengine.Remove[value]())
		Expect(fold.Kind).To(Equal(metaengine.FoldRemove))
	})

	It("works with different value types in the same query", func() {
		type other struct{ Count int }
		fold := metaengine.On(struct{ ID string }{}, metaengine.Remove[other]())
		Expect(fold.Kind).To(Equal(metaengine.FoldRemove))
	})
})

var _ = Describe("Skip sentinel", func() {
	It("is classified as FoldSkip", func() {
		fold := metaengine.On(
			struct{ ID string }{},
			func(e struct{ ID string }) metaengine.Skip { return metaengine.Skip{} },
		)
		Expect(fold.Kind).To(Equal(metaengine.FoldSkip))
	})
})

var _ = Describe("EventTypeName", func() {
	It("extracts the struct name", func() {
		type MyEvent struct{ X int }
		Expect(metaengine.EventTypeName(MyEvent{})).To(Equal("MyEvent"))
	})

	It("unwraps pointers", func() {
		type MyEvent struct{ X int }
		Expect(metaengine.EventTypeName(&MyEvent{})).To(Equal("MyEvent"))
	})
})
