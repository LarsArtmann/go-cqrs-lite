# Status Report — 2026-06-10 23:33

> **go-cqrs-lite** — Lightweight CQRS/Event Sourcing Library for Go
> v2.2.0 released 2026-06-08 | 35 workspace modules | 623 Go files | 84.0% total coverage

---

## Executive Summary

The project is in **excellent shape**. All 37 test packages pass (including race detection), lint is zero across all 28 modules, module layers check passes, and total coverage is 84.0%. The last 48 hours have been an intensive polish sprint: panic elimination, test deduplication, error safety cleanup, and immutability hardening. A new `encryption/` module (AES-256-GCM event payload encryption) is ready for review but not yet integrated into CI. One genuine data race in `projection/golden_test.go` was fixed this session.

---

## a) FULLY DONE ✅

### Core Library (22 library modules — ALL GREEN)

| Module         | Tests | Race | Lint | Coverage | Notes                                                                      |
| -------------- | ----- | ---- | ---- | -------- | -------------------------------------------------------------------------- |
| `event/`       | ✅    | ✅   | 0    | ~95%     | Core event sourcing: 19 functional options, immutable events, reactive bus |
| `command/`     | ✅    | ✅   | 0    | ~93%     | Typed handlers, middleware chain, reactive bus                             |
| `query/`       | ✅    | ✅   | 0    | ~92%     | Pagination, typed dispatch, reactive bus                                   |
| `decider/`     | ✅    | ✅   | 0    | ~95%     | Pure-function aggregate pattern                                            |
| `id/`          | ✅    | ✅   | 0    | 97.8%    | Branded IDs: `id.Of[T]` phantom types                                      |
| `dispatcher/`  | ✅    | ✅   | 0    | ~90%     | Generic Dispatcher[H, M]                                                   |
| `schema/`      | ✅    | ✅   | 0    | ~85%     | Upcaster, VersionedStore                                                   |
| `snapshot/`    | ✅    | ✅   | 0    | ~88%     | Snapshot strategies                                                        |
| `codec/`       | ✅    | ✅   | 0    | ~90%     | JSON + Raw payload encoding                                                |
| `memory/`      | ✅    | ✅   | 0    | ~88%     | In-memory test implementations                                             |
| `catalog/`     | ✅    | ✅   | 0    | ~85%     | AsyncAPI, OpenAPI, D2, EventCatalog exports                                |
| `middleware/`  | ✅    | ✅   | 0    | ~87%     | 24 factories (8 concerns × 3 message types)                                |
| `projection/`  | ✅    | ✅   | 0    | ~88%     | Replay+live runner, builder, retry, dead-letter                            |
| `signing/`     | ✅    | ✅   | 0    | ~92%     | HMAC-SHA256, Ed25519, multi-sig                                            |
| `storage/`     | ✅    | ✅   | 0    | ~34%\*   | SQL stores (PG/SQLite/Turso)                                               |
| `watermill/`   | ✅    | ✅   | 0    | ~83%     | Watermill protocol adapter                                                 |
| `listing/`     | ✅    | ✅   | 0    | ~88%     | Aggregate listing, tombstone detection                                     |
| `otel/`        | ✅    | ✅   | 0    | ~73%     | Shared OTel helpers                                                        |
| `pebble/`      | ✅    | ✅   | 0    | ~82%     | Embedded KV event store                                                    |
| `turso/`       | ✅    | ✅   | 0    | ~29%\*   | Turso/libSQL connector                                                     |
| `integration/` | ✅    | ✅   | 0    | ~85%     | Cross-module tests                                                         |
| `cmd/cqrs-gen` | ✅    | ✅   | 0    | ~80%     | Code generator                                                             |

\* Low coverage is shared infra / connector glue — not business logic.

### Infrastructure & Quality

- **Build**: `nix run .#build` — clean ✅
- **Vet**: `nix run .#vet` — clean ✅
- **Lint**: All 28 modules — zero issues ✅
- **Format**: `nix fmt` — clean ✅
- **Race**: All 37 packages pass with `-race` ✅ (fixed `projection/golden_test.go` data race this session)
- **Module layers**: `nix run .#check-layers` — passes ✅
- **CI pipeline**: 11 jobs defined (build, test, race, lint, coverage gate, per-module, benchmark, gosec, module-layers, file-size, go.work sync)
- **14 ADRs**: All accepted, none pending
- **Zero TODOs/FIXMEs** in production code
- **Zero deprecated API markers**
- **art-dupl t50 clones**: 0 (industry standard)

### Recent Sprint Completions (Jun 8–10)

