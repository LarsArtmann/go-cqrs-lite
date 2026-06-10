# Comprehensive Status Report — 2026-06-11 00:51

> **go-cqrs-lite**: Lightweight CQRS library/SDK for Go with Event Sourcing, branded IDs, and auto-documentation.

## Executive Summary

**All 40 test packages green. Zero failures. Total coverage: 84.7%.** 144 unreleased commits since v2.2.0 across 31 modules (22 library + 6 examples + 1 integration + 2 cmd). The encryption module is complete with AES-256-GCM and XChaCha20-Poly1305. Sprint work delivered: SchemaVersion arithmetic, JSON marshal, graceful shutdown panic fix, error context preservation, storage/sql coverage bump.

---

## a) FULLY DONE ✅

### Module Health (all green)

| Module | Coverage | Status |
|--------|----------|--------|
| event/v2 | 89.6% | ✅ |
| event/v2/eventtest | 17.8% (test infra) | ✅ |
| command/v2 | 97.1% | ✅ |
| query/v2 | 94.3% | ✅ |
| decider/v2 | 100.0% | ✅ |
| id/v2 | 97.5% | ✅ |
| dispatcher/v2 | 98.0% | ✅ |
| schema/v2 | 89.7% | ✅ |
| snapshot/v2 | 88.9% | ✅ |
| codec/v2 | 93.3% | ✅ |
| memory/v2 | 98.2% | ✅ |
| catalog/v2 | 95.9% | ✅ |
| catalog/v2/asyncapi | 93.9% | ✅ |
| catalog/v2/d2 | 95.0% | ✅ |
| catalog/v2/docserver | 90.1% | ✅ |
| catalog/v2/eventcatalog | 92.7% | ✅ |
| catalog/v2/internal/caseutil | 100.0% | ✅ |
| catalog/v2/openapi | 100.0% | ✅ |
| catalog/v2/schema | 86.0% | ✅ |
| middleware/v2 | 95.7% | ✅ |
| integration/v2 | [no stmts] | ✅ |
| integration/v2/command | [no stmts] | ✅ |
| integration/v2/event | [no stmts] | ✅ |
| integration/v2/query | [no stmts] | ✅ |
| integration/v2/signing | [no stmts] | ✅ |
| integration/v2/simulation | 92.3% | ✅ |
| projection/v2 | 91.4% | ✅ |
| signing/v2 | 94.1% | ✅ |
| signing/v2/multisig | 94.2% | ✅ |
| storage/v2 | 89.2% | ✅ |
| storage/v2/sql | 61.2% | ✅ |
| watermill/v2 | 28.6% | ✅ |
| encryption/v2 | 73.0% | ✅ |
| listing/v2 | 84.3% | ✅ |
| otel/v2 | 89.9% | ✅ |
| pebble/v2 | 86.1% | ✅ |
| turso/v2 | 94.3% | ✅ |
| cmd/cqrs-gen/v2 | 94.9% | ✅ |

### Completed This Session

1. **Archived 97 old status reports** (pre-Jun 5 → `docs/status/archive/`)
2. **Added `encryption` to CI per-module matrix** (`.github/workflows/ci.yml`)
3. **SchemaVersion.Add() and Sub()** with underflow protection (`event/types.go`)
4. **Version + SchemaVersion JSON marshal/unmarshal** (`event/types.go`)
5. **6 new event tests** for Add/Sub/JSON in `event/batch_test.go`
6. **storage/sql coverage: 37.4% → 61.2%** — 13 new tests in `storage/sql/coverage_test.go`
7. **Graceful shutdown panic fix** — added `recover()` in goroutine (`pkg/gracefulshutdown/shutdown.go`)
8. **4 error context loss fixes** — replaced `fmt.Errorf` with classified error wrappers:
   - `watermill/protocol.go` (3× `event.WrapRejection`)
   - `listing/middleware.go` (1× `event.WrapInfrastructure`)
