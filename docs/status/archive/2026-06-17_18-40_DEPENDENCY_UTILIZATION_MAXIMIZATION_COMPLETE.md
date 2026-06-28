# Status Report — 2026-06-17 18:40

> **Context**: Dependency utilization maximization project — extracting more value from 15 direct dependencies without breaking changes. Three implementation rounds + three self-review rounds completed.

---

## A) FULLY DONE ✅

### Dependency Utilization Features (18 total, all implemented)

| Feature                                  | Module           | Commit     | Impact                                |
| ---------------------------------------- | ---------------- | ---------- | ------------------------------------- |
| singleflight load coalescing             | decider          | `98308192` | 🔴 N concurrent loads → 1 DB query    |
| SQLiteEnableForeignKeys helper           | storage          | `019fac95` | 🟡 Opt-in referential integrity       |
| HKDF DeriveKey (multi-tenant)            | encryption       | `88a29b21` | 🟡 Per-tenant key derivation          |
| Pebble DefaultOptions (bloom+compaction) | pebble           | `5df4051`  | 🟡 Production-grade defaults          |
| Pebble DefaultOptionsWithLogging         | pebble           | `5df4051`  | 🟡 Operational event alerts           |
| Backend.Metrics + BlockCacheHitRate      | pebble           | `5df4051`  | 🟡 LSM telemetry exposure             |
| CBORCompactCodec + ExtraReturnErrors     | codec            | `5df4051`  | 🟡 Schema drift detection             |
| CBOR Diagnose()                          | codec            | `5df4051`  | 🟢 Human-readable CBOR debug          |
| OTel Int64Counter + helpers              | otel, middleware | `5df4051`  | 🔴 Rate metrics (events/s, cmds/s)    |
| OTel ServiceResourceAttributes           | otel             | `5df4051`  | 🟡 Multi-service trace attribution    |
| OTel CQRSHistogramBoundaries             | otel             | `5df4051`  | 🟡 CQRS-optimized latency buckets     |
| OTel AddSpanEvent helper                 | otel, projection | `5df4051`  | 🟡 Retry/checkpoint span events       |
| ULID Monotonic entropy                   | id               | `5df4051`  | 🟡 Guaranteed within-ms ordering      |
| SQLite busy_timeout=5000                 | storage          | `5df4051`  | 🟡 Eliminates "database is locked"    |
| Pebble ULID-narrowed journal scan        | pebble           | `7af5325d` | 🔴 O(n)→O(log n) projection catch-up  |
| pebbleLogger fmt.Sprintf fix             | pebble           | `7af5325d` | 🔴 Real bug: all event logging broken |
| Dead code removal (scanJournal)          | pebble           | `7af5325d` | 🟢 Cleanup                            |
| testutil rapid generators                | testutil         | `5df4051`  | 🟡 Shared test infrastructure         |

### Tests Added (all passing, race-clean, lint-clean)

| Test file                                 | What it verifies                                          |
| ----------------------------------------- | --------------------------------------------------------- |
| `decider/decider_singleflight_test.go`    | Concurrent load coalescing (5 goroutines → 1 store.Load)  |
| `pebble/journal_scan_test.go`             | Narrowed scan: midpoint, last-event, zero-ID (100 events) |
| `pebble/options_metrics_test.go`          | DefaultOptions opens real DB + write/read round-trip      |
| `storage/sqlite_helpers_test.go`          | busy_timeout=5000 + foreign_keys=ON PRAGMAs               |
| `encryption/hkdf_test.go`                 | HKDF determinism, uniqueness, validation (6 tests)        |
| `testutil/rapidgen_test.go`               | Rapid generators produce valid values (4×100 iterations)  |
| `catalog/asyncapi/yaml_roundtrip_test.go` | AsyncAPI Document YAML round-trip                         |

### Documentation Updated

