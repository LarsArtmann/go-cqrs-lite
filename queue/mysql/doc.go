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
// Scope note: the metaengine Engine surface (claimkit capabilities,
// metaengine.RegisterDriver) rides the SAME pattern as queue/sqlite and
// queue/postgres — NewEngine/NewEngineFromDB expose DueClaimer + FactSink +
// DedupStore over the queue's own database (ADR-0142 T09).
package mysql
