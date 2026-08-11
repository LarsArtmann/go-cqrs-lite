package bboltengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// FuzzMapSetGet exercises MapSet + MapGet with arbitrary string keys and
// integer values. Regression guard for encoding/decoding correctness.
func FuzzMapSetGet(f *testing.F) {
	f.Add("key1", int64(42))
	f.Add("", int64(0))
	f.Add("unicode-key-æøå", int64(-1))
	f.Add("very-long-key-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", int64(999999999))

	f.Fuzz(func(t *testing.T, key string, value int64) {
		t.Parallel()

		ctx := context.Background()
		eng := mustNewBboltEngine(t)
		mb := eng.(metaengine.MapBackend)

		if err := mb.MapSet(ctx, "fuzz", key, value); err != nil {
			t.Fatalf("MapSet %q: %v", key, err)
		}

		got, found, err := mb.MapGet(ctx, "fuzz", key)
		if err != nil {
			t.Fatalf("MapGet %q: %v", key, err)
		}
		if !found {
			t.Fatalf("MapGet %q: not found after Set", key)
		}

		// Values are JSON-encoded, so int64 may decode as float64.
		switch v := got.(type) {
		case float64:
			if int64(v) != value {
				t.Errorf("MapGet %q = %v, want %d", key, v, value)
			}
		case int64:
			if v != value {
				t.Errorf("MapGet %q = %d, want %d", key, v, value)
			}
		default:
			t.Errorf("MapGet %q = %T (%v), want int64", key, got, got)
		}
	})
}
