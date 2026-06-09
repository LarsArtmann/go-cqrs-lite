# Status Report — 2026-06-09 02:12

> **Self-Review & Improvement Arc Completion**
> Session: Post-golden-test self-critique → improvement execution
> Commits this arc: 7 (a9024883..51fbc8ab)
> Net change: +218 / −475 lines across 25 files

---

## TL;DR

| Metric                   | Value                                                                 |
| ------------------------ | --------------------------------------------------------------------- |
| Build                    | ✅ PASS                                                               |
| Tests                    | ✅ 37/37 packages                                                     |
| Lint                     | ✅ 0 issues across 22 modules                                         |
| Modules                  | 31 (22 library + 6 examples + 1 integration + 2 cmd)                  |
| Coverage (avg)           | ~89% (top: decider/dispatcher 100%, low: turso 28.6%)                 |
| Committed binaries       | ❌ 1 tracked (saga-pattern, 7.5MB ELF)                                |
| Uncommitted golden drift | ⚠️ 2 files (codec + middleware golden files reformatted by `nix fmt`) |
| CI (GitHub Actions)      | ❌ BLOCKED by billing ("recent account payments have failed")         |

---

## a) FULLY DONE

### Type Model Improvements

| Item                                                | Commit     | Detail                                                                       |
| --------------------------------------------------- | ---------- | ---------------------------------------------------------------------------- |
| `listing.AggregateRef` → `listing.AggregateListing` | `a9024883` | Disambiguated from `event.AggregateRef` across 7 files (listing/ + storage/) |
| JSON tags on `listing.AggregateListing`             | `a93f0d97` | 5 fields tagged (id, type, version, event_count, last_event_at)              |
| JSON tags on `listing.Page[T]`                      | `a93f0d97` | items + hasMore tagged                                                       |
| JSON tags on `snapshot.Snapshot`                    | `886d88fc` | 5 fields tagged (aggregateId, aggregateType, version, state, createdAt)      |
| Removed `aggregateStatusJSON` helper                | `a93f0d97` | Simplified to struct embedding in `MarshalJSON`                              |

### Golden Test Deduplication

| Item                                   | Commit     | Detail                                                                                    |
| -------------------------------------- | ---------- | ----------------------------------------------------------------------------------------- |
| Shared `eventtest.AssertGolden` helper | `e29add52` | `event/eventtest/golden.go` — 37 lines, takes `(t, path, got, update bool)`               |
| 10 modules migrated to shared helper   | `a433226d` | listing, signing, memory, schema, watermill, middleware, pebble, storage, turso, snapshot |
| Net line reduction                     | `a433226d` | **−183 lines** of duplicated helper code removed                                          |

### Naming & File Hygiene

| Item                                          | Commit     | Detail                                               |
| --------------------------------------------- | ---------- | ---------------------------------------------------- |
| `command/coverage_test.go` → `errors_test.go` | `16efe8bd` | File tests error family aliases, not "coverage"      |
| AGENTS.md updated                             | `51fbc8ab` | listing type name + golden helper pattern documented |

### Previously Done (Earlier Sessions, Already Pushed)

| Item                                  | Detail                                                                                                   |
| ------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| Golden tests in 12 modules            | signing, schema, snapshot, memory, listing, watermill, middleware, otel, pebble, storage, turso, command |
| `command/coverage_test.go` created    | Error family alias tests, WithCommandMetadata, MustParseAggregateType, closed-dispatch                   |
| Benchmark baselines regenerated       | `benchmarks/benchmark-baseline.txt`                                                                      |
| Golden file tracked in codec          | `codec/testdata/golden/raw_passthrough.bin`                                                              |
| Dep budget CI                         | `check-module-layers.sh` + Nix wiring                                                                    |
| OTel abstraction principle documented | AGENTS.md + docs                                                                                         |
| Saga pattern example                  | `example/saga-pattern/` with Go module, tests, README                                                    |

---

## b) PARTIALLY DONE

### `nix fmt` Golden File Drift ⚠️

`nix fmt` reformats JSON golden files (golines breaks long JSON lines), causing test failures on CI:

