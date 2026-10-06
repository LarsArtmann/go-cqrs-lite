package adttest

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// AssertConcurrentVectorInsert pins the memory-engine concurrency contract
// (M05/F18): N writers inserting M embeddings each into the same collection
// must land every row (count == N*M after the join) without losing writes to
// unsynchronized map access. Run under -race to catch the lock omissions the
// count alone can miss.
func AssertConcurrentVectorInsert(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	ctx := context.Background()

	vb, ok := eng.(metaengine.VectorBackend)
	if !ok {
		t.Fatal("engine does not implement metaengine.VectorBackend")
	}

	const writers = 8
	const perWriter = 50

	suffix := fmt.Sprintf("_%d", time.Now().UnixNano())
	col := "concurrent_vec" + suffix

	var wg sync.WaitGroup

	errs := make(chan error, writers*perWriter)

	for w := 0; w < writers; w++ {
		wg.Add(1)

		go func(w int) {
			defer wg.Done()

			for i := 0; i < perWriter; i++ {
				id := fmt.Sprintf("w%d_i%d", w, i)

				if err := vb.VectorInsert(
					ctx,
					col,
					metaengine.Embedding{ID: id, Values: []float32{float32(w), float32(i)}},
				); err != nil {
					errs <- fmt.Errorf("writer %d insert %s: %w", w, id, err)
				}
			}
		}(w)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent insert: %v", err)
	}

	if counter, ok := eng.(metaengine.VectorCounter); ok {
		got, err := counter.VectorCount(ctx, col)
		if err != nil {
			t.Fatalf("VectorCount: %v", err)
		}

		if want := int64(writers * perWriter); got != want {
			t.Fatalf("VectorCount after concurrent inserts = %d, want %d (lost writes)", got, want)
		}
	}
}

// AssertConcurrentScanDuringWrite pins the memory-engine concurrency contract
// (M05/F18) from the reader side: while a writer keeps inserting, concurrent
// readers hammer every brute-force read path (vector search/count, spatial
// range, full-text query). Under -race this fails on any unsynchronized map
// access between reader and writer, which a post-hoc count cannot detect.
func AssertConcurrentScanDuringWrite(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	ctx := context.Background()

	suffix := fmt.Sprintf("_%d", time.Now().UnixNano())

	stop := make(chan struct{})

	var writer sync.WaitGroup

	writer.Add(1)

	go func() {
		defer writer.Done()

		vb, isVector := eng.(metaengine.VectorBackend)
		sb, isSpatial := eng.(metaengine.SpatialBackend)
		search, isSearch := eng.(metaengine.SearchBackend)

		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}

			id := fmt.Sprintf("row_%d", i)
			vec := []float32{float32(i % 10), 1}

			if isVector {
				if err := vb.VectorInsert(ctx, "scan_race_vec"+suffix,
					metaengine.Embedding{ID: id, Values: vec}); err != nil {
					t.Errorf("writer vector insert: %v", err)
					return
				}
			}

			if isSpatial {
				if err := sb.SpatialInsert(ctx, "scan_race_spatial"+suffix,
					metaengine.Point{ID: id, X: float64(i % 360) - 180, Y: 0}); err != nil {
					t.Errorf("writer spatial insert: %v", err)
					return
				}
			}

			if isSearch {
				if err := search.SearchInsert(ctx, "scan_race_text"+suffix,
					metaengine.IndexedText{ID: id, Content: fmt.Sprintf("row %d needle", i)}); err != nil {
					t.Errorf("writer search insert: %v", err)
					return
				}
			}
		}
	}()

	const readers = 4

	var readerWG sync.WaitGroup

	for r := 0; r < readers; r++ {
		readerWG.Add(1)

		go func() {
			defer readerWG.Done()

			deadline := time.Now().Add(300 * time.Millisecond)

			for time.Now().Before(deadline) {
				if vb, ok := eng.(metaengine.VectorBackend); ok {
					if _, err := vb.VectorSearch(
						ctx, "scan_race_vec"+suffix, []float32{1, 1}, 5, "cosine",
					); err != nil {
						t.Errorf("concurrent VectorSearch: %v", err)
						return
					}
				}

				if sb, ok := eng.(metaengine.SpatialBackend); ok {
					if _, err := sb.SpatialRange(
						ctx, "scan_race_spatial"+suffix, 0, 0, 1_000_000, 5,
					); err != nil {
						t.Errorf("concurrent SpatialRange: %v", err)
						return
					}
				}

				if search, ok := eng.(metaengine.SearchBackend); ok {
					if _, err := search.SearchQuery(
						ctx, "scan_race_text"+suffix, "needle", 5,
					); err != nil {
						t.Errorf("concurrent SearchQuery: %v", err)
						return
					}
				}
			}
		}()
	}

	readerWG.Wait()
	close(stop)
	writer.Wait()
}
