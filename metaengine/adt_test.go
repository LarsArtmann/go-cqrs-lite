package metaengine_test

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// ─── MULTIMAP ADT ───

type TasksForUser struct {
	User UserID
}

type TasksForUserResult struct {
	TaskIDs []TaskID
}

func tasksForUserQuery() metaengine.QueryDecl[TasksForUser, TasksForUserResult] {
	return metaengine.Query[TasksForUser, TasksForUserResult](
		"tasks_for_user",
		metaengine.On(TaskAssigned{}, func(e TaskAssigned) metaengine.MultiEntry {
			return metaengine.MultiEntry{Key: e.Assignee, Value: e.TaskID}
		}),
	)
}

var _ = Describe("Multimap ADT", func() {
	var store *metaengine.Store

	BeforeEach(func() {
		var err error
		store, err = metaengine.Plan(
			[]metaengine.Engine{metaengine.NewMemoryEngine()},
			tasksForUserQuery(),
		)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(store.Close()).To(Succeed())
	})

	When("multiple tasks are assigned to the same user", func() {
		BeforeEach(func() {
			Expect(store.Apply(context.Background(), "TaskAssigned", TaskAssigned{
				TaskID: "t1", Assignee: "alice", At: time.Now(),
			})).To(Succeed())
			Expect(store.Apply(context.Background(), "TaskAssigned", TaskAssigned{
				TaskID: "t2", Assignee: "alice", At: time.Now(),
			})).To(Succeed())
			Expect(store.Apply(context.Background(), "TaskAssigned", TaskAssigned{
				TaskID: "t3", Assignee: "bob", At: time.Now(),
			})).To(Succeed())
		})

		It("returns all task IDs for the user", func() {
			result, err := metaengine.ExecuteTyped[TasksForUser, TasksForUserResult](
				context.Background(), store, TasksForUser{User: "alice"},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.TaskIDs).To(HaveLen(2))
			Expect(result.TaskIDs).To(ContainElement(TaskID("t1")))
			Expect(result.TaskIDs).To(ContainElement(TaskID("t2")))
		})

		It("returns an empty list for a user with no tasks", func() {
			result, err := metaengine.ExecuteTyped[TasksForUser, TasksForUserResult](
				context.Background(), store, TasksForUser{User: "nobody"},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.TaskIDs).To(BeEmpty())
		})
	})

	It("is classified as ADTMultimap by the planner", func() {
		plan := store.Plan()
		Expect(plan.Queries[0].ADT).To(Equal(metaengine.ADTMultimap))
		Expect(plan.Queries[0].ReadPattern).To(Equal(metaengine.ReadMultiLookup))
	})
})

// ─── LOG ADT ───

type RecentTasks struct {
	Limit int
}

type RecentTasksResult struct {
	Events []TaskCreated
}

func recentTasksQuery() metaengine.QueryDecl[RecentTasks, RecentTasksResult] {
	return metaengine.Query[RecentTasks, RecentTasksResult](
		"recent_tasks",
		metaengine.On(TaskCreated{}, func(e TaskCreated) metaengine.Append {
			return metaengine.Append{Value: e}
		}),
	)
}

var _ = Describe("Log ADT", func() {
	var store *metaengine.Store

	BeforeEach(func() {
		var err error
		store, err = metaengine.Plan(
			[]metaengine.Engine{metaengine.NewMemoryEngine()},
			recentTasksQuery(),
		)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(store.Close()).To(Succeed())
	})

	When("events are appended to the log", func() {
		BeforeEach(func() {
			base := time.Now()
			for i := range 5 {
				Expect(store.Apply(context.Background(), "TaskCreated", TaskCreated{
					ID:     TaskID(rune('a' + i)),
					Title:  string(rune('A' + i)),
					Status: "open",
					At:     base.Add(time.Duration(i) * time.Hour),
				})).To(Succeed())
			}
		})

		It("returns the last N events in insertion order", func() {
			result, err := metaengine.ExecuteTyped[RecentTasks, RecentTasksResult](
				context.Background(), store, RecentTasks{Limit: 3},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Events).To(HaveLen(3))
			// Should be the last 3: c, d, e
			Expect(result.Events[0].Title).To(Equal("C"))
			Expect(result.Events[2].Title).To(Equal("E"))
		})

		It("returns all events when limit exceeds log size", func() {
			result, _ := metaengine.ExecuteTyped[RecentTasks, RecentTasksResult](
				context.Background(), store, RecentTasks{Limit: 100},
			)
			Expect(result.Events).To(HaveLen(5))
		})
	})

	It("is classified as ADTLog by the planner", func() {
		plan := store.Plan()
		Expect(plan.Queries[0].ADT).To(Equal(metaengine.ADTLog))
		Expect(plan.Queries[0].ReadPattern).To(Equal(metaengine.ReadLogTail))
	})
})

var _ = Describe("On classification for new ADTs", func() {
	type event struct{ ID, Val string }

	It("classifies MultiEntry return as FoldMultiInsert", func() {
		fold := metaengine.On(event{}, func(e event) metaengine.MultiEntry {
			return metaengine.MultiEntry{Key: e.ID, Value: e.Val}
		})
		Expect(fold.Kind).To(Equal(metaengine.FoldMultiInsert))
	})

	It("classifies Append return as FoldAppend", func() {
		fold := metaengine.On(event{}, func(e event) metaengine.Append {
			return metaengine.Append{Value: e}
		})
		Expect(fold.Kind).To(Equal(metaengine.FoldAppend))
	})
})