9. **Fixed 2 golden test drifts** (codec + middleware)

### Previously Completed (v2.2.0 → HEAD, 144 commits)

- Encryption module with AES-256-GCM + XChaCha20-Poly1305
- Composable codec wrapper (`encryption.NewCodec`)
- Integration test helpers extracted to `testutil/v2`
- Panic elimination sprint (MustParse removal)
- Code deduplication across 27 test files
- nlreturn lint fixes across all test files
- Dead API surface cleanup (storage, pebble, turso)
- Architecture improvement plan
- Streaming API v4 with tombstone read model
- Command/Query Type.IsZero, ParseType helpers

---

## b) PARTIALLY DONE ⚠️

| Item | Status | Details |
|------|--------|---------|
| encryption/v2 coverage | 73.0% | Below 80% threshold. Middleware tests and edge cases needed. |
| watermill/v2 coverage | 28.6% | Very low. Most protocol.go paths untested without Watermill broker. |
| eventtest coverage | 17.8% | Test infrastructure — low coverage is expected (helpers used by other tests). |
| Strong-ID migration | Deferred | `tracing_logging.go` and `sse.go` still use `string` for IDs. Breaking public API change — deferred to v3. |

---

## c) NOT STARTED 📋

- **v2.2.1 tag** — 144 unreleased commits. Needs tag review and CHANGELOG update.
- **CommandSchema coverage** — `PostgresDialect.CommandSchema()` and `SQLiteDialect.CommandSchema()` at 0% in `storage/sql/dialect.go`
- **OTel helpers coverage** — `storage/sql/otel.go` at 0% (Tracer, StartAggregateSpan, StartSaveSpan)
- **QueryEngine LoadWithSpan coverage** — 20% (main happy path untested with real DB + spans)
- **Encryption BDD test expansion** — Current BDD suite covers basic encrypt/decrypt roundtrip
- **Strong-ID migration for SSE + tracing** — Would break public API
- **Release notes / CHANGELOG for v2.2.1**

---

## d) TOTALLY FUCKED UP 💥

| Issue | Severity | Details |
|-------|----------|---------|
| 144 unreleased commits | HIGH | v2.2.0 tagged but HEAD has 144 commits with no release. Encryption module, panic fixes, coverage improvements, deduplication — all unreleased. |
| Golden test drift | LOW | codec and middleware golden files drifted from production output. Fixed this session but indicates golden tests aren't run in CI's format check. |
| `testutil/go.sum` drift | LOW | `go.sum` has unstaged changes suggesting `go mod tidy` hasn't been run since last module edit. |
| Pre-existing `GOWORK=off` build failure in middleware | MEDIUM | `memory/v2@v2.2.0` references `event.StreamKey` which doesn't exist in published version. Only affects per-module CI with `GOWORK=off`. Workspace builds fine. |

---

## e) WHAT WE SHOULD IMPROVE 📈

### Critical
1. **Tag v2.2.1** — 144 unreleased commits is irresponsible for a library consumers depend on
2. **Fix `GOWORK=off` middleware build** — memory/v2 needs republishing with StreamKey fix
3. **Run golden tests in CI** — Prevent drift by running `go test -run TestGolden` in CI pipeline

### Important
4. **encryption/v2 coverage → 80%+** — Currently 73%, below project threshold
5. **watermill/v2 coverage** — 28.6% is the lowest in the project. Needs integration tests or mark as experimental
6. **storage/sql OTel helpers** — 0% coverage on Tracer, StartAggregateSpan, StartSaveSpan
7. **Remove dead `docs/status/` status reports at root** — 97+ files still at root (only pre-Jun-5 archived this session)

### Nice-to-Have
8. **Strong-ID migration plan for v3** — Document breaking changes needed
9. **Benchmark baseline** — CI has benchmark regression detection but no baseline file committed
10. **File size audit** — Some files may exceed the 350-line CI gate