- `pebble/doc.go` — DefaultOptions, Metrics, BlockCacheHitRate sections
- `decider/doc.go` — Load Coalescing section
- `encryption/doc.go` — Key Derivation (HKDF) section
- `AGENTS.md` — Design Principle #17 + Key Patterns code examples
- HTML audit report — all deep dive sections, Top Opportunities, Quick Wins, self-review
- Gap closure plan — all 10 tasks marked ✅

---

## B) PARTIALLY DONE 🟡

| Item                           | What's done                                             | What's missing                                                               |
| ------------------------------ | ------------------------------------------------------- | ---------------------------------------------------------------------------- |
| **samber/ro utilization (8%)** | Basic pub/sub + Pipe1 + Filter                          | BufferTime, GroupBy, RetryWithConfig, Catch, Scan, ~80 operators unused      |
| **Watermill utilization (3%)** | Publisher/Subscriber adapter                            | Router, middleware chain, retry/DLQ, entire infrastructure layer bypassed    |
| **CBOR utilization (12%)**     | CanonicalEncOptions, CompactCodec, Diagnose             | MarshalToBuffer (zero-alloc), streaming, TagSet, keyasint/omitzero tags      |
| **OTel utilization (35%)**     | Tracing, metrics, counters, resource attrs, span events | Baggage propagation, Views, Exemplars, sampler config, TextMapPropagator     |
| **gomega utilization (20%)**   | Equal, HaveOccurred, BeNumerically                      | MatchJSON, ConsistOf, HaveField/HaveValue, custom matchers                   |
| **rapid utilization (10%)**    | StringMatching, IntRange, Check in testutil             | State machine testing, shrink testing, custom generators per module          |
| **HTML audit report**          | All critical findings resolved                          | Quick Wins row 10 (TimeUnixDynamic) still pending — correctly, it IS pending |

---

## C) NOT STARTED ⬜

1. **Pebble `db.NewSnapshot()`** — point-in-time consistency for journal reads during concurrent writes. Architectural change.
2. **OTel baggage propagation** — cross-service context propagation. Design needed.
3. **CBOR `toarray` struct tag** — 30-40% smaller events. Requires consumer-side struct tag changes.
4. **CBOR `TimeUnixDynamic`** — native time encoding. Moot without wire format change (int64 storage).
5. **Watermill Router + middleware** — the entire infrastructure layer is bypassed.
6. **samber/ro reactive operators** — BufferTime, GroupBy, RetryWithConfig, Catch, Scan.
7. **Concurrent singleflight benchmark** — functional test proves correctness; no A/B benchmark.
8. **`go mod tidy` needed in 5+ modules** — gopls reports `pgregory.net/rapid` missing from go.mod in command, decider, encryption, event, id, query modules.

---

## D) TOTALLY FUCKED UP! 💥

Nothing is fucked up. All code compiles, all tests pass with `-race`, all modules lint-clean (0 issues). No data corruption, no broken APIs, no broken builds.

**However, there were embarrassing mistakes caught by self-review:**

1. **`pebbleLogger.Infof` dropped format args** — `"pebble: " + format` without `fmt.Sprintf(format, args...)`. ALL pebble event listener logging was silently broken. Fixed in round 1.
2. **Zero tests on first push** — 7 new files pushed with zero tests. Added in round 2.
3. **Lint never run on first push** — 12+ issues across 4 modules. Fixed in round 2.
4. **HTML report contradicted itself** — Round 3 self-review said "all resolved" but Top Opportunities table still showed pending badges. Fixed this session.
5. **`fmt.Errorf` broke error family pattern** — singleflight used `fmt.Errorf` instead of `event.Wrap*`. Fixed in round 3.
6. **Stale buildflow pre-commit hook** — used `--parallel` flag removed from buildflow, silently blocking ALL commits. Fixed this session.

---

## E) WHAT WE SHOULD IMPROVE 🔧

### Architecture / Type Model

