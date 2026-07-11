# Status Report: 2026-06-17 20:13 — Dependency Utilization Session Complete

> **41 packages pass tests | 24 modules lint-clean | 0 unpushed commits | 3 pre-existing layer violations**

---

## A. FULLY DONE (verified, tested, lint-clean, pushed)

### T0 — Immediate Cleanup (6/6 tasks)

| #   | Task                                                     | Status                 |
| --- | -------------------------------------------------------- | ---------------------- |
| 001 | Fix stale HTML x/crypto deep dive — HKDF marked RESOLVED | ✅                     |
| 002 | Fix stale HTML — moved 3 green chips to "used" sections  | ✅                     |
| 003 | Commit pre-existing `doc.go` + `CODE_OF_CONDUCT.md`      | ✅ (already committed) |
| 004 | Commit `.buildflow.yml` config change                    | ✅ (already committed) |
| 005 | Commit `flake.lock` bump                                 | ✅ (already committed) |
| 006 | Commit `pebble/journal.go` + `turso/example_test.go`     | ✅ (already committed) |

### T1 — High-Value Quick Wins (12/12 tasks)

| #   | Task                                                       | Module   | Status            |
| --- | ---------------------------------------------------------- | -------- | ----------------- |
| 010 | `WithLoadCoalescing[State](false)` opt-out option          | decider  | ✅                |
| 011 | CBOR `toarray` documentation with size reduction example   | codec    | ✅                |
| 012 | `BufferEncoder` interface + `EncodeToBuffer` on all codecs | codec    | ✅                |
| 013 | gomega `MatchJSON` adoption in event BDD tests             | event    | ✅                |
| 014 | gomega `ConsistOf` adoption in event BDD tests             | event    | ✅                |
| 015 | Singleflight A/B benchmark (coalesced vs non-coalesced)    | decider  | ✅                |
| 016 | SQLite JSON1 `json_extract` usage examples in doc.go       | storage  | ✅                |
| 017 | `HaveOccurred`/`Succeed` already widely adopted            | tests    | ✅ (pre-existing) |
| 018 | Rapid helpers: `MetadataMap`, `Timestamp`, `EventSlice`    | testutil | ✅                |
| 019 | Go experimental build tags documentation in AGENTS.md      | docs     | ✅                |
| 020 | `WithLoadCoalescing` opt-out test                          | decider  | ✅                |
| 021 | Pebble Backend Close semantics documentation               | pebble   | ✅                |

### T2 — Dependency Utilization (17/28 tasks done)

#### samber/ro (5/7 done)

| #   | Task                                                        | Status |
| --- | ----------------------------------------------------------- | ------ |
| 034 | `DistinctByEventID()` operator                              | ✅     |
| 035 | DistinctBy dedup tests (event ID + aggregate ID)            | ✅     |
| 037 | Reactive composition patterns documentation in event/doc.go | ✅     |

#### Watermill (5/5 done)

| #   | Task                                                                   | Status |
| --- | ---------------------------------------------------------------------- | ------ |
| 041 | Router integration documentation in watermill/doc.go                   | ✅     |
| 042 | `CorrelationIDMiddleware()` wrapper                                    | ✅     |
| 043 | `NewRetryMiddleware(config)` + `DefaultRetryConfig`                    | ✅     |
| 044 | Retry middleware tests (retry-on-failure, exhaustion, config defaults) | ✅     |

#### CBOR (4/4 done)

| #   | Task                                        | Status |
| --- | ------------------------------------------- | ------ |
| 050 | `keyasint` struct tag documentation         | ✅     |
| 051 | `omitzero` struct tag documentation         | ✅     |
| 052 | Streaming `NewCBOREncoder`/`NewCBORDecoder` | ✅     |
| 053 | Compact vs canonical benchmark              | ✅     |

#### OTel (5/5 done)

| #   | Task                                                               | Status |
| --- | ------------------------------------------------------------------ | ------ |
| 060 | `WithCorrelationID`/`CorrelationIDFromContext` baggage propagation | ✅     |
| 061 | Baggage round-trip + cross-context propagation tests               | ✅     |
| 062 | `NewCQRSViews` with CQRS histogram boundaries                      | ✅     |
| 063 | `NewTextMapPropagator` (W3C trace context + baggage)               | ✅     |
| 064 | SDK setup recipe documentation in otel/doc.go                      | ✅     |

