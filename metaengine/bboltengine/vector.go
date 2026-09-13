package bboltengine

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"strings"

	bolt "go.etcd.io/bbolt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
)

// --- VectorBackend (degraded, brute-force) ---
//
// Vector embeddings are stored under the "vec\x00<col>\x00" prefix inside
// the cqrs_meta bucket as binary float32 payloads
// (metaengine.EncodeVectorBinary); rows written by pre-binary versions as
// bare JSON arrays keep decoding via metaengine.DecodeVectorAuto.
// VectorSearch scans the prefix and computes every distance in Go —
// O(N·D) per query, declared as ComplexityON + degraded in the profile.
// Suitable for small collections; for production scale, pair the store with
// an engine that provides ANN search.

func (e *bboltEngine) VectorInsert(
	_ context.Context,
	collection string,
	emb metaengine.Embedding,
) error {
	k := keycodec.VectorKey(collection, emb.ID)
	metaKey := keycodec.VectorMetaKey(collection, emb.ID)
	metaValue := []byte(nil)
	if emb.Metadata != nil {
		metaValue = encodeJSON(emb.Metadata)
	}

	return e.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if err := bucket.Put(k, metaengine.EncodeVectorBinary(emb.Values)); err != nil {
			return err //nolint:wrapcheck // bbolt error is self-describing
		}

		if metaValue == nil {
			return bucket.Delete(metaKey) //nolint:wrapcheck // bbolt error is self-describing
		}

		return bucket.Put(metaKey, metaValue) //nolint:wrapcheck // bbolt error is self-describing
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

			vec, err := metaengine.DecodeVectorAuto(v)
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

// VectorSearchFiltered is the metadata-filtered k-NN path: it scans the
// vector prefix, loads each embedding's metadata (vecm key family — absent
// means no metadata), applies the filters, and only then scores survivors.
// Embeddings excluded by a filter never pay the distance computation.
func (e *bboltEngine) VectorSearchFiltered(
	_ context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
	filters []metaengine.VectorFilter,
) ([]metaengine.VectorResult, error) {
	prefix := keycodec.VectorPrefix(collection)

	var results []metaengine.VectorResult

	err := e.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		c := bucket.Cursor()

		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			id := strings.TrimPrefix(string(k), string(prefix))

			var meta map[string]any
			if raw := bucket.Get(keycodec.VectorMetaKey(collection, id)); raw != nil {
				if err := json.Unmarshal(raw, &meta); err != nil {
					return fmt.Errorf("bboltengine.VectorSearchFiltered: metadata %s: %w", id, err)
				}
			}

			if !metaengine.VectorMatchesFilters(meta, filters) {
				continue
			}

			vec, err := metaengine.DecodeVectorAuto(v)
			if err != nil {
				return fmt.Errorf("bboltengine.VectorSearchFiltered: decode %s: %w", id, err)
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
