# metaengine/tursoengine — Turso/libSQL-Backed Engine

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4)

Turso (libSQL)-backed [metaengine](../README.md) Engine. Pure Go (`turso`
driver, no CGo). A thin wrapper over `sqliteengine` that adds remote-deployment
awareness: remote DSNs declare a same-datacenter network-RTT prior via
calibration, so the cost-based planner routes with honest network latency
instead of assuming an embedded disk.

```bash
go get github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4
```

## Quick Start

```go
import "github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"

engine, err := tursoengine.New("libsql://myapp.turso.io?authToken=...")
```

Empty DSN defaults to `:memory:`; plain file paths and `file:` DSNs work for
embedded libSQL use.

## Materialized Views (operator option, ADR-0135)

Turso's incremental view maintenance turns declared aggregate shapes into
precomputed, transactionally-consistent views. The operator declares WHAT to
accelerate; the engine derives the DDL, enables the required `views`
experimental feature on embedded DSNs, and serves matching unfiltered
aggregates from the views automatically:

```go
eng, err := tursoengine.New("libsql://myapp.turso.io?authToken=...",
    tursoengine.WithMaterializedViews([]metaengine.MaterializedViewSpec{
        {Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount"},
        {Collection: "orders", Fn: metaengine.MatViewAvg, Column: "amount", GroupBy: "customer"},
    }))
```

Serving rules (exact algebraic rewrites only, everything else falls through
to the base tables): scalar SUM/COUNT/MIN/MAX/AVG ↔ scalar views; grouped
aggregates ↔ grouped views (AVG stores SUM+COUNT and divides); a grouped view
also serves its scalar aggregate via exact derivation. Filtered aggregates and
planned-table collections are never served from views. Declare views in
`system` deployment YAML via `EngineConfig.MaterializedViews` instead of code
when composing through the system package. Verify registrations in
`Store.Doctor` ("Materialized views" section) or
`metaengine.ExplainableAggregate` (shows the view SQL that will run).

The corresponding `sqlite` driver REJECTS specs at construction — materialized
views require Turso/libSQL with the `views` experimental feature.

## Capabilities

Inherits the full `sqliteengine` capability set (Map, Set, Counter, Scan,
PushdownScan, StreamingScan, LayoutPlanner/Applier, raw-value reads) — the
engine embeds a `sqliteEngine` over the turso driver connection.

## Notes

- The connection is capped at `MaxOpenConns(1)` (libSQL replication semantics).
- Remote DSNs (`libsql://`, `https://`) contribute a live-RTT prior that
  `ProbeEngine` replaces with runtime measurements once the probe loop runs.
- Health: `db.PingContext` round-trip to the remote server.
