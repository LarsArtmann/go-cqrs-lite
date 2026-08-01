package pebbleengine_test

import (
	"context"
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// FuzzScanRawValues exercises the filter index path with arbitrary filter values.
// Regression guard for the cursor pagination fix and numeric encoding.
func FuzzScanRawValues(f *testing.F) {
	f.Add(int64(0))
	f.Add(int64(10))
	f.Add(int64(100))
	f.Add(int64(-1))
	f.Add(int64(999999999))

	f.Fuzz(func(t *testing.T, threshold int64) {
		t.Parallel()

		ctx := context.Background()
		eng, err := pebbleengine.NewPebbleEngine("")
		if err != nil {
			t.Skipf("pebble engine: %v", err)
		}

		defer eng.Close()

		lp := eng.(metaengine.LayoutPlanner)
		if err := lp.ApplyLayout("items", []string{"score"}, nil); err != nil {
			t.Fatalf("ApplyLayout: %v", err)
		}

		mb := eng.(metaengine.MapBackend)

		for i := int64(1); i <= 5; i++ {
			val := map[string]any{"score": float64(i * 10)}
			if err := mb.MapSet(ctx, "items", string(rune('a'+i-1)), val); err != nil {
				t.Fatalf("MapSet: %v", err)
			}
		}

		rsr := eng.(metaengine.RawScanReader)
		sortSpec := &metaengine.SortSpec{Column: "score", Desc: false}

		results, err := rsr.ScanRawValues(
			ctx, "items",
			[]metaengine.FilterSpec{{Column: "score", Op: metaengine.FilterGe, Value: threshold}},
			sortSpec, nil, 3,
		)
		if err != nil {
			t.Fatalf("ScanRawValues: %v", err)
		}

		for _, raw := range results.Items {
			var decoded map[string]any
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Errorf("unmarshal result: %v", err)

				continue
			}

			score, ok := decoded["score"].(float64)
			if !ok {
				t.Errorf("score field is %T, not float64", decoded["score"])
			}

			if score < float64(threshold) {
				t.Errorf("score %v < threshold %v (FilterGe violated)", score, threshold)
			}
		}
	})
}