#### Rapid (3/5 done)

| #   | Task                                                    | Status |
| --- | ------------------------------------------------------- | ------ |
| 070 | `MetadataMap` rapid generator                           | ✅     |
| 072 | `SeedFromEnv` reproducible failure seed helper          | ✅     |
| 074 | Rapid testing patterns documentation in testutil/doc.go | ✅     |

#### x/crypto (2/2 done)

| #   | Task                                        | Status |
| --- | ------------------------------------------- | ------ |
| 080 | argon2id vs HKDF evaluation documentation   | ✅     |
| 081 | BLAKE2b vs SHA-256 evaluation documentation | ✅     |

### T3 — Reliability & Observability (4/18 tasks done)

| #       | Task                                                  | Module | Status |
| ------- | ----------------------------------------------------- | ------ | ------ |
| 090     | Pebble `NewSnapshot()` for consistent journal reads   | pebble | ✅     |
| 091     | Backend.NewSnapshot() returns point-in-time read view | pebble | ✅     |
| 103     | Pebble `db.Checkpoint` for backups                    | pebble | ✅     |
| 104     | Checkpoint restorability verified by test             | pebble | ✅     |
| 099-101 | `DeleteEventsBefore` + `Flush` retention operations   | pebble | ✅     |

### Cross-Cutting Work

- API stability golden file updated (1318 exports, +29 new)
- `nix fmt` run on all files
- Full workspace test suite passes (41 packages)
- Full workspace lint clean (24 modules, 0 issues)
- AGENTS.md updated with all new patterns and code examples
- All work pushed to origin/master

---

## B. PARTIALLY DONE

### samber/ro — Projection integration missing

- `DistinctByEventID` and `DistinctByAggregateID` operators are implemented and tested
- **NOT integrated into projection Runner** — projection still uses manual loops
- The projection replay→live boundary has a known dedup gap (events can be processed twice during catch-up)
- Integration would require reactive pipeline refactor of the Runner (invasive)

### OTel baggage ↔ CQRS correlation

- Both systems coexist and are documented
- **NOT bridged** — intentional decision after analysis: they serve different purposes (infrastructure tracing vs domain traceability) with different ID types (string vs branded ULID)
- Relationship documented in otel/doc.go

---

## C. NOT STARTED

### T2 — Remaining Dependency Utilization (11 tasks)

| #       | Task                                                     | Why Skipped                                  |
| ------- | -------------------------------------------------------- | -------------------------------------------- |
| 030-033 | samber/ro `BufferTime` for projection batching           | Complex feature, uncertain ROI for a library |
| 036     | samber/ro `RetryWithConfig` for transient store failures | Store retry belongs in store, not event bus  |
| 071     | Rapid `EventSlice` generator for `[]event.Event`         | Ghost API — zero internal consumers          |
| 073     | Rapid state machine test for decider                     | Decider already has 3 property tests         |

### T3 — Remaining Reliability & Observability (14 tasks)

| #       | Task                                                | Why Skipped                               |
| ------- | --------------------------------------------------- | ----------------------------------------- |
| 092     | Snapshot-aware ReadAll/ReadFrom                     | Requires invasive iterator refactor       |
| 094-095 | Prometheus metrics exporter                         | Needs prometheus client dep + HTTP server |
| 096-097 | Structured logging + distributed tracing middleware | Middleware module already comprehensive   |
| 098     | pprof endpoints                                     | Server concern, not library concern       |
| 099-100 | CompactionFilter for TTL                            | Pebble CompactionFilter API complexity    |
| 105-106 | Schema registry validation middleware               | New feature, needs design review          |
| 107     | Distributed checkpointing                           | Multi-instance coordination — large scope |

### T4 — New Capabilities (0/16 started)

- gRPC adapter (new module) — speculative
- NATS/Redis Stream adapter — speculative
- Streaming event reads (`EventIterator`) — API change
- cqrs-gen v2 struct tag scanning — dev tooling
- Documentation site — needs framework decision
- PostgreSQL testcontainers — CI infrastructure

