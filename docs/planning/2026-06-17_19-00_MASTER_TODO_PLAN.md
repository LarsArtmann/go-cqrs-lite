# MASTER TODO PLAN — All Tasks, Max 12min Each

> **Date**: 2026-06-17 19:00
> **Sources**: TODO_LIST.md, ROADMAP.md, status reports, HTML audit findings, planning docs
> **Sorting**: Importance × Impact × Customer Value ÷ Effort
> **Rule**: Every task ≤ 12 minutes. Large features split into sequential sub-tasks.

---

## Summary

| Tier | Focus | Tasks | Total Est |
|------|-------|-------|-----------|
| T0 | Immediate cleanup — fix stale state | 6 | 45m |
| T1 | High-value quick wins — customer-facing | 12 | 2h |
| T2 | Dependency utilization — deeper lib adoption | 28 | 5.5h |
| T3 | Reliability & observability | 18 | 3.5h |
| T4 | New capabilities & adapters | 16 | 3h |
| T5 | Experimental / long-term | 12 | 2h |
| T6 | Deferred breaking changes (v3/v4) | 8 | 4h |
| **TOTAL** | | **100 tasks** | **~20.5h** |

---

## T0 — Immediate Cleanup (fix stale state, unblock everything)

| # | Task | Module | Impact | Effort |
|---|------|--------|--------|--------|
| 001 | Fix stale HTML: x/crypto deep dive — mark HKDF finding as RESOLVED | docs | Accuracy | 3m |
| 002 | Fix stale HTML: move 3 green chips (Diagnose, CoreDetEncOptions, Levels tuning) from "Also unused" to "What's used" sections | docs | Accuracy | 8m |
| 003 | Commit pre-existing changes: `doc.go` (root package doc), `CODE_OF_CONDUCT.md` | root | Clean tree | 3m |
| 004 | Review & commit pre-existing `.buildflow.yml` change (todo_min_severity config) | root | Clean tree | 3m |
| 005 | Review & commit pre-existing `flake.lock` bump | root | Clean tree | 2m |
| 006 | Review & commit pre-existing `pebble/journal.go` comment fix + `turso/indexing/example_test.go` | pebble, turso | Clean tree | 5m |

---

## T1 — High-Value Quick Wins (customer-facing, under 30m each)

| # | Task | Module | Impact | Effort |
|---|------|--------|--------|--------|
| 010 | Add `WithLoadCoalescing(false)` option to decider Repository — let consumers opt out of singleflight | decider | API flexibility | 12m |
| 011 | Add CBOR `toarray` documentation example in codec/doc.go showing 30-40% size reduction | codec | Customer education | 10m |
| 012 | Add CBOR `MarshalToBuffer` to codec interface for zero-alloc encoding path | codec | Performance | 12m |
| 013 | Add gomega `MatchJSON` adoption — replace manual JSON comparison in 3 test files | tests | Test quality | 12m |
| 014 | Add gomega `ConsistOf` adoption — replace manual slice ordering checks in event tests | tests | Test quality | 12m |
| 015 | Add concurrent singleflight benchmark — A/B comparison with/without coalescing | decider | Proven benefit | 12m |
| 016 | Fix stale HTML: SQLite JSON1 finding — document `json_extract` usage example in storage/doc.go | storage, docs | Customer education | 10m |
| 017 | Add `HaveOccurred` / `Succeed` gomega adoption in remaining test files (pebble, storage) | tests | Test quality | 10m |
| 018 | Add rapid `.Filter()` / `.Map()` helpers to testutil — reduce generator boilerplate | testutil | Test infra | 12m |
| 019 | Document Go `build tags` strategy in AGENTS.md (goexperiment.arenas, simd, etc.) | docs | Dev experience | 8m |
| 020 | Add `WithLoadCoalescing` test — verify opt-out disables singleflight coalescing | decider | Test coverage | 10m |
| 021 | Add `GoAway`/`Close` method documentation to pebble Backend — clarify Close semantics | pebble | Customer education | 8m |

---

## T2 — Dependency Utilization (deeper library adoption)

### samber/ro (currently 8% → target 25%)

