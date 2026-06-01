// Package pebble provides an embedded key-value event store implementation
// using CockroachDB Pebble. It implements event.Store with optimistic
// concurrency control via per-aggregate locking.
//
// Use NewStore to create a store from an existing *pebble.DB.
package pebble
