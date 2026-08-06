package sqliteengine_test

import (
	"context"
	"database/sql"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func newSQLiteEngine() (metaengine.Engine, *sql.DB) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	Expect(err).NotTo(HaveOccurred())
	db.SetMaxOpenConns(1)

	eng, err := metaengine.NewSQLiteEngine(db)
	Expect(err).NotTo(HaveOccurred())

	return eng, db
}

var _ = Describe("SQLiteEngine", func() {
	var (
		eng metaengine.Engine
		db  *sql.DB
	)

	BeforeEach(func() {
		eng, db = newSQLiteEngine()
	})

	AfterEach(func() {
		_ = eng.Close()
		_ = db.Close()
	})

	Describe("Profile", func() {
		It("returns the SQLite cost profile", func() {
			p := eng.Profile()
			Expect(p.Name).To(Equal("sqlite"))
			Expect(p.Supports).To(HaveKey(metaengine.ADTMap))
		})
	})

	Describe("MapBackend", func() {
		It("sets and gets a value", func() {
			mb := eng.(metaengine.MapBackend)
			ctx := context.Background()

			err := mb.MapSet(ctx, "tasks", "task-1", map[string]any{"title": "Build API"})
			Expect(err).NotTo(HaveOccurred())

			val, exists, err := mb.MapGet(ctx, "tasks", "task-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(val).NotTo(BeNil())
		})

		It("returns false for missing key", func() {
			mb := eng.(metaengine.MapBackend)
			ctx := context.Background()

			_, exists, err := mb.MapGet(ctx, "tasks", "nonexistent")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("deletes a value", func() {
			mb := eng.(metaengine.MapBackend)
			ctx := context.Background()

			_ = mb.MapSet(ctx, "tasks", "task-1", "value")
			err := mb.MapDelete(ctx, "tasks", "task-1")
			Expect(err).NotTo(HaveOccurred())

			_, exists, _ := mb.MapGet(ctx, "tasks", "task-1")
			Expect(exists).To(BeFalse())
		})

		It("isolates collections", func() {
			mb := eng.(metaengine.MapBackend)
			ctx := context.Background()

			_ = mb.MapSet(ctx, "col-a", "key-1", "val-a")
			_ = mb.MapSet(ctx, "col-b", "key-1", "val-b")

			valA, _, _ := mb.MapGet(ctx, "col-a", "key-1")
			valB, _, _ := mb.MapGet(ctx, "col-b", "key-1")
			Expect(valA).NotTo(Equal(valB))
		})
	})

	Describe("MapUpdater", func() {
		It("performs atomic read-modify-write", func() {
			mu := eng.(metaengine.MapUpdater)
			ctx := context.Background()

			_ = eng.(metaengine.MapBackend).MapSet(ctx, "counters", "c1", float64(5))

			err := mu.MapUpdate(ctx, "counters", "c1", func(prev any) any {
				if v, ok := prev.(float64); ok {
					return v + 10
				}

				return 10
			})
			Expect(err).NotTo(HaveOccurred())

			val, _, _ := eng.(metaengine.MapBackend).MapGet(ctx, "counters", "c1")
			Expect(val.(float64)).To(Equal(float64(15)))
		})
	})

	Describe("SetBackend", func() {
		It("adds and checks membership", func() {
			sb := eng.(metaengine.SetBackend)
			ctx := context.Background()

			err := sb.SetAdd(ctx, "active-users", "user-1")
			Expect(err).NotTo(HaveOccurred())

			contains, err := sb.SetContains(ctx, "active-users", "user-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(contains).To(BeTrue())
		})

		It("returns false for non-member", func() {
			sb := eng.(metaengine.SetBackend)
			ctx := context.Background()

			contains, err := sb.SetContains(ctx, "active-users", "unknown")
			Expect(err).NotTo(HaveOccurred())
			Expect(contains).To(BeFalse())
		})
	})

	Describe("CounterBackend", func() {
		It("increments and reads counters", func() {
			cb := eng.(metaengine.CounterBackend)
			ctx := context.Background()

			err := cb.CounterIncrement(ctx, "stats", metaengine.Delta{"total": 5, "errors": 1})
			Expect(err).NotTo(HaveOccurred())

			err = cb.CounterIncrement(ctx, "stats", metaengine.Delta{"total": 3})
			Expect(err).NotTo(HaveOccurred())

			counters, err := cb.CounterGet(ctx, "stats")
			Expect(err).NotTo(HaveOccurred())
			Expect(counters["total"]).To(Equal(int64(8)))
			Expect(counters["errors"]).To(Equal(int64(1)))
		})
	})

	Describe("MultimapBackend", func() {
		It("stores multiple values per key", func() {
			mb := eng.(metaengine.MultimapBackend)
			ctx := context.Background()

			_ = mb.MultiAdd(ctx, "assignee-tasks", "user-1", "task-a")
			_ = mb.MultiAdd(ctx, "assignee-tasks", "user-1", "task-b")
			_ = mb.MultiAdd(ctx, "assignee-tasks", "user-1", "task-c")

			values, err := mb.MultiGet(ctx, "assignee-tasks", "user-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(HaveLen(3))
		})
	})

	Describe("LogBackend", func() {
		It("appends and tails in order", func() {
			lb := eng.(metaengine.LogBackend)
			ctx := context.Background()

			_ = lb.LogAppend(ctx, "audit", "entry-1")
			_ = lb.LogAppend(ctx, "audit", "entry-2")
			_ = lb.LogAppend(ctx, "audit", "entry-3")

			tail, err := lb.LogTail(ctx, "audit", 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(tail).To(HaveLen(2))
			// LogTail returns the last N entries in chronological order.
			Expect(tail[0]).To(Equal("entry-2"))
			Expect(tail[1]).To(Equal("entry-3"))
		})
	})

	Describe("GraphBackend", func() {
		It("finds neighbors via BFS", func() {
			gb := eng.(metaengine.GraphBackend)
			ctx := context.Background()

			// Build: A → B → C, A → D
			_ = gb.GraphAddEdge(ctx, "dep-graph", metaengine.Edge{From: "A", To: "B"})
			_ = gb.GraphAddEdge(ctx, "dep-graph", metaengine.Edge{From: "B", To: "C"})
			_ = gb.GraphAddEdge(ctx, "dep-graph", metaengine.Edge{From: "A", To: "D"})

			// Depth 1: neighbors of A are B and D
			n1, err := gb.GraphNeighbors(ctx, "dep-graph", "A", 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(n1).To(ConsistOf("B", "D"))

			// Depth 2: neighbors of A are B, C, D
			n2, err := gb.GraphNeighbors(ctx, "dep-graph", "A", 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(n2).To(ConsistOf("B", "C", "D"))
		})
	})

	Describe("ScanBackend", func() {
		It("scans with sorting and limit", func() {
			sb := eng.(metaengine.ScanBackend)
			mb := eng.(metaengine.MapBackend)
			ctx := context.Background()

			_ = mb.MapSet(
				ctx,
				"products",
				"p1",
				map[string]any{"name": "Zebra", "price": float64(100)},
			)
			_ = mb.MapSet(
				ctx,
				"products",
				"p2",
				map[string]any{"name": "Apple", "price": float64(50)},
			)
			_ = mb.MapSet(
				ctx,
				"products",
				"p3",
				map[string]any{"name": "Mango", "price": float64(75)},
			)

			sortFunc := func(a, b any) int {
				an := a.(map[string]any)["name"].(string)
				bn := b.(map[string]any)["name"].(string)
				if an < bn {
					return -1
				}
				if an > bn {
					return 1
				}

				return 0
			}

			results, err := sb.MapScan(ctx, "products", nil, sortFunc, nil, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(results.Items).To(HaveLen(2))
			// First result should be "Apple" (alphabetically first).
			Expect(results.Items[0].(map[string]any)["name"]).To(Equal("Apple"))
		})
	})
})

var _ = Describe("SQLiteEngine FoldUpdate reify (regression)", func() {
	It("applies a typed update fold through SQLite without panicking", func() {
		// Regression: SQLite MapUpdate decodes the stored value into any,
		// producing map[string]any. The fold's typed update handler
		// (func(TaskCompleted, FindTaskResult) must receive
		// reified FindTaskResult, not the raw map — otherwise reflect.Call
		// panics on the type mismatch. Memory engines store typed values, so
		// only the SQLite path exhibited this. Found by planner_bench_test.go.
		eng, db := newSQLiteEngine()
		defer eng.Close()
		defer db.Close()

		store, err := metaengine.Plan([]metaengine.Engine{eng}, findTaskQuery())
		Expect(err).NotTo(HaveOccurred())

		ctx := context.Background()
		Expect(store.Apply(ctx, "TaskCreated", TaskCreated{
			ID: "t-1", Title: "ship it", Assignee: "u-1", Status: "open", Priority: 3,
		})).To(Succeed())

		// This used to panic: callUpdate reflect-called the handler with a
		// map[string]any prev.
		Expect(store.Apply(ctx, "TaskCompleted", TaskCompleted{ID: "t-1"})).To(Succeed())

		got, err := metaengine.ExecuteTyped[FindTask, FindTaskResult](
			ctx,
			store,
			FindTask{ID: "t-1"},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Status).To(Equal("completed"))
		Expect(got.Title).To(Equal("ship it"))
	})
})

var _ = Describe("SQLiteEngine cost-based selection", func() {
	It("selects MemoryEngine for O(1) Map over SQLiteEngine O(logN)", func() {
		memEngine := metaengine.NewMemoryEngine()
		defer memEngine.Close()

		sqlEngine, db := newSQLiteEngine()
		defer sqlEngine.Close()
		defer db.Close()

		store, err := metaengine.Plan(
			[]metaengine.Engine{sqlEngine, memEngine}, // SQLite first, Memory second
			findTaskQuery(),
		)
		Expect(err).NotTo(HaveOccurred())
		defer store.Close()

		// Memory is O(1) for Map; SQLite is O(logN). Memory should win.
		plan := store.Plan()
		Expect(plan).NotTo(BeNil())

		var assignedEngine string

		for _, q := range plan.Queries {
			if q.QueryName == "find_task" {
				assignedEngine = q.EngineName
			}
		}
		Expect(assignedEngine).To(Equal("memory"))
	})
})