| Sprint                          | Status  | Key Results                                              |
| ------------------------------- | ------- | -------------------------------------------------------- |
| v2.2.0 Release                  | ✅ Done | 81 commits since v2.1.0, tagged all modules              |
| Immuntability Audit (Phase 1–3) | ✅ Done | 7 mutability leaks fixed, `payloadForDecode()` zero-copy |
| Panic Elimination               | ✅ Done | All `MustParse`/`MustNew` removed from test code         |
| Test Deduplication              | ✅ Done | -133 lines, shared helpers extracted, nlreturn fixed     |
| Error Safety Cleanup            | ✅ Done | Classified errors replace `fmt.Errorf` in 4 paths        |
| Dead API Surface Removal        | ✅ Done | Storage, pebble, turso dead code removed                 |
| Brute Self-Review               | ✅ Done | Findings catalogued, improvement arc planned             |
| Branching Flow Lint             | ✅ Done | 389 issues catalogued into 10 sprints                    |

### New Module: `encryption/` (ready for review)

AES-256-GCM event payload encryption with:

- `Encrypter`/`Decrypter` interfaces + AES-256-GCM implementation
- `Ciphertext` type with JSON (base64) marshal/unmarshal
- Event integration helpers (`AttachEncryption`, `ExtractCiphertext`)
- Publisher/handler middleware (encrypt on publish, decrypt on handle)
- BDD test suite, unit tests, benchmarks, example tests
- 83.7% coverage, passes with `-race`
- In `go.work`, builds and tests clean, NOT yet in `flake.nix` CI `testModules` list

---

## b) PARTIALLY DONE ⚠️

### CI Integration of `encryption/` Module

- **Done**: Code complete, tests pass, in `go.work`
- **Not done**: Not added to `flake.nix` `testModules` list, not in CI pipeline, no `nix run .#test` coverage

### Unreleased CHANGELOG

- **Done**: 12+ fixes and changes documented in `[Unreleased]`
- **Not done**: Not yet tagged as a release. Should become v2.2.1 or v2.3.0

### Golden File Protection

- **Done**: Tests work, golden files exist
- **Not done**: `nix fmt` reformats `testdata/golden/*.json`, breaking golden tests. Needs `treefmt` exclusion

### Encryption Module Maturity

- **Done**: Core functionality, tests, BDD suite
- **Not done**: No key rotation, no KDF, no envelope encryption, no other cipher implementations

---

## c) NOT STARTED 📋

### From Execution Plan v2 (49 tasks, 0 started)

| Tier                       | Tasks    | Est. Time | Description                                                       |
| -------------------------- | -------- | --------- | ----------------------------------------------------------------- |
| **A: Type Safety & API**   | 5 tasks  | 33 min    | `SchemaVersion.Add/Cmp`, Version JSON marshal, `TypeOf[T]` design |
| **B: Test Coverage**       | 14 tasks | 156 min   | storage/sql 34%→70%, otel 73%→85%, turso 29%→50%                  |
| **C: Code Quality Polish** | 5 tasks  | 32 min    | New method tests, otel audit, `TypeOf` design                     |
| **D: CI & DevEx**          | 3 tasks  | 36 min    | Docker multi-arch, nolint audit, go vet in CI                     |
| **E: Developer Docs**      | 5 tasks  | 44 min    | godoc examples for decider, projection, signing, schema, listing  |
| **F: Experiments**         | 2 tasks  | 24 min    | jsonv2 codec, arena allocation                                    |

**Total actionable: ~325 min (~5.8 hours). None started.**

### From Branching Flow Plan (44 non-deferred issues, 0 started)

| Sprint                          | Issues | Est.   | Priority   |
| ------------------------------- | ------ | ------ | ---------- |
| Sprint 1: Panic + Error Context | 5      | 1 hr   | HIGH       |
| Sprint 2: Strong IDs            | 5      | 30 min | MEDIUM     |
| Sprint 3: Duplicate Types       | 15     | 1.5 hr | MEDIUM     |
| Sprint 4: Anti-patterns         | 5      | 30 min | LOW-MEDIUM |
| Sprint 5–9: Composition/Mixins  | 19     | 2 hr   | LOW        |

**Note**: The 315 phantom-type violations are explicitly deferred to incremental adoption.

### Other Not Started

- **v3 Breaking Changes** (5 items, ~12 hr): `TransactionID` branded type, `io.Closer` removal, Store split, `event.Core` immutability, HTTP→transport/ move
- **Test containers**: PostgreSQL integration tests (blocked on Docker)
- **Docs site**: Static site with pkg.go.dev hosting
- **Schema migration tool**: CLI for schema evolution

---

## d) TOTALLY FUCKED UP 💥

### 1. Golden Files vs `nix fmt` (Process Issue)

