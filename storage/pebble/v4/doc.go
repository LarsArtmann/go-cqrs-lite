// Package pebble provides embedded key-value stores backed by CockroachDB Pebble.
// It implements the full CQRS persistence stack: event store with journal,
// snapshot store, and checkpoint store.
//
// A single *pebble.DB backs all stores via disjoint key prefixes. The Backend
// facade owns the database and exposes every store; it is the recommended path:
//
//	backend, err := pebble.Open("data", pebble.DefaultOptions(), nil)
//	defer backend.Close()
//
//	eventStore := backend.EventStore()
//	snapStore  := backend.SnapshotStore()
//	cpStore    := backend.CheckpointStore()
//
// For manual wiring from an existing *pebble.DB, use NewStore,
// NewSnapshotStore, and NewCheckpointStore (each returns (store, error)).
//
// # Event Store
//
// EventStore implements event.Store with optimistic concurrency control via
// per-stream locking (sharded mutex pool). It also implements event.Journal
// and event.SeekableJournal for global event replay.
//
// Use NewStore to create a store from an existing *pebble.DB.
//
// # Snapshot Store
//
// SnapshotStore implements snapshot.SnapshotStore. One snapshot per stream
// is retained; saving an older version is silently ignored to prevent state
// regressions. Use NewSnapshotStore to create a store.
//
// # Checkpoint Store
//
// CheckpointStore implements event.CheckpointStore for projection positioning.
// One checkpoint per projection is retained. Load returns a zero-value
// event.Checkpoint when no checkpoint exists, enabling first-run detection
// without error inspection. Use NewCheckpointStore to create a store.
//
// # Envelope Format
//
// All three stores use CBOR-encoded envelopes with canonical (deterministic)
// encoding (RFC 7049). Event payloads are stored as raw bytes — no base64
// encoding. For backward compatibility, deserialization reads both CBOR
// envelopes (current) and JSON envelopes (legacy) via format sniffing.
//
// # Key Prefixes
//
//	cqrs_event:{type}:{id}:{version}      — per-stream event log
//	cqrs_journal:{nanoseconds}:{eventID}  — global event ordering index
//	cqrs_snapshot:{type}:{id}             — one snapshot per stream
//	cqrs_checkpoint:{projectionName}      — one checkpoint per projection
//
// The prefixes are disjoint — no prefix is a substring of another. This
// guarantees that scans over one store's keyspace never accidentally
// surface another store's data. If you add custom key prefixes via
// future options, ensure they do not collide with these four.
//
// # Logger
//
// All stores accept a *slog.Logger. Pass nil to disable logging — nil-safe
// logging is enforced throughout. When a logger is provided, stores emit
// Debug-level entries for save/load operations and Warn-level entries for
// corrupt data detection.
//
// # Recommended Options
//
// Use DefaultOptions for production-grade defaults: bloom filter policy per
// level, concurrent compactions, and 64MB block cache. For operational
// observability, use DefaultOptionsWithLogging which attaches an EventListener
// that logs write stalls, flushes, and compactions via slog.
//
//	opts := pebble.DefaultOptions()
//	backend, _ := pebble.Open("data", opts, slog.Default())
//
// # Metrics
//
// Backend.Metrics exposes pebble's internal telemetry: block cache hit rate,
// compaction debt, WAL bytes, memtable count. Use BlockCacheHitRate for the
// single most useful operational signal.
//
//	backend, _ := pebble.Open("data", nil, slog.Default())
//	rate := backend.Metrics().BlockCacheHitRate()
//
// # Close Semantics
//
// Unlike the SQL Backend (which borrows the *sql.DB and does NOT close it),
// the Pebble Backend OWNS the *pebble.DB. Calling backend.Close() closes the
// database AND all stores created from it. After Close, all store operations
// return ErrClosed.
//
//	backend, _ := pebble.Open("data", opts, slog.Default())
//	defer backend.Close() // closes DB + all stores
//
// When wiring stores manually (NewStore, NewSnapshotStore, etc.) from a
// shared *pebble.DB, the caller is responsible for closing the DB after all
// stores are done. Using Backend avoids this manual lifecycle management.
//
// # Backup & Recovery
//
// Backend.Checkpoint creates a point-in-time DB snapshot for backups:
//
//	backend.Checkpoint("backups/" + time.Now().Format("2006-01-02"))
//
// The checkpoint directory contains a complete, restorable Pebble DB.
// Upload to S3/GCS or copy to another machine.
//
// # Consistent Reads
//
// Backend.NewSnapshot returns a point-in-time consistent read view:
//
//	snap := backend.NewSnapshot()
//	defer snap.Close()
//	iter, _ := snap.NewIter(&pebble.IterOptions{
//	    LowerBound: []byte("cqrs_journal:"),
//	    UpperBound: []byte("cqrs_journal:\xff"),
//	})
package pebble
