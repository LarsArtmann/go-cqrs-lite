// Package postgres is the networked twin of the SQLite engine: the same
// facts-first semantics over PostgreSQL, for deployments where many
// producers and workers share one queue across machines. Claims use
// SELECT ... FOR UPDATE SKIP LOCKED instead of SQLite's single
// serialized writer: competing workers lock disjoint rows instead of
// queueing behind one connection.
//
// The storage mapping mirrors the SQLite engine exactly (unix-milli
// BIGINT timestamps, deps table, partial unique dedup index) so
// projections and the journal remain compatible across backends. Both
// engines are held to the identical semantics by the shared
// queue/conformance suite — the parity bar transcribed from the donor's
// mirrored suites.
package postgres
