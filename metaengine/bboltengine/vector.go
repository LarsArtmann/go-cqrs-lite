package bboltengine

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	bolt "go.etcd.io/bbolt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
)

// --- VectorBackend (degraded, brute-force) ---
//
// Vector embeddings are stored JSON-encoded under the "vec\x00<col>\x00"
// prefix inside the cqrs_meta bucket. VectorSearch scans the prefix and
// computes every distance in Go — O(N·D) per query, declared as
// ComplexityON + degraded in the profile. Suitable for small collections;
// for production scale, pair the store with an engine that provides ANN
// search.

func (e *bboltEngine) VectorInsert(
	_ context.Context,
	collection string,
	emb metaengine.Embedding,
) error {
	k := keycodec.VectorKey(collection, emb.ID)

	return e.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		return bucket.Put(
			k,
			encodeJSON(emb.Values),
		) //nolint:wrapcheck // bbolt error is self-describing
	})
}

func (e *bboltEngine) VectorSearch(
	_ context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
) ([]metaengine.VectorResult, error) {
	prefix := keycodec.VectorPrefix(collection)

	var results []metaengine.VectorResult

	err := e.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		c := bucket.Cursor()

		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			id := strings.TrimPrefix(string(k), string(prefix))

			vec, err := metaengine.DecodeVectorJSON(v)
			if err != nil {
				return fmt.Errorf("bboltengine.VectorSearch: decode %s: %w", id, err)
			}

			results = append(results, metaengine.VectorResult{
				ID:       id,
				Distance: metaengine.VectorDistance(query, vec, metric),
			})
		}

		return nil
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	return metaengine.TopKNearest(results, k), nil
}
