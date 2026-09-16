package badgerengine

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"

	"github.com/dgraph-io/badger/v4"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
)

// --- VectorBackend (degraded, brute-force) ---
//
// Mirrors the pebble/bbolt precedent: embeddings are stored under the
// "vec\x00<col>\x00" prefix as binary float32 payloads
// (metaengine.EncodeVectorBinary); rows written by pre-binary versions as
// bare JSON arrays keep decoding via metaengine.DecodeVectorAuto.
// VectorSearch prefix-scans and computes every distance in Go — O(N·D) per
// query, declared ComplexityON + degraded in the profile. Metadata lives in
// a separate "vecm" key family so the vector format is byte-identical across
// the LSM engines.

func (e *badgerEngine) VectorInsert(
	_ context.Context,
	collection string,
	emb metaengine.Embedding,
) error {
	established, err := e.firstVectorDimension(collection)
	if err != nil {
		return err //nolint:wrapcheck // wrapped by the probe helper
	}

	if err := metaengine.CheckVectorDimension(collection, established, len(emb.Values)); err != nil {
		return fmt.Errorf("badgerengine.VectorInsert: %w", err)
	}

	k := keycodec.VectorKey(collection, emb.ID)
	val := metaengine.EncodeVectorBinary(emb.Values)

	metaKey := keycodec.VectorMetaKey(collection, emb.ID)
	var metaVal []byte
	if emb.Metadata != nil {
		metaVal = encodeJSON(emb.Metadata)
	}

	return e.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(k, val); err != nil {
			return fmt.Errorf("badgerengine.VectorInsert: %w", err)
		}

		// Upsert semantics: a metadata-free insert clears stale metadata.
		if metaVal == nil {
			return txn.Delete(metaKey)
		}

		return txn.Set(metaKey, metaVal)
	})
}

func (e *badgerEngine) VectorSearch(
	_ context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
) ([]metaengine.VectorResult, error) {
	prefix := keycodec.VectorPrefix(collection)

	var results []metaengine.VectorResult

	err := e.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{Prefix: prefix})
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			id := strings.TrimPrefix(string(item.Key()), string(prefix))

			val, err := item.ValueCopy(nil)
			if err != nil {
				return fmt.Errorf("badgerengine.VectorSearch: value %s: %w", id, err)
			}

			vec, err := metaengine.DecodeVectorAuto(val)
			if err != nil {
				return fmt.Errorf("badgerengine.VectorSearch: decode %s: %w", id, err)
			}

			results = append(results, metaengine.VectorResult{
				ID:       id,
				Distance: metaengine.VectorDistance(query, vec, metric),
			})
		}

		return nil
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // already prefixed inside the view
	}

	return metaengine.TopKNearest(results, k), nil
}

// VectorSearchFiltered is the metadata-filtered k-NN path: scan the vector
// prefix, load each embedding's metadata ("vecm" keys — absent means no
// metadata), filter, and only then score survivors.
func (e *badgerEngine) VectorSearchFiltered(
	_ context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
	filters []metaengine.VectorFilter,
) ([]metaengine.VectorResult, error) {
	prefix := keycodec.VectorPrefix(collection)

	var results []metaengine.VectorResult

	err := e.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{Prefix: prefix})
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			id := strings.TrimPrefix(string(item.Key()), string(prefix))

			meta, err := vectorMetadata(txn, collection, id)
			if err != nil {
				return fmt.Errorf("badgerengine.VectorSearchFiltered: %w", err)
			}

			if !metaengine.VectorMatchesFilters(meta, filters) {
				continue
			}

			val, err := item.ValueCopy(nil)
			if err != nil {
				return fmt.Errorf("badgerengine.VectorSearchFiltered: value %s: %w", id, err)
			}

			vec, err := metaengine.DecodeVectorAuto(val)
			if err != nil {
				return fmt.Errorf("badgerengine.VectorSearchFiltered: decode %s: %w", id, err)
			}

			results = append(results, metaengine.VectorResult{
				ID:       id,
				Distance: metaengine.VectorDistance(query, vec, metric),
			})
		}

		return nil
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // already prefixed inside the view
	}

	return metaengine.TopKNearest(results, k), nil
}

// vectorMetadata loads an embedding's metadata map inside an open view
// transaction; nil when none is stored.
func vectorMetadata(txn *badger.Txn, collection, id string) (map[string]any, error) {
	item, err := txn.Get(keycodec.VectorMetaKey(collection, id))
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, nil //nolint:nilnil // no metadata stored is not an error
		}

		return nil, err //nolint:wrapcheck // wrapped by caller
	}

	val, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err //nolint:wrapcheck // wrapped by caller
	}

	var meta map[string]any
	if err := json.Unmarshal(val, &meta); err != nil {
		return nil, fmt.Errorf("metadata %s: %w", id, err)
	}

	return meta, nil
}

// VectorSearchPath reports the Go-scored scan path (implements
// [metaengine.VectorPathReporter]): this engine has no native vector
// distance function, so k-NN scans rows and scores via
// metaengine.VectorDistance (ADR-0140).
func (e *badgerEngine) VectorSearchPath() string {
	return metaengine.VectorPathScan
}

var (
	_ metaengine.VectorPathReporter = (*badgerEngine)(nil)
)

// firstVectorDimension reads the stored dimension of the collection's first
// vector (0 when the collection is empty) for the insert-time dimension lock.
func (e *badgerEngine) firstVectorDimension(collection string) (int, error) {
	var established int

	err := e.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{Prefix: keycodec.VectorPrefix(collection)})
		defer iter.Close()

		iter.Rewind()
		if !iter.Valid() {
			return nil
		}

		val, err := iter.Item().ValueCopy(nil)
		if err != nil {
			return fmt.Errorf("badgerengine.VectorInsert: dimension probe: %w", err)
		}

		values, err := metaengine.DecodeVectorAuto(val)
		if err != nil {
			return fmt.Errorf("badgerengine.VectorInsert: dimension probe: %w", err)
		}

		established = len(values)

		return nil
	})
	if err != nil {
		return 0, err //nolint:wrapcheck // wrapped at the probe site
	}

	return established, nil
}
