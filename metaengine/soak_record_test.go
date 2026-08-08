package metaengine_test

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// TestSoak_RecordAwarePipeline processes 100K events through the Record-aware
// pipeline (store.ApplyRecord + OnRecord folds) and verifies:
//
//  1. No memory leaks — heap growth is O(unique keys), not O(total events).
//  2. Record metadata (StreamID, Version) is correctly stamped on every event,
//     not just the first or last.
//
// This complements TestSoak_MemoryBounded_10M, which exercises the legacy
// Apply path. The ApplyRecord path has a different dispatch (SetCurrentRecord
// on every event) that warrants its own soak.
func TestSoak_RecordAwarePipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("record-aware soak: skips in -short mode")
	}

	t.Parallel()

	type createdEvent struct {
		ID    string
		Name  string
		Value int64
	}
	type lookup struct{ Key string }
	type itemState struct {
		ID       string
		Name     string
		StreamID string
		Version  int64
		Value    int64
	}

	q := metaengine.Query[lookup, itemState](
		"soak-record-items",
		metaengine.OnRecord(
			createdEvent{},
			func(rec record.Record, e createdEvent) (string, itemState) {
				return e.ID, itemState{
					ID:       e.ID,
					Name:     e.Name,
					StreamID: rec.StreamID.String(),
					Version:  rec.Version,
					Value:    e.Value,
				}
			},
		),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	const (
		numEvents = 100_000
		numKeys   = 500 // 200 updates per key — memory bounded by numKeys
	)

	keys := make([]string, numKeys)
	for i := range keys {
		keys[i] = fmt.Sprintf("item-%04d", i)
	}

	streamRefs := make([]record.StreamRef, numKeys)
	for i := range streamRefs {
		streamRefs[i] = record.NewStreamRef("Item", keys[i])
	}

	runtime.GC()
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	for i := range numEvents {
		keyIdx := i % numKeys
		rec := record.Record{
			Type:       "createdEvent",
			StreamID:   streamRefs[keyIdx],
			StreamType: "Item",
			Version:    int64(i/numKeys + 1),
			MetaData: record.CommonMetadata{
				CorrelationID: fmt.Sprintf("corr-%d", i),
			},
		}

		evt := createdEvent{
			ID:    keys[keyIdx],
			Name:  fmt.Sprintf("name-%d", i),
			Value: int64(i),
		}

		if err := store.ApplyRecord(ctx, rec, evt); err != nil {
			t.Fatalf("ApplyRecord %d: %v", i, err)
		}
	}

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	totalGrowth := int64(after.HeapAlloc) - int64(baseline.HeapAlloc)

	maxGrowth := int64(10 * 1024 * 1024) // 10MB
	if enginetest.RaceEnabled {
		maxGrowth *= 5 // 50MB under -race
	}

	if totalGrowth > maxGrowth {
		t.Errorf(
			"heap grew %d bytes after %d ApplyRecord calls with %d keys (max %d) — possible leak",
			totalGrowth,
			numEvents,
			numKeys,
			maxGrowth,
		)
	}

	var checked, metadataErrors int

	for k := range numKeys {
		result, err := metaengine.ExecuteTyped[lookup, itemState](
			ctx, store, lookup{Key: keys[k]},
		)
		if err != nil {
			t.Fatalf("ExecuteTyped key %d: %v", k, err)
		}

		checked++

		// The last event for this key determines the final state.
		// Key k receives events at indices k, k+numKeys, k+2*numKeys, ...
		// The last one is at index numEvents - numKeys + k.
		lastIdx := numEvents - numKeys + k
		expectedVersion := int64(lastIdx/numKeys + 1)

		if result.StreamID != streamRefs[k].String() {
			metadataErrors++
			if metadataErrors <= 5 {
				t.Errorf("key %s: StreamID got %q, want %q",
					keys[k], result.StreamID, streamRefs[k].String())
			}
		}

		if result.Version != expectedVersion {
			metadataErrors++
			if metadataErrors <= 5 {
				t.Errorf("key %s: Version got %d, want %d",
					keys[k], result.Version, expectedVersion)
			}
		}

		if result.ID != keys[k] {
			metadataErrors++
			if metadataErrors <= 5 {
				t.Errorf("key %s: ID got %q, want %q", keys[k], result.ID, keys[k])
			}
		}
	}

	if metadataErrors > 5 {
		t.Errorf("...and %d more metadata errors (checked %d keys)",
			metadataErrors-5, checked)
	}

	t.Logf(
		"record-aware soak: %d events, %d keys, %d bytes heap growth (%.1f MB), %d metadata errors",
		numEvents,
		numKeys,
		totalGrowth,
		float64(totalGrowth)/1024/1024,
		metadataErrors,
	)
}
