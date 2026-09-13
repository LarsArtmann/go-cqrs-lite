// Package claiming is the dialect-correct SQL core for lease-based row
// claiming: one atomic statement stamps a lease deadline on every due row
// and returns exactly the rows THIS caller claimed, so N workers can share
// one table without double-processing. The lease predicate re-opens rows
// whose previous claim expired, which is the whole crash-reclaim story: a
// worker that dies mid-processing delays its rows only until the lease
// lapses.
//
// The package owns no store. Consuming stores (timer stores, task queues)
// bring their table shape as a [Spec], their scan and metrics machinery,
// and their transaction boundaries; this package supplies the statements
// that make the claim atomic per dialect:
//
//   - Postgres: CTE fences due rows FOR UPDATE SKIP LOCKED, then UPDATE
//     stamps the lease and RETURNING hands back exactly the claimed rows.
//   - SQLite: no SKIP LOCKED exists, but the single-writer model serializes
//     claim transactions, which is equivalent for the no-double-processing
//     guarantee — a plain UPDATE..RETURNING suffices.
//   - MySQL/MariaDB 10.6+: SELECT ... FOR UPDATE SKIP LOCKED fences the
//     rows, then a plain UPDATE stamps the lease inside the same
//     transaction (no UPDATE..FROM..RETURNING on MySQL-compatible servers).
//
// Semantics are production-proven: scheduling/sqlstore's claiming timer
// store ran this exact SQL across all three dialects before the core was
// extracted (2026-09-13); the timer store now delegates here.
package claiming

import (
	"errors"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Dialect selects SQL syntax for claim statements and placeholders.
// scheduling/sqlstore re-exports this type as an alias, so its published
// constants keep their values by construction; idempotency/sqlstore
// carries an intentional duplicate whose values MUST match.
type Dialect int

const (
	// DialectSQLite uses ?N placeholders and stores timestamps as RFC3339
	// text; claims rely on SQLite's single-writer serialization.
	DialectSQLite Dialect = iota

	// DialectPostgres uses $N placeholders and native TIMESTAMP WITH TIME
	// ZONE; claims fence rows with FOR UPDATE SKIP LOCKED.
	DialectPostgres

	// DialectMySQL uses ? placeholders and native DATETIME(3); claims use
	// FOR UPDATE SKIP LOCKED (MySQL 8.0+ or MariaDB 10.6+).
	DialectMySQL
)

// ErrUnsupported is returned when claiming is requested for a dialect that
// cannot honor the claim contract. MySQL/MariaDB 10.6+ (FOR UPDATE SKIP
// LOCKED — verified live on MariaDB 11.4) IS supported; only unknown
// dialects are rejected.
var ErrUnsupported = errors.New(
	"claiming: requires Postgres, SQLite, or MySQL/MariaDB 10.6+ (FOR UPDATE SKIP LOCKED)",
)

// ErrLeaseNotHeld is returned by lease renewal when the caller no longer
// owns the row's claim: the row completed or disappeared, or the lease
// expired and another worker may have re-claimed it. Classified as
// Orchestration — a distributed-coordination race, not a caller bug.
var ErrLeaseNotHeld = errorfamily.NewOrchestration(
	"claiming.lease_not_held",
	"no live claim for this row (completed, canceled, or lease expired)",
)

// DefaultLease is how long a claim lasts when the operator does not name a
// duration. A claimed row becomes claimable again only after the lease
// expires, so the lease bounds how long a crashed worker delays the row —
// and how long concurrent workers are guaranteed not to double-process it.
const DefaultLease = time.Minute