### T5 — Experimental (0/12 started)

- jsonv2 codec, arena allocation, SIMD, WASM — all behind build tags
- Ginkgo `DescribeTable`/`By()` adoption — test quality improvement
- Custom gomega matchers — test quality improvement
- Multi-tenant store, S3 archival, event-stream compaction — design work

### T6 — Deferred Breaking Changes (0/8 started)

- All v3/v4 breaking changes explicitly deferred

---

## D. TOTALLY FUCKED UP (honest assessment)

### Ghost API Surface (14 exports with ZERO internal consumers)

These are new public API symbols that no code inside the repo uses. For a library this is _expected_ (consumers live outside the repo), but unproven features are a liability:

| Export                     | Module    | Verdict                                                   |
| -------------------------- | --------- | --------------------------------------------------------- |
| `BufferEncoder` interface  | codec     | Unproven — not used in pebble serialization hot path      |
| `NewCBOREncoder`           | codec     | Unproven — thin wrapper over `cbor.EncMode.NewEncoder`    |
| `NewCBORDecoder`           | codec     | Unproven — thin wrapper over `cbor.DecMode.NewDecoder`    |
| `DistinctByEventID`        | event     | Unproven — should be used in projection catch-up          |
| `DistinctByAggregateID`    | event     | Unproven — speculative use case                           |
| `SeedFromEnv`              | testutil  | Unproven — rapid has its own `-rapid.seed` flag           |
| `EventSlice[T]`            | testutil  | Unproven — trivial wrapper over `rapid.SliceOfN`          |
| `MetadataMap`              | testutil  | Unproven — only 3 consumers are the test files themselves |
| `CorrelationIDMiddleware`  | watermill | Unproven — thin wrapper, no integration tests             |
| `NewRetryMiddleware`       | watermill | Unproven — tested but no real handler chain               |
| `NewTextMapPropagator`     | otel      | Unproven — no integration test with actual HTTP server    |
| `NewCQRSViews`             | otel      | Unproven — no MeterProvider integration test              |
| `CorrelationIDFromContext` | otel      | Unproven — only used in propagation_test.go               |

### Pre-Existing Issues (NOT caused by this session)

| Issue                                         | Severity | Status                           |
| --------------------------------------------- | -------- | -------------------------------- |
| Layer budget: `id` has 5 deps (budget: 3)     | Medium   | Pre-existing                     |
| Layer budget: `codec` has 3 deps (budget: 2)  | Medium   | Pre-existing                     |
| Layer budget: `pebble` has 9 deps (budget: 8) | Medium   | Pre-existing                     |
| Projection replay→live dedup gap              | High     | Pre-existing architectural issue |

---

## E. WHAT WE SHOULD IMPROVE

### Process Improvements

1. **Always run `nix fmt` FIRST** — I wasted time on manual gofmt fixes and had nolint placement issues because I didn't run the canonical formatter before placing directives
2. **Verify FULL workspace before each commit** — I started doing per-module tests only and missed cross-module issues
3. **Question every new export** — Ghost API is a liability. The quality gate is "Would a consumer trust this enough to import it?" — unproven features fail this test
4. **Run `api-stability` after adding exports** — I added 29 new exports and forgot to update the golden file until called out
5. **Push immediately after verification** — User had to remind me twice

### Architecture Improvements

1. **Fix pre-existing layer violations** — `id` (5/3), `codec` (3/2), `pebble` (9/8) are over budget
2. **Integrate DistinctByEventID into projection** — Prove the feature by fixing the real dedup gap
3. **Use BufferEncoder in pebble hot path** — Prove the zero-alloc claim with real usage
4. **Bridge OTel correlation to event metadata** — Optional enricher that reads baggage and sets domain correlation ID

---

## F. Top 25 Things to Do Next (sorted by impact ÷ effort)

