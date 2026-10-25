# Engine Coverage and Gaps

> Status snapshot of the metaengine storage-engine fleet: what exists, which storage classes are
> entirely absent, and what is deliberately refused. **Audience:** operators choosing engines and
> contributors weighing a new engine. Date: 2026-10-25. Per-engine capability facts (graph, edge
> removal, build tags) live in the generated, gated [`engine-capabilities.md`](engine-capabilities.md)
> — this doc covers coverage at the storage-class level only.

## Fleet today (13 engines)

| Storage class | Engine(s) | Registry driver | Notes |
| --- | --- | --- | --- |
| In-process | memory | `memory` | default; no persistence |
| Embedded relational | sqliteengine, tursoengine | `sqlite`, `turso` | turso wraps the sqlite engine (embedded libSQL) |
| Server relational | pgengine, mysqlengine | `postgres`, `mysql` | MySQL 8.0+ recursive CTE with probed iterative fallback |
| Embedded KV (LSM) | pebbleengine, badgerengine | `pebble`, `badger` | pure Go |
| Embedded KV (B-tree) | bboltengine | `bbolt` | pure Go, single-writer |
| Embedded columnar OLAP | duckdbengine | `duckdb` | CGo; registration behind `//go:build cgo` |
| Distributed graph | dgraphengine | `dgraph` | DQL over gRPC |
| Cloud wide-column | bigtableengine | `bigtable` | GCP only; heavy deps |
| P2P / CRDT | irohengine | — (programmatic) | loopback + QUIC transports |
| In-process graph | graphadapter | — (programmatic) | directed-only, flat node identity |

## Missing storage classes (gaps, not roadmap)

Each row below would need the ADR-0142 owner gate (see
[ADR-0142](adr/0142-universal-storage-substrate.md), AGENTS.md contract 30) before any driver is
written. None has a driver, an ADR, or a planned train today.

| Class | Engines named | Evidence / notes |
| --- | --- | --- |
| Server columnar / DWH | ClickHouse | Named throughout the original vision docs ([`planning/meta-engine-design.md`](planning/meta-engine-design.md) Deployments C/D) but never built; pure-Go client exists, making it the most credible consumer-pulled gap |
| Distributed wide-column | Cassandra, ScyllaDB | Also named in the vision docs; only GCP Bigtable covers the class |
| Native graph server | Neo4j, Memgraph | **Deliberately superseded** — dgraphengine + the SQL recursive-CTE fallback (`metaengine/graph_fallback.go`) replaced it (ADR-0119, ADR-0129). `graph.GraphDriver` is the sanctioned extension point ("consumer-pulled sibling modules", `graph/graph.go:29-32`) |
| Document | MongoDB | No document engine in the fleet |
| Full-text search | Elasticsearch, Meilisearch, Typesense | Class absent entirely; engines offer predicate filter/sort pushdown only |
| Vector-native ANN | Qdrant, Milvus, Weaviate, pgvector | Vector search exists as an engine-generic capability (cosine/dot/L2, `TopKNearest`, ADR-0140) but there is no dedicated ANN engine |
| NewSQL / distributed SQL | CockroachDB, TiDB, YugabyteDB | Untested and undeclared. CockroachDB speaks the PostgreSQL wire protocol, so pgengine may work — **unverified** |
| Cloud KV / wide-column beyond GCP | DynamoDB | Cloud coverage is GCP-only (bigtableengine) |
| Time-series | InfluxDB, TimescaleDB | Temporal as-of reads exist over the journal (`metaengine/temporal.go`), but no TSDB engine |
| Object / blob storage | S3, GCS | Export/import exists (`metaengine/export_import.go`); no blob engine |

## Deliberately refused (not gaps)

| Candidate | Why refused |
| --- | --- |
| Redis (as storage engine) | Sits at the broker/transport layer (watermill suite, `integration-redis`). ADR-0142 gate A: transport is not durable state — delivery layers stay at their delivery layer |
| RocksDB, LevelDB, LMDB | The embedded-KV niche is covered by pebble/bbolt/badger. RocksDB additionally requires CGo, which would create a second CGo island (policy: CGo is isolated to duckdb alone, ADR-0071) |
| Per-engine reimplementation of shared storage capabilities | ADR-0142: one shared runtime per storage class in metaengine core / claimkit / Map runtimes — never satellite-owned SQL, never per-engine forks |

## Extension points (bring your own engine)

- **Registry driver** — implement `metaengine.Engine`, call `RegisterDriver` from an `init()` in a
  dep-isolated `metaengine/<name>engine` module; consumers select it by blank import in
  `DeploymentConfig`. The per-module `register.go` clones are intentional (Go has no central
  registration) and carry `//art-dupl:accept`.
- **Programmatic engine** — construct in code without a registry driver (the irohengine and
  graphadapter pattern).
- **Graph server** — implement `graph.GraphDriver` and pass the `graphtest` contract (the Neo4j path
  documented at `graph/graph.go:29-32`).

## Checklist for adding an engine

1. **ADR-0142 gate first** — run the owner A/B pass: is there a GOOD reason the substrate lacks the
   class (gate A: don't add), or should it exist (gate B: design the capability WITH the owner,
   one shared runtime per storage class)?
2. New dep-isolated module `metaengine/<name>engine` (own `go.mod`, dependency budget review via `check-arch`).
3. `init()` + `RegisterDriver`; engine passes the `enginetest` conformance suite.
4. Register everywhere the fleet is census-gated: `go.work`, `flake.nix` module lists,
   `scripts/check-module-layers.sh` (tier + dep budget), `cmd/api-stability` (golden + module catalog meta-test),
   `cmd/cqrs-lint` module catalog.
5. Regenerate `docs/engine-capabilities.md` (`scripts/check-engine-capabilities.sh --update`) — the
   gate fails on source/doc drift.

## Provenance

- Fleet table cross-checked against the generated `engine-capabilities.md` census (2026-10-25).
- "Missing" rows verified by repo-wide search (no RocksDB/driver hits beyond test comments; Neo4j
  documented as designed-for-but-unbuilt in the 2026-09-13 event-query-model reconciliation audit,
  `status/archived/2026-09-13_18-35_event-query-model-truth-reconciliation-execution.md`).
- CockroachDB wire compatibility with pgengine is reasoning, not a tested claim.
