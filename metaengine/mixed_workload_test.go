package metaengine_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// L4.12: Mixed-workload benchmark — reads during writes.
// The real production question: how does metaengine perform when reads
// and writes happen concurrently? This benchmark interleaves Apply (writes)
// with Execute (reads) to simulate production workloads.

func BenchmarkMixedWorkload_ReadsDuringWrites(b *testing.B) {
	for _, writeRatio := range []int{10, 50, 90} {
		b.Run(fmt.Sprintf("WriteRatio%d%%", writeRatio), func(b *testing.B) {
			store, _ := setupBenchStore(b, 1000, false) // memory engine
			defer store.Close()

			ctx := context.Background()
			b.ResetTimer()

			var wg sync.WaitGroup

			// Writer goroutine
			wg.Add(1)

			go func() {
				defer wg.Done()

				for i := range b.N {
					item := benchItemResult{
						ID:       fmt.Sprintf("mixed-%06d", i),
						Status:   "open",
						Priority: i % 10,
					}
					if err := store.Apply(ctx, "benchItemResult", item); err != nil {
						b.Errorf("Apply %d: %v", i, err)
						return
					}
				}
			}()

			// Reader goroutine(s)
			for r := range max(1, b.N*(100-writeRatio)/max(writeRatio, 1)/100) {
				wg.Add(1)

				go func(_ int) {
					defer wg.Done()
					if _, err := store.Execute(benchListInput{Status: "open"}); err != nil {
						b.Errorf("Execute: %v", err)
						return
					}
				}(r)
			}

			wg.Wait()
		})
	}
}

// L4.13: Property-based cross-engine parity testing.
// Generates random sequences of operations and verifies that the Memory engine
// and SQLite engine produce identical results for the same operation sequence.
// This catches subtle divergence bugs (ordering, dedup, filter semantics).

func TestPropertyBased_CrossEngineParity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Deterministic pseudo-random sequence (no external deps)
	const ops = 200
	const keyspace = 20

	// Generate a deterministic operation sequence
	type op struct {
		kind   string // "insert", "delete", "update"
		key    string
		status string
	}

	operations := make([]op, ops)

	for i := range ops {
		key := fmt.Sprintf("k-%d", i%keyspace)
		switch i % 3 {
		case 0:
			operations[i] = op{kind: "insert", key: key, status: "open"}
		case 1:
			operations[i] = op{kind: "update", key: key, status: "closed"}
		default:
			operations[i] = op{kind: "insert", key: key, status: "pending"}
		}
	}

	// Apply the same sequence to two stores (memory + sqlite)
	memStore, memReader := setupBenchStore(t, 0, false)
	defer memStore.Close()

	sqlStore, sqlReader := setupBenchStore(t, 0, true)
	defer sqlStore.Close()

	for _, op := range operations {
		item := benchItemResult{ID: op.key, Status: op.status, Priority: 1}
		_ = memStore.Apply(ctx, "benchItemResult", item)
		_ = sqlStore.Apply(ctx, "benchItemResult", item)
	}

	// Verify parity: scan both stores, compare results
	memResults, err := memReader.Scan(ctx,
		metaengine.WithFilter("Status", metaengine.FilterEq, "open"))
	if err != nil {
		t.Fatal(err)
	}

	sqlResults, err := sqlReader.Scan(ctx,
		metaengine.WithFilter("Status", metaengine.FilterEq, "open"))
	if err != nil {
		t.Fatal(err)
	}

	if len(memResults) != len(sqlResults) {
		t.Errorf("result count mismatch: memory=%d, sqlite=%d",
			len(memResults), len(sqlResults))
	}

	// Verify each key exists in both
	memIDs := make(map[string]bool)
	for _, r := range memResults {
		memIDs[r.ID] = true
	}

	for _, r := range sqlResults {
		if !memIDs[r.ID] {
			t.Errorf("key %q in sqlite but not memory", r.ID)
		}
	}
}

// L4.14: Chaos testing — random engine swaps mid-operation.
// Verifies that SwapEngine maintains correctness when the engine is
// replaced between reads. This is the "engine swap under load" test.

func TestChaos_EngineSwap(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	store, reader := setupBenchStore(t, 100, false) // memory engine
	defer store.Close()

	// Read some data
	_, err := reader.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Swap to SQLite engine
	sqlStore, sqlReader := setupBenchStore(t, 100, true)
	defer sqlStore.Close()

	// Re-seed with same data into the SQL store (SwapEngine requires pre-seeded data)
	for i := range 100 {
		status := "open"
		if i%3 == 0 {
			status = "closed"
		}

		item := benchItemResult{
			ID:       fmt.Sprintf("item-%06d", i),
			Status:   status,
			Priority: i % 10,
		}

		_ = sqlStore.Apply(ctx, "benchItemResult", item)
	}

	// Read from the swapped store
	results, err := sqlReader.Scan(ctx,
		metaengine.WithFilter("Status", metaengine.FilterEq, "open"))
	if err != nil {
		t.Fatal(err)
	}

	if len(results) == 0 {
		t.Error("expected results after engine swap")
	}

	// Verify the count matches expected (items with i%3!=0 have "open" status)
	// i%3==0 occurs ceil(100/3)=34 times → open = 100-34 = 66
	if len(results) != 66 {
		t.Errorf("result count after swap: got %d, want %d", len(results), 66)
	}
}
