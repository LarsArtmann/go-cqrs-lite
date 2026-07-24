package metaengine_test

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Apply and Execute", func() {
	var store *metaengine.Store
	var ctx context.Context

	BeforeEach(func() {
		var err error
		store, err = metaengine.Plan(
			[]metaengine.Engine{metaengine.NewMemoryEngine()},
			allQueries()...,
		)
		Expect(err).NotTo(HaveOccurred())
		ctx = context.Background()
	})

	AfterEach(func() {
		Expect(store.Close()).To(Succeed())
	})

	Describe("Map ADT: FindTask", func() {
		BeforeEach(func() {
			Expect(store.Apply(context.Background(), "TaskCreated", TaskCreated{
				ID: "t1", Title: "Write tests", Assignee: "alice",
				Status: "open", Priority: 3, At: time.Now(),
			})).To(Succeed())
		})

		It("returns the stored task by ID", func() {
			result, err := metaengine.ExecuteTyped[FindTask, FindTaskResult](
				ctx,
				store,
				FindTask{ID: "t1"},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Title).To(Equal("Write tests"))
			Expect(result.Status).To(Equal("open"))
		})

		When("the task is completed via FoldUpdate", func() {
			BeforeEach(func() {
				Expect(
					store.Apply(
						context.Background(),
						"TaskCompleted",
						TaskCompleted{ID: "t1", At: time.Now()},
					),
				).To(Succeed())
			})

			It("updates the status to completed", func() {
				result, err := metaengine.ExecuteTyped[FindTask, FindTaskResult](
					ctx,
					store,
					FindTask{ID: "t1"},
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Status).To(Equal("completed"))
			})
		})

		When("the task is deleted via Remove sentinel", func() {
			BeforeEach(func() {
				Expect(
					store.Apply(
						context.Background(),
						"TaskDeleted",
						TaskDeleted{ID: "t1", At: time.Now()},
					),
				).To(Succeed())
			})

			It("returns a zero-value result", func() {
				result, err := metaengine.ExecuteTyped[FindTask, FindTaskResult](
					ctx,
					store,
					FindTask{ID: "t1"},
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Title).To(BeEmpty())
			})
		})

		It("returns a zero-value result for an unknown ID", func() {
			result, err := metaengine.ExecuteTyped[FindTask, FindTaskResult](
				ctx,
				store,
				FindTask{ID: "nonexistent"},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Title).To(BeEmpty())
		})
	})

	Describe("Set ADT: CheckAssignee", func() {
		BeforeEach(func() {
			Expect(store.Apply(context.Background(), "TaskAssigned", TaskAssigned{
				TaskID: "t1", Assignee: "alice", At: time.Now(),
			})).To(Succeed())
		})

		It("reports true for a known assignee", func() {
			taken, err := metaengine.ExecuteTyped[CheckAssignee, bool](
				ctx,
				store,
				CheckAssignee{User: "alice"},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(taken).To(BeTrue())
		})

		It("reports false for an unknown assignee", func() {
			taken, err := metaengine.ExecuteTyped[CheckAssignee, bool](
				ctx,
				store,
				CheckAssignee{User: "nobody"},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(taken).To(BeFalse())
		})
	})

	Describe("Counter ADT: CountByStatus", func() {
		BeforeEach(func() {
			Expect(store.Apply(context.Background(), "TaskCreated", TaskCreated{
				ID: "t1", Title: "A", Status: "open", At: time.Now(),
			})).To(Succeed())
			Expect(store.Apply(context.Background(), "TaskCreated", TaskCreated{
				ID: "t2", Title: "B", Status: "open", At: time.Now(),
			})).To(Succeed())
			Expect(store.Apply(context.Background(), "TaskCreated", TaskCreated{
				ID: "t3", Title: "C", Status: "open", At: time.Now(),
			})).To(Succeed())
			Expect(
				store.Apply(
					context.Background(),
					"TaskCompleted",
					TaskCompleted{ID: "t1", At: time.Now()},
				),
			).To(Succeed())
		})

		It("counts open and completed tasks correctly", func() {
			counts, err := metaengine.ExecuteTyped[CountByStatus, map[string]int64](
				ctx,
				store,
				CountByStatus{},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(counts["open"]).To(Equal(int64(2)))
			Expect(counts["completed"]).To(Equal(int64(1)))
		})
	})

	Describe("Graph ADT: TasksByAssignee", func() {
		BeforeEach(func() {
			Expect(store.Apply(context.Background(), "TaskAssigned", TaskAssigned{
				TaskID: "t1", Assignee: "alice", At: time.Now(),
			})).To(Succeed())
			Expect(store.Apply(context.Background(), "TaskAssigned", TaskAssigned{
				TaskID: "t2", Assignee: "alice", At: time.Now(),
			})).To(Succeed())
		})

		It("returns all task IDs for an assignee at depth 1", func() {
			result, err := metaengine.ExecuteTyped[TasksByAssignee, TasksByAssigneeResult](
				ctx, store, TasksByAssignee{User: "alice", Depth: 1},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.TaskIDs).To(HaveLen(2))
			Expect(result.TaskIDs).To(ContainElement(TaskID("t1")))
			Expect(result.TaskIDs).To(ContainElement(TaskID("t2")))
		})
	})

	Describe("SortedMap ADT: ListTasksByStatus", func() {
		BeforeEach(func() {
			Expect(store.Apply(context.Background(), "TaskCreated", TaskCreated{
				ID: "t1", Title: "Low", Status: "open", Priority: 5, At: time.Now(),
			})).To(Succeed())
			Expect(store.Apply(context.Background(), "TaskCreated", TaskCreated{
				ID: "t2", Title: "High", Status: "open", Priority: 1, At: time.Now(),
			})).To(Succeed())
			Expect(store.Apply(context.Background(), "TaskCreated", TaskCreated{
				ID: "t3", Title: "Done", Status: "completed", Priority: 3, At: time.Now(),
			})).To(Succeed())
		})

		It("filters by status and sorts by priority", func() {
			result, err := metaengine.ExecuteTyped[ListTasksByStatus, ListTasksByStatusResult](
				ctx, store, ListTasksByStatus{Status: "open", Limit: 10},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Tasks).To(HaveLen(2))
			Expect(
				result.Tasks[0].ID,
			).To(Equal(TaskID("t2")), "highest priority (1) should come first")
			Expect(
				result.Tasks[1].ID,
			).To(Equal(TaskID("t1")), "lower priority (5) should come second")
		})

		It("excludes tasks not matching the filter", func() {
			result, _ := metaengine.ExecuteTyped[ListTasksByStatus, ListTasksByStatusResult](
				ctx, store, ListTasksByStatus{Status: "completed", Limit: 10},
			)
			Expect(result.Tasks).To(HaveLen(1))
			Expect(result.Tasks[0].ID).To(Equal(TaskID("t3")))
		})
	})
})

var _ = Describe("Concurrent FoldUpdate atomicity", func() {
	type evt struct {
		ID     string
		Amount int
	}
	type val struct {
		ID    string
		Total int
	}
	type input struct{ ID string }

	It("preserves all increments under concurrent access", func() {
		q := metaengine.Query[input, val](
			"counters",
			metaengine.On(evt{}, func(e evt) (string, val) {
				return e.ID, val{ID: e.ID, Total: e.Amount}
			}),
			metaengine.On(evt{}, func(e evt, prev val) val {
				prev.Total += e.Amount
				return prev
			}),
		)

		store, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, q)
		Expect(err).NotTo(HaveOccurred())
		defer store.Close()

		done := make(chan struct{}, 100)
		for range 100 {
			go func() {
				defer func() { done <- struct{}{} }()
				_ = store.Apply(context.Background(), "evt", evt{ID: "c1", Amount: 1})
			}()
		}
		for range 100 {
			<-done
		}

		result, err := metaengine.ExecuteTyped[input, val](
			context.Background(),
			store,
			input{ID: "c1"},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Total).To(Equal(100))
	})
})

