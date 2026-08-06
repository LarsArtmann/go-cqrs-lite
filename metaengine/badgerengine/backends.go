package badgerengine

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"

	"github.com/dgraph-io/badger/v4"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- SetBackend ---

func (e *badgerEngine) SetAdd(_ context.Context, col string, key any) error {
	k := setKey(col, encodeKeyStr(key))

	return e.db.Update(func(txn *badger.Txn) error {
		return txn.Set(k, []byte{})
	})
}

func (e *badgerEngine) SetContains(_ context.Context, col string, key any) (bool, error) {
	k := setKey(col, encodeKeyStr(key))

	found := false

	err := e.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(k)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil
			}

			return err
		}

		found = true

		return nil
	})

	return found, err
}

// --- CounterBackend ---

func (e *badgerEngine) CounterIncrement(
	_ context.Context,
	col string,
	deltas metaengine.Delta,
) error {
	return e.db.Update(func(txn *badger.Txn) error {
		for k, d := range deltas {
			ck := counterKey(col, k)

			var current int64

			item, err := txn.Get(ck)
			if err == nil {
				val, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}

				current = decodeCounterValue(val)
			} else if !errors.Is(err, badger.ErrKeyNotFound) {
				return err
			}

			if err := txn.Set(ck, encodeCounterValue(current+d)); err != nil {
				return err
			}
		}

		return nil
	})
}

func (e *badgerEngine) CounterGet(_ context.Context, col string) (map[string]int64, error) {
	prefix := counterPrefix(col)
	result := make(map[string]int64)

	err := e.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{Prefix: prefix})
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()

			keyStr := string(item.Key())

			parts := strings.SplitN(keyStr, sep, 3)
			if len(parts) < 3 {
				continue
			}

			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			result[parts[2]] = decodeCounterValue(val)
		}

		return nil
	})

	return result, err
}

// --- GraphBackend ---

func (e *badgerEngine) GraphAddEdge(_ context.Context, col string, edge metaengine.Edge) error {
	from := encodeKeyStr(edge.From)
	to := encodeKeyStr(edge.To)

	return e.db.Update(func(txn *badger.Txn) error {
		// Store edge in both directions for efficient neighbor lookup.
		if err := txn.Set(graphEdgeKey(col, from, to), []byte{}); err != nil {
			return err
		}

		return txn.Set(graphEdgeKey(col, to, from), []byte{})
	})
}

func (e *badgerEngine) GraphNeighbors(
	_ context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	nodeStr := encodeKeyStr(node)
	visited := map[string]bool{nodeStr: true}
	frontier := []string{nodeStr}

	var result []string

	for d := 0; d < depth && len(frontier) > 0; d++ {
		var next []string

		for _, n := range frontier {
			neighbors := e.scanGraphNeighbors(col, n)
			for _, nb := range neighbors {
				if !visited[nb] {
					visited[nb] = true
					result = append(result, nb)
					next = append(next, nb)
				}
			}
		}

		frontier = next
	}

	decoded := make([]any, len(result))
	for i, r := range result {
		decoded[i] = decodeJSON([]byte(r))
	}

	return decoded, nil
}

func (e *badgerEngine) scanGraphNeighbors(col, node string) []string {
	prefix := graphPrefixForward(col, node)

	var neighbors []string

	_ = e.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{Prefix: prefix})
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			keyStr := string(iter.Item().Key())

			parts := strings.SplitN(keyStr, sep, 4)
			if len(parts) < 4 {
				continue
			}

			neighbors = append(neighbors, parts[3])
		}

		return nil
	})

	return neighbors
}

// --- MultimapBackend ---

func (e *badgerEngine) MultiAdd(_ context.Context, col string, key, value any) error {
	seq := e.nextMmSeq(col)
	k := multimapKey(col, encodeKeyStr(key), seq)
	val := encodeJSON(value)

	return e.db.Update(func(txn *badger.Txn) error {
		return txn.Set(k, val)
	})
}

func (e *badgerEngine) MultiGet(_ context.Context, col string, key any) ([]any, error) {
	prefix := multimapPrefix(col, encodeKeyStr(key))

	var out []any

	err := e.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{Prefix: prefix})
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			val, err := iter.Item().ValueCopy(nil)
			if err != nil {
				return err
			}

			out = append(out, decodeJSON(val))
		}

		return nil
	})

	return out, err
}

func (e *badgerEngine) nextMmSeq(col string) int64 {
	actual, _ := e.mmSeq.LoadOrStore(col, &atomic.Int64{})

	return actual.(*atomic.Int64).Add(1)
}

// --- LogBackend ---

func (e *badgerEngine) LogAppend(_ context.Context, col string, value any) error {
	seq := e.nextLogSeq(col)
	k := logKey(col, seq)
	val := encodeJSON(value)

	return e.db.Update(func(txn *badger.Txn) error {
		return txn.Set(k, val)
	})
}

func (e *badgerEngine) LogTail(_ context.Context, col string, limit int) ([]any, error) {
	prefix := logPrefix(col)

	if limit <= 0 {
		var entries []any

		err := e.db.View(func(txn *badger.Txn) error {
			iter := txn.NewIterator(badger.IteratorOptions{Prefix: prefix})
			defer iter.Close()

			for iter.Rewind(); iter.Valid(); iter.Next() {
				val, err := iter.Item().ValueCopy(nil)
				if err != nil {
					return err
				}

				entries = append(entries, decodeJSON(val))
			}

			return nil
		})

		return entries, err
	}

	// For limit > 0: we need to iterate in reverse to get the tail.
	// Badger doesn't support reverse iteration directly, so we collect all
	// and take the last `limit` entries.
	var all []any

	err := e.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{Prefix: prefix})
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			val, err := iter.Item().ValueCopy(nil)
			if err != nil {
				return err
			}

			all = append(all, decodeJSON(val))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(all) <= limit {
		return all, nil
	}

	return all[len(all)-limit:], nil
}

func (e *badgerEngine) nextLogSeq(col string) int64 {
	actual, _ := e.logSeq.LoadOrStore(col, &atomic.Int64{})

	return actual.(*atomic.Int64).Add(1)
}
