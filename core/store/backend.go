package store

import (
	"context"
	"errors"
	"io"
)

// Backend is the universal storage primitive for CQRS data.
// All persistence implementations (SQL, Pebble, Memory, etc.) satisfy this.
//
// Keys are byte slices with structured encoding determined by domain adapters:
//
//	"evt:{type}:{id}:{version:010d}"   — event store
//	"cmd:{id}"                          — command store
//	"proj:{name}:{key}"                 — projection store
//	"snap:{type}:{id}"                  — snapshot store
//	"chk:{name}"                        — checkpoint store
//
// Domain-specific stores are built as type-safe adapters over Backend.
// Adding a new backend = implement Backend once → ALL domain stores work.
type Backend interface {
	// Get retrieves the value for key. Returns ErrNotFound if missing.
	Get(ctx context.Context, key []byte) ([]byte, error)

	// Put stores value at key, overwriting if present.
	Put(ctx context.Context, key, value []byte) error

	// Delete removes key. No error if key doesn't exist.
	Delete(ctx context.Context, key []byte) error

	// Scan returns an iterator over all key-value pairs where key starts
	// with prefix. Keys are yielded in lexicographic order.
	Scan(ctx context.Context, prefix []byte) (Iterator, error)

	// Batch executes fn atomically. All operations within fn are committed
	// together or none are. Use for optimistic concurrency (read-then-write).
	Batch(ctx context.Context, fn func(Transaction) error) error

	io.Closer
}

// Iterator scans over key-value pairs in key order.
// Usage:
//
//	it, _ := backend.Scan(ctx, prefix)
//	defer it.Close()
//	for it.Next() {
//	    key := it.Key()
//	    val := it.Value()
//	}
type Iterator interface {
	Next() bool
	Key() []byte
	Value() []byte
	Err() error
	Close() error
}

// Transaction provides atomic read-modify-write within Batch.
type Transaction interface {
	Get(key []byte) ([]byte, error)
	Put(key, value []byte) error
	Delete(key []byte) error
}

// ErrNotFound indicates the requested key does not exist.
var ErrNotFound = errors.New("store: key not found")
