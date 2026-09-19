// Package queue is the durable work-queue CONTRACT: a task lifecycle
// (pending → running → completed/dead/cancelled) with lease-based claims,
// retries with backoff, a dead-letter queue, priorities with bounded aging,
// DAG dependency gating, idempotent (dedup-keyed) enqueue, and an
// append-only fact journal written in the SAME transaction as every state
// change.
//
// The semantics are production-proven, not designed on paper: they are
// transcribed from go-taskqueue's internal queue stores, which ran the
// dogfood agent pool over live repositories for five weeks (its Store
// contract is the spec donor; the mirrored conformance suites that held
// its SQLite and Postgres backends to identical semantics are upstreamed
// here as queue/conformance). Engines live in sibling modules
// (queue/sqlite, queue/postgres) and are held to this contract by that
// one shared suite.
//
// Load-bearing invariant (the journal-first lineage): every method that
// mutates task state appends its fact(s) inside the SAME transaction — a
// state change without its fact did not happen, and the journal can never
// disagree with the task table. The conformance suite pins this.
//
// Deliberate omissions vs the donor (lean v1 contract; all additive later):
// severity/dashboard sort orders, per-project count rollups, the AI
// priority-score cache, journal archiving, operator watermark
// overrides, and the donor app's priority BANDS (P1–P4 markers with
// backlog clamping) — bands are application policy over the stored
// int; the contract keeps priorities unbounded and provenance rides
// the Reprioritized fact. The donor keeps those at its app layer.
//
// One deliberate semantic ADDITION vs the donor: enqueue validates
// that every task.New.Deps entry references an existing task
// (queue.ErrDanglingDep) on every engine — the donor inserts blindly
// and lets SQLite's foreign keys reject dangling rows (Postgres did
// not). Uniform validation is also the cycle policy: deps are fixed at
// enqueue and a store-minted task ID cannot be referenced by any
// existing task, so a dependency cycle's closing edge would have to
// name a task that does not exist yet — rejected as dangling. Cycles
// are unrepresentable by construction; no runtime cycle detection is
// needed or provided. There is also no unblock-bump (the donor's
// ADR-0015 concept): gating is evaluated at claim time, so completing
// a dependency unblocks its waiters in the same write — and bounded
// aging already lifts long-blocked tasks without distorting stored
// priority.
package queue
