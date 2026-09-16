package adttest

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// AssertVectorDimensionGuard pins the cross-engine dimension-lock contract on
// one engine instance: the first insert establishes the collection's
// dimension; a mismatching insert is rejected with errors.Is(...,
// metaengine.ErrVectorDimensionMismatch) (Rejection family); zero-dimension
// embeddings are rejected; same-dimension upserts and other collections are
// unaffected. Run it from every engine module's test suite so the lock is
// enforced uniformly (mismatched behavior here is a split-brain class).
func AssertVectorDimensionGuard(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	ctx := context.Background()

	vb, ok := eng.(metaengine.VectorBackend)
	if !ok {
		t.Fatal("engine does not implement metaengine.VectorBackend")
	}

	suffix := fmt.Sprintf("_%d", time.Now().UnixNano())
	col := "dim_guard" + suffix
	col3d := "dim_guard_3d" + suffix
	colEmpty := "dim_guard_empty" + suffix

	if err := vb.VectorInsert(
		ctx,
		col,
		metaengine.Embedding{ID: "a", Values: []float32{1, 0}},
	); err != nil {
		t.Fatalf("establishing insert: %v", err)
	}

	err := vb.VectorInsert(ctx, col, metaengine.Embedding{ID: "b", Values: []float32{1, 0, 0}})
	if !errors.Is(err, metaengine.ErrVectorDimensionMismatch) {
		t.Fatalf("mismatching insert err = %v, want ErrVectorDimensionMismatch", err)
	}

	err = vb.VectorInsert(ctx, colEmpty, metaengine.Embedding{ID: "z", Values: nil})
	if !errors.Is(err, metaengine.ErrVectorDimensionMismatch) {
		t.Fatalf("zero-dimension insert err = %v, want ErrVectorDimensionMismatch", err)
	}

	if err := vb.VectorInsert(
		ctx,
		col,
		metaengine.Embedding{ID: "a", Values: []float32{0, 1}},
	); err != nil {
		t.Fatalf("same-dimension upsert: %v", err)
	}

	if err := vb.VectorInsert(
		ctx,
		col3d,
		metaengine.Embedding{ID: "c", Values: []float32{1, 0, 0}},
	); err != nil {
		t.Fatalf("other-collection insert: %v", err)
	}
}
