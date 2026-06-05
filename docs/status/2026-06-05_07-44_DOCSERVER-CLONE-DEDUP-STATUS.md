# go-cqrs-lite — Comprehensive Status Report

**Date:** 2026-06-05 07:44 CEST
**Branch:** master @ `c9a098f7`
**Release:** v2.1.0 (tagged 2026-06-03, **not yet pushed to remote**)
**Go:** 1.26.3 · **LOC:** ~67,300 · **Files:** ~500 · **Modules:** 30 (22 library + 6 examples + 2 cmd)
**Session focus:** Code-quality dedup (docserver test clones) + planning-doc research

---

## Executive Summary

The v2.1.0 release is **frozen locally but un-pushed**. The post-release period has been dominated by **architectural brainstorming** toward a v3.0 design (`docs/brainstorming/toward-perfect-go-cqrs-lite.html` 48 KB, 9 commits), culminating in a storage-environment mapping research piece (`docs/research/storage-environment-mapping.html` 1,680 lines, untracked). The only code change in this session was eliminating 8 code clones in `catalog/docserver/docserver_test.go` by extracting a `newTestRequest` helper — the file now reports **0 clone groups at threshold 50**. All 9 catalog test packages still pass; `go vet` and `gofmt` are clean.

The repository is **stable but stale** on the public face: v2.1.0 needs to be pushed, ADR-0005 is still missing, ROADMAP.md is still missing, and the buildFlow pre-commit hook is still bypassed with `--no-verify`. The brainstorming/ folder is rich (2 large design pieces) but the actionable TODO list is empty.

---

## A) FULLY DONE ✅

### This Session (2026-06-05)

- [x] **Eliminated 8 code clones in `catalog/docserver/docserver_test.go`** — extracted `newTestRequest(path string) *http.Request` helper alongside the existing `testProvider`/`testServer`/`decodeJSON` helpers; replaced 8 literal-path 3-line `httptest.NewRequestWithContext(...)` blocks + 2 loop-based occurrences using `tc.path`
- [x] Verified: `go test ./...` (catalog module) — 9/9 packages PASS
- [x] Verified: `go vet ./docserver/...` — clean
- [x] Verified: `gofmt -l docserver/` — clean
- [x] Verified: `art-dupl --semantic -t 50` — **0 clone groups** (was 8 at threshold 10)

### v2.1.0 Release (committed 2026-06-03, `d629b7b3`)

- [x] All inter-module `require` bumped v2.0.0 → v2.1.0 (22 go.mod files)
- [x] CHANGELOG.md rewritten with full Added/Changed/Performance/Fixed/Removed
- [x] 25 annotated tags created (1 root + 22 library + 2 cmd) — **all local only**
- [x] Build clean, 40 test packages pass, lint 11/12 zero issues
- [x] 9 performance improvements (alloc reductions across event/signing/listing/catalog/memory)
- [x] 6 production bug fixes (HealthCheck OOM, SQL race, Pebble close leak, etc.)
- [x] 2 new features (`query.TypedHandler[Q,R]`, `listing.CacheInvalidationMiddleware`)
- [x] 17 scale benchmarks + `nix run .#bench` + `benchstat-compare` script

### Post-Release Brainstorming (2026-06-03 → 2026-06-05, 17 commits)

- [x] **`docs/brainstorming/toward-perfect-go-cqrs-lite.html`** (48,880 bytes) — full architectural redesign proposal: per-backend packages (no more god-factory), `relay/` (renamed from `journalpublisher/`), `readmodel/` (renamed from `data/`), env-detector, runtime backend selection
- [x] **`docs/research/storage-environment-mapping.html`** (1,680 lines, untracked) + `.md` (17,682 bytes) — exhaustive mapping of every storage touchpoint to native backend per runtime environment
- [x] Editorial cleanups across brainstorm pieces (removed AI-slop, modernized language, removed phantom outbox, removed Tombstone from Sink, fixed two domain model contradictions)
- [x] Visual redesign with modern Inter/JetBrains Mono/Newsreader typography + syntax highlighting

### Module State (unchanged from v2.1.0 release)