---

## f) Top 25 Things We Should Get Done Next

### Immediate (this session or next)

| # | Task | Impact | Effort | Module |
|---|------|--------|--------|--------|
| 1 | **Tag v2.2.1** with unreleased changes + CHANGELOG | HIGH | 15 min | git tags |
| 2 | Fix `GOWORK=off` middleware build (re-tag memory/v2) | HIGH | 10 min | memory |
| 3 | Run `go mod tidy` on testutil + all modules | MEDIUM | 5 min | testutil |
| 4 | Add golden test step to CI workflow | HIGH | 5 min | .github |
| 5 | encryption/v2 coverage → 80%+ (middleware tests) | HIGH | 30 min | encryption |
| 6 | storage/sql CommandSchema tests | MEDIUM | 15 min | storage/sql |
| 7 | storage/sql OTel helper tests | MEDIUM | 15 min | storage/sql |

### Short-term (this week)

| # | Task | Impact | Effort | Module |
|---|------|--------|--------|--------|
| 8 | watermill/v2 coverage → 50%+ | HIGH | 60 min | watermill |
| 9 | Commit benchmark baseline file | MEDIUM | 5 min | root |
| 10 | LoadWithSpan integration test | MEDIUM | 20 min | storage/sql |
| 11 | Encryption BDD: multi-event batch, key rotation | MEDIUM | 30 min | encryption |
| 12 | Strong-ID v3 migration plan (ADR) | LOW | 15 min | docs/adr |
| 13 | SSE handler integration test | MEDIUM | 20 min | middleware |
| 14 | Graceful shutdown integration test | MEDIUM | 15 min | pkg |
| 15 | CI: add `nix run .#check-layers` step | LOW | 5 min | .github |

### Medium-term (next 2 weeks)

| # | Task | Impact | Effort | Module |
|---|------|--------|--------|--------|
| 16 | v3 roadmap: strong-ID migration, API cleanup | HIGH | 60 min | docs |
| 17 | Event store benchmark suite (PG vs SQLite vs Pebble vs Turso) | MEDIUM | 90 min | storage |
| 18 | catalog/v2/schema coverage → 90%+ | LOW | 20 min | catalog |
| 19 | pebble/v2: add compaction/tombstone benchmarks | LOW | 30 min | pebble |
| 20 | turso/v2: add sync failure recovery tests | LOW | 30 min | turso |
| 21 | projection/v2: add pause/resume test | LOW | 20 min | projection |
| 22 | signing/v2/multisig: add threshold signing test | LOW | 15 min | signing |
| 23 | cmd/api-stability: add encryption to tracked surface | LOW | 10 min | cmd |
| 24 | docs: add encryption module to architecture diagram | LOW | 10 min | docs |
| 25 | example/: add encryption example | LOW | 20 min | example |

---

## g) Top #1 Question I Cannot Figure Out Myself 🤔

**Should we release v2.2.1 now with 144 commits (including encryption, panic fixes, error context), or wait to bundle it with encryption coverage improvements and the GOWORK=off fix as v2.3.0?**

The 144 commits include both new features (encryption module, SchemaVersion arithmetic) and bug fixes (graceful shutdown panic, error context loss, golden test drift). Bug fixes should reach consumers ASAP, but the memory/v2 GOWORK=off build failure means per-module CI will fail for middleware until memory is re-tagged. The call on release timing and version number is yours.

---

## Metrics Snapshot

| Metric | Value |
|--------|-------|
| Total modules | 33 (22 library + 6 examples + 1 integration + 2 cmd + 2 testutil) |
| Production LOC | 22,442 |
| Test LOC | 51,452 |
| Total coverage | 84.7% |
| Packages green | 40/40 (100%) |
| Packages FAIL | 0 |
| Unreleased commits | 144 |
| Last tagged release | v2.2.0 |
| Status reports in archive | ~130 |
| Status reports at root | ~97 |
