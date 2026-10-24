// Package kv defines a minimal, backend-agnostic interface for embedded
// key-value stores with ordered iteration and atomic batch writes.
//
// # Durable Backing (contract 30 / ADR-0142, status 2026-10-24)
//
// Under the all-durable-I/O-through-metaengine rule, NEW durable KV state
// belongs on a metaengine engine (MapBackend/SetBackend collections), not on
// a hand-wired kv.Store over a database driver. This package's durable scope
// going forward is the TYPED layer over engines and caches
// ([TypedStore], [ViewStore], [Cache]); its only durable implementors today
// are the legacy storage/pebble and storage/bbolt adapters, which are
// removed at v5. [NewMemStore] stays the canonical in-memory/testing Store.
// Whether the durable adapters grow a metaengine Map backing or the package
// fully demotes to cache/typed layering is the open G1 owner verdict — until
// then this note IS the guidance: do not add new satellite-SQL kv backends.
//
// The interface matches the common denominator across Pebble, BadgerDB, and
// bbolt: byte-slice keys with lexicographic ordering, prefix iteration, and
// atomic multi-key batches.
//
// # Why Another KV Interface?
//
// No existing Go KV meta-API provides all three operations that an event store
// needs: ordered iteration, atomic batch writes, and byte-slice keys. gokv
// (828★) lacks iteration and batch writes. valkeyrie (307★) targets
// distributed stores and has no batch writes. See
// docs/research/kv-store-abstraction-research.md for the full analysis.
//
// # Interface Segregation
//
// [Store] composes [Reader], [Writer], and [io.Closer]. Callers can accept
// [Reader] or [Writer] individually to express narrow dependencies:
//
//	func loadSnapshot(r kv.Reader, key []byte) ([]byte, error) {
//	    return r.Get(key)
//	}
//
// # Keys and Values
//
// Keys are raw byte slices with lexicographic ordering. Values are raw byte
// slices. The package has no marshalling opinion — callers handle serialization.
//
// # In-Memory Implementation
//
// [NewMemStore] returns a [Store] backed by a sorted in-memory map. It is
// safe for concurrent use and intended for testing and single-process
// scenarios.
//
//	s := kv.NewMemStore()
//	defer s.Close()
//
//	_ = s.Set([]byte("user:1"), []byte("alice"))
//	val, _ := s.Get([]byte("user:1"))
//	fmt.Println(string(val)) // alice
//
// # Atomic Batches
//
// Use [Store.Batch] to group multiple Set/Delete operations into a single
// atomic commit:
//
//	batch, _ := s.Batch()
//	_ = batch.Set([]byte("a"), []byte("1"))
//	_ = batch.Set([]byte("b"), []byte("2"))
//	_ = batch.Delete([]byte("old"))
//	_ = batch.Commit()
//
// If [Batch.Close] is called before [Batch.Commit], all pending operations
// are discarded.
//
// # Iteration
//
// [Store.NewIterator] returns an [Iterator] over keys matching a prefix in
// lexicographic order. A nil prefix iterates over all keys:
//
//	iter, _ := s.NewIterator([]byte("user:"))
//	defer iter.Close()
//
//	for iter.Next() {
//	    fmt.Printf("%s = %s\n", iter.Key(), iter.Value())
//	}
//
//	if err := iter.Error(); err != nil {
//	    log.Fatal(err)
//	}
package kv
