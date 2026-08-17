package pebbleengine

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"

	"github.com/cockroachdb/pebble"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
)

// --- VectorBackend (degraded, brute-force) ---
//
// Vector embeddings are stored under the "vec\x00<col>\x00" prefix as
// binary float32 payloads (metaengine.EncodeVectorBinary); rows written by
// pre-binary versions as bare JSON arrays keep decoding via
// metaengine.DecodeVectorAuto. VectorSearch scans the collection prefix and
// computes every distance in Go — O(N·D) per query, declared as ComplexityON
// + degraded in the profile. Suitable for small collections; for production
// scale, pair the store with an engine that provides ANN search.

func (e *pebbleEngine) VectorInsert(
	_ context.Context,
	collection string,
	emb metaengine.Embedding,
) error {
	if err := e.db.Set(
		keycodec.VectorKey(collection, emb.ID),
		metaengine.EncodeVectorBinary(emb.Values),
		pebble.Sync,
	); err != nil {
		return fmt.Errorf("pebbleengine.VectorInsert: %w", err)
	}

	metaKey := keycodec.VectorMetaKey(collection, emb.ID)
	if emb.Metadata == nil {
		if err := e.db.Delete(metaKey, pebble.Sync); err != nil {
			return fmt.Errorf("pebbleengine.VectorInsert: clear metadata: %w", err)
		}

		return nil
	}

	if err := e.db.Set(metaKey, encodeJSON(emb.Metadata), pebble.Sync); err != nil {
		return fmt.Errorf("pebbleengine.VectorInsert: metadata: %w", err)
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
	// art-dupl:accept scan prologue; pgengine vector.go is a dep-isolated dialect twin
	if err != nil {
		return nil, fmt.Errorf("pebbleengine.VectorSearch: %w", err)
	}
	defer metaengine.DeferClose(iter)

	var results []metaengine.VectorResult

	for iter.First(); iter.Valid(); iter.Next() {
		id := strings.TrimPrefix(string(iter.Key()), string(prefix))

		vec, err := metaengine.DecodeVectorAuto(iter.Value())
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

	return metaengine.TopKNearest(results, k), nil
}

// VectorSearchFiltered is the metadata-filtered k-NN path: it scans the
// vector prefix, loads each embedding's metadata (vecm key family — absent
// means no metadata), applies the filters, and only then scores survivors.
// Embeddings excluded by a filter never pay the distance computation.
func (e *pebbleEngine) VectorSearchFiltered(
	_ context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
	filters []metaengine.VectorFilter,
) ([]metaengine.VectorResult, error) {
	prefix := keycodec.VectorPrefix(collection)

	iter, err := e.newPrefixIter(prefix)
	// art-dupl:accept scan prologue; pgengine vector.go is a dep-isolated dialect twin
	if err != nil {
		return nil, fmt.Errorf("pebbleengine.VectorSearchFiltered: %w", err)
	}
	defer metaengine.DeferClose(iter)

	var results []metaengine.VectorResult

	for iter.First(); iter.Valid(); iter.Next() {
		id := strings.TrimPrefix(string(iter.Key()), string(prefix))

		meta, err := e.vectorMetadata(collection, id)
		if err != nil {
			return nil, fmt.Errorf("pebbleengine.VectorSearchFiltered: %w", err)
		}

		if !metaengine.VectorMatchesFilters(meta, filters) {
			continue
		}

		vec, err := metaengine.DecodeVectorAuto(iter.Value())
		if err != nil {
			return nil, fmt.Errorf("pebbleengine.VectorSearchFiltered: decode %s: %w", id, err)
		}

		results = append(results, metaengine.VectorResult{
			ID:       id,
			Distance: metaengine.VectorDistance(query, vec, metric),
		})
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("pebbleengine.VectorSearchFiltered: %w", err)
	}

	return metaengine.TopKNearest(results, k), nil
}

// vectorMetadata loads an embedding's metadata map; nil when none is stored.
func (e *pebbleEngine) vectorMetadata(collection, id string) (map[string]any, error) {
	value, closer, err := e.db.Get(keycodec.VectorMetaKey(collection, id))
	if errors.Is(err, pebble.ErrNotFound) {
		return nil, nil //nolint:nilnil // no metadata stored is not an error
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // wrapped by caller
	}
	defer func() { _ = closer.Close() }()

	var meta map[string]any
	if err := json.Unmarshal(value, &meta); err != nil {
		return nil, fmt.Errorf("metadata %s: %w", id, err)
	}

	return meta, nil
}
