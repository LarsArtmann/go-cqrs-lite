package metaengine_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

var _ = Describe("Regression: cost model picks the cheaper engine", func() {
	It("assigns a point-lookup query to memory over SQLite at equal complexity", func() {
		// find_task is a Map ADT point lookup. Both engines serve ADTMap at
		// O(logN), but memory's NsPerOp (500) is ~14x cheaper than SQLite
		// (7000). The planner must therefore prefer memory. This guards the
		// cost model's core promise: pick the obviously-right engine.
		sqliteEng, db := newSQLiteEngine()
		DeferCleanup(func() { _ = db.Close() })

		store, err := metaengine.Plan(
			[]metaengine.Engine{metaengine.NewMemoryEngine(), sqliteEng},
			findTaskQuery(),
		)
		Expect(err).NotTo(HaveOccurred())
		defer store.Close()

		plan := store.Plan()
		Expect(plan.Queries).To(HaveLen(1))
		Expect(plan.Queries[0].EngineName).To(Equal("memory"))
		Expect(plan.Queries[0].Cost.EstimatedLatencyMs).To(BeNumerically("<",
			plan.Queries[0].Cost.EstimatedLatencyMs+1)) // sanity: finite cost
	})

	It("estimates a lower latency for memory than SQLite for the same query", func() {
		// Memory serves a Map lookup at O(1) (hash map) with NsPerOp=500; SQLite
		// serves it at O(logN) (B-tree index) with NsPerOp=7000. Memory wins on
		// BOTH axes, which is why the planner assigns the point lookup to it.
		mem := metaengine.NewMemoryEngine().Profile()
		sqlite := metaengine.SQLiteEngineProfile()

		Expect(mem.NsPerOp).To(BeNumerically("<", sqlite.NsPerOp))
		Expect(mem.Supports[metaengine.ADTMap]).To(Equal(metaengine.ComplexityO1))
		Expect(sqlite.Supports[metaengine.ADTMap]).To(Equal(metaengine.ComplexityOLogN))
	})

	It("distributes different queries to different engines based on cost", func() {
		// The headline promise of the metaengine: different queries land on
		// different engines based on the cost model. A Counter query (O(1) on
		// both engines, but Memory has 14x lower NsPerOp) should go to Memory.
		// A Map query with FilterOnField (SQLite pushdown: O(logN) vs Memory
		// closure scan O(N)) should go to SQLite. This proves the planner
		// distributes work, not just picks one engine for everything.
		sqliteEng, db := newSQLiteEngine()
		DeferCleanup(func() { _ = db.Close() })

		counterQ := countByStatusQuery()

		filteredMapQ := metaengine.Query[FindTask, FindTaskResult](
			"filtered_find_task",
			metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, FindTaskResult) {
				return e.ID, FindTaskResult{ID: e.ID, Title: e.Title, Status: e.Status}
			}),
			metaengine.On(TaskDeleted{}, metaengine.Remove[FindTaskResult]()),
			metaengine.FilterOnField[FindTaskResult]("Status", metaengine.FilterEq),
			metaengine.Volume(100_000),
		)

		store, err := metaengine.Plan(
			[]metaengine.Engine{metaengine.NewMemoryEngine(), sqliteEng},
			counterQ, filteredMapQ,
		)
		Expect(err).NotTo(HaveOccurred())
		defer store.Close()

		plan := store.Plan()
		Expect(plan.Queries).To(HaveLen(2))

		assignments := map[string]string{}
		for _, q := range plan.Queries {
			assignments[q.QueryName] = q.EngineName
		}

		// Counter → Memory (O(1) on both, but Memory is 14x cheaper per op).
		Expect(assignments["count_by_status"]).To(Equal("memory"),
			"Counter query should go to Memory (cheaper NsPerOp at equal complexity)")

		// Filtered Map → SQLite (O(logN) pushdown beats Memory O(N) scan).
		Expect(assignments["filtered_find_task"]).To(Equal("sqlite"),
			"Filtered Map query should go to SQLite (pushdown O(logN) beats Memory O(N))")

		// The core assertion: they went to DIFFERENT engines.
		Expect(assignments["count_by_status"]).NotTo(Equal(assignments["filtered_find_task"]),
			"Distribution test: queries must land on different engines")
	})
})
