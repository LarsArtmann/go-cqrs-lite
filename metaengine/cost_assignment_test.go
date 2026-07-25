package metaengine_test

import (
	"database/sql"

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
		db, err := sql.Open("sqlite", "file::memory:?cache=shared")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { _ = db.Close() })
		sqliteEng, err := metaengine.NewSQLiteEngine(db)
		Expect(err).NotTo(HaveOccurred())

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
})
