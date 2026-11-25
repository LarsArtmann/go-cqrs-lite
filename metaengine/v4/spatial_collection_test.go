package metaengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestMemorySpatialIndexPerCollection pins that two spatial collections sharing
// one index do not see each other's points, even for the same point ID.
func TestMemorySpatialIndexPerCollection(t *testing.T) {
	t.Parallel()

	idx := metaengine.NewMemorySpatialIndex()
	ctx := context.Background()

	// Berlin and (far away) Munich, both under the ID "same".
	err := idx.Insert(ctx, "places", metaengine.Point{ID: "same", X: 13.405, Y: 52.52})
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	err = idx.Insert(ctx, "stores", metaengine.Point{ID: "same", X: 11.582, Y: 48.135})
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	for col, want := range map[string]int{"places": 1, "stores": 0} {
		got, err := idx.Range(ctx, col, 13.405, 52.52, 1000, 10)
		if err != nil {
			t.Fatalf("Range %s: %v", col, err)
		}

		if len(got) != want {
			t.Errorf("Range(%s) = %d results, want %d", col, len(got), want)
		}
	}

	if err := idx.Insert(ctx, "places", metaengine.Point{ID: "same", X: 13.405, Y: 52.52}); err != nil {
		t.Fatalf("Insert: %v", err)
	}
}