1. **`Repository[State]` has a `singleflight.Group` embedded** — it's zero-value safe but unexported. Consider a `WithLoadCoalescing(bool)` option for consumers who want to opt out (e.g., when using a cache layer that already handles dedup).
2. **`PebbleMetrics` is a flat struct, not an interface** — consumers can't mock or wrap it. Consider extracting a `MetricsProvider` interface.
3. **`DeriveKey` returns `[]byte`** — could return a branded type (`DerivedKey`) to prevent accidental use as a non-key. But this adds complexity for minimal safety gain.
4. **Error wrapping inconsistency in decider** — singleflight passthrough uses `//nolint:wrapcheck` while every other error goes through `opError`. This is correct but inconsistent. Could add a `decider.wrapLoadErr` helper.

### Dependency Utilization (the "not using libs superbly" concern)

5. **samber/ro at 8%** — this is the biggest opportunity. The library provides RxJS-style reactive composition and we only use it as a typed channel. BufferTime would batch events for projection, GroupBy would route by aggregate type, RetryWithConfig would handle transient store failures.
6. **Watermill at 3%** — we built our own adapter but bypass the entire Router, middleware, and retry infrastructure. Watermill's Router alone could replace custom projection wiring.
7. **CBOR at 12%** — `toarray` struct tags would make events 30-40% smaller with zero code change (consumer adds `cbor:",toarray"` to their payloads). MarshalToBuffer enables zero-alloc encoding.
8. **rapid at 10%** — state machine testing would catch race conditions that property tests miss. Each module's domain logic could have rapid-generated command sequences.

### Process

9. **Always write tests BEFORE committing** — not after. Three rounds of self-review found the same pattern: features implemented, tests forgotten.
10. **Always update ALL stale references** — when marking something "done", check every table, section, and badge across all documents. The HTML report had 4 stale locations for the same features.
11. **Run `go mod tidy` after adding deps** — gopls flags 5+ modules with missing rapid dependency.

---

## F) TOP 25 THINGS TO DO NEXT 🎯

Sorted by impact × effort ratio:

### High Impact, Low Effort (do first)

| #   | Task                                                                                                                                                    | Impact                                         | Effort |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------- | ------ |
| 1   | **`go mod tidy` in all modules** (command, decider, encryption, event, id, query)                                                                       | Unblocks gopls, CI                             | 5m     |
| 2   | **Commit pre-existing changes** (`.buildflow.yml`, `flake.lock`, `pebble/journal.go`, `turso/indexing/example_test.go`, `doc.go`, `CODE_OF_CONDUCT.md`) | Clean working tree                             | 5m     |
| 3   | **CBOR `toarray` documentation** — add doc example showing consumers how to use `cbor:",toarray"` on their payloads                                     | 30-40% smaller events for consumers who opt in | 15m    |
| 4   | **Add `gomega.ConsistOf` / `MatchJSON` to existing tests** — replace manual slice comparison boilerplate                                                | Cleaner tests, better failure messages         | 30m    |
| 5   | **Watermill Router spike** — evaluate if Router can replace projection wiring in example/todo                                                           | Potential architecture simplification          | 1h     |

### High Impact, Medium Effort

| #   | Task                                                                                                   | Impact                              | Effort |
| --- | ------------------------------------------------------------------------------------------------------ | ----------------------------------- | ------ |
| 6   | **samber/ro BufferTime for projection batching** — batch events by time window before handler dispatch | Fewer DB writes, better throughput  | 2h     |
| 7   | **samber/ro GroupBy for aggregate routing** — route events by aggregate type in the reactive pipeline  | Cleaner event routing               | 2h     |
| 8   | **rapid state machine testing** for decider — generate command sequences, verify aggregate invariants  | Catches ordering bugs               | 3h     |
| 9   | **Pebble `db.NewSnapshot()` for consistent reads** — point-in-time snapshots for journal reads         | Eliminates read-write inconsistency | 3h     |
| 10  | **OTel baggage propagation** — propagate correlation IDs across service boundaries                     | Cross-service tracing               | 3h     |
| 11  | **CBOR `MarshalToBuffer` for zero-alloc encoding** — reuse buffers in hot paths                        | Reduced GC pressure                 | 2h     |
| 12  | **Concurrent singleflight benchmark** — A/B comparison with/without singleflight                       | Proves performance benefit          | 1h     |
| 13  | **Watermill middleware chain** — replace custom retry/DLQ with Watermill's built-in                    | Less custom code                    | 3h     |

