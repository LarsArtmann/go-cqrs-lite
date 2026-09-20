// Package mysql is the MySQL/MariaDB engine for the durable work queue:
// the queue.Store contract over one MySQL (8.0+) or MariaDB (10.6+)
// database, with two-statement claims — SELECT ... FOR UPDATE SKIP
// LOCKED picks the row, the guarded UPDATE takes it — so any number of
// workers can poll concurrently. Semantics are pinned by the shared
// queue/conformance suite; this engine is its third dialect.
//
// Timestamps are BIGINT unix milliseconds everywhere (the queue/sqlite
// and queue/postgres convention), NOT DATETIME columns: one encoding,
// no timezone or fractional-second dialect traps.
//
// InnoDB deadlock scope: only ClaimDue retries deadlocks internally
// (bounded, backoff+jitter). Claims are the one constant-contention
// path — every worker polls the same index range in a tight loop — so
// occasional 1213/1205 kills there are routine and safe to absorb.
// Enqueue and the finalize transactions touch disjoint rows (keyed by
// task ID / dedup key), so deadlocks there are rare; on the rare hit
// the error surfaces to the caller, whose retry is both safe and
// visible: Enqueue is idempotent under its DedupKey (and a keyless
// enqueue is deliberately non-idempotent — a silent in-store retry
// would create duplicates), and every finalize is token-fenced, so a
// retry either lands or reports ErrLeaseNotHeld after a theft. Hiding
// that behind in-store retries would mask pathological lock contention
// from operators.
//
// Scope note: the metaengine Engine surface (claimkit capabilities,
// metaengine.RegisterDriver) rides the SAME pattern as queue/sqlite and
// queue/postgres — NewEngine/NewEngineFromDB expose DueClaimer + FactSink +
// DedupStore over the queue's own database (ADR-0142 T09).
package mysql
