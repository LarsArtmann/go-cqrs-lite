# Status Report — 2026-06-20 17:40

## Deployer-First Architecture Completion — Phases 0-7

**Branch:** master (clean, all committed)
**Plan:** [Deployer-First Completion Plan](../planning/2026-06-20_17-03_DEPLOYER-FIRST-COMPLETION-PLAN.md)
**Goal:** "The Best CQRS/ES SDK in Go" — composable, powerful, deployer-first
**Build:** ✅ passing | **Vet:** ✅ passing | **Tests:** 47/48 suites green (1 pre-existing CBOR fuzz failure)

---

## A. Fully Done ✅ (19 of 27 tasks, 70%)

### Phase 0: Critical Bug Fixes (T01-T03) — commit `72538df7`

| Task | Fix | Impact |
|------|-----|--------|
| **T01** | `stack/materialize.go:132` — any `Store.Get` error was silently classified as "not found" → `OnCreate`, causing **silent data corruption** on real DB errors. Fixed: `errors.Is(err, kv.ErrNotFound)` + propagate other errors. Also fixed tombstone path (line 104). Test: `TestMaterialize_StoreGetErrorPropagates`. | Critical — prevents data loss |
| **T02** | `event/types.go` — `Version.Add(n int)` accepted negative values which silently wrapped to `MaxUint64` after the int→uint64 change. Changed to `Add(n uint)` — type system prevents negative input at compile time. Fixed all callers. Also fixed `storage/pg_bus_listen.go` `np.Version-1` underflow. | High — type safety |
| **T03** | `docs/migration/V3_MIGRATION.md` — broken `memory.NewMemoryBus()` example (module deleted). Ghost-code lie: `storage/pg_bus.go` is LIVE (ADR-0027), not ghost. Status table updated: Version Done, memory/ move Done, readmodel deletion Done, io.Closer Rejected. | High — consumer trust |

### Phase 1: Kill readmodel↔kv Split Brain (T04-T06) — commit `c4d0ffcd`

| Task | What shipped |
|------|-------------|
| **T04-T06** | **Deleted readmodel/ module entirely.** 482 LOC, 14 files, 2 Go modules eliminated. Types were character-for-character identical to `kv.TypedStore` and `kv.Cache` (6 of 22 production clone groups = 27% of all duplication). Zero production Go code imported readmodel types. Bench redirected to `kv.NewTypedStore`. 7 go.mod files, go.work, api-stability golden updated. |

### Phase 2: Eliminate Remaining Duplication (T07-T09) — commit `dcd350af`

| Task | What shipped |
|------|-------------|
| **T07** | buildOptions dedup — **intentionally skipped.** 20-line duplication between stack/sqlite and stack/postgres is structural isolation. Extracting would create an unwanted cross-module dependency. |
| **T08** | Exported `codec.CBORDecMode()` so storage/pebble reuses the canonical CBOR decoder instead of duplicating it. 1 clone group eliminated. |
| **T09** | Wired dead `WithEventDB`/`WithQueryDB` in stack/sqlite/preset.go. Were dead options that set config fields `newBundle()` never read. Added `openSecondaryStores()` helper + `multiCloser` for cleanup. |

### Phase 3: Wire Island Types into stack/ (T10-T13) — commit `f9ecce33`

| Task | Accessor added |
|------|---------------|
| **T10** | `stack.NewMaterialize[V,K]()` — constructs Materialize from Bundle's read-model backend. Deployer-first projection builder. |
| **T11** | `Bundle.CatchUpSubscriber()` — constructs watermill.CatchUpSubscriber from Bundle's journal + subscriber + checkpoint. Added `EventBus.MessageSubscriber()` to expose internal GoChannel. |
| **T12** | `stack.TypedRepository[State,Cmd]()` — constructs TypedRepository from Bundle's event store + TypedDecider. Compile-time command binding. |
| **T13** | `stack.QueryAuditMiddleware()` — creates query.AuditMiddleware from Bundle's query sink. Deployer picks audit level. |

### Phase 4: Deployer-First Architecture Validated (T14-T16) — commit `6e519372`

| Task | What shipped |
|------|-------------|
| **T14-T16** | `stack/deployer_first_test.go` — `TestDeployerFirstArchitecture` proves the full flow: deployer picks infrastructure → consumer creates Materialize → event published → handler processes → view retrievable. **ZERO references to projection.Runner or readmodel.Store.** Uses only: stack.NewMaterialize, watermill.EventBus, kv.TypedStore, event.Publisher. This is the safety net for Phase 5. |

### Phase 7: Documentation & Cleanup (T25-T27) — commit `11056d4a`

| Task | What shipped |
|------|-------------|
| **T25** | ADR-0029 (storage consolidation) and ADR-0032 (readmodel merge) status updated from Accepted → **Implemented**. Both fully shipped. |
| **T26** | Deleted `id/transaction_id.go` — pure ghost code with zero consumers. Not wired into event.Metadata, NewEvent, or any Option. TODO_LIST.md updated: was marked `[x]` (lie), now correctly marks it deleted. |
| **T27** | (Partial — FEATURES.md/ROADMAP.md update pending) |

