package bboltengine

import (
	"bytes"
	"context"
	"sort"
	"slices"

	bolt "go.etcd.io/bbolt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- MapBackend ---
//art-dupl:accept cross-module KV engine pattern — separate go.mod

func (e *bboltEngine) MapSet(_ context.Context, col string, key, value any) error {
	k := mapKey(col, encodeKeyStr(key))
	val := encodeJSON(value)

	return e.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		return bucket.Put(k, val) //nolint:wrapcheck // bbolt error is self-describing
	})
}

func (e *bboltEngine) MapGet(_ context.Context, col string, key any) (any, bool, error) {
	k := mapKey(col, encodeKeyStr(key))

	var result any

	found := false

	err := e.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		val := bucket.Get(k)
		if val == nil {
			return nil
		}

		result = decodeJSON(slices.Clone(val))
		found = true

		return nil
	})

	return result, found, err //nolint:wrapcheck // passthrough
}

func (e *bboltEngine) MapDelete(_ context.Context, col string, key any) error {
	k := mapKey(col, encodeKeyStr(key))

	return e.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		return bucket.Delete(k) //nolint:wrapcheck // bbolt error is self-describing
	})
}

// --- MapUpdater ---

func (e *bboltEngine) MapUpdate(
	_ context.Context,
	col string,
	key any,
	update func(prev any) any,
) error {
	k := mapKey(col, encodeKeyStr(key))

	// bbolt's Update is single-writer: the read and write are atomic within
	// the same transaction. The e.mu prevents concurrent MapUpdate calls from
	// deadlocking on the write lock.
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))

		var prev any

		val := bucket.Get(k)
		if val != nil {
			prev = decodeJSON(slices.Clone(val))
		}

		newVal := update(prev)

		return bucket.Put(k, encodeJSON(newVal)) //nolint:wrapcheck // bbolt error is self-describing
	})
}

// --- ScanBackend ---

// kvPair pairs a bbolt key with its decoded value for sorting.
type kvPair struct {
	key   []byte
	value any
}

func (e *bboltEngine) MapScan(
	_ context.Context,
	col string,
	filterFn func(item any) bool,
	sortFunc func(a, b any) int,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	prefix := collectionPrefix(col)

	var pairs []kvPair

	err := e.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		c := bucket.Cursor()

		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			decoded := decodeJSON(slices.Clone(v))

			if filterFn != nil && !filterFn(decoded) {
				continue
			}

			pairs = append(pairs, kvPair{
				key:   append([]byte(nil), k...),
				value: decoded,
			})
		}

		return nil
	})
	if err != nil {
		return metaengine.ScanResult{}, err //nolint:wrapcheck // passthrough
	}

	pairs = sortAndPaginateKV(pairs, sortFunc, cursor, limit)

	hasMore := limit > 0 && len(pairs) > limit
	if hasMore {
		pairs = pairs[:limit]
	}

	results := make([]any, len(pairs))
	for i, p := range pairs {
		results[i] = p.value
	}

	return metaengine.ScanResult{Items: results, HasMore: hasMore}, nil
}

// sortAndPaginateKV sorts pairs by value (with byte-key tiebreak for
// determinism), applies keyset pagination, and truncates to limit+1.
//art-dupl:accept cross-module KV engine pattern — separate go.mod
func sortAndPaginateKV(pairs []kvPair, sortFn func(a, b any) int, cursor any, limit int) []kvPair {
	if sortFn != nil {
		sort.Slice(pairs, func(i, j int) bool {
			if c := sortFn(pairs[i].value, pairs[j].value); c != 0 {
				return c < 0
			}

			return bytes.Compare(pairs[i].key, pairs[j].key) < 0
		})
	}

	if cursor != nil && sortFn != nil {
		filtered := pairs[:0]

		for _, p := range pairs {
			if sortFn(p.value, cursor) <= 0 {
				continue
			}

			filtered = append(filtered, p)
		}

		pairs = filtered
	}

	if limit > 0 && len(pairs) > limit {
		pairs = pairs[:limit+1]
	}

	return pairs
}
