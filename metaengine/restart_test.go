package metaengine_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// These specs cover restart-safety (persistence across DB reopen) and
// concurrent-write idempotency for the Set/Log/Graph backends — the same class
// of guarantee already locked in for Multimap/MapUpdate in hardening_test.go.

var _ = Describe("Regression: LogBackend restart safety", func() {
	It("preserves appended entries across a database reopen", func() {
		dir, err := os.MkdirTemp("", "metaengine-log-restart-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { _ = os.RemoveAll(dir) })

		dbPath := filepath.Join(dir, "log.db")
		ctx := context.Background()

		// Phase 1: append 3 ordered values.
		eng1, db1 := newSQLiteEngineForPath(dbPath)
		lb1 := eng1.(metaengine.LogBackend)
		for _, v := range []string{"one", "two", "three"} {
			Expect(lb1.LogAppend(ctx, "audit", v)).To(Succeed())
		}
		_ = eng1.Close()
		Expect(db1.Close()).To(Succeed())

		// Phase 2: reopen, append a 4th, and tail — all 4 must survive in order.
		eng2, db2 := newSQLiteEngineForPath(dbPath)
		DeferCleanup(func() { _ = db2.Close() })
		lb2 := eng2.(metaengine.LogBackend)
		Expect(lb2.LogAppend(ctx, "audit", "four")).To(Succeed())

		tail, err := lb2.LogTail(ctx, "audit", 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(tail).To(HaveLen(4))
	})
})

var _ = Describe("Regression: SetBackend concurrent idempotency", func() {
	It("is idempotent under concurrent SetAdd of the same key", func() {
		// Set membership is idempotent: N concurrent SetAdd(col, sameKey) must
		// not error and must leave exactly one membership record.
		ctx := context.Background()
		eng := metaengine.NewMemoryEngine()
		defer eng.Close()
		sb := eng.(metaengine.SetBackend)

		const goroutines = 100
		var wg sync.WaitGroup
		wg.Add(goroutines)
		start := make(chan struct{})
		var errCount int
		var mu sync.Mutex

		for range goroutines {
			go func() {
				defer wg.Done()
				<-start
				if err := sb.SetAdd(ctx, "members", "alice"); err != nil {
					mu.Lock()
					errCount++
					mu.Unlock()
				}
			}()
		}
		close(start)
		wg.Wait()

		Expect(errCount).To(BeZero())
		ok, err := sb.SetContains(ctx, "members", "alice")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
	})
})

var _ = Describe("Regression: GraphBackend restart safety", func() {
	It("preserves edges across a database reopen", func() {
		dir, err := os.MkdirTemp("", "metaengine-graph-restart-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { _ = os.RemoveAll(dir) })

		dbPath := filepath.Join(dir, "graph.db")
		ctx := context.Background()

		// Phase 1: build a small graph user→[t1, t2].
		eng1, db1 := newSQLiteEngineForPath(dbPath)
		gb1 := eng1.(metaengine.GraphBackend)
		for _, to := range []string{"t1", "t2"} {
			Expect(
				gb1.GraphAddEdge(ctx, "assign", metaengine.Edge{From: "alice", To: to}),
			).To(Succeed())
		}
		_ = eng1.Close()
		Expect(db1.Close()).To(Succeed())

		// Phase 2: reopen, add one more edge, and verify all three neighbors.
		eng2, db2 := newSQLiteEngineForPath(dbPath)
		DeferCleanup(func() { _ = db2.Close() })
		gb2 := eng2.(metaengine.GraphBackend)
		Expect(
			gb2.GraphAddEdge(ctx, "assign", metaengine.Edge{From: "alice", To: "t3"}),
		).To(Succeed())

		neighbors, err := gb2.GraphNeighbors(ctx, "assign", "alice", 1)
		Expect(err).NotTo(HaveOccurred())
		Expect(neighbors).To(ConsistOf("t1", "t2", "t3"))
	})
})
