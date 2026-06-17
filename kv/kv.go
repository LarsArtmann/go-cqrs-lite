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
