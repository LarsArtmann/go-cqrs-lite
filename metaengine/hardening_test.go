package metaengine_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// These specs guard the metaengine SQLite-engine hardening fixes:
// MapUpdate transactional atomicity, multimap restart-safe seq seeding, and
// cross-engine JSON reification of struct results.
var _ = Describe("SQLite engine hardening", func() {
	Describe("MapUpdate atomicity", func() {
		It("does not lose concurrent increments", func() {
			eng, db := newSQLiteEngine()
			DeferCleanup(func() {
				_ = eng.Close()
				_ = db.Close()
			})

			mu := eng.(metaengine.MapUpdater)
			mb := eng.(metaengine.MapBackend)
			ctx := context.Background()

			Expect(mb.MapSet(ctx, "contention", "shared", float64(0))).To(Succeed())

			// With a non-atomic Get-then-Set (the pre-hardening shape), concurrent
			// updaters interleave their read and write on the single pooled
			// connection and silently drop increments. The tx-wrapped MapUpdate
			// reserves the connection for the whole read-modify-write, so every
			// increment is applied exactly once.
			const n = 50
			var wg sync.WaitGroup
			start := make(chan struct{})

			wg.Add(n)
			for range n {
				go func() {
					defer wg.Done()
					<-start
					_ = mu.MapUpdate(ctx, "contention", "shared", func(prev any) any {
						v, _ := prev.(float64)

						return v + 1
					})
				}()
			}
			close(start)
			wg.Wait()

			val, _, err := mb.MapGet(ctx, "contention", "shared")
			Expect(err).NotTo(HaveOccurred())
			Expect(val.(float64)).To(Equal(float64(n)))
		})
	})

	Describe("Multimap restart safety", func() {
		It("does not collide on seq after reopening the database", func() {
			dir, err := os.MkdirTemp("", "metaengine-restart-*")
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() { _ = os.RemoveAll(dir) })

			dbPath := filepath.Join(dir, "restart.db")
			ctx := context.Background()

			// Phase 1: persist 3 values, then close.
			eng1, db1 := newSQLiteEngineForPath(dbPath)
			mb1 := eng1.(metaengine.MultimapBackend)
			for _, v := range []string{"a", "b", "c"} {
				Expect(mb1.MultiAdd(ctx, "restart-mm", "k", v)).To(Succeed())
			}
			_ = eng1.Close() // no-op; caller owns the DB
			Expect(db1.Close()).To(Succeed())

			// Phase 2: reopen the SAME file and add 2 more. A fresh engine with
			// a zeroed in-memory seq counter would reuse seq 0,1 and collide on
			// the (collection,key,seq) primary key. The sync.Once MAX(seq) seed
			// must keep the sequence monotonic across process restarts.
			eng2, db2 := newSQLiteEngineForPath(dbPath)
			DeferCleanup(func() { _ = db2.Close() })
			mb2 := eng2.(metaengine.MultimapBackend)
			for _, v := range []string{"d", "e"} {
				Expect(mb2.MultiAdd(ctx, "restart-mm", "k", v)).To(Succeed())
			}

			values, err := mb2.MultiGet(ctx, "restart-mm", "k")
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(HaveLen(5))
		})
	})

	Describe("Cross-engine reification", func() {
		It("reifies a SQLite map[string]any into a typed struct result", func() {
			eng, db := newSQLiteEngine()
			DeferCleanup(func() {
				_ = eng.Close()
				_ = db.Close()
			})

			store, err := metaengine.Plan([]metaengine.Engine{eng}, findTaskQuery())
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() { _ = store.Close() })

			ctx := context.Background()
			Expect(store.Apply(ctx, "TaskCreated", TaskCreated{
				ID: "t1", Title: "Reify me", Assignee: "alice",
				Status: "open", Priority: 3, At: time.Now(),
			})).To(Succeed())

			// SQLite returns the stored value as map[string]any (JSON round-trip
			// through any); the direct type assertion in ExecuteTyped fails and
			// reify[R] rebuilds the struct via JSON. Without reification this
			// would return an execute-type-mismatch error.
			result, err := metaengine.ExecuteTyped[FindTask, FindTaskResult](
				ctx, store, FindTask{ID: "t1"},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Title).To(Equal("Reify me"))
			Expect(result.Status).To(Equal("open"))
			Expect(result.Assignee).To(Equal(UserID("alice")))
			Expect(result.Priority).To(Equal(3))
		})
	})
})
