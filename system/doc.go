// ─── EventAdapter Save atomicity: the backend contract ───
//
// EventAdapter.Save performs an optimistic-concurrency write (expected
// version check + append). The atomicity guarantee depends on which
// OPTIONAL capability the engine's StreamLogBackend implements, checked in
// this order:
//
//  1. AtomicAppender (StreamAppendExpected) — ATOMIC. The version check and
//     append happen in one engine operation (single lock or single SQL
//     statement): concurrent writers cannot interleave between check and
//     append, and a crash mid-operation leaves either the old stream or the
//     fully appended stream. ALL engines shipped with metaengine implement
//     this (memory, sqlite, turso, pebble, bbolt, badger, dgraph, duckdb,
//     postgres, mysql).
//
//  2. Transactional (RunInTx) — TRANSACTIONAL. Check and append run in one
//     transaction: atomic against concurrent writers, with crash durability
//     delegated to the engine's durability tier (see
//     metaengine.DurabilityTier). This path is only reached for third-party
//     engines that implement Transactional but not AtomicAppender.
//
//  3. Neither — RACY. A bare check-then-append with no atomicity between
//     the two steps: two concurrent writers can both pass the version check
//     and both append, corrupting the version sequence. This fallback exists
//     so minimal third-party engines still function single-threaded; do NOT
//     rely on it under concurrency.
//
// Consumers reasoning about crash windows should therefore confirm their
// engine is an AtomicAppender (all shipped engines are) — then Save is
// all-or-nothing per call, and what survives a process crash is governed
// entirely by the engine's durability settings, not by Save itself.

package system