| # | Task | Module | Impact | Effort |
|---|------|--------|--------|--------|
| 030 | Spike: samber/ro `BufferTime` for projection event batching — write prototype | projection | Throughput | 12m |
| 031 | Implement `BufferTime` wrapper in projection Runner — batch events by time window | projection | Throughput | 12m |
| 032 | Test: BufferTime batching — verify events coalesced within window, flushed on timeout | projection | Confidence | 12m |
| 033 | Spike: samber/ro `GroupBy` for per-aggregate-type reactive routing | event | Architecture | 12m |
| 034 | Implement `DistinctBy(eventID)` dedup operator in event EventBus | event | Correctness | 12m |
| 035 | Test: DistinctBy dedup — verify duplicate event IDs filtered | event | Confidence | 10m |
| 036 | Spike: samber/ro `RetryWithConfig` for transient store failures | event | Resilience | 12m |
| 037 | Document samber/ro reactive composition patterns in event/doc.go | event | Customer education | 10m |

### Watermill (currently 3% → target 15%)

| # | Task | Module | Impact | Effort |
|---|------|--------|--------|--------|
| 040 | Spike: Watermill `Router` — evaluate replacing custom projection wiring in example/todo | watermill | Architecture | 12m |
| 041 | Document Watermill Router integration path in watermill/doc.go | watermill | Customer education | 10m |
| 042 | Add Watermill `CorrelationID` middleware wrapper | watermill | Tracing | 12m |
| 043 | Add Watermill `Retry` middleware wrapper with exponential backoff | watermill | Resilience | 12m |
| 044 | Test: Watermill Retry middleware — verify retry count and backoff timing | watermill | Confidence | 12m |

### CBOR (currently 12% → target 25%)

| # | Task | Module | Impact | Effort |
|---|------|--------|--------|--------|
| 050 | Add CBOR `keyasint` struct tag documentation — integer keys for compact maps | codec | Size reduction | 10m |
| 051 | Add CBOR `omitzero` struct tag documentation — omit zero-value fields | codec | Size reduction | 10m |
| 052 | Add CBOR streaming Encoder/Decoder support for large event batches | codec | Memory | 12m |
| 053 | Benchmark: CBORCompactCodec vs CBORCodec — size and speed comparison | codec | Evidence | 12m |

### OTel (currently 35% → target 50%)

| # | Task | Module | Impact | Effort |
|---|------|--------|--------|--------|
| 060 | Add OTel baggage propagation helper — propagate correlation IDs across services | otel | Distributed tracing | 12m |
| 061 | Test: OTel baggage propagation — verify baggage entries survive context propagation | otel | Confidence | 10m |
| 062 | Add OTel Views helper — CQRS-specific metric aggregation views | otel | Observability | 12m |
| 063 | Add OTel TextMapPropagator setup helper — W3C trace context + baggage | otel | Standards | 12m |
| 064 | Document OTel SDK setup recipe in otel/doc.go — provider, exporter, sampler | otel | Customer education | 10m |

### Rapid (currently 10% → target 20%)

| # | Task | Module | Impact | Effort |
|---|------|--------|--------|--------|
| 070 | Add rapid `MapOf` generator to testutil — generate event.Metadata maps | testutil | Test infra | 10m |
| 071 | Add rapid `EventSlice` generator to testutil — generate []event.Event sequences | testutil | Test infra | 12m |
| 072 | Add rapid seed control helper — reproducible failure seeds for CI | testutil | Debugging | 10m |
| 073 | Add rapid state machine test for decider — generate command sequences, verify invariants | decider | Correctness | 12m |
| 074 | Document rapid testing patterns in testutil/doc.go | testutil | Customer education | 8m |

### x/crypto (currently 8% → target 20%)

| # | Task | Module | Impact | Effort |
|---|------|--------|--------|--------|
| 080 | Evaluate argon2id for password-based key derivation — document vs HKDF tradeoffs | encryption | Security | 10m |
| 081 | Add blake2b hasher option for faster non-cryptographic checksums | codec | Performance | 12m |

---

## T3 — Reliability & Observability