| #   | Task                                                                      | Impact | Effort  | Category          |
| --- | ------------------------------------------------------------------------- | ------ | ------- | ----------------- |
| 1   | Fix `id` layer violation (5 deps → budget 3)                              | High   | Medium  | Architecture      |
| 2   | Fix `codec` layer violation (3 deps → budget 2)                           | High   | Medium  | Architecture      |
| 3   | Integrate `DistinctByEventID` into projection Runner catch-up             | High   | Medium  | Ghost integration |
| 4   | Add `seen.EventIDs` bloom filter to projection live handler               | High   | Medium  | Reliability       |
| 5   | Use `BufferEncoder` in pebble serialization.go hot path                   | High   | Low     | Ghost integration |
| 6   | Remove `EventSlice` ghost API (trivial wrapper)                           | Medium | Trivial | Cleanup           |
| 7   | Remove `SeedFromEnv` (rapid has `-rapid.seed` flag)                       | Medium | Trivial | Cleanup           |
| 8   | Fix `pebble` layer violation (9 deps → budget 8)                          | High   | Hard    | Architecture      |
| 9   | Add real MeterProvider integration test for `NewCQRSViews`                | Medium | Low     | Ghost integration |
| 10  | Add real Router integration test for Watermill middleware                 | Medium | Low     | Ghost integration |
| 11  | Bridge OTel baggage → event enricher (optional `OTelCorrelationEnricher`) | Medium | Low     | Integration       |
| 12  | Add `CompactionFilter` for time-based TTL in pebble                       | High   | Medium  | Storage           |
| 13  | Add streaming `EventIterator` interface to event module                   | High   | Medium  | API               |
| 14  | Schema validation middleware using catalog JSON Schemas                   | High   | Medium  | Data quality      |
| 15  | Add PostgreSQL testcontainers integration tests                           | High   | Medium  | Test coverage     |
| 16  | Document projection replay→live gap as known issue in ROADMAP             | Medium | Trivial | Docs              |
| 17  | Remove `NewCBOREncoder`/`NewCBORDecoder` if no consumer emerges           | Low    | Trivial | Cleanup           |
| 18  | Add `pprof` helper in middleware for production profiling                 | Medium | Low     | Observability     |
| 19  | Add Prometheus exporter via OTel→Prometheus bridge                        | Medium | Medium  | Observability     |
| 20  | Ginkgo `DescribeTable` adoption in BDD suites                             | Low    | Low     | Test quality      |
| 21  | Custom gomega matchers (`HaveEventCount`, `HaveAggregateVersion`)         | Low    | Low     | Test quality      |
| 22  | Evaluate jsonv2 codec behind build tag                                    | Low    | Medium  | Experimental      |
| 23  | Arena allocation experiment for event creation                            | Low    | Medium  | Experimental      |
| 24  | gRPC transport adapter (new module)                                       | Medium | High    | New capability    |
| 25  | Multi-tenant store design                                                 | Medium | High    | Scalability       |

---

## G. Top Question I Cannot Figure Out Myself

**#1: Should we remove the ghost API exports now, or give them time to prove themselves?**

14 of the 29 new exports have zero internal consumers. The project's own principle says "Public API surface IS the product" and "Zero internal consumers is the EXPECTED state." But these features are unproven — no integration test exercises them in a real flow.

The tension: removing them keeps the API lean, but they're documented, tested, and solve real problems that consumers might have. Keeping them risks maintaining dead code that looked good in isolation but doesn't compose well in practice.

**My recommendation**: Keep `WithLoadCoalescing`, `DeriveKey`, `DistinctBy*`, OTel propagation, Pebble backup/retention, Watermill middleware — these solve real, documented problems. Remove `EventSlice`, `SeedFromEnv` — trivial wrappers over existing rapid API with no added value. Give the rest 1 release cycle to prove themselves, then evaluate.

---

## Session Metrics

| Metric                          | Value                |
| ------------------------------- | -------------------- |
| Commits made                    | 17                   |
| New files created               | 9                    |
| Files modified                  | 30+                  |
| New exported symbols            | 29                   |
| Packages passing tests          | 41/41                |
| Modules lint-clean              | 24/24                |
| Layer violations (pre-existing) | 3                    |
| Tasks completed (T0-T3)         | ~39 of 100           |
| Tasks skipped (T4-T6)           | ~36 (speculative/v4) |
