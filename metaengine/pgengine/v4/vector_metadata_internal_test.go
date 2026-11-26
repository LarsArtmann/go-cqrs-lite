package pgengine

import (
	"context"
	"math"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Embedding.Metadata is caller-supplied map[string]any: a value JSON cannot
// encode must come back as an error, not a panic. No database is needed —
// the metadata is rejected before anything is probed.
func TestVectorInsert_RejectsUnmarshalableMetadata(t *testing.T) {
	t.Parallel()

	e := &pgEngine{}
	ctx := context.Background()

	for name, metadata := range map[string]map[string]any{
		"func":    {"f": func() {}},
		"chan":    {"c": make(chan int)},
		"complex": {"c": complex(1, 2)},
		"nan":     {"n": math.NaN()},
		"nested":  {"m": map[string]any{"deep": func() {}}},
	} {
		if metadata == nil {
			continue
		}

		err := e.VectorInsert(ctx, "c", metaengine.Embedding{
			ID: "x", Values: []float32{1}, Metadata: metadata,
		})
		if err == nil {
			t.Errorf("%s: VectorInsert metadata = nil error, want a marshal error", name)
		}
	}
}