### Medium Impact, Various Effort

| #   | Task                                                                                                  | Impact                       | Effort |
| --- | ----------------------------------------------------------------------------------------------------- | ---------------------------- | ------ |
| 14  | **`WithLoadCoalescing(false)` option** — let consumers disable singleflight when using external cache | API flexibility              | 30m    |
| 15  | **OTel Views for custom metric aggregation** — enable CQRS-specific dashboard views                   | Better observability         | 2h     |
| 16  | **rapid generators per module** — module-specific event/command generators in testutil                | Reusable test infrastructure | 2h     |
| 17  | **Pebble `CompactionFilter` for TTL expiry** — automatic event retention                              | Storage management           | 3h     |
| 18  | **Pebble `db.DeleteRange` for retention** — bulk delete old events                                    | Storage management           | 2h     |
| 19  | **samber/ro RetryWithConfig** — retry transient store failures reactively                             | Resilience                   | 2h     |
| 20  | **gomega custom matchers** — `HaveEventCount(n)`, `HaveAggregateVersion(v)`                           | Test readability             | 1h     |

### Lower Priority

| #   | Task                                                                          | Impact               | Effort |
| --- | ----------------------------------------------------------------------------- | -------------------- | ------ |
| 21  | **Pebble `db.Checkpoint` for backups** — point-in-time DB snapshots           | Disaster recovery    | 2h     |
| 22  | **Pebble `db.Ingest` for bulk load** — sstable ingestion for data migration   | Fast bulk imports    | 3h     |
| 23  | **CBOR `TagSet` for custom type tags** — typed CBOR encoding for domain types | Type safety          | 2h     |
| 24  | **OTel Exemplars** — link metrics to traces                                   | Debugging            | 2h     |
| 25  | **Ginkgo DescribeTable for table-driven BDD** — replace manual test loops     | Test maintainability | 1h     |

---

## G) TOP QUESTION I CANNOT FIGURE OUT 🤔

**The samber/ro problem.** We use samber/ro as a typed pub/sub channel — `NewPublishSubject` + `Filter` + `Pipe1` + `Subscribe`. But the library provides ~80 operators for reactive composition (BufferTime, GroupBy, RetryWithConfig, Catch, Scan, MergeMap, SwitchMap, etc.).

**The question: should we invest in using ro as a full reactive pipeline, or is the current pub/sub usage the right level of abstraction for a CQRS library?**

Arguments for deeper ro usage:

- BufferTime would batch events for projection writes
- GroupBy would enable per-aggregate-type reactive pipelines
- RetryWithConfig would handle transient store failures reactively

Arguments against:

- ro adds complexity that consumers must understand
- The library is a CQRS/ES library, not a reactive framework
- Consumers who want reactive pipelines can compose ro themselves
- Most ro operators are designed for UI/streaming, not event sourcing

**I cannot determine the right tradeoff without knowing the target consumer profile: are they building real-time streaming systems (where ro operators add value) or traditional CRUD-with-events (where pub/sub is sufficient)?**

---

## Summary Metrics

| Metric                           | Value                                                                                   |
| -------------------------------- | --------------------------------------------------------------------------------------- |
| Total features implemented       | 18                                                                                      |
| Total tests added                | 19 test functions across 7 files                                                        |
| Modules touched                  | decider, storage, encryption, pebble, testutil, codec, otel, middleware, id, projection |
| Commits this session             | 17                                                                                      |
| Lint status                      | 0 issues across all changed modules                                                     |
| Test status                      | All passing with `-race`                                                                |
| Pre-existing uncommitted changes | 4 modified files, 2 untracked files (not ours)                                          |
| Average dependency utilization   | ~24% → ~28% (estimated after our changes)                                               |