| File                                                    | Issue                                                                     |
| ------------------------------------------------------- | ------------------------------------------------------------------------- |
| `codec/testdata/golden/json_encode.json`                | Pretty-printed → compact (codec uses `json.Marshal`, not `MarshalIndent`) |
| `middleware/testdata/golden/health-check-response.json` | Array formatting changed                                                  |

**Status**: Golden files regenerated locally (`-update` flag), but not yet committed. Tests pass locally. This will break again next time `nix fmt` runs.

**Root cause**: `golines` formatter in `treefmt-nix` treats `.json` files in `testdata/` as Go-adjacent and reformats them.

**Fix needed**: Either exclude `testdata/**/*.json` from golines in treefmt config, or use `eventtest.AssertGolden` in codec too (codec doesn't depend on event, so needs a different approach).

### Error Re-export Inconsistency

`command/errors.go` and `event/errors.go` re-export `errorfamily` functions, but internal code (`command/dispatcher.go`, `command/store.go`, `query/`, `dispatcher/`) imports `errorfamily` directly, undermining the abstraction.

**Status**: Identified but not fixed. Requires decision: enforce re-exports everywhere or remove them (breaking change → v3).

### `otel/` Golden Test Not Migrated

`otel/golden_test.go` still has a local `assertOtelGolden` helper because otel doesn't depend on event.

**Status**: Accepted. Adding event dependency just for a test helper isn't worth it.

---

## c) NOT STARTED

| Item                                          | Effort                   | Impact | Notes                                                                              |
| --------------------------------------------- | ------------------------ | ------ | ---------------------------------------------------------------------------------- |
| **`go-snaps` migration**                      | Medium (13 go.mod files) | High   | Eliminates ~30 lines boilerplate per module, proper diff output, snapshot cleaning |
| **OTel interface abstraction**                | Large (2-3 days)         | Medium | Phase 3 from packaging hygiene plan. ~10 interfaces, 15+ files                     |
| **`example/user` dead `runServer()` removal** | Trivial                  | Low    | Dead code in example                                                               |
| **Benchmark-regression.sh in CI**             | Small                    | Medium | Script exists but not wired into Nix flake                                         |
| **Coverage regression gate in CI**            | Small                    | Medium | No minimum coverage check                                                          |
| **Playwright E2E tests**                      | Large                    | Low    | SSE client testing                                                                 |
| **`.gitignore` for `saga-pattern` binary**    | Trivial                  | High   | 7.5MB binary committed to root                                                     |
| **Remove tracked binaries from git history**  | Medium                   | High   | `saga-pattern` (7.5MB) at root                                                     |
| **`treefmt-nix` JSON exclusion**              | Small                    | High   | Prevents golden file drift                                                         |

---

## d) TOTALLY FUCKED UP 💥

### 1. Committed Binary: `saga-pattern` (7.5MB ELF)

**Commit**: `8ed46406` — A compiled Go binary was committed directly to the repo root. This bloats the git history permanently.

**Impact**: Every clone downloads 7.5MB of unnecessary binary data. BuildFlow detects it and flags it as HIGH severity.

**Fix**: `git rm saga-pattern`, add to `.gitignore`, optionally use `git-filter-repo` to remove from history.

### 2. Golden Files Fragile to `nix fmt`

`nix fmt` → golines reformats JSON in testdata → golden tests break. This happened twice this session (codec + middleware). Not caught before push because the golden files were regenerated in a different order than `nix fmt` ran.

**Impact**: Tests fail after formatting. Silent regression if `nix fmt` runs after golden regeneration.

### 3. CI Completely Blocked

GitHub Actions CI fails with billing error: "recent account payments have failed". Not a code issue, but means no automated CI validation.

**Impact**: All CI-dependent quality gates (build/vet/test/lint/race/coverage/layers/gosec) are offline.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture & Type Model

1. **`listing.AggregateListing` vs `event.AggregateRef` naming** — Fixed this session, but the fact that two types with similar names existed was a design smell. The rename makes it clear: `AggregateRef` = identity, `AggregateListing` = enriched summary.

2. **Error taxonomy re-exports** — `command/errors.go` has 71 lines of pure delegation to `errorfamily`, but callers bypass them. Either enforce the abstraction or remove it. Current state is the worst of both worlds.

3. **Golden test infrastructure** — Shared `eventtest.AssertGolden` is good, but `go-snaps` would be strictly better: proper diff output, snapshot cleaning, matchers. The main blocker is adding a dep to 13 go.mod files.

### Process & Hygiene

4. **Never commit binaries** — The saga-pattern binary should have been in `.gitignore`. Add a pre-commit hook or CI check.

5. **Golden files + formatters don't mix** — Need to exclude `testdata/` from golines in treefmt config, or golden tests will keep breaking.

6. **Incremental commits** — Previous session dumped 12 golden test files in one massive commit. This session did better (one commit per logical change).

7. **Test before push** — Two golden files were broken after formatting. Should have run full test suite after `nix fmt` + before `git push`.

### Coverage Gaps

8. **`turso` at 28.6%** — Lowest coverage by far. Module has real production code but minimal tests.

9. **`otel` dropped to 73.0%** — Was 96.6% in previous report. Something changed (likely new code without tests).

10. **`storage/sql` sub-package at 34.7%** — SQL dialect helpers undertested.

---

## f) Top #25 Things We Should Get Done Next

Sorted by impact × effort (Pareto order):

| #   | Item                                                                     | Impact | Effort | Type           |
| --- | ------------------------------------------------------------------------ | ------ | ------ | -------------- |
| 1   | **Fix golden file drift: exclude `testdata/` from golines in treefmt**   | High   | Tiny   | Bug fix        |
| 2   | **Remove `saga-pattern` binary from git, add to .gitignore**             | High   | Tiny   | Hygiene        |
| 3   | **Commit the 2 regenerated golden files**                                | High   | Tiny   | Fix            |
| 4   | **Add `go-snaps` to replace manual golden helpers**                      | High   | Medium | DX improvement |
| 5   | **Wire benchmark-regression.sh into Nix flake**                          | Medium | Small  | CI             |
| 6   | **Add coverage regression gate (minimum 80% per module)**                | Medium | Small  | CI             |
| 7   | **Fix `turso` coverage (28.6% → 80%+)**                                  | Medium | Medium | Quality        |
| 8   | **Investigate `otel` coverage drop (96.6% → 73.0%)**                     | Medium | Small  | Quality        |
| 9   | **Add `storage/sql` sub-package tests (34.7% → 80%+)**                   | Medium | Medium | Quality        |
| 10  | **Remove dead `runServer()` from example/user**                          | Low    | Tiny   | Cleanup        |
| 11  | **Consolidate error re-exports (command/errors.go, event/errors.go)**    | Medium | Medium | Architecture   |
| 12  | **Add `go vet` + `staticcheck` to CI (when unblocked)**                  | Medium | Small  | Quality        |
| 13  | **Add doc.go with examples to all modules missing them**                 | Medium | Medium | DX             |
| 14  | **Migrate otel/ and codec/ golden tests to shared helper**               | Low    | Small  | Consistency    |
| 15  | **Add `check-compile-example` to verify all examples compile**           | Medium | Small  | CI             |
| 16  | **Create integration test for listing → storage roundtrip**              | Medium | Medium | Testing        |
| 17  | **Add API stability check to CI (cmd/api-stability already exists)**     | Medium | Small  | CI             |
| 18  | **Write CONTRIBUTING.md (mentioned in sprint 1, status unknown)**        | Medium | Medium | DX             |
| 19  | **Add `gosec` findings to CI output (not just flag them)**               | Low    | Small  | Security       |
| 20  | **Add pre-commit hook to prevent binary commits**                        | High   | Small  | Hygiene        |
| 21  | **Evaluate `govulncheck` integration for dependency scanning**           | Medium | Small  | Security       |
| 22  | **Add property-based tests to remaining modules (only 3/22 have rapid)** | Medium | Large  | Quality        |
| 23  | **OTel interface abstraction (Phase 3 from packaging plan)**             | Medium | Large  | Architecture   |
| 24  | **Add Docker multi-stage build for examples**                            | Low    | Medium | DX             |
| 25  | **Add Playwright E2E for SSE client**                                    | Low    | Large  | Testing        |

---

## g) Top #1 Question I Can NOT Figure Out Myself

**Should we remove the error re-exports in `command/errors.go` and `event/errors.go`?**

These modules re-export ~15 functions from `errorfamily` (Classify, Wrapf, NewRejection, etc.), but internal code (`command/dispatcher.go`, `query/`, `dispatcher/`) imports `errorfamily` directly. This creates an inconsistent abstraction:

- **Option A**: Enforce re-exports everywhere (rename direct imports → use wrapper). Maintains the abstraction boundary but adds overhead.
- **Option B**: Remove re-exports entirely. Consumers import `errorfamily` directly. Simpler, but loses the per-module error namespace.
- **Option C**: Keep as-is (worst of both worlds — some code uses wrappers, some doesn't).

This is a product/design decision that affects the public API surface. It can't be resolved by searching the code.

---

## Coverage by Module (Live Run)

| Module         | Coverage | Trend                                    |
| -------------- | -------- | ---------------------------------------- |
| decider        | 100.0%   | —                                        |
| dispatcher     | 100.0%   | —                                        |
| catalog/schema | 100.0%   | —                                        |
| memory         | 98.2%    | ↓ (was 99.1%)                            |
| command        | 97.3%    | ↑ (was 93.8%, boosted by errors_test.go) |
| id             | 96.4%    | ↑ (was 94.5%)                            |
| catalog        | 95.9%    | —                                        |
| listing        | 94.9%    | ↑ (was 93.8%)                            |
| signing        | 94.1%    | ↑ (was 94.0%)                            |
| query          | 94.3%    | ↓ (was 95.5%)                            |
| watermill      | 94.3%    | ↑ (was 92.6%)                            |
| codec          | 93.3%    | —                                        |
| middleware     | 93.5%    | ↓ (was 98.5%)                            |
| snapshot       | 92.3%    | —                                        |
| projection     | 91.2%    | ↑ (was 90.9%)                            |
| event          | 89.4%    | ↑ (was 88.9%)                            |
| schema         | 89.7%    | —                                        |
| pebble         | 87.2%    | ↓ (was 88.1%)                            |
| storage        | 86.8%    | ↑ (was 71.4%)                            |
| integration    | 80.0%    | —                                        |
| otel           | 73.0%    | ↓ (was 96.6%) ⚠️                         |
| turso          | 28.6%    | — ⚠️                                     |

**Modules with coverage drops**: otel (−23.6%), middleware (−5%), memory (−0.9%), query (−1.2%), pebble (−0.9%). These likely have new untested code added since last measurement.

---

## Session Timeline

```
Session 1 (2026-06-08): Golden test sprint — 12 modules, coverage boost
Session 2 (2026-06-08): Self-review + improvement plan
Session 3 (2026-06-08): Type model fixes (listing rename, json tags, snapshot tags)
Session 4 (2026-06-09): Golden test dedup (10 modules → shared helper), naming fix, push
```

---

## File Change Summary This Arc

```
AGENTS.md                                          | 4 +-
codec/testdata/golden/json_encode.json             | 2 +-
command/{coverage_test.go => errors_test.go}       | 0
event/eventtest/golden.go                          | 37 +++
listing/aggregate_reader.go                        | 8 +-
listing/builder.go                                 | 2 +-
listing/builder_test.go                            | 12 +-
listing/golden_test.go                             | 37 +--
listing/in_memory.go                               | 8 +-
listing/types.go                                   | 12 +-
memory/golden_test.go                              | 42 +---
middleware/golden_test.go                          | 34 +--
middleware/testdata/golden/health-check-response.json | 4 +-
pebble/golden_test.go                              | 35 +--
schema/golden_test.go                              | 35 +--
signing/golden_test.go                             | 42 +---
snapshot/golden_test.go                            | 38 +--
snapshot/store.go                                  | 10 +-
storage/golden_test.go                             | 68 +++--
storage/sql_aggregate_reader.go                    | 8 +-
turso/golden_test.go                               | 35 +--
watermill/golden_test.go                           | 35 +--
25 files changed, 218 insertions(+), 475 deletions(-)
```