| Module          | Status              | Coverage | Tests | Lint Issues      |
| --------------- | ------------------- | -------- | ----- | ---------------- |
| `event/`        | ✅ FULLY_FUNCTIONAL | 89–94%   | 180+  | 0                |
| `command/`      | ✅ FULLY_FUNCTIONAL | 100%     | 42    | 0                |
| `query/`        | ✅ FULLY_FUNCTIONAL | 94%      | 22    | 0                |
| `decider/`      | ✅ FULLY_FUNCTIONAL | 100%     | 41    | 0                |
| `id/`           | ✅ FULLY_FUNCTIONAL | 98%      | 48    | 0                |
| `dispatcher/`   | ✅ FULLY_FUNCTIONAL | 100%     | 17    | 0                |
| `schema/`       | ✅ FULLY_FUNCTIONAL | 90%      | 18    | 0                |
| `snapshot/`     | ✅ FULLY_FUNCTIONAL | 92%      | 11    | 0                |
| `codec/`        | ✅ FULLY_FUNCTIONAL | 93%      | 13    | 0                |
| `memory/`       | 🧪 TESTING_ONLY     | 99%      | 40+   | 0                |
| `catalog/`      | ✅ FULLY_FUNCTIONAL | 94%      | 242   | 7 (pre-existing) |
| `middleware/`   | ✅ FULLY_FUNCTIONAL | 99%      | 89    | 0                |
| `signing/`      | ✅ FULLY_FUNCTIONAL | 95%      | 63    | N/A              |
| `storage/`      | ✅ FULLY_FUNCTIONAL | 89%      | 125   | N/A              |
| `projection/`   | ✅ FULLY_FUNCTIONAL | 91%      | 54    | N/A              |
| `listing/`      | ✅ FULLY_FUNCTIONAL | 95%      | 12    | N/A              |
| `watermill/`    | ✅ FULLY_FUNCTIONAL | 90%      | 14    | N/A              |
| `pebble/`       | ✅ FULLY_FUNCTIONAL | 87%      | 33    | N/A              |
| `turso/`        | ✅ FULLY_FUNCTIONAL | 29%      | 16    | N/A              |
| `otel/`         | ✅ FULLY_FUNCTIONAL | 96%      | 21    | N/A              |
| `integration/`  | ✅ FULLY_FUNCTIONAL | —        | 38    | N/A              |
| `cmd/cqrs-gen/` | 🔧 TOOL             | 71%      | 22    | N/A              |

### Documentation State

- [x] 9 ADRs in `docs/adr/` (0001, 0002, 0003, 0004, **0006, 0007, 0008, 0009** — 0005 still missing)
- [x] `docs/DOMAIN_LANGUAGE.md` (10,301 bytes)
- [x] `docs/ARCHITECTURE_PATTERNS.md` (3,172 bytes)
- [x] `docs/STORAGE_GUIDE.md` — performance comparison
- [x] `docs/CONTRIBUTING.md` rewritten
- [x] `docs/api_surface.txt` (28,468 bytes) — golden file for `cmd/api-stability`
- [x] **110+ status reports** in `docs/status/` (growing, no cleanup)

---

## B) PARTIALLY DONE ⚠️

### 1. `docs/research/storage-environment-mapping.html` — Untracked Render

- 1,680 lines, styled HTML render of the source `.md`
- Source `.md` (17,682 bytes) **is tracked** in commit `c9a098f7`
- The HTML sibling is untracked — needs review and `git add` if we want to keep rendered versions in tree

### 2. `docs/brainstorming/toward-perfect-go-cqrs-lite.html` — Big Idea, No Code

- 48 KB of architecture proposal: replace `data/` with `readmodel/`, replace `journalpublisher/` with `relay/`, kill god-factory for env-detector + per-backend packages
- Beautifully rendered, multiple editorial passes complete
- **Zero implementation has started.** Still in "design" state.

### 3. Turso Module — 28.6% Coverage

- 16 tests pass; CRUD tests added in v2.1.0
- **Gap:** Remote `SyncDB` Push/Pull/Checkpoint operations untested locally (need testcontainers or remote Turso server)

### 4. Pebble Module — Missing Two Interfaces

- ✅ Store, Journal, SeekableJournal implemented
- ❌ `BackwardsSource` not implemented
- ❌ `StreamLoader` not implemented
- Coverage 86.5%

### 5. SQL Storage — No Journal Implementation

- Memory and Pebble both implement `Journal` / `SeekableJournal`
- SQL storage has `Store` only — no `ReadAll()` / `ReadFrom()` for cross-aggregate replay
- **Impact:** Projection replay from SQL requires full table scan workaround

### 6. Catalog Module — 7 Pre-existing Lint Issues

- `forcetypeassert` ×1 (`reflect.go:68`)
- `gochecknoglobals` ×1 (`schemaCache` sync.Map)
- `goconst` ×2 (`"CreateOrder"` repeated 3× in tests)
- `godoclint` ×1
- `unused` ×1 (`jsonKeyType`)
- `wrapcheck` ×1 (`schema.ToAny`)

