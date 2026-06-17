package pebble

import (
	"errors"
	"fmt"
	"slices"
	"sync/atomic"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/kv/v2"
)

// maxByteValue is the largest value a single byte can hold (0xff).
const maxByteValue = 0xff

// KVAdapter adapts a [*pebble.DB] to the [kv.Store] interface.
//
// It is the first concrete consumer of the kv.Store abstraction, proving the
// interface is fit for purpose against a real LSM-tree storage engine.
//
// Ownership semantics:
//   - By default the adapter owns the database: Close calls db.Close.
//   - Pass [WithBorrowedDB] when the caller manages the lifecycle (e.g.
//     when sharing one *pebble.DB across multiple stores via Backend).
type KVAdapter struct {
	database   *pebble.DB
	syncWrites bool
	owned      bool
	closed     atomic.Bool
}

// KVOption configures a [KVAdapter].
type KVOption func(*KVAdapter)

// WithKVSyncWrites enables synchronous writes (pebble.Sync).
// Without this option writes are asynchronous for higher throughput.
func WithKVSyncWrites() KVOption {
	return func(adapter *KVAdapter) { adapter.syncWrites = true }
}

// WithBorrowedDB tells the adapter it does NOT own the *pebble.DB.
// Close becomes a no-op; the caller is responsible for closing the database.
func WithBorrowedDB() KVOption {
	return func(adapter *KVAdapter) { adapter.owned = false }
}

// NewKVStore wraps a [*pebble.DB] as a [kv.Store].
// Panics if db is nil.
//
// By default the adapter owns the database. Use [WithBorrowedDB] to share
// the database across multiple consumers.
func NewKVStore(database *pebble.DB, opts ...KVOption) kv.Store {
	if database == nil {
		panic("pebble: NewKVStore called with nil db")
	}

	adapter := &KVAdapter{ //nolint:exhaustruct // closed is zero-value
		database:   database,
		syncWrites: false,
		owned:      true,
	}

	for _, opt := range opts {
		opt(adapter)
	}

	return adapter
}

// compile-time interface check.
var _ kv.Store = (*KVAdapter)(nil)

func (adapter *KVAdapter) writeOptions() *pebble.WriteOptions {
	if adapter.syncWrites {
		return pebble.Sync
	}

	return nil
}

func (adapter *KVAdapter) checkClosed() error {
	if adapter.closed.Load() {
		return kv.ErrClosed
	}

	return nil
}

// ── Reader ───────────────────────────────────────────────────

// Get implements [kv.Reader.Get].
func (adapter *KVAdapter) Get(key []byte) ([]byte, error) {
	err := adapter.checkClosed()
	if err != nil {
		return nil, err
	}

	val, closer, err := adapter.database.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, kv.ErrNotFound
		}

		return nil, fmt.Errorf("pebble: get %q: %w", key, err)
	}

	defer func() { _ = closer.Close() }()

	return slices.Clone(val), nil
}

// Has implements [kv.Reader.Has].
func (adapter *KVAdapter) Has(key []byte) (bool, error) {
	err := adapter.checkClosed()
	if err != nil {
		return false, err
	}

	_, closer, err := adapter.database.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return false, nil
		}

		return false, fmt.Errorf("pebble: has %q: %w", key, err)
	}

	_ = closer.Close()

	return true, nil
}

// NewIterator implements [kv.Reader.NewIterator].
// A nil or empty prefix iterates over all keys.
func (adapter *KVAdapter) NewIterator(prefix []byte) (kv.Iterator, error) {
	err := adapter.checkClosed()
	if err != nil {
		return nil, err
	}

	opts := &pebble.IterOptions{} //nolint:exhaustruct // only Lower/Upper bound needed
	if len(prefix) > 0 {
		opts.LowerBound = prefix
		opts.UpperBound = prefixUpperBound(prefix)
	}

	iter, err := adapter.database.NewIter(opts)
	if err != nil {
		return nil, fmt.Errorf("pebble: new iterator: %w", err)
	}

	return &pebbleIterator{iter: iter}, nil //nolint:exhaustruct // positioned is zero-value
}

// ── Writer ───────────────────────────────────────────────────

// Set implements [kv.Writer.Set].
func (adapter *KVAdapter) Set(key, value []byte) error {
	err := adapter.checkClosed()
	if err != nil {
		return err
	}

	err = adapter.database.Set(key, value, adapter.writeOptions())
	if err != nil {
		return fmt.Errorf("pebble: set %q: %w", key, err)
	}

	return nil
}