`nix fmt` (treefmt with gofumpt/goimports/golines) reformats `testdata/golden/*.json` files, causing golden test mismatches. This has been a **recurring problem across multiple sessions**. The fix (adding `settings.excludes` to treefmt config) is trivial but keeps getting deprioritized.

**Impact**: Tests fail after `nix fmt`. Forces `-update` flag to regenerate. Wastes time every session.

### 2. CI Not Running (Billing/Access)

CI pipeline is defined (`.github/workflows/ci.yml`) but appears to have been running with `-parallel=4` which Ginkgo rejects. The paste from the user showed the CI run with extra flags not in the current `flake.nix`, suggesting either an older CI config or external CI runner with different settings.

**Impact**: CI red. Blocks merge confidence. The signing "build failed" was a red herring — it was Ginkgo rejecting `-parallel`.

### 3. ~298 Status Reports Accumulated

There are **133 active + 165 archived** status reports in `docs/status/`. This is excessive. Many are redundant or superseded. The directory has become a dumping ground.

**Impact**: Hard to find relevant historical context. Noise drowns signal.

### 4. Encryption Module Not in CI

The new `encryption/` module is in `go.work` but not in `flake.nix` `testModules`. It won't be tested/linted by CI.

**Impact**: Regression risk. Module could break without detection.

---

## e) WHAT WE SHOULD IMPROVE 🎯

### High Priority

1. **Protect golden files from `nix fmt`** — Add `testdata/golden/**` to treefmt excludes in `flake.nix`. One-line fix. Been broken for multiple sessions.

2. **Add `encryption/` to CI pipeline** — Add to `testModules` in `flake.nix`. Module is ready, just needs wiring.

3. **Tag v2.2.1 release** — Unreleased changes are accumulating. The CHANGELOG already has 12+ items. Ship it.

4. **Archive old status reports** — Move anything older than 2026-06-05 to `archive/`. Keep the last week visible.

### Medium Priority

5. **Fix CI pipeline flags** — Ensure CI doesn't pass `-parallel=4` to `go test` when Ginkgo suites are present. Either use `ginkgo` CLI or remove `-parallel`.

6. **Start Tier A (Type Safety)** — 33 min of high-impact work. `SchemaVersion.Add/Cmp` and Version JSON marshal are straightforward.

7. **Improve storage/sql coverage** — 34% → 70% is the single largest coverage gap. Most of the remaining uncovered code is SQL path integration tests.

8. **Improve turso coverage** — 29% is the lowest. Sync operations at 0% coverage.

9. **Document immutability contract** — Add clear doc comments to `Event` interface about the clone-on-access contract. ADR 0013 covers the decision but the interface itself doesn't document the guarantee.

### Lower Priority

10. **Consolidate error wrapper patterns** — `command/errors.go` and `event/errors.go` have identical wrapper functions. Keep per-module (library pattern) but consider a shared internal package.

11. **Generic `Type[T]` for string types** — Unify `event.Type`, `command.Type`, `query.Type` with a generic `stringtype.Of[T]`. Large change, deferred.

12. **Add `testkit` package** — Extract shared test helpers into a dedicated `testkit/` module instead of scattering across `eventtest` and local helpers.

13. **Key management for encryption** — Add key rotation, KDF, or envelope encryption support.

---

## f) Top 25 Things We Should Get Done Next

### Immediate (this session or next)

| #   | Task                                           | Impact | Effort | Module      |
| --- | ---------------------------------------------- | ------ | ------ | ----------- |
| 1   | Add `testdata/golden/**` to treefmt excludes   | HIGH   | 1 min  | flake.nix   |
| 2   | Add `encryption` to `testModules` in flake.nix | HIGH   | 2 min  | flake.nix   |
| 3   | Tag v2.2.1 release with unreleased changes     | HIGH   | 5 min  | git tags    |
| 4   | Archive old status reports (>Jun 5 → archive/) | MEDIUM | 2 min  | docs/status |
| 5   | Verify CI workflow matches current flake.nix   | HIGH   | 5 min  | .github     |
| 6   | Add encryption module README.md                | LOW    | 10 min | encryption/ |

### Short-term (this week)

