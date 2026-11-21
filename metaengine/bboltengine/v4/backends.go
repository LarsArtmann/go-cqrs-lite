package bboltengine

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	bolt "go.etcd.io/bbolt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
)

// keySegments is the collection/key/value split arity of the KV key form.
const keySegments = 3

// seqCounter returns (storing on first use) the per-key monotonic counter.
// The type assertion cannot fail: every value in m is stored by this helper
// as *atomic.Int64 at the same call site.
func seqCounter(m *sync.Map, key string) *atomic.Int64 {
	v, _ := m.LoadOrStore(key, new(atomic.Int64))

	return v.(*atomic.Int64) //nolint:forcetypeassert // stored by this helper as *atomic.Int64
}

// --- SetBackend ---
//art-dupl:accept cross-module KV engine pattern — separate go.mod

func (e *bboltEngine) SetAdd(_ context.Context, col string, key any) error {
	k := setKey(col, encodeKeyStr(key))

	return e.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))

		return bucket.Put(k, []byte{})
	})
}

func (e *bboltEngine) SetContains(_ context.Context, col string, key any) (bool, error) {
	k := setKey(col, encodeKeyStr(key))

	found := false

	err := e.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		val := bucket.Get(k)
		found = val != nil

		return nil
	})

	return found, err
}

// --- CounterBackend ---
//art-dupl:accept cross-module KV engine pattern — separate go.mod

func (e *bboltEngine) CounterIncrement(
	_ context.Context,
	col string,
	deltas metaengine.Delta,
) error {
	return e.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))

		for k, d := range deltas {
			counterK := counterKey(col, k)

			var current int64

			val := bucket.Get(counterK)
			if val != nil {
				current = decodeCounterValue(val)
			}

			if err := bucket.Put(counterK, encodeCounterValue(current+d)); err != nil {
				return err
			}
		}

		return nil
	})
}

func (e *bboltEngine) CounterGet(_ context.Context, col string) (map[string]int64, error) {
	prefix := counterPrefix(col)

	result := make(map[string]int64)

	err := e.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		c := bucket.Cursor()

		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			keyStr := string(k)

			parts := strings.SplitN(keyStr, sep, keySegments)
			if len(parts) < keySegments {
				continue
			}

			result[parts[2]] = decodeCounterValue(v)
		}

		return nil
	})

	return result, err
}

// --- MultimapBackend ---
//art-dupl:accept cross-module KV engine pattern — separate go.mod

func (e *bboltEngine) MultiAdd(_ context.Context, col string, key, value any) error {
	seq := e.nextMmSeq(col)
	k := multimapKey(col, encodeKeyStr(key), seq)

	val, err := keycodec.EncodeJSONErr(value)
	if err != nil {
		return fmt.Errorf("bboltengine.MultiAdd: %w", err)
	}

	return e.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))

		return bucket.Put(k, val)
	})
}

func (e *bboltEngine) MultiGet(_ context.Context, col string, key any) ([]any, error) {
	prefix := multimapPrefix(col, encodeKeyStr(key))

	var out []any

	err := e.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		c := bucket.Cursor()

		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			out = append(out, decodeJSON(cloneBytes(v)))
		}

		return nil
	})

	return out, err
}

func (e *bboltEngine) nextMmSeq(col string) int64 {
	return seqCounter(&e.mmSeq, col).Add(1)
}

// --- LogBackend ---
//art-dupl:accept cross-module KV engine pattern — separate go.mod

func (e *bboltEngine) LogAppend(_ context.Context, col string, value any) error {
	seq := e.nextLogSeq(col)
	k := logKey(col, seq)

	val, err := keycodec.EncodeJSONErr(value)
	if err != nil {
		return fmt.Errorf("bboltengine.LogAppend: %w", err)
	}

	return e.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))

		return bucket.Put(k, val)
	})
}

func (e *bboltEngine) LogTail(_ context.Context, col string, limit int) ([]any, error) {
	prefix := logPrefix(col)

	var all []any

	err := e.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		c := bucket.Cursor()

		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			all = append(all, decodeJSON(cloneBytes(v)))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if limit > 0 && len(all) > limit {
		all = all[len(all)-limit:]
	}

	if all == nil {
		all = []any{}
	}

	return all, nil
}

func (e *bboltEngine) nextLogSeq(col string) int64 {
	return seqCounter(&e.logSeq, col).Add(1)
}

// cloneBytes copies a byte slice, since bbolt values are only valid during the
// transaction. Uses append to avoid an extra import.
// art-dupl:accept cross-module KV engine pattern — separate go.mod
func cloneBytes(b []byte) []byte {
	return append([]byte(nil), b...)
}
