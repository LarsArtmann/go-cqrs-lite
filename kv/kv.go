// Package kv defines a minimal, backend-agnostic interface for embedded
// key-value stores with ordered iteration and atomic batch writes.
//
// The interface is designed to match the common denominator across Pebble,
// BadgerDB, and bbolt: byte-slice keys with lexicographic ordering, prefix
// iteration, and atomic multi-key batches. No existing Go KV meta-API (gokv,
// valkeyrie, etc.) provides all three capabilities. See
// docs/research/kv-store-abstraction-research.md for the full analysis.
//
// # Interface Segregation
//
// The [Store] interface is split into [Reader], [Writer], and [Batch] so that
// callers can express narrow dependencies:
//
//	func loadSnapshot(r kv.Reader, key []byte) ([]byte, error) {
//	    return r.Get(key)
//	}
//
//	func saveSnapshot(w kv.Writer, key, val []byte) error {
//	    return w.Set(key, val)
//	}
//
// # Keys
//
// Keys are raw byte slices. Callers are responsible for key encoding and value
// serialization — the package intentionally has no marshalling opinion.
// Lexicographic byte ordering is the only ordering guarantee.
//
// # In-Memory Implementation
//
// The [NewMemStore] function returns a [Store] backed by a sorted in-memory map.
// It is safe for concurrent use and intended for testing and single-process
// scenarios.
//
//	s := kv.NewMemStore()
//	defer s.Close()
//
//	_ = s.Set([]byte("user:1"), []byte("alice"))
//	val, _ := s.Get([]byte("user:1"))
//	fmt.Println(string(val)) // alice
package kv

import "io"

// Store is the core key-value store interface combining read and write access.
type Store interface {
	Reader
	Writer
	io.Closer
}

// Reader provides read-only access to the store.
type Reader interface {
	// Get retrieves the value for the given key.
	// Returns [ErrNotFound] if the key does not exist.
	Get(key []byte) ([]byte, error)

	// Has reports whether a key exists without reading the value.
	Has(key []byte) (bool, error)

	// NewIterator returns an iterator over keys matching the given prefix.
	// A nil prefix iterates over all keys.
	// Keys are yielded in lexicographic order.
	// The caller must call Close on the returned iterator.
	NewIterator(prefix []byte) (Iterator, error)
}

// Writer provides write access to the store.
type Writer interface {
	// Set stores the value for the given key.
	Set(key, value []byte) error

	// Delete removes the value for the given key.
	// Deleting a non-existent key is a no-op.
	Delete(key []byte) error

	// Batch returns a new [Batch] for atomic writes.
	// All operations queued on the batch are committed atomically on
	// [Batch.Commit].
	Batch() (Batch, error)
}

// Iterator yields key-value pairs in lexicographic key order.
// Iterator is not safe for concurrent use by multiple goroutines.
type Iterator interface {
	// Next advances to the next key-value pair.
	// Returns false when exhausted or on error (check [Iterator.Error]).
	Next() bool

	// Key returns the current key. Only valid after [Iterator.Next] returns true.
	Key() []byte

	// Value returns the current value. Only valid after [Iterator.Next] returns true.
	Value() []byte

	// Error returns any error encountered during iteration.
	// Returns nil if no error occurred.
	Error() error

	// Close releases iterator resources.
	// Must be called exactly once when iteration is complete.
	Close() error
}

// Batch collects write operations for atomic commit.
// A Batch is not safe for concurrent use by multiple goroutines.
type Batch interface {
	// Set queues a set operation.
	Set(key, value []byte) error

	// Delete queues a delete operation.
	Delete(key []byte) error

	// Commit applies all queued operations atomically.
	// After Commit the batch is closed and cannot be reused.
	Commit() error

	// Close releases batch resources.
	// Uncommitted operations are discarded.
	// Calling Close after Commit is a no-op.
	Close() error
}