### 7. BuildFlow Pre-Commit Hook — Broken

- Exits code 1 for `pkg/` / `internal/` suggestions that don't apply to this monorepo
- Bypassed with `--no-verify` on every commit
- Undermines the safety net

### 8. Stale Formatting Remnants (carryover from 06-03 release)

- 3 files with minor leftover formatting: `listing/in_memory.go`, `pebble/helpers.go`, `watermill/benchmark_test.go`
- Not committed since v2.1.0

---

## C) NOT STARTED 🔴

### From v3.0 Brainstorming — Zero Implementation

The `toward-perfect-go-cqrs-lite.html` proposal implies breaking changes that have **zero code** behind them:

1. Rename `data/` package → `readmodel/` (renamed in docs only)
2. Rename `journalpublisher/` → `relay/` (renamed in docs only)
3. Replace `readmodel.NewFactory(...)` god-factory with per-backend packages + env detector
4. Add runtime backend selection to `readmodel/` proposal
5. Move `Tombstone` out of `Sink` interface (tombstoning is a projection concern)

### Open TODO Items (carried forward, no progress)

**`[v2]` Deferred (next major):**
- TransactionID branded type
- `io.Closer` removal from core interfaces
- Split `event.Store` into Writer/Reader/Deleter
- Make event core truly immutable

**`[FUTURE]` Long-term:**
- Outbox pattern implementation
- Schema registry (JSON Schema middleware)
- Catalog diff / breaking-change detection tool
- High-level test utilities (AggregateTester, ProjectionTester)
- Server-side timestamps (ServerReceivedAt, ServerStoredAt)
- Bi-temporal support (ValidAt, WithValidAt)
- HLC (Hybrid Logical Clock)
- Pull-before-push sync protocol
- Rebase mechanism
- Network simulator for testing
- Multi-client test harness
- Thin PostgreSQL store adapter (no Watermill)
- Thin NATS bus adapter
- Distributed consensus (Raft/CRDT)
- Time-series event query language
- Documentation site (Docusaurus/MkDocs)
- pkg.go.dev hosting
- Transactional projection contract
- Filter/Predicate types
- ContextQuerier/ContextAppender interfaces
- Hybrid service example
- Multi-engine storage via sqlc
- Schema migration tool

**`[BLOCKED]` External:**
- PostgreSQL integration tests (testcontainers) — needs Docker
- Move `example/todo` to own repo — manual repo creation
- Push signing v1.0.0 tag — manual tag + push
- Change LICENSE to MIT/Apache-2.0 — owner decision

---

## D) TOTALLY FUCKED UP 💥

### 1. 110+ Status Reports, Zero Cleanup

- `docs/status/` has 110+ markdown files. 37+ are from May 27 alone.
- `archive/` subdirectory exists but is mostly empty
- **Problem:** Signal-to-noise ratio is catastrophic. Cannot find relevant status.
- **Root Cause:** Every session dumps a status file without cleanup.

### 2. 38+ Planning Documents, No Consolidation

- `docs/planning/` has 75 files (40 in `archive/`)
- Multiple overlapping execution plans
- **No single authoritative plan** — you have to read 5+ files to understand priorities

### 3. No `ROADMAP.md`

- Flagged as missing in multiple sessions. **Never created.**
- Long-term direction is scattered across 110 status files and 75 planning docs

### 4. ADR Sequence Gap — ADR-0005 Missing

- ADRs: 0001, 0002, 0003, 0004, 0006, 0007, 0008, 0009
- **0005 was skipped during creation and never filled in**
- Confusing for anyone reading the ADR index

### 5. v2.1.0 Tags Not Pushed to Remote

- 25 tags exist locally only (1 root + 22 library + 2 cmd)
- `git push --follow-tags` not yet executed
- **Risk:** Machine loss = tags gone. Remote still shows v2.0.0 as latest.

### 6. Every Commit Bypasses Pre-Commit Hook

- `--no-verify` used on every commit because BuildFlow exits 1
- Safety net is dead

### 7. Untracked HTML Render of Committed Research

- `docs/research/storage-environment-mapping.html` (56,987 bytes) is untracked
- The source `.md` is committed in `c9a098f7` — the HTML is a styled render
- Decision needed: keep rendered HTML in tree, or build it on demand?

### 8. Fresh Planning Doc Just Appeared

- `docs/planning/2026-06-05_MODULE_IMPROVEMENT_PLAN.md` (15,065 bytes) — untracked
- Self-described as: "Generated 2026-06-05, full audit of all 21 library modules (scc, coverage, lint, API surface, docs, code smells, existing plans), 62 tasks all ≤12 min each"
- **Not committed** — review contents before commit, as it overlaps with the brainstorming pieces

