// Package pebble provides embedded key-value stores backed by CockroachDB Pebble.
// It implements the full CQRS persistence stack: event store with journal,
// snapshot store, and checkpoint store.
//
// A single *pebble.DB can back all three stores via disjoint key prefixes:
//
//	db, _ := pebble.Open("data", &pebble.Options{})
//	eventStore  := pebble.NewStore(db, slog.Default())
//	snapStore   := pebble.NewSnapshotStore(db, slog.Default())
//	cpStore     := pebble.NewCheckpointStore(db, slog.Default())
//
// # Event Store
//
// EventStore implements event.Store with optimistic concurrency control via
// per-aggregate locking (sharded mutex pool). It also implements event.Journal
// and event.SeekableJournal for global event replay.
//
// Use NewStore to create a store from an existing *pebble.DB.
//
// # Snapshot Store
//
// SnapshotStore implements snapshot.SnapshotStore. One snapshot per aggregate
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
//	cqrs_event:{type}:{id}:{version}      — per-aggregate event log
//	cqrs_journal:{nanoseconds}:{eventID}  — global event ordering index
//	cqrs_snapshot:{type}:{id}             — one snapshot per aggregate
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
package pebble
