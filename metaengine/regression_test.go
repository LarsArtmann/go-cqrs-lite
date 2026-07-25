package metaengine_test

import (
	"context"
	"strconv"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// These specs lock in honest cost claims and prior-session correctness fixes
// (ADT complexity demotion, Counter atomicity, ExecuteTyped reify mismatch,
// SQLite caller-owns-DB Close contract) that previously had no dedicated test.

var _ = Describe("Regression: engine profile honesty", func() {
	It("reports ADTSortedMap as O(NlogN) for SQLite, NOT the optimistic O(logN)", func() {
		// The SQLite MapScan loads every row then sorts in Go: O(N) load +
		// O(N log N) sort. The profile must claim O(NlogN), not O(logN), until
		// sort-column pushdown lands (ADR-0063). This guards against a
		// regression that would make the planner over-pick SQLite for scans.
		p := metaengine.SQLiteEngineProfile()
		Expect(p.Supports[metaengine.ADTSortedMap]).To(Equal(metaengine.ComplexityONLogN))
		Expect(p.Supports[metaengine.ADTSortedMap]).NotTo(Equal(metaengine.ComplexityOLogN))
	})

	It("reports ADTSortedMap for the memory engine", func() {
		p := metaengine.NewMemoryEngine().Profile()
		Expect(p.Supports).To(HaveKey(metaengine.ADTSortedMap))
		Expect(p.Supports[metaengine.ADTSortedMap]).NotTo(Equal(metaengine.ComplexityOLogN))
	})
})

var _ = Describe("Regression: Counter atomicity under concurrency", func() {
	It("loses no increments when many events fire concurrently", func() {
		ctx := context.Background()
		store, err := metaengine.Plan(
			[]metaengine.Engine{metaengine.NewMemoryEngine()},
			countByStatusQuery(),
		)
		Expect(err).NotTo(HaveOccurred())
		defer store.Close()

		const goroutines = 100
		var wg sync.WaitGroup
		wg.Add(goroutines)
		start := make(chan struct{})

		for i := range goroutines {
			go func(i int) {
				defer wg.Done()
				<-start
				// Every event increments the "open" counter by 1.
				Expect(store.Apply(ctx, "TaskCreated", TaskCreated{
					ID: TaskID(strconv.Itoa(i)), Title: "t", Status: "open",
				})).To(Succeed())
			}(i)
		}
		close(start)
		wg.Wait()

		raw, err := store.ExecuteCtx(ctx, CountByStatus{})
		Expect(err).NotTo(HaveOccurred())
		counts, ok := raw.(map[string]int64)
		Expect(ok).To(BeTrue())
		// All 100 increments must be present (no lost updates).
		Expect(counts["open"]).To(Equal(int64(goroutines)))
	})
})

var _ = Describe("Regression: ExecuteTyped reify mismatch", func() {
	It("returns a type-mismatch error when R is incompatible with the result", func() {
		// A query declared to return FindTaskResult cannot be read into an
		// unrelated struct. ExecuteTyped must surface an error (via the reify
		// fallback failing), not silently return a zero/garbage value.
		ctx := context.Background()
		store, err := metaengine.Plan(
			[]metaengine.Engine{metaengine.NewMemoryEngine()},
			findTaskQuery(),
		)
		Expect(err).NotTo(HaveOccurred())
		defer store.Close()

		Expect(store.Apply(ctx, "TaskCreated", TaskCreated{
			ID: "t1", Title: "T", Status: "open",
		})).To(Succeed())

		// A struct result cannot be reified into a numeric type: JSON cannot
		// unmarshal an object into an int, so reify fails and ExecuteTyped must
		// surface the execute-type-mismatch error rather than a silent zero.
		type wrongResult int
		_, execErr := metaengine.ExecuteTyped[FindTask, wrongResult](ctx, store, FindTask{ID: "t1"})
		Expect(execErr).To(HaveOccurred())
		Expect(execErr.Error()).To(ContainSubstring("does not match"))
	})
})

var _ = Describe("Regression: SQLite engine Close is a no-op (caller owns the DB)", func() {
	It("does not close the caller's *sql.DB", func() {
		eng, db := newSQLiteEngine()

		// Closing the engine must be a no-op: the caller owns the *sql.DB.
		Expect(eng.Close()).To(Succeed())
		Expect(eng.Close()).To(Succeed(), "Close must be idempotent")

		// The DB must still be usable after the engine Close.
		var one int
		Expect(db.QueryRowContext(context.Background(), "SELECT 1").Scan(&one)).To(Succeed())
		Expect(one).To(Equal(1))

		Expect(db.Close()).To(Succeed())
	})

	It("does not return a *sql.DB pointer for the memory engine", func() {
		// Sanity: memory engine has no DB and Close is a harmless no-op.
		eng := metaengine.NewMemoryEngine()
		Expect(eng.Close()).To(Succeed())
	})
})

