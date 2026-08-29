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

## Capabilities

Inherits the full `sqliteengine` capability set (Map, Set, Counter, Scan,
PushdownScan, StreamingScan, LayoutPlanner/Applier, raw-value reads) — the
engine embeds a `sqliteEngine` over the turso driver connection.

## Notes

- The connection is capped at `MaxOpenConns(1)` (libSQL replication semantics).
- Remote DSNs (`libsql://`, `https://`) contribute a live-RTT prior that
  `ProbeEngine` replaces with runtime measurements once the probe loop runs.
- Health: `db.PingContext` round-trip to the remote server.
