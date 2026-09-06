package pebbleengine

import (
	"context"
	"slices"
	"testing"

	"github.com/cockroachdb/pebble"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
)

// TestVectorSearch_LegacyJSONPayloadReadable pins the upgrade contract: rows
// written by pre-binary versions as bare JSON arrays keep decoding, and a
// collection may hold both formats at once.
func TestVectorSearch_LegacyJSONPayloadReadable(t *testing.T) {
	t.Parallel()

	eng := mustNewPebbleEngineInternal(t)
	e, ok := eng.(*pebbleEngine)
	if !ok {
		t.Fatalf("unexpected engine type %T", eng)
	}

	ctx := context.Background()
	col := "vec_legacy_json"

	if err := e.db.Set(keycodec.VectorKey(col, "old"), []byte("[1,0,0]"), pebble.Sync); err != nil {
		t.Fatalf("write legacy JSON row: %v", err)
	}

	if err := e.db.Set(
		keycodec.VectorKey(col, "orthogonal"),
		[]byte("[0,1,0]"),
		pebble.Sync,
	); err != nil {
		t.Fatalf("write legacy JSON row: %v", err)
	}

	if err := e.VectorInsert(
		ctx,
		col,
		metaengine.Embedding{ID: "fresh", Values: []float32{1, 0, 0}},
	); err != nil {
		t.Fatalf("VectorInsert fresh: %v", err)
	}

	raw, closer, err := e.db.Get(keycodec.VectorKey(col, "fresh"))
	if err != nil {
		t.Fatalf("read fresh payload: %v", err)
	}
	defer func() { _ = closer.Close() }()

	if len(raw) == 0 || raw[0] != 'b' {
		t.Errorf("fresh payload marker = %v, want 'b' (binary format)", raw[:1])
	}

	results, err := e.VectorSearch(ctx, col, []float32{1, 0, 0}, 2, "cosine")
	if err != nil {
		t.Fatalf("VectorSearch: %v", err)
	}

	ids := make([]string, 0, len(results))
	for _, r := range results {
		ids = append(ids, r.ID)
	}
	slices.Sort(ids)

	// The two [1,0,0] rows — one legacy JSON, one binary — both match at
	// distance 0; the orthogonal legacy row loses to k=2.
	if want := []string{"fresh", "old"}; !slices.Equal(ids, want) {
		t.Errorf("search IDs = %v, want %v", ids, want)
	}
}
