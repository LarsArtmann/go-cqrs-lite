# metaengine/tursoengine — Turso Database Engine (embedded)

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4)

Turso Database-backed [metaengine](../README.md) Engine (Turso Database is the Rust SQLite rewrite — libSQL is the legacy C fork; this module embeds the former). Pure Go (`turso`
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
embedded Turso Database use.

## Encryption at Rest (embedded, experimental)

> Reachability note: `WithEncryption` is a direct `tursoengine.New(...)`
> option — the `metaengine.DriverConfig`/`system` YAML layer has no encryption
> slot yet (a v5 `DriverConfig.Encryption` design is planned). Composing
> through the driver registry cannot reach it today.

The embedded Turso engine encrypts every page and the WAL with a native AEAD
cipher (experimental upstream — not yet third-party audited; Turso Cloud BYOK
is the production-ready path for regulated workloads). Pass a typed cipher
and a hex-encoded key — the Turso Database convention (Turso Cloud BYOK uses
base64 keys and is a separate, per-connection mechanism):

```go
// openssl rand -hex 32 — store the key in a secret manager, never in code.
eng, err := tursoengine.New("/data/app.db",
    tursoengine.WithEncryption(tursoengine.CipherAES256GCM, hexKey))
```

Ciphers: `CipherAEGIS256` (default recommendation), `CipherAEGIS128L` (faster,
128-bit), `CipherAEGIS128X2/X4`, `CipherAEGIS256X2/X4` (SIMD variants), and
`CipherAES128GCM`/`CipherAES256GCM` (NIST-approved — the auditor-friendly
choice for HIPAA/PCI-DSS). ChaCha20-Poly1305 exists only in Turso Cloud, not
in the local engine.

Semantics:

- A wrong or missing key fails decryption explicitly (`Decryption failed for
  page=N`), never silently returns garbage — that is the AEAD auth tag.
- The key is merged into the DSN at construction (`experimental=encryption`,
  `encryption_cipher`, `encryption_hexkey`), and `redactDSN` strips
  encryption parameters from every engine error message, so key material
  never reaches logs. `WithEncryption` refuses a DSN that already carries
  encryption parameters — exactly one key source must remain.
- Remote DSNs are rejected: Cloud BYOK keys ride the connection/sync layer,
  which this engine does not manage (configure via Turso tooling today).
- Upstream has no native rekeying yet — rotation is export/reimport.

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
views require the Turso Database engine with the `views` experimental feature.

> **Grouped-view caveat (upstream turso-go defect):** grouped views silently
> lose cross-transaction deltas at scale on driver versions through
> `metaengine.TursoGoIVMVerifiedThrough` — scalar views are exact. The
> `Store.Doctor` WARN flags it at runtime; `ivm_repro_test.go` in this module
> (`-tags ivmrepro`) is the one-command check for new driver releases, and
> `docs/turso-go-ivm-fix-flip-runbook.md` is the lift-the-caveat procedure.

## Capabilities

Inherits the full `sqliteengine` capability set (Map, Set, Counter, Scan,
PushdownScan, StreamingScan, LayoutPlanner/Applier, raw-value reads) — the
engine embeds a `sqliteEngine` over the turso driver connection.

## Notes

- The connection is capped at `MaxOpenConns(1)` (Turso replication semantics).
- Remote DSNs (`libsql://`, `https://`) contribute a live-RTT prior that
  `ProbeEngine` replaces with runtime measurements once the probe loop runs.
- Health: `db.PingContext` round-trip to the remote server.