| # | Task | Module | Impact | Effort |
|---|------|--------|--------|--------|
| 090 | Add `PebbleMetricsProvider` interface — enable wrapping/mocking metrics | pebble | Testability | 12m |
| 091 | Pebble `db.NewSnapshot()` for consistent journal reads — point-in-time snapshots | pebble | Correctness | 12m |
| 092 | Pebble `db.NewSnapshot()` — implement snapshot-aware ReadAll/ReadFrom | pebble | Correctness | 12m |
| 093 | Test: Pebble NewSnapshot — verify consistent reads during concurrent writes | pebble | Confidence | 12m |
| 094 | Add Prometheus metrics exporter — `/prometheus` endpoint helper in middleware | middleware | Observability | 12m |
| 095 | Add Prometheus metrics exporter — implement OTel→Prometheus bridge | middleware | Observability | 12m |
| 096 | Add structured logging middleware — configurable slog levels for cmd/event/query | middleware | Observability | 12m |
| 097 | Add distributed tracing propagation — span context across module boundaries | otel, middleware | Distributed tracing | 12m |
| 098 | Add `pprof` endpoints helper — `/debug/pprof` setup for production profiling | middleware | Debugging | 10m |
| 099 | Pebble `CompactionFilter` for TTL expiry — auto-delete events older than N days | pebble | Storage mgmt | 12m |
| 100 | Pebble `CompactionFilter` — implement filter logic and test retention window | pebble | Storage mgmt | 12m |
| 101 | Pebble `db.DeleteRange` for bulk retention — delete events before timestamp | pebble | Storage mgmt | 12m |
| 102 | Test: Pebble DeleteRange — verify bulk deletion preserves recent events | pebble | Confidence | 10m |
| 103 | Pebble `db.Checkpoint` for backups — point-in-time DB snapshot for disaster recovery | pebble | Disaster recovery | 12m |
| 104 | Test: Pebble Checkpoint — verify checkpoint is consistent and restorable | pebble | Confidence | 12m |
| 105 | Schema registry — JSON Schema validation middleware for events (design) | schema | Data quality | 12m |
| 106 | Schema registry — implement validation middleware with go-error-family rejections | schema | Data quality | 12m |
| 107 | Distributed checkpointing — multi-instance projection coordination (design) | projection | Scalability | 12m |

---

## T4 — New Capabilities & Adapters

| # | Task | Module | Impact | Effort |
|---|------|--------|--------|--------|
| 110 | gRPC transport adapter — design command/query dispatch over gRPC (new module) | grpc (new) | Integration | 12m |
| 111 | gRPC transport adapter — implement server-side command handler | grpc (new) | Integration | 12m |
| 112 | gRPC transport adapter — implement client-side dispatch | grpc (new) | Integration | 12m |
| 113 | gRPC transport adapter — test end-to-end command dispatch | grpc (new) | Confidence | 12m |
| 114 | NATS/Redis Stream adapter — design message broker integration (new module) | nats (new) | Integration | 12m |
| 115 | NATS adapter — implement publisher/subscriber | nats (new) | Integration | 12m |
| 116 | Streaming event reads — `StreamLoader` iterator without materializing full slice | event | Memory | 12m |
| 117 | Streaming event reads — implement `EventIterator` interface | event | Memory | 12m |
| 118 | Streaming event reads — test iterator with large event streams | event | Confidence | 10m |
| 119 | cqrs-gen v2 — struct tag scanning improvements (design) | cmd/cqrs-gen | Dev experience | 12m |
| 120 | cqrs-gen v2 — implement struct tag scanning | cmd/cqrs-gen | Dev experience | 12m |
| 121 | Pebble `db.Ingest` for bulk load — sstable ingestion for data migration | pebble | Data migration | 12m |
| 122 | Documentation site — evaluate Docusaurus vs MkDocs vs Hugo | docs | Customer experience | 12m |
| 123 | Documentation site — scaffold chosen framework with API docs | docs | Customer experience | 12m |
| 124 | PostgreSQL integration tests — set up testcontainers-go CI pipeline | storage, ci | Test coverage | 12m |
| 125 | PostgreSQL integration tests — write real PG event store tests | storage | Test coverage | 12m |

