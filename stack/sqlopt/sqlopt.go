// Package sqlopt provides shared helpers for converting a storage.SQLBackend
// into stack.Option values. It is the single home for the option-assembly
// logic used by every SQL-backed preset (sqlite, postgres, turso).
//
// This is a separate package within the stack module so that the base stack
// package does not acquire a storage dependency; consumers using non-SQL
// backends (memory, pebble) never import this package and never pull storage
// into their build.
package sqlopt

import (
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

// AllOptions assembles the full stack.Option set from a storage.SQLBackend.
// The event store is always present (eager); the lazy stores (command, query,
// snapshot, checkpoint) are included only when they construct successfully.
// This is the common case: one database backs every store.
func AllOptions(backend *storage.SQLBackend) []stack.Option {
	return append(EventStoreOptions(backend), QueryStoreOptions(backend)...)
}

// EventStoreOptions assembles options for the event-sourcing write model:
// the event store (eager), snapshots, and checkpoints. Used by multi-DB
// presets that isolate event stores on their own database.
func EventStoreOptions(backend *storage.SQLBackend) []stack.Option {
	opts := []stack.Option{stack.WithEventStore(backend.EventStore())}

	if snapStore, err := backend.SnapshotStore(); err == nil {
		opts = append(opts, stack.WithSnapshotStore(snapStore))
	}

	if cpStore, err := backend.CheckpointStore(); err == nil {
		opts = append(opts, stack.WithCheckpointStore(cpStore))
	}

	return opts
}

// QueryStoreOptions assembles options for the command and query audit stores.
// Used by multi-DB presets that isolate audit stores on their own database.
func QueryStoreOptions(backend *storage.SQLBackend) []stack.Option {
	var opts []stack.Option

	if cmdStore, err := backend.CommandStore(); err == nil {
		opts = append(opts, stack.WithCommandStore(cmdStore))
	}

	if queryStore, err := backend.QueryStore(); err == nil {
		opts = append(opts, stack.WithQueryStore(queryStore))
	}

	return opts
}