### 8. Brainstorming has No Linkage to Code

- `toward-perfect-go-cqrs-lite.html` is a 48 KB beautiful document
- Zero issues, PRs, or branches reference it
- Risk: design crystallizes in docs but never lands in code

---

## E) WHAT WE SHOULD IMPROVE 🔧

### Critical (Do First)

1. **Push v2.1.0 tags to remote** — risk of data loss; remote still shows v2.0.0
2. **Commit the untracked research file** — `docs/research/storage-environment-mapping.{html,md}` is untracked work
3. **Fix or remove BuildFlow pre-commit hook** — currently useless, blocks every commit
4. **Archive old status reports** — move 100+ stale reports to `docs/status/archive/`
5. **Fill ADR-0005 gap** — either write it or renumber 0006-0009

### High Priority

6. **Create `ROADMAP.md`** — long-term direction document, missing for months
7. **Implement SQL Journal** — parity gap with Memory/Pebble; projections need it
8. **Implement Pebble `BackwardsSource`** — interface exists but no implementation
9. **Turso coverage** — 28.6% is the lowest in the project
10. **Fix 7 catalog lint issues** — last module with lint debt
11. **`eventtest/fake_store.go` audit** — 273 lines duplicating MemoryStore

### Medium Priority

12. **Convert brainstorming to actionable tasks** — turn 48 KB of v3.0 design into a structured issue list
13. **Update example `go.mod` files** to reference v2.1.0 (currently v2.0.0 in replace)
14. **Benchmark baseline update** post-v2.1.0 perf work
15. **Stale doc audit** — `docs/benchmarks`, `docs/planning` may reference pre-v2 APIs
16. **Coverage gate** — enforce 80% minimum in CI

### Nice to Have

17. **Documentation site** (Docusaurus/MkDocs)
18. **pkg.go.dev readiness** — ensure godoc is complete
19. **GitHub Release Notes** — auto-generate from CHANGELOG on tag push
20. **API stability CI** — integrate `cmd/api-stability` into pipeline

---

## F) Top 25 Things We Should Get Done Next

### Tier 1: Immediate (This Session / Today)

| #   | Task                                                | Impact                       | Effort    |
| --- | --------------------------------------------------- | ---------------------------- | --------- |
| 1   | Decide on `storage-environment-mapping.html` (tracked `.md` exists; render is untracked) | Tree hygiene             | 30 sec   |
| 2   | Commit `catalog/docserver/docserver_test.go` dedup | Clean working tree           | 1 minute  |
| 3   | Push v2.1.0 tags to remote (`git push --follow-tags`) | Prevents data loss        | 1 command |

### Tier 2: This Week

| #   | Task                                                          | Impact                       | Effort    |
| --- | ------------------------------------------------------------- | ---------------------------- | --------- |
| 4   | Fix or remove BuildFlow pre-commit hook                       | Unblocks normal git workflow | 15 min    |
| 5   | Archive 100+ stale status reports to `docs/status/archive/`   | Signal over noise            | 10 min    |
| 6   | Fill ADR-0005 gap (or renumber 0006-0009)                     | Documentation integrity      | 15 min    |
| 7   | Fix 7 catalog lint issues                                     | Zero-lint across ALL modules | 30 min    |
| 8   | Create `ROADMAP.md` with v3.0 vision from brainstorm pieces   | Long-term clarity            | 2 hours   |
| 9   | Implement SQL Journal (`ReadAll`/`ReadFrom`)                  | Storage parity               | 2–4 hours |
| 10  | Implement Pebble `BackwardsSource`                           | Interface completeness       | 1 hour    |

### Tier 3: Next Sprint

| #   | Task                                                                                | Impact                   | Effort  |
| --- | ----------------------------------------------------------------------------------- | ------------------------ | ------- |
| 11  | Turso integration tests (testcontainers)                                            | Coverage 29% → 80%+      | 4 hours |
| 12  | Update all examples to v2.1.0                                                       | Consumer experience      | 30 min  |
| 13  | `eventtest/fake_store.go` audit — reduce duplication with MemoryStore               | Maintainability          | 1 hour  |
| 14  | Benchmark baseline update post-v2.1.0 perf work                                     | Regression detection     | 1 hour  |
| 15  | Integrate `cmd/api-stability` into CI pipeline                                      | API contract enforcement | 2 hours |
| 16  | Convert `toward-perfect-go-cqrs-lite.html` into GitHub issues / ROADMAP.md sections | Actionable design        | 2 hours |
| 17  | Apply brainstorm renames in code (`data/`→`readmodel/`, `journalpublisher/`→`relay/`) | v3.0 first cut         | 4 hours |