var _ = Describe("ApplyEncoded", func() {
	var store *metaengine.Store

	BeforeEach(func() {
		var err error
		store, err = metaengine.Plan(
			[]metaengine.Engine{metaengine.NewMemoryEngine()},
			findTaskQuery(),
		)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(store.Close()).To(Succeed())
	})

	It("decodes JSON payloads and applies them", func() {
		payload := `{"ID":"t9","Title":"From JSON","Assignee":"bob","Status":"open","Priority":2,"At":"2026-01-01T00:00:00Z"}`
		Expect(
			store.ApplyEncoded(context.Background(), "TaskCreated", []byte(payload)),
		).To(Succeed())

		result, err := metaengine.ExecuteTyped[FindTask, FindTaskResult](
			context.Background(), store, FindTask{ID: "t9"},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Title).To(Equal("From JSON"))
	})
})

var _ = Describe("EventTypeNames", func() {
	It("returns all event type names the store reacts to, sorted", func() {
		store, err := metaengine.Plan(
			[]metaengine.Engine{metaengine.NewMemoryEngine()},
			findTaskQuery(),
		)
		Expect(err).NotTo(HaveOccurred())
		defer store.Close()

		names := store.EventTypeNames()
		Expect(names).To(Equal([]string{"TaskCompleted", "TaskCreated", "TaskDeleted"}))
	})
})

var _ = Describe("Execute error handling", func() {
	var store *metaengine.Store

	BeforeEach(func() {
		var err error
		store, err = metaengine.Plan(
			[]metaengine.Engine{metaengine.NewMemoryEngine()},
			findTaskQuery(),
		)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(store.Close()).To(Succeed())
	})

	When("the input type is not declared in any query", func() {
		It("returns an error naming the unknown type", func() {
			type Unknown struct{ X int }
			_, err := store.Execute(Unknown{X: 42})
			Expect(err).To(MatchError(MatchRegexp("no query declared for input type")))
		})
	})

	When("the context is cancelled", func() {
		It("returns the context error", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			_, err := store.ExecuteCtx(ctx, FindTask{ID: "x"})
			Expect(err).To(MatchError(context.Canceled))
		})
	})
})
