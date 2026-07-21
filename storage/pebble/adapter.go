package pebble

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/cockroachdb/pebble"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
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
	casMu      sync.Mutex // serializes SetIfAbsent check-then-set within this instance
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
// Returns ErrNilDatabase if db is nil.
func NewKVStore(database *pebble.DB, opts ...KVOption) (kv.Store, error) {
	if database == nil {
		return nil, ErrNilDatabase
	}

	adapter := &KVAdapter{
		database:   database,
		syncWrites: false,
		owned:      true,
	}

	for _, opt := range opts {
		opt(adapter)
	}

	return adapter, nil
}

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
func (adapter *KVAdapter) Get(_ context.Context, key []byte) ([]byte, error) {
	err := adapter.checkClosed()
	if err != nil {
		return nil, err
	}

	val, closer, err := adapter.database.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, kv.ErrNotFound
		}

		return nil, errorfamily.WrapInfrastructure(err, "pebble.adapter.get",
			fmt.Sprintf("get %q", key))
	}

	defer func() { _ = closer.Close() }()

	return slices.Clone(val), nil
}

// Has implements [kv.Reader.Has].
func (adapter *KVAdapter) Has(_ context.Context, key []byte) (bool, error) {
	err := adapter.checkClosed()
	if err != nil {
		return false, err
	}

	_, closer, err := adapter.database.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return false, nil
		}

		return false, errorfamily.WrapInfrastructure(err, "pebble.adapter.has",
			fmt.Sprintf("has %q", key))
	}

	_ = closer.Close()

	return true, nil
}

// NewIterator implements [kv.Reader.NewIterator].
// A nil or empty prefix iterates over all keys.
func (adapter *KVAdapter) NewIterator(_ context.Context, prefix []byte) (kv.Iterator, error) {
	err := adapter.checkClosed()
	if err != nil {
		return nil, err
	}

	opts := &pebble.IterOptions{}
	if len(prefix) > 0 {
		opts.LowerBound = prefix
		opts.UpperBound = prefixUpperBound(prefix)
	}

	iter, err := adapter.database.NewIter(opts)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "pebble.adapter.new_iterator",
			"create iterator")
	}

	return &pebbleIterator{iter: iter}, nil
}

// ── Writer ───────────────────────────────────────────────────

// Set implements [kv.Writer.Set].
func (adapter *KVAdapter) Set(_ context.Context, key, value []byte) error {
	err := adapter.checkClosed()
	if err != nil {
		return err
	}

	err = adapter.database.Set(key, value, adapter.writeOptions())
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "pebble.adapter.set",
			fmt.Sprintf("set %q", key))
	}

	return nil
}

// Delete implements [kv.Writer.Delete].
func (adapter *KVAdapter) Delete(_ context.Context, key []byte) error {
	err := adapter.checkClosed()
	if err != nil {
		return err
	}

	err = adapter.database.Delete(key, adapter.writeOptions())
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "pebble.adapter.delete",
			fmt.Sprintf("delete %q", key))
	}

	return nil
}

// SetIfAbsent implements [kv.ConditionalWriter].
//
// Pebble has no native compare-and-set, so check-then-set is serialized by a
// per-adapter mutex. This makes the operation atomic within a single KVAdapter
// instance (matching [kv.MemStore.SetIfAbsent]'s process-local guarantee). It is
// NOT safe against concurrent writers using a DIFFERENT KVAdapter on the same
// underlying *pebble.DB — a single shared adapter (the default) must be used.
func (adapter *KVAdapter) SetIfAbsent(_ context.Context, key, value []byte) (bool, error) {
	if err := adapter.checkClosed(); err != nil {
		return false, err
	}

	adapter.casMu.Lock()
	defer adapter.casMu.Unlock()

	_, closer, err := adapter.database.Get(key)
	if err != nil {
		if !errors.Is(err, pebble.ErrNotFound) {
			return false, errorfamily.WrapInfrastructure(err, "pebble.adapter.set_if_absent_get",
				fmt.Sprintf("set-if-absent get %q", key))
		}
	} else {
		_ = closer.Close()

		return false, nil // key already exists
	}

	if err := adapter.database.Set(key, value, adapter.writeOptions()); err != nil {
		return false, errorfamily.WrapInfrastructure(err, "pebble.adapter.set_if_absent_set",
			fmt.Sprintf("set-if-absent set %q", key))
	}

	return true, nil
}

// compile-time interface checks.
var (
	_ kv.Store             = (*KVAdapter)(nil)
	_ kv.ConditionalWriter = (*KVAdapter)(nil)
)

// Batch implements [kv.Writer.Batch].
func (adapter *KVAdapter) Batch(_ context.Context) (kv.Batch, error) {
	err := adapter.checkClosed()
	if err != nil {
		return nil, err
	}

	return &pebbleBatch{
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

	return closeAndWrap(adapter.database, "pebble.adapter.close", "close database")
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