### Tier 4: Strategic

| #   | Task                                                          | Impact                    | Effort  |
| --- | ------------------------------------------------------------- | ------------------------- | ------- |
| 18  | Documentation site (Docusaurus/MkDocs)                        | Public visibility         | 1 day   |
| 19  | pkg.go.dev readiness audit                                    | Developer experience      | 4 hours |
| 20  | Thin PostgreSQL adapter (no Watermill dep)                    | Adoption                  | 1 day   |
| 21  | High-level test utilities (AggregateTester, ProjectionTester) | Testing ergonomics        | 1 day   |
| 22  | Schema registry / event validation middleware                 | Runtime safety            | 1 day   |
| 23  | Outbox pattern implementation                                 | Transactional consistency | 2 days  |
| 24  | Bi-temporal support (ValidAt, WithValidAt)                    | Compliance use cases      | 1 day   |
| 25  | Thin NATS bus adapter                                         | Transport flexibility     | 1 day   |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should we pivot now from "polish v2.1.0 / push tags" to "implement v3.0 brainstorm", or keep grinding the existing debt first?**

Specifically, I see a tension I cannot resolve without you:

- The **brainstorming pieces** (48 KB toward-perfect design, 56 KB storage-environment mapping) describe a v3.0 that **abandons the `data/` package**, **renames `journalpublisher/` to `relay/`**, and **kills the god-factory pattern**. That's a major rewrite, not an extension.
- The **v2.1.0 debt** (catalog lint, SQL Journal gap, Turso coverage, missing ROADMAP, ADR-0005, unpushed tags) is real and growing.
- If we land v3.0 brainstorm code first, the v2.x debt becomes **abandoned** (lint fixes for code that's about to be deleted, ADR-0005 about a `data/` package that no longer exists, etc.).
- If we grind v2.x debt first, the v3.0 design crystallizes further in docs but no closer to code.

I cannot decide which is more valuable because:
- **Push-tags-first** is the safe move (prevents data loss, low risk) but freezes v2.1.0 as the public release while we work on v3.0
- **Brainstorm-first** is the ambitious move (real progress toward the next major) but might leave v2.x debt forever
- **Parallel** risks half-done work in both directions

What is your priority signal here — "ship v2.1.0 fully then plan v3.0", or "v3.0 is the goal, v2.1.0 is already done enough"?

---

## Appendix: Raw Build / Lint / Test Results (This Session)

### Build

```
✅ Clean — catalog/docserver compiles
```

### Test (catalog module, 9 packages, 0 failures)

```
ok  github.com/larsartmann/go-cqrs-lite/catalog/v2                 0.003s
ok  github.com/larsartmann/go-cqrs-lite/catalog/v2/asyncapi         0.002s
ok  github.com/larsartmann/go-cqrs-lite/catalog/v2/d2               0.002s
ok  github.com/larsartmann/go-cqrs-lite/catalog/v2/docserver        0.009s   (12/12 tests, including 8 that used the deduplicated pattern)
ok  github.com/larsartmann/go-cqrs-lite/catalog/v2/eventcatalog    0.004s
ok  github.com/larsartmann/go-cqrs-lite/catalog/v2/internal/caseutil 0.001s
?   github.com/larsartmann/go-cqrs-lite/catalog/v2/internal/cattest  [no test files]
ok  github.com/larsartmann/go-cqrs-lite/catalog/v2/openapi         0.002s
ok  github.com/larsartmann/go-cqrs-lite/catalog/v2/schema          0.002s
```

### Lint / Format

```
$ go vet ./docserver/...          → (no output, clean)
$ gofmt -l docserver/             → (no output, clean)
$ art-dupl -t 50 docserver_test.go → 0 clone groups
```

### Diff Summary

```
catalog/docserver/docserver_test.go | 46 +++++++++++++------------------------
1 file changed, 16 insertions(+), 30 deletions(-)
```

### Git State (at time of report)

```
On branch master
Your branch is up to date with 'origin/master'.

Changes not staged for commit:
        modified:   catalog/docserver/docserver_test.go

Untracked files:
        docs/planning/2026-06-05_MODULE_IMPROVEMENT_PLAN.md
        docs/research/storage-environment-mapping.html   (render of tracked .md)
        docs/status/2026-06-05_07-44_DOCSERVER-CLONE-DEDUP-STATUS.md  (this report)
```
