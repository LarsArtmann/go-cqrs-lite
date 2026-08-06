package badgerengine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/dgraph-io/badger/v4"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- MapBackend ---

func (e *badgerEngine) MapSet(_ context.Context, col string, key, value any) error {
	k := mapKey(col, encodeKeyStr(key))
	val := encodeJSON(value)

	return e.db.Update(func(txn *badger.Txn) error {
		return txn.Set(k, val)
	})
}

func (e *badgerEngine) MapGet(_ context.Context, col string, key any) (any, bool, error) {
	k := mapKey(col, encodeKeyStr(key))

	var result any

	found := false

	err := e.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(k)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil
			}

			return err
		}

		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		result = decodeJSON(val)
		found = true

		return nil
	})

	return result, found, err
}

func (e *badgerEngine) MapDelete(_ context.Context, col string, key any) error {
	k := mapKey(col, encodeKeyStr(key))

	return e.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(k)
	})
}

// --- MapUpdater ---

func (e *badgerEngine) MapUpdate(
	_ context.Context,
	col string,
	key any,
	update func(prev any) any,
) error {
	k := mapKey(col, encodeKeyStr(key))

	// The read-modify-write must be atomic: concurrent MapUpdate calls on the
	// same key would each read the same prev value and the last writer wins,
	// silently dropping updates. The engine-wide mutex ensures atomicity.
	e.mu.Lock()
	defer e.mu.Unlock()

	var prev any

	err := e.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(k)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil
			}

			return err
		}

		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		prev = decodeJSON(val)

		return nil
	})
	if err != nil {
		return err
	}

	newVal := update(prev)

	return e.db.Update(func(txn *badger.Txn) error {
		return txn.Set(k, encodeJSON(newVal))
	})
}

// --- ScanBackend ---

// kvPair pairs a Badger key with its decoded value for sorting.
type kvPair struct {
	key   []byte
	value any
}

func (e *badgerEngine) MapScan(
	_ context.Context,
	col string,
	filterFn func(item any) bool,
	sortFunc func(a, b any) int,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	prefix := collectionPrefix(col)

	var pairs []kvPair

	err := e.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{
			Prefix: prefix,
		})
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()

			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			decoded := decodeJSON(val)

			if filterFn != nil && !filterFn(decoded) {
				continue
			}

			pairs = append(pairs, kvPair{
				key:   append([]byte(nil), item.Key()...),
				value: decoded,
			})
		}

		return nil
	})
	if err != nil {
		return metaengine.ScanResult{}, err
	}

	pairs = sortAndPaginate(pairs, sortFunc, cursor, limit)

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

// sortAndPaginate sorts pairs by value (with byte-key tiebreak for determinism),
// applies keyset pagination, and truncates to limit+1.
func sortAndPaginate(pairs []kvPair, sortFn func(a, b any) int, cursor any, limit int) []kvPair {
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

	truncLimit := 0
	if limit > 0 {
		truncLimit = limit + 1
	}

	if truncLimit > 0 && len(pairs) > truncLimit {
		pairs = pairs[:truncLimit]
	}

	return pairs
}

var _ = fmt.Sprintf // suppress unused import
