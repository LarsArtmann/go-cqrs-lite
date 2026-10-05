package bboltengine

import (
	"context"
	"slices"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
	bolt "go.etcd.io/bbolt"
)

// TestVectorSearch_LegacyJSONPayloadReadable pins the upgrade contract: rows
// written by pre-binary versions as bare JSON arrays keep decoding, and a
// collection may hold both formats at once.
func TestVectorSearch_LegacyJSONPayloadReadable(t *testing.T) {
	t.Parallel()

	eng, err := NewBboltEngine("")
	if err != nil {
		t.Skipf("bbolt not available: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })

	e, ok := eng.(*bboltEngine)
	if !ok {
		t.Fatalf("unexpected engine type %T", eng)
	}

	ctx := context.Background()
	col := "vec_legacy_json"

	legacy := map[string]string{
		"old":        "[1,0,0]",
		"orthogonal": "[0,1,0]",
	}

	err = e.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		for id, payload := range legacy {
			if err := bucket.Put(keycodec.VectorKey(col, id), []byte(payload)); err != nil {
				return err //nolint:wrapcheck // bbolt error is self-describing
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("write legacy JSON rows: %v", err)
	}

	if err := e.VectorInsert(
		ctx,
		col,
		metaengine.Embedding{ID: "fresh", Values: []float32{1, 0, 0}},
	); err != nil {
		t.Fatalf("VectorInsert fresh: %v", err)
	}

	err = e.db.View(func(tx *bolt.Tx) error {
		raw := tx.Bucket([]byte(bucketName)).Get(keycodec.VectorKey(col, "fresh"))
		if len(raw) == 0 || raw[0] != 'b' {
			t.Errorf("fresh payload marker = %v, want 'b' (binary format)", raw[:1])
		}

		return nil
	})
	if err != nil {
		t.Fatalf("read fresh payload: %v", err)
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
