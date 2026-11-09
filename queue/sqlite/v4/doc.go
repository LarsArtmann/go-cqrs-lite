// Package sqlite is the embedded, durable queue engine over one SQLite
// file: the production dialect of the donor's five-week dogfood,
// transcribed statement-for-statement and held to the shared
// queue/conformance suite.
//
// Concurrency model (the donor's load-bearing invariant): a single
// serialized write connection (MaxOpenConns(1)) plus WAL journal mode.
// All task mutations and their journal facts happen in one transaction,
// so the journal can never disagree with the task table. Multiple
// processes may open the same file; busy_timeout + WAL serialize
// cross-process writers.
//
// Claim SQL is engine-owned, in the claiming/ SHAPE: the queue's claim
// predicate (status-aware due-ness, dependency gating, priority+aging
// order, owner-checked renewal) is strictly richer than a claiming.Spec
// expresses, and extending Spec with those knobs is exactly the
// speculative-surface growth the claiming extraction forbade. The shapes
// stay aligned by the shared conformance suite instead.
package sqlite
