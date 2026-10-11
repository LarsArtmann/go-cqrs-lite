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
	//art-dupl:accept dialect twin — dimension-lock insert body mirrors the sibling LSM engine (dep-isolated modules)
	established, err := e.firstVectorDimension(collection)
	if err != nil {
		return err //nolint:wrapcheck // wrapped by the probe helper
	}

	if err := metaengine.CheckVectorDimension(
		collection,
		established,
		len(emb.Values),
	); err != nil {
		return fmt.Errorf("pebbleengine.VectorInsert: %w", err)
	}

	if err := e.db.Set(
		keycodec.VectorKey(collection, emb.ID),
		metaengine.EncodeVectorBinary(emb.Values),
		e.writeOptions(),
	); err != nil {
		return fmt.Errorf("pebbleengine.VectorInsert: %w", err)
	}

	metaKey := keycodec.VectorMetaKey(collection, emb.ID)
	if emb.Metadata == nil {
		if err := e.db.Delete(metaKey, e.writeOptions()); err != nil {
			return fmt.Errorf("pebbleengine.VectorInsert: clear metadata: %w", err)
		}

		return nil
	}

	if err := e.db.Set(metaKey, encodeJSON(emb.Metadata), e.writeOptions()); err != nil {
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
	defer metaengine.DeferClose(closer)

	var meta map[string]any
	if err := json.Unmarshal(value, &meta); err != nil {
		return nil, fmt.Errorf("metadata %s: %w", id, err)
	}

	return meta, nil
}

// VectorSearchPath reports the Go-scored scan path (implements
// [metaengine.VectorPathReporter]): this engine has no native vector
// distance function, so k-NN scans rows and scores via
// metaengine.VectorDistance (ADR-0140).
func (e *pebbleEngine) VectorSearchPath() string {
	return metaengine.VectorPathScan
}

var _ metaengine.VectorPathReporter = (*pebbleEngine)(nil)

// firstVectorDimension reads the stored dimension of the collection's first
// vector (0 when the collection is empty) for the insert-time dimension lock.
func (e *pebbleEngine) firstVectorDimension(collection string) (int, error) {
	iter, err := e.newPrefixIter(keycodec.VectorPrefix(collection))
	if err != nil {
		return 0, fmt.Errorf("pebbleengine.VectorInsert: dimension probe: %w", err)
	}
	defer metaengine.DeferClose(iter)

	if !iter.First() || !iter.Valid() {
		return 0, nil
	}

	values, err := metaengine.DecodeVectorAuto(iter.Value())
	if err != nil {
		return 0, fmt.Errorf("pebbleengine.VectorInsert: dimension probe: %w", err)
	}

	return len(values), nil
}