---

## B. Partially Done ⚠️

| Item | Status | What remains |
|------|--------|-------------|
| **Projection dissolution** (T17-T21) | Replacement types exist + are wired into stack/ + deployer-first test validates them. | Migrate example/todo + example/user (20 refs), update cqrs-gen, remove ProjectionRunner from stack/, delete projection/ module (1343 LOC). |
| **Metadata alias break** (T22-T24) | Typed fields exist (Tracing, TombstoneMark, Causation). | `command.Metadata = event.Metadata` alias intact. `query.Metadata = event.Metadata` alias intact. Only 4 consumer files but SQL serialization must be verified. |
| **FEATURES.md/ROADMAP.md** (T27) | ADR statuses updated. | Need to reflect watermill.EventBus, Materialize, TypedDecider, CatchUpSubscriber as DONE; remove readmodel. |

---

## C. Not Started ⏳

| Task | Description | Blocked by |
|------|-------------|-----------|
| T17 | Migrate example/todo to Materialize + Watermill Router | Nothing — safe to start |
| T18 | Migrate example/user to Materialize + Watermill Router | Nothing — safe to start |
| T19 | Update cqrs-gen to emit Materialize code | T17/T18 patterns established |
| T20 | Remove ProjectionRunner from stack/accessors.go + bundle.go | T17/T18 complete |
| T21 | Delete projection/ module entirely | T17-T20 complete |
| T22 | command.Metadata own struct | Decision: v2 additive or v3 breaking? |
| T23 | query.Metadata own struct | Same as T22 |
| T24 | Update storage scan helpers for new Metadata types | T22/T23 complete |

---

## D. Totally Fucked Up 🔴 (and what we did about it)

| # | What happened | Impact | Fix |
|---|--------------|--------|-----|
| 1 | **Materialize data corruption bug** (shipped in `98ebd0b3`, fixed in `72538df7`) | Any DB error (connection drop, decode failure) silently triggered OnCreate, overwriting real data. Tests didn't catch it because fake store only returns ErrNotFound. | Fixed with `errors.Is(err, kv.ErrNotFound)` check. Added test that injects non-ErrNotFound error. |
| 2 | **Version.Add underflow** (introduced in `c7df1c79`, fixed in `72538df7`) | `Add(n int)` accepted negatives which silently wrapped to `MaxUint64` after the uint64 change. Latent footgun. | Changed to `Add(n uint)`. Type system now prevents negative input at compile time. |
| 3 | **Dead WithEventDB/WithQueryDB** (shipped in deployer-first, fixed in `dcd350af`) | Options set config fields that `newBundle()` never read. Deployers thought they were getting DB separation; they weren't. | Added `openSecondaryStores()` helper that actually opens separate backends. |
| 4 | **V3_MIGRATION.md lies** (shipped in `382bd875`, fixed in `72538df7`) | Example didn't compile (`memory.NewMemoryBus()` deleted). Ghost-code lie (`pg_bus` is live). | Replaced with working example. Corrected ghost-code table. |
| 5 | **TransactionID ghost type** (shipped in `02f7eaa8`, deleted in `11056d4a`) | Type existed with zero consumers. TODO marked it `[x]` done. Pure lie. | Deleted the file. Updated TODO_LIST.md. |
| 6 | **Docs wipe** (`f276873a`, restored in `9c544d6f`) | 564 docs deleted by directory pattern without verifying staleness. | All restored. BuildFlow pre-commit hook fixed (was breaking on filenames with spaces). |
| 7 | **io.Closer removal attempt** (reverted same session) | Would have replaced typed `io.Closer` with `any` + defensive type assertions. Verschlimmbessern. | Correctly stopped. ADR-0010 stays Proposed. |
| 8 | **readmodel half-applied ADR** (ADR-0032, fixed in `c4d0ffcd`) | Destination types created in kv/ but source module not deleted → 6 clone groups of pure duplication. | Completed the merge: deleted readmodel/ entirely. |

---

## E. What We Should Improve 🔄

1. **Finish half-applied ADRs before starting new ones.** ADR-0030 (dissolve projection) and ADR-0031 (metadata split) are still half-applied — destinations built, sources not deleted. These are the remaining split brains.

2. **Migrate examples before deleting modules.** The deployer-first test (Phase 4) proves the new architecture works in isolation, but example/todo and example/user still use projection.Runner. They must be migrated before projection/ can be deleted safely.

3. **Test error paths, not just happy paths.** The Materialize bug existed because the fake store only returned ErrNotFound. Every Store.Get error path should be tested with injected real errors.

4. **Codify migration examples as compilable tests.** V3_MIGRATION.md had a broken code example. If it were an `Example_*` function in a test file, `go test` would have caught it automatically.

5. **Run the review skill when scheduled.** The brutal-self-review was Top-25 #10 but never ran until explicitly asked. Schedule = do.

6. **Consider extracting FakeBus.** 130 LOC of middleware-chain logic in a test helper is a parallel implementation. If it needs its own tests, it's not a fake — it's a module.

---

