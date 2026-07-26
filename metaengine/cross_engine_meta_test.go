package metaengine_test

import (
	"context"
	"database/sql"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// cross_engine_meta_test.go guards the Store read-contract across every Engine
// implementation. The bug class: a reflective fold/query layer where one engine
// returns typed Go values and another returns JSON-decoded map[string]any. If
// any user-code boundary (fold update func, FilterOn/SortOn closures, collection
// reconstruction, ExecuteTyped) forgets to reify, that path panics or silently
// returns empty results — but ONLY on the diverging engine.
//
// This test runs the full Apply → ExecuteTyped scenario (FoldInsert,
// FoldUpdate, filtered+sorted scan, counter aggregate) against both the memory
// and SQLite engines and asserts they produce IDENTICAL typed results. If a
// future engine impl diverges, or a new user-code boundary is added without
// reify, this test fails on the diverging engine.
//
// This is the test that was missing when ADR-0067 (tx-atomic MapUpdate) shipped
// with a panicking FoldUpdate-on-SQLite path — the benchmark caught it, the test
// suite did not. See docs/status/2026-07-26_16-13 (d1).

// crossEngineResults captures the typed outputs of the full scenario, so we can
// deep-compare across engines without reflecting over the Store internals.
type crossEngineResults struct {
	pointAfterInsert FindTaskResult
	pointAfterUpdate FindTaskResult
	scanTasks        []FindTaskResult
	counterByStatus  map[string]int64
	membership       bool
}

func runCrossEngineScenario(eng metaengine.Engine) crossEngineResults {
	ctx := context.Background()

	store, err := metaengine.Plan(
		[]metaengine.Engine{eng},
		findTaskQuery(),
		listTasksByStatusQuery(),
		countByStatusQuery(),
		checkAssigneeQuery(),
	)
	Expect(err).NotTo(HaveOccurred())
	defer store.Close()

	now := time.Now()

	// FoldInsert (Map ADT).
	Expect(store.Apply(ctx, "TaskCreated", TaskCreated{
		ID: "t1", Title: "Write meta-test", Assignee: "alice",
		Status: "open", Priority: 2, At: now,
	})).To(Succeed())
	Expect(store.Apply(ctx, "TaskCreated", TaskCreated{
		ID: "t2", Title: "Reify boundaries", Assignee: "bob",
		Status: "open", Priority: 1, At: now,
	})).To(Succeed())

	// Point lookup after insert — exercises ExecuteTyped reify (execute.go:333).
	r1, err := metaengine.ExecuteTyped[FindTask, FindTaskResult](ctx, store, FindTask{ID: "t1"})
	Expect(err).NotTo(HaveOccurred())

	// FoldUpdate (MapUpdater RMW) — exercises callUpdate reify (fold_classify.go).
	Expect(store.Apply(ctx, "TaskCompleted", TaskCompleted{ID: "t1", At: now})).To(Succeed())

	r2, err := metaengine.ExecuteTyped[FindTask, FindTaskResult](ctx, store, FindTask{ID: "t1"})
	Expect(err).NotTo(HaveOccurred())

	// Filtered + sorted scan — exercises buildFilterPredicates, buildSortFunc,
	// reconstructCollection reify (execute.go, collection.go).
	scan, err := metaengine.ExecuteTyped[ListTasksByStatus, ListTasksByStatusResult](
		ctx, store, ListTasksByStatus{Status: "open", Limit: 10},
	)
	Expect(err).NotTo(HaveOccurred())

	// Counter aggregate — map[string]int64 (non-struct, scalar-safe).
	counts, err := metaengine.ExecuteTyped[CountByStatus, map[string]int64](
		ctx,
		store,
		CountByStatus{},
	)
	Expect(err).NotTo(HaveOccurred())

	// Set membership (bool result — scalar-safe).
	Expect(store.Apply(ctx, "TaskAssigned", TaskAssigned{
		TaskID: "t1", Assignee: "alice", Previous: "", At: now,
	})).To(Succeed())
	mem, err := metaengine.ExecuteTyped[CheckAssignee, bool](
		ctx,
		store,
		CheckAssignee{User: "alice"},
	)
	Expect(err).NotTo(HaveOccurred())

	return crossEngineResults{
		pointAfterInsert: r1,
		pointAfterUpdate: r2,
		scanTasks:        scan.Tasks,
		counterByStatus:  counts,
		membership:       mem,
	}
}

var _ = Describe("Cross-engine read-contract meta-test", func() {
	var (
		memEng    metaengine.Engine
		sqliteEng metaengine.Engine
		db        *sql.DB
	)

	BeforeEach(func() {
		memEng = metaengine.NewMemoryEngine()

		var err error
		db, err = sql.Open("sqlite", "file:xmeta1?mode=memory&cache=shared")
		Expect(err).NotTo(HaveOccurred())
		db.SetMaxOpenConns(1)

		sqliteEng, err = metaengine.NewSQLiteEngine(db)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(memEng.Close()).To(Succeed())
		Expect(sqliteEng.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})

	// The headline assertion: every typed read path returns IDENTICAL results
	// across both engines. A divergence here is the reify-boundary bug class.
	It("produces identical typed results across memory and SQLite engines", func() {
		mem := runCrossEngineScenario(memEng)
		sql := runCrossEngineScenario(sqliteEng)

		// Point lookup after FoldInsert.
		Expect(sql.pointAfterInsert).To(Equal(mem.pointAfterInsert))
		Expect(sql.pointAfterInsert.Title).To(Equal("Write meta-test"),
			"point lookup must return the typed struct, not map[string]any")

		// Point lookup after FoldUpdate — the path that panicked before reify.
		Expect(sql.pointAfterUpdate).To(Equal(mem.pointAfterUpdate))
		Expect(sql.pointAfterUpdate.Status).To(Equal("completed"),
			"FoldUpdate must reify prev before calling the typed update handler")

		// Filtered+sorted scan collection — the path that panicked + silently
		// returned empty before reify.
		Expect(sql.scanTasks).To(Equal(mem.scanTasks))
		Expect(len(sql.scanTasks)).To(BeNumerically(">", 0),
			"scan must reconstruct typed items, not silently drop map[string]any rows")
		if len(sql.scanTasks) > 0 {
			Expect(sql.scanTasks[0].Title).NotTo(BeEmpty(),
				"scan items must be typed FindTaskResult, not zero-value structs from failed reify")
		}

		// Counter + membership (scalar results — guard against future divergence).
		Expect(sql.counterByStatus).To(Equal(mem.counterByStatus))
		Expect(sql.membership).To(Equal(mem.membership))
	})
})
