# Comprehensive Execution Plan v3 — go-cqrs-lite

**Date:** 2026-06-11
**Scope:** ALL remaining open items from TODO_LIST.md, ROADMAP.md, code quality audit, coverage gaps, and planning docs.
**Task Size:** Max 12 minutes each.
**Sort:** Impact × (1/Effort) × Customer Value — highest first.

---

## Summary Statistics

| Tier                  | Items  | Est. Time   | Avg Impact   |
| --------------------- | ------ | ----------- | ------------ |
| **A: Lint & Hygiene** | 5      | ~30min      | HIGH         |
| **B: Test Coverage**  | 6      | ~66min      | HIGH         |
| **C: Code Quality**   | 8      | ~76min      | MED-HIGH     |
| **D: Documentation**  | 7      | ~64min      | MED          |
| **E: Performance**    | 3      | ~36min      | MED          |
| **F: CI & DevEx**     | 3      | ~36min      | MED          |
| **G: Experiments**    | 2      | ~24min      | LOW          |
| **X: v3 Breaking**    | 5      | ~12hr       | HIGH (defer) |
| **Y: Blocked**        | 11     | N/A         | N/A          |
| **Z: Future**         | 30+    | N/A         | N/A          |
| **ACTIONABLE TOTAL**  | **39** | **~332min** |              |

---

## Tier A: Lint & Hygiene (HIGH impact, LOW effort)

_Why first: Zero lint is table stakes for a library. Suppressions hide real bugs._

| #   | Task                                                                                   | Module  | Impact | Effort |
| --- | -------------------------------------------------------------------------------------- | ------- | ------ | ------ |
| A1  | Remove unused `backend` field from Pebble store                                        | pebble  | MED    | 3min   |
| A2  | Fix CBOR `cborEncMode` init error handling (replace `_, _` with explicit check)        | pebble  | MED    | 5min   |
| A3  | Verify all `//nolint` comments have `// reason` format — audit 120 suppressions        | all     | HIGH   | 12min  |
| A4  | Fix 31 `nolint:errcheck` suppressions in `defer .Close()` — use `defer func()` pattern | all     | HIGH   | 8min   |
| A5  | Reduce catalog/ nolint suppressions (36 total — suggest design issues)                 | catalog | MED    | 12min  |

---

## Tier B: Test Coverage (HIGH impact, MEDIUM effort)

_Why second: Untested code is untrusted code. Consumers check coverage before importing._

| #   | Task                                                                         | Module            | Current → Target | Impact | Effort |
| --- | ---------------------------------------------------------------------------- | ----------------- | ---------------- | ------ | ------ |
| B1  | Add `cmd/api-stability` tests — tool guards breaking changes but is untested | cmd/api-stability | 0% → 70%         | HIGH   | 12min  |
| B2  | Add turso `OpenSync` error-path tests (nil/empty path, invalid URL)          | turso             | 39% → 50%        | MED    | 12min  |
| B3  | Add `storage/sql` `LoadWithSpan` error branch tests                          | storage/sql       | 89.2% → 92%      | MED    | 10min  |
| B4  | Add `go-snaps` golden tests for `signing` (HMAC/Ed25519 signatures)          | signing           | 94% → golden     | MED    | 12min  |
| B5  | Add `go-snaps` golden tests for `storage` (DDL schemas, metadata roundtrip)  | storage           | 89% → golden     | MED    | 12min  |
| B6  | Add `go-snaps` golden tests for `middleware` (SSE frames, circuit breaker)   | middleware        | 94% → golden     | MED    | 12min  |

---

## Tier C: Code Quality (MED-HIGH impact, MED effort)

_Why: Type safety gaps and design smells erode consumer trust._

| #   | Task                                                                                  | Module  | Impact | Effort |
| --- | ------------------------------------------------------------------------------------- | ------- | ------ | ------ |
| C1  | Fix `event.ImmutableEvent.Clone` shared opts pointer — deep-copy opts                 | event   | HIGH   | 8min   |
| C2  | Add `query.BasicQuery` metadata (correlation/tracing) — parity with BasicCommand      | query   | MED    | 12min  |
| C3  | Clean test deps from 12 production go.mod files — move to test targets where possible | all     | MED    | 12min  |
| C4  | Extract `eventtest/` to own go.mod — removes 5 test-only transitive deps from event   | event   | HIGH   | 12min  |
| C5  | Fix ADR numbering gap — ADR-0005 missing, README lists only ADR-0001–0003             | docs    | LOW    | 5min   |
| C6  | Audit `decider/load.go:60` `err113` suppression — consider sentinel error             | decider | LOW    | 5min   |
| C7  | Clean up pebble backward-compat aliases — check if any consumers use old names        | pebble  | LOW    | 8min   |
| C8  | Evaluate `CoreDetEncOptions` vs `CanonicalEncOptions` for CBOR signing safety         | codec   | MED    | 12min  |

---

## Tier D: Documentation (MED impact, MED effort)

_Why: pkg.go.dev is the primary consumer touchpoint. Examples drive adoption._