| #   | Task                                                     | Impact | Effort | Module     |
| --- | -------------------------------------------------------- | ------ | ------ | ---------- |
| 7   | Tier A1: `SchemaVersion.Add()` method                    | HIGH   | 5 min  | event      |
| 8   | Tier A2: `SchemaVersion.Cmp()` method                    | HIGH   | 5 min  | event      |
| 9   | Tier A3: `Version` JSON marshal/unmarshal                | HIGH   | 10 min | event      |
| 10  | Tier A4: `SchemaVersion` arithmetic helpers              | MEDIUM | 8 min  | event      |
| 11  | Tier B1: storage/sql coverage 34%→50%                    | HIGH   | 60 min | storage    |
| 12  | Tier C1: New method tests for event                      | MEDIUM | 10 min | event      |
| 13  | Sprint 1 from branching flow: fix gracefulshutdown panic | HIGH   | 15 min | middleware |
| 14  | Sprint 1: 4 error context loss fixes                     | MEDIUM | 30 min | multiple   |
| 15  | Sprint 1: 2 strong-id fixes                              | MEDIUM | 15 min | middleware |

### Medium-term (next 2 weeks)

| #   | Task                                              | Impact | Effort | Module     |
| --- | ------------------------------------------------- | ------ | ------ | ---------- |
| 16  | Tier B2: otel coverage 73%→85%                    | MEDIUM | 30 min | otel       |
| 17  | Tier B3: turso coverage 29%→50%                   | MEDIUM | 45 min | turso      |
| 18  | Tier D1: Docker multi-arch build                  | MEDIUM | 20 min | flake.nix  |
| 19  | Tier E1: godoc examples for decider               | MEDIUM | 10 min | decider    |
| 20  | Tier E2: godoc examples for projection            | MEDIUM | 10 min | projection |
| 21  | Tier E3: godoc examples for signing               | MEDIUM | 10 min | signing    |
| 22  | Document immutability contract on Event interface | HIGH   | 15 min | event      |
| 23  | Key management design for encryption module       | HIGH   | 60 min | encryption |

### Longer-term

| #   | Task                                                                | Impact    | Effort | Module     |
| --- | ------------------------------------------------------------------- | --------- | ------ | ---------- |
| 24  | Encryption key rotation + envelope encryption                       | HIGH      | 4 hr   | encryption |
| 25  | v3 breaking changes (Store split, TransactionID, io.Closer removal) | VERY HIGH | 12 hr  | core       |

---

## g) Top #1 Question I Cannot Figure Out Myself 🤔

**Is the `encryption/` module intended to be a v2.3.0 feature release, or should it be included in a v2.2.1 patch release alongside the current unreleased fixes?**

The module is fully functional with 83.7% coverage, BDD tests, benchmarks, and examples. But it's new external-facing API surface in a library — adding it to a patch release feels wrong semantically, yet the unreleased changes in CHANGELOG are already substantial enough to warrant shipping. The decision affects:

- Whether to add it to `flake.nix` `testModules` now or after versioning
- Whether to write a module README and update FEATURES.md
- Whether to add it to the v2.2.1 tag or wait for v2.3.0

---

## Metrics Dashboard

| Metric                       | Value                                                                        |
| ---------------------------- | ---------------------------------------------------------------------------- |
| **Go files**                 | 623 (325 production, 298 test)                                               |
| **Workspace modules**        | 35 (23 library + 6 examples + 1 integration + 2 cmd + 1 encryption + 2 root) |
| **Test packages**            | 37 — ALL GREEN                                                               |
| **Lint issues**              | 0 across all 28 modules                                                      |
| **Race issues**              | 0 (fixed 1 this session)                                                     |
| **Total coverage**           | 84.0%                                                                        |
| **TODOs in production code** | 0                                                                            |
| **ADRs**                     | 14 (all accepted)                                                            |
| **Status reports**           | 133 active + 165 archived                                                    |
| **Latest release**           | v2.2.0 (2026-06-08)                                                          |
| **Commits since v2.2.0**     | ~30                                                                          |
| **art-dupl t50 clones**      | 0                                                                            |
| **Module layer violations**  | 0                                                                            |

## Uncommitted Changes

| File                                                    | Change                                           |
| ------------------------------------------------------- | ------------------------------------------------ |
| `projection/golden_test.go`                             | Fix data race: add sync.Mutex to trackedHandler  |
| `encryption/*` (18 files)                               | New module: AES-256-GCM event payload encryption |
| `go.work`                                               | Added encryption module                          |
| `middleware/testdata/golden/health-check-response.json` | Golden file update                               |
| `codec/testdata/golden/json_encode.json`                | Golden file update                               |
| `docs/status/2026-06-10_23-19_*`                        | Previous status report edits                     |
| `docs/status/2026-06-10_23-20_*`                        | Previous status report edits                     |

## Test Results (as of this report)

```
nix run .#build  → clean
nix run .#vet    → clean
nix run .#lint   → 0 issues across 28 modules
nix run .#test   → 37/37 packages OK
nix run .#test-race → 37/37 packages OK (race-free)
nix fmt          → clean
nix run .#check-layers → PASS
Coverage: 84.0%
```
