// Package bbolt implements go-cqrs-lite event/snapshot/checkpoint stores
// backed by an embedded bbolt (B+tree) database.
//
// bbolt is a pure-Go, single-file, B+tree key-value store maintained by the
// etcd team. Unlike Pebble (LSM tree), bbolt uses a B+tree with a single-writer
// model: one read-write transaction at a time, unlimited concurrent readers.
// This makes writes fully serialized — no per-stream locking is needed.
//
// The module mirrors the storage/pebble package structure: EventStore,
// SnapshotStore, CheckpointStore, and a kv.Store adapter for read models.
// All stores share a single *bbolt.DB via disjoint buckets.
package bbolt