| #   | Task                                                           | Module     | Impact | Effort |
| --- | -------------------------------------------------------------- | ---------- | ------ | ------ |
| D1  | Add `ExampleCBORCodec` — runnable example for pkg.go.dev       | codec      | MED    | 6min   |
| D2  | Add godoc example: `decider` Execute + Load patterns           | decider    | HIGH   | 10min  |
| D3  | Add godoc example: `projection` Runner + Builder + On[T](<>)   | projection | HIGH   | 10min  |
| D4  | Add godoc example: `signing` HMAC + Ed25519 + tamper detection | signing    | MED    | 8min   |
| D5  | Add godoc example: `schema` Upcaster + VersionedStore          | schema     | MED    | 8min   |
| D6  | Document CBOR usage patterns in `codec/README.md`              | codec      | LOW    | 8min   |
| D7  | Add README section linking to `docs/benchmarks/`               | root       | LOW    | 4min   |

---

## Tier E: Performance (MED impact, MED effort)

| #   | Task                                                                              | Module  | Impact | Effort |
| --- | --------------------------------------------------------------------------------- | ------- | ------ | ------ |
| E1  | Optimize `listing/InMemoryAggregateReader` — cache sorted result (269x potential) | listing | HIGH   | 12min  |
| E2  | Profile JSON vs CBOR allocation patterns (`go test -benchmem`)                    | codec   | MED    | 12min  |
| E3  | Benchmark `MemoryStore` with concurrent writers                                   | memory  | MED    | 12min  |

---

## Tier F: CI & DevEx (MED impact, MED effort)

| #   | Task                                                        | Module  | Impact | Effort |
| --- | ----------------------------------------------------------- | ------- | ------ | ------ |
| F1  | Add Docker build CI step: linux/amd64 + linux/arm64         | CI      | MED    | 12min  |
| F2  | Add Playwright E2E test for `example/user/` health endpoint | example | MED    | 12min  |
| F3  | Add Playwright E2E test: command → event → query flow       | example | MED    | 12min  |

---

## Tier G: Experiments (LOW impact, speculative)

| #   | Task                                          | Module | Impact | Effort |
| --- | --------------------------------------------- | ------ | ------ | ------ |
| G1  | `jsonv2` codec experiment behind build tag    | codec  | LOW    | 12min  |
| G2  | Arena allocation experiment in event creation | event  | LOW    | 12min  |

---

## Tier X: v3 Breaking Changes — DEFERRED

_These are high impact but require major version bump and migration guide._

| #   | Task                                                 | Effort | Note                         |
| --- | ---------------------------------------------------- | ------ | ---------------------------- |
| X1  | Remove `io.Closer` from core interfaces              | 4hr    | ADR-0010 accepted            |
| X2  | Add global `TransactionID` branded type              | 60min  | ADR needed first             |
| X3  | Split `event.Store` into Writer/Reader/Deleter       | 3hr    | Breaking change              |
| X4  | Make event Core truly immutable                      | 2hr    | Breaking change              |
| X5  | Move HTTP code from middleware → `transport/` module | 2hr    | SSE/healthcheck/metrics_http |

---

## Tier Y: [BLOCKED] — External action required

| #   | Blocker                                                     | What's needed        |
| --- | ----------------------------------------------------------- | -------------------- |
| Y1  | Move `example/todo` to own repository                       | Manual repo creation |
| Y2  | Add PostgreSQL integration tests (testcontainers)           | Docker setup         |
| Y3  | Remove cockroachdb/errors from go-localsync                 | Different repo       |
| Y4  | Create go-branded-id v0.2.0                                 | Different repo       |
| Y5  | Design ActaFlow event sourcing overlay                      | Different project    |
| Y6  | Extract shared golangci.yml into larsartmann/library-policy | Different repo       |
| Y7  | Change LICENSE from proprietary → MIT/Apache-2.0            | Owner decision       |
| Y8  | Migrate ActaFlow build to flake.nix                         | Different project    |
| Y9  | Integrate TypeSpec types → catalog.Registry                 | Different project    |
| Y10 | Playwright CI step                                          | Infrastructure setup |
| Y11 | Push signing v1.0.0 tag                                     | Manual tag + push    |

---

## Tier Z: [FUTURE] — Speculative, no design yet

27+ items from ROADMAP Sprint 6 and docs/planning/. Full list in TODO_LIST.md.
Includes: outbox pattern, saga module, schema registry, bi-temporal support, HLC,
distributed checkpointing, WASM target, gRPC adapter, NATS/Redis adapter, etc.

---

## Execution Order

```
Phase 1  (30min):  A1 → A2 → A4 → A3 → A5          — Lint & hygiene zero
Phase 2  (66min):  B1 → B2 → B3 → B4 → B5 → B6     — Coverage push
Phase 3  (76min):  C1 → C2 → C4 → C3 → C5 → C6 → C7 → C8 — Code quality
Phase 4  (64min):  D1 → D2 → D3 → D4 → D5 → D6 → D7 — Documentation
Phase 5  (36min):  E1 → E2 → E3                     — Performance
Phase 6  (36min):  F1 → F2 → F3                     — CI & DevEx
Phase 7  (24min):  G1 → G2                          — Experiments
```

**Total actionable: ~332 min (~5.5 hr)**
**Deferred: X1–X5 (~12 hr), Y1–Y11 (blocked), Z (30+ speculative)**