---

## T5 — Experimental / Long-Term

| # | Task | Module | Impact | Effort |
|---|------|--------|--------|--------|
| 130 | jsonv2 codec experiment — behind build tag, evaluate API | codec | Future | 12m |
| 131 | Arena allocation experiment — behind build tag + Go experiment flag | event | Future perf | 12m |
| 132 | SIMD-accelerated serialization — evaluate Go experiment flag impact | codec | Future perf | 12m |
| 133 | WASM compilation target — verify decider module compiles to WASM | decider | Edge compute | 12m |
| 134 | Ginkgo `DescribeTable` adoption — replace manual test loops in BDD suites | tests | Test quality | 12m |
| 135 | Ginkgo `By()` step tracking — add structured test steps to BDD suites | tests | Test quality | 10m |
| 136 | Gomega custom matchers — `HaveEventCount(n)`, `HaveAggregateVersion(v)` | tests | Test quality | 12m |
| 137 | Gomega custom matchers — `HaveEventType(t)`, `HaveEventPayload(p)` | tests | Test quality | 12m |
| 138 | Multi-tenant store — design tenant-isolated event store API | event (new) | Scalability | 12m |
| 139 | S3/GCS archival — design cold event storage adapter | storage (new) | Cost | 12m |
| 140 | Event-stream compaction — design snapshot+delete compaction strategy | event | Storage | 12m |
| 141 | Performance regression dashboard — set up benchstat CI comparison | ci | Reliability | 12m |

---

## T6 — Deferred Breaking Changes (v3/v4)

| # | Task | Module | Impact | Effort |
|---|------|--------|--------|--------|
| 150 | [v3] Remove `io.Closer` from core interfaces (ADR-0010) | event, snapshot, command | API break | 12m |
| 151 | [v3] Add global `TransactionID` branded type for cross-aggregate consistency | event | API break | 12m |
| 152 | [v3] Make event Core truly immutable — deep-copy all pointer fields | event | API break | 12m |
| 153 | [v3] Move HTTP code out of middleware → transport/ module | middleware, transport (new) | API break | 12m |
| 154 | [v3] Fix `query.Handler` returns `any` → generic `TypedHandler[T]` returning `(T, error)` | query | API break | 12m |
| 155 | [v4] Split `catalog.Message` into Message + MessageMeta (17 fields → structured) | catalog | API break | 12m |
| 156 | [v4] Split `catalog.Service` into Service + ServiceMeta (16 fields → structured) | catalog | API break | 12m |
| 157 | [v3] `command` module — stop re-exporting event types (module boundary violation) | command | API break | 12m |

---

## Deduplicated / Excluded

These items appeared in multiple sources and are consolidated above:

| Item | Resolution |
|------|-----------|
| `go mod tidy` in 5+ modules | **EXCLUDED** — gopls false positives (replace directives via go.work; all modules pass `go test` and `go build`) |
| Replace `go-faster/yaml` | **REJECTED** — churn for zero value |
| Outbox pattern | **REJECTED** — explicitly HARD NO in ROADMAP.md |
| Saga pattern | **REJECTED** — explicitly HARD NO in ROADMAP.md |
| GraphQL adapter | **REJECTED** — explicitly HARD NO in ROADMAP.md |
| Change default CBOR encoding to toarray | **REJECTED** — breaks stored data |
| Enable foreign_keys=ON by default | **REJECTED** — breaks existing DBs |
| Add ExtraReturnErrors as default decoding | **REJECTED** — breaks forward compat |
| CBOR TimeUnixDynamic | **DEFERRED** — moot without wire format change (int64 storage) |

---

## Execution Priority

**Do first (T0 + T1):** 18 tasks, ~2.5h. Fixes stale state, adds high-value customer-facing features.

**Do second (T2):** 28 tasks, ~5.5h. Deeper dependency utilization — the "using libs superbly" work.

**Do third (T3):** 18 tasks, ~3.5h. Reliability and observability gaps.

**Do fourth (T4+T5):** 28 tasks, ~5h. New capabilities and experiments.

**Last (T6):** 8 tasks, ~4h. Breaking changes for v3/v4.