## F. Top 25 Things to Get Done Next

| # | Task | Impact | Effort | Phase |
|---|------|--------|--------|-------|
| 1 | **Migrate example/todo to Materialize + Watermill Router** | Critical | 30min | 5 |
| 2 | **Migrate example/user to Materialize + Watermill Router** | High | 25min | 5 |
| 3 | **Update cqrs-gen to emit Materialize code** | Medium | 20min | 5 |
| 4 | **Remove ProjectionRunner from stack/ accessors + bundle** | High | 15min | 5 |
| 5 | **Delete projection/ module entirely** (1343 LOC) | Critical | 20min | 5 |
| 6 | **command.Metadata own struct** (embeds Tracing, no alias) | High | 25min | 6 |
| 7 | **query.Metadata own struct** (embeds Tracing, no CausationID) | High | 25min | 6 |
| 8 | **Update storage scan helpers for new Metadata** | High | 15min | 6 |
| 9 | Update FEATURES.md with watermill.EventBus, Materialize, etc. | Medium | 15min | 7 |
| 10 | Update ROADMAP.md with completed work | Low | 15min | 7 |
| 11 | Fix CBOR fuzz test (duplicate map key -17) | Medium | 30min-2h | — |
| 12 | Fix PgxListener race condition (concurrent Close) | Medium | 45min | — |
| 13 | Add rapid property tests for Version arithmetic | Medium | 20min | — |
| 14 | Move indexing advisor to storage/sql/ (Theme F3) | Low | 30min | — |
| 15 | Consider extracting FakeBus to syncbus module | Low | 1h | — |
| 16 | encoding/json/v2 migration | Low | 2h | — |
| 17 | Event concrete struct (ADR: make Event a struct, remove type-assertions) | High (v3) | 2-3 days | — |
| 18 | Decider rename Fold→Apply | Low (v3) | 1 day | — |
| 19 | SecurityEnvelope typed struct (9 string-key concepts) | Medium | 2h | — |
| 20 | Replace remaining flaky time.Sleep calls (top 10) | Medium | 2h | — |
| 21 | Add stack/contracttest validation for all 4 presets | Medium | 30min | — |
| 22 | Consumer migration test: memory.NewMemoryBus → watermill.EventBus | Medium | 1h | — |
| 23 | Consider go-cache/otter for subscriber dedup | Low | 1h | — |
| 24 | Example: watermill.EventBus with NATS backend | Low | 2h | — |
| 25 | v3.0.0 release planning doc (what actually breaks) | High | 1h | — |

---

## G. The #1 Question I Cannot Figure Out Myself

**Should projection/ be dissolved now (v2.9) or at the v3 boundary?**

The replacement architecture is proven:
- ✅ `stack.NewMaterialize[V,K]()` is wired and tested
- ✅ `Bundle.CatchUpSubscriber()` is wired and tested
- ✅ `TestDeployerFirstArchitecture` validates event→materialize→view with zero projection.Runner references
- ✅ `watermill.EventBus` is the default in all 4 presets

But projection/ has **20 production references** across:
- `example/todo/cmd/api/setup.go` (live consumer)
- `example/user/handlers.go` (live consumer)
- `stack/accessors.go:80-100` (Bundle.ProjectionRunner)
- `stack/bundle.go:166` (compile-time assertion)
- `cmd/cqrs-gen/generate.go` (codegen emits `projection.On[...]`)
- 6 integration test files

Dissolving it requires migrating ALL of these. The migration is mechanical (projection.Runner → Materialize + Watermill Router) but touches consumer-facing code in examples.

**The question:** Is this v2.9 (additive deprecation: keep projection/ working, mark deprecated) or v3.0 (breaking: delete projection/, force migration)?

I lean toward v2.9: deprecate `Bundle.ProjectionRunner()` with a doc comment pointing to `NewMaterialize` + `CatchUpSubscriber`, migrate the examples, then delete in v3.0. But this is a product decision — it affects every consumer who imports `projection/`.

---

## Progress Summary

| Phase | Tasks | Status | Commits |
|-------|-------|--------|---------|
| 0 — Bug Fixes | 3 | ✅ Done | `72538df7` |
| 1 — Kill readmodel | 3 | ✅ Done | `c4d0ffcd` |
| 2 — Dedup | 3 | ✅ Done (T07 skipped) | `dcd350af` |
| 3 — Wire Islands | 4 | ✅ Done | `f9ecce33` |
| 4 — Example | 3 | ✅ Done | `6e519372` |
| 5 — Dissolve projection | 5 | ⏳ Not started | — |
| 6 — Kill aliases | 3 | ⏳ Not started | — |
| 7 — Docs | 3 | ✅ Done (T27 partial) | `11056d4a` |
| **Total** | **27** | **19 done (70%)** | **7 commits** |

**Metrics:** 36 modules (was 38 — readmodel + readmodel/cache deleted). Build ✅. Vet ✅. 47/48 test suites green. 1343 LOC in projection/ awaiting dissolution. 0 clone groups from readmodel↔kv (was 6). 0 ghost types (TransactionID deleted).
