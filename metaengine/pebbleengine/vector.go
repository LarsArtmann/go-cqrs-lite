package pebbleengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"sort"
	"strings"

	"github.com/cockroachdb/pebble"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
)

// --- VectorBackend (degraded, brute-force) ---
//
// Vector embeddings are stored JSON-encoded under the "vec\x00<col>\x00"
// prefix. VectorSearch scans the collection prefix and computes every
// distance in Go — O(N·D) per query, declared as ComplexityON + degraded in
// the profile. Suitable for small collections; for production scale, pair
// the store with an engine that provides ANN search.

func (e *pebbleEngine) VectorInsert(
	_ context.Context,
	collection string,
	emb metaengine.Embedding,
) error {
	if err := e.db.Set(keycodec.VectorKey(collection, emb.ID), encodeJSON(emb.Values), pebble.Sync); err != nil {
		return fmt.Errorf("pebbleengine.VectorInsert: %w", err)
	}

	return nil
}

func (e *pebbleEngine) VectorSearch(
	_ context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
) ([]metaengine.VectorResult, error) {
	prefix := keycodec.VectorPrefix(collection)

	iter, err := e.newPrefixIter(prefix)
	if err != nil {
		return nil, fmt.Errorf("pebbleengine.VectorSearch: %w", err)
	}
	defer metaengine.DeferClose(iter)

	var results []metaengine.VectorResult

	for iter.First(); iter.Valid(); iter.Next() {
		id := strings.TrimPrefix(string(iter.Key()), string(prefix))

		vec, err := decodeVector(iter.Value())
		if err != nil {
			return nil, fmt.Errorf("pebbleengine.VectorSearch: decode %s: %w", id, err)
		}

		results = append(results, metaengine.VectorResult{
			ID:       id,
			Distance: metaengine.VectorDistance(query, vec, metric),
		})
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("pebbleengine.VectorSearch: %w", err)
	}

	return topKNearest(results, k), nil
}

// decodeVector decodes a JSON-encoded embedding.
func decodeVector(data []byte) ([]float32, error) {
	var vec []float32
	if err := json.Unmarshal(data, &vec); err != nil {
		return nil, err
	}

	return vec, nil
}

// topKNearest sorts ascending by distance (the "dot" metric is negated by
// VectorDistance so ascending is always nearest-first) and truncates to k.
func topKNearest(results []metaengine.VectorResult, k int) []metaengine.VectorResult {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	if k > 0 && k < len(results) {
		results = results[:k]
	}

	return results
}