// Delete implements [kv.Writer.Delete].
func (adapter *KVAdapter) Delete(key []byte) error {
	err := adapter.checkClosed()
	if err != nil {
		return err
	}

	err = adapter.database.Delete(key, adapter.writeOptions())
	if err != nil {
		return fmt.Errorf("pebble: delete %q: %w", key, err)
	}

	return nil
}

// Batch implements [kv.Writer.Batch].
func (adapter *KVAdapter) Batch() (kv.Batch, error) {
	err := adapter.checkClosed()
	if err != nil {
		return nil, err
	}

	return &pebbleBatch{ //nolint:exhaustruct // committed is zero-value
		batch:      adapter.database.NewBatch(),
		commitOpts: adapter.writeOptions(),
	}, nil
}

// ── Closer ───────────────────────────────────────────────────

// Close releases the database if the adapter owns it.
// With [WithBorrowedDB], Close is a no-op.
func (adapter *KVAdapter) Close() error {
	if !adapter.closed.CompareAndSwap(false, true) {
		return nil
	}

	if !adapter.owned {
		return nil
	}

	err := adapter.database.Close()
	if err != nil {
		return fmt.Errorf("pebble: close database: %w", err)
	}

	return nil
}

// ── prefixUpperBound ─────────────────────────────────────────

// prefixUpperBound returns the smallest byte slice that sorts immediately
// after all keys with the given prefix. It increments the last byte; if that
// overflows (0xff), it drops that byte and tries the previous one.
func prefixUpperBound(prefix []byte) []byte {
	end := make([]byte, len(prefix))
	copy(end, prefix)

	for i := range slices.Backward(end) {
		if end[i] < maxByteValue {
			end[i]++

			return end[:i+1]
		}
	}

	// All bytes are 0xff — need prefix + 0xff.
	result := make([]byte, len(prefix)+1)
	copy(result, prefix)
	result[len(prefix)] = maxByteValue

	return result
}

// ── pebbleIterator ───────────────────────────────────────────

type pebbleIterator struct {
	iter       *pebble.Iterator
	positioned bool
	closed     bool
}

var _ kv.Iterator = (*pebbleIterator)(nil)

func (it *pebbleIterator) Next() bool {
	if it.closed {
		return false
	}

	if !it.positioned {
		it.positioned = true

		return it.iter.First()
	}

	return it.iter.Next()
}

func (it *pebbleIterator) Key() []byte {
	return slices.Clone(it.iter.Key())
}

func (it *pebbleIterator) Value() []byte {
	return slices.Clone(it.iter.Value())
}

func (it *pebbleIterator) Error() error {
	err := it.iter.Error()
	if err != nil {
		return fmt.Errorf("pebble: iterator error: %w", err)
	}

	return nil
}

func (it *pebbleIterator) Close() error {
	if it.closed {
		return nil
	}

	it.closed = true

	err := it.iter.Close()
	if err != nil {
		return fmt.Errorf("pebble: close iterator: %w", err)
	}

	return nil
}

// ── pebbleBatch ──────────────────────────────────────────────

type pebbleBatch struct {
	batch      *pebble.Batch
	commitOpts *pebble.WriteOptions
	committed  bool
}

var _ kv.Batch = (*pebbleBatch)(nil)

func (batch *pebbleBatch) Set(key, value []byte) error {
	err := batch.batch.Set(key, value, nil)
	if err != nil {
		return fmt.Errorf("pebble: batch set %q: %w", key, err)
	}

	return nil
}

func (batch *pebbleBatch) Delete(key []byte) error {
	err := batch.batch.Delete(key, nil)
	if err != nil {
		return fmt.Errorf("pebble: batch delete %q: %w", key, err)
	}

	return nil
}

func (batch *pebbleBatch) Commit() error {
	if batch.committed {
		return nil
	}

	batch.committed = true

	err := batch.batch.Commit(batch.commitOpts)
	if err != nil {
		return fmt.Errorf("pebble: batch commit: %w", err)
	}

	return nil
}

func (batch *pebbleBatch) Close() error {
	if batch.committed {
		return nil
	}

	err := batch.batch.Close()
	if err != nil {
		return fmt.Errorf("pebble: batch close: %w", err)
	}

	return nil
}
