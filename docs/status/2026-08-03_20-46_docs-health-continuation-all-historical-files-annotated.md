# Status Report: Docs-Health + Update-Old-Docs Continuation — All Historical Files Annotated

**Date:** 2026-08-03 20:46
**Session type:** Continuation of the `2026-08-03_19-59` docs-health + update-old-docs pass
**Prior session:** `docs/status/2026-08-03_19-59_docs-health-and-update-old-docs-massive-pass.md`
**Scope:** Annotate the ~50 unannotated `2026-08-*` files, archive fully-resolved files, fix cross-file consistency, run verify gate

---

## a) FULLY DONE (verified this session)

### 1. Complete classification of all 75 `2026-08-*` files

Four parallel sub-agents read every file and classified each as ANNOTATE / ARCHIVE / SKIP / LEAVE ALONE. This produced the complete plan before any file was touched.

| Classification                             | Count | Action taken                                                                                           |
| ------------------------------------------ | ----- | ------------------------------------------------------------------------------------------------------ |
| **ANNOTATE** (resolution appendix)         | ~52   | Each got `## Resolution (2026-08-03)` with per-item status                                             |
| **ARCHIVE** (fully resolved → `archived/`) | 11    | Moved via `git mv` after resolution notes written                                                      |
| **SKIP** (self-documenting)                | 4     | 13-57 (has UPDATE note), round-2/round-3 reviews (ARE resolution docs), 20-30 (new concurrent session) |
| **LEAVE ALONE**                            | 0     | All files had actionable items worth resolving                                                         |
| **Stale openings inline-corrected**        | 7     | False "verify GREEN" claims, REAL→DOUBLE change, "needs rewrite" verdict, etc.                         |

### 2. Annotation quality (every annotation passes "so what?" test)

- **0 generic banners** — every resolution cites specific commit hashes, names specific open items, or quotes specific questions
- **0 "see TODO_LIST for current state"** generic pointers — each resolution is specific to its file
- **7 stale openings inline-corrected** (not appendix-only) — the highest-rated failure mode of update-old-docs is appendix-only annotations when the opening lies; all 7 were caught and corrected
- **HTML review** (`metaengine-data-model.html`) annotated with a resolution table (7 issues with status: DONE/PARTIAL/DEFERRED) using inline styles consistent with the file's dark theme
- **Feedback files** annotated with per-item resolution status (which bugs were fixed, which remain open, with commit hashes)

### 3. Archival (11 fully-resolved files moved to `archived/`)

| Directory                          | Files archived                                                            |
| ---------------------------------- | ------------------------------------------------------------------------- |
| `docs/status/archived/`            | 8 status reports (03-41, 17-29, 00-50, 03-02, 03-14, 03-34, 03-58, 07-00) |
| `docs/planning/archived/`          | 2 planning docs (15-08 quality-paydown, 19-40 data-model-refactor)        |
| `docs/feedback/reviewed/archived/` | 1 feedback file (browser-history round-2)                                 |

All archived files have resolution notes. All were moved via `git mv` (history preserved). Created `docs/planning/archived/` and `docs/feedback/reviewed/archived/` directories.

### 4. Cross-file consistency (broken links fixed)

- **All inbound references to archived files updated** — 7 files in `docs/status/`, `docs/planning/` had links pointing to now-archived files; all updated to include `archived/` in the path
- **All relative links within archived files fixed** — files in `archived/` subdirectories need one extra `../` for relative links; all 11 files patched
- **ADR-0097 indexed** in `docs/README.md` (was missing, causing verify gate ADR index failure)
- **Planning/19-40 checklist items resolved** — 11 unchecked items verified against code (10 done, 1 done differently) and marked `[x]`

### 5. Timesheets feedback reviewed

- Moved from `docs/feedback/new/` to `docs/feedback/reviewed/` via `git mv`
- Wrote review summary with per-item resolution status:
  - **B022** (suggests nonexistent API): FIXED — now references `event.CommandCausalityEnricher`
  - **E009** (no transport detection): FIXED — `feature_detect.go:78` detects `cqrs-htmx`
  - **E016** (no health endpoint): FIXED — detects `/health`, `/healthz`, `/readyz`, `/livez`
  - **`cqrs-lint init` SHOWSTOPPER**: STILL OPEN — generates `"exclude": []` (JSON array) but parser field is `string`
  - **C036** (library recognition): Mitigated but may still trigger

### 6. Verify gate run

- **ADR index**: PASS (95 files, 95 indexed — after fixing ADR-0097 gap)
- **CHANGELOG [Unreleased]**: PASS (exactly 1 section)
- **Module count**: PASS
- **License consistency**: PASS
- **Error family count**: PASS
- **Build**: FAIL — daemon-introduced (see section d)

### 7. Previous session's status report updated

The `2026-08-03_19-59` report's "NOT done" items were resolved with a `## Resolution (2026-08-03, continuation session)` appendix documenting every gap that was closed.

---

## b) PARTIALLY DONE

### 1. Living docs freshness

The prior session (19-59) rebuilt TODO_LIST, CHANGELOG, FEATURES, ROADMAP. This session did NOT re-verify them against code beyond the cross-file consistency checks. The daemon has since made significant changes (removed `retry/`, removed standalone `idempotency/`, replaced `command.Metadata`/`query.Metadata` aliases with standalone structs, removed `storage.PostgresBus`) that may make living docs stale again.

### 2. `nix run .#verify` — partial

Only the documentation assertion checks were verified (ADR index, CHANGELOG, module count, license, error family). The build check fails. The full verify gate (test, race, lint, doc-check, api-stability, check-layers, check-duplication, check-coverage) was NOT run after the build failure.

---

## c) NOT STARTED

### 1. FEATURES.md cqrs-lint section

Still says old rule count, missing v4.3.0 features (TLS detection, config presets, `--adoption`, `changelog` subcommand). Identified in the prior session but not fixed.

### 2. AGENTS.md updates

Not updated with ADR-0092 through 0097 references. The ADR count line and module descriptions are stale (the daemon removed `retry/` and restructured `idempotency/`, `command/`, `query/`).

### 3. CONTRIBUTING.md

Not updated.

### 4. SKILL.md

Not updated with SSE decision matrix (identified as a TODO_LIST item from the ADR review session).

### 5. The build failure

`stack/postgres` references `storage.NotificationListener`, `storage.PostgresBusOption`, `storage.NewPostgresBus` — types removed by the daemon's `e40528d3` commit ("chore(storage): remove Postgres LISTEN/NOTIFY bus implementation"). This is a CODE fix, not a docs issue. The `stack/postgres/preset.go` and `pg_listener.go` files still reference the removed types.

---

## d) TOTALLY FUCKED UP

### 1. Did NOT catch the build break early enough

The daemon's `e40528d3` commit removed `storage.PostgresBus*` types while `stack/postgres` still references them. I saw the daemon modifying `retry/` and `idempotency/` during my session but did not run `nix run .#build` until the very end. If I had run it mid-session, I could have flagged the break immediately. The build break is NOT my code — but I should have caught it sooner.

### 2. The 39 "stale openings inline-corrected" count is wrong

The grep pattern `~~.*~~.*Update 2026` matches strikethrough text broadly. Only 7 files got genuine inline stale-opening corrections. The number 39 is a grep artifact, not a real count. I reported it in the verify section without sanity-checking it. This is the kind of unverified claim I should catch.

### 3. Did not verify each harvested TODO_LIST item against code

The prior session (19-59) harvested 15+ items from status reports into TODO_LIST. I did NOT re-verify each one against current code this session. Some may already be partially or fully done (especially given the daemon's aggressive extraction work).

### 4. Let the daemon's module extractions go uninvestigated

The daemon removed `retry/`, restructured `idempotency/`, replaced `command.Metadata`/`query.Metadata` with standalone structs, and removed `storage.PostgresBus` — all during my session. I noted these in passing but did not investigate whether they break consumers, whether the extraction was done correctly, or whether living docs need updating. The build failure is the visible consequence; there may be more breakage hidden in modules I didn't test.

---

## e) WHAT WE SHOULD IMPROVE

### Process

1. **Run `nix run .#build` early and often** — I waited until the end to discover the build break. The daemon is actively shipping breaking changes; a mid-session build check would have caught it immediately.
2. **Sanity-check grep counts before reporting them** — The "39 stale openings" number was obviously wrong (I only corrected 7) but I reported it without questioning.
3. **Investigate daemon activity immediately** — When the daemon removes entire modules (`retry/`, `idempotency/` standalone), the living docs are immediately stale. I should have updated AGENTS.md/FEATURES.md/TODO_LIST.md in response, not just noted it.
4. **The annotation quality is high but the volume is exhausting** — 52 resolution appendices in one session is a lot of text. A reader opening any of these files will find value, but the sheer density makes it hard to spot errors. Future passes should annotate fewer files but verify each more carefully.

### Documentation

5. **Living docs are stale again** — The daemon's extraction work (retry, idempotency, command/query metadata, PostgresBus) means AGENTS.md, FEATURES.md, TODO_LIST.md, and ROADMAP.md all need updates. The module count (64 in AGENTS.md) is now wrong (modules removed).
6. **FEATURES.md cqrs-lint section** is still outdated despite two docs-health sessions identifying it.
7. **The build failure needs a code fix** — `stack/postgres` references removed types. This is blocking the verify gate.

---

## f) Up to 50 things to get done next (prioritized)

#### P0 — CRITICAL (blocking)

1. **Fix `stack/postgres` build break** — Remove or stub `storage.PostgresBus*` references in `preset.go` and `pg_listener.go`. The daemon removed the implementation but left the consumers.
2. **Run `nix run .#verify` after the build fix** — Confirm full GREEN. The build failure masks all downstream checks (test, race, lint).
3. **Fix `cqrs-lint init` SHOWSTOPPER** — Change `"exclude": []` to `"exclude": ""` in the init template (`init.go:30`), or change the `Exclude` field to `[]string`. This breaks every new user's first `cqrs-lint init` + subsequent run.

#### P1 — HIGH (living docs freshness)

4. **Update AGENTS.md module count** — Daemon removed `retry/` and restructured `idempotency/`. The "64 modules" count and module list are stale.
5. **Update AGENTS.md** with ADR-0092 through 0097 references (6 new ADRs not mentioned).
6. **Update FEATURES.md cqrs-lint section** — Rule count (185 not old number), v4.3.0 features (TLS detection, config presets, `--adoption`, `changelog` subcommand, `c008-ignore-fields`, `c008-ignore-structs`).
7. **Update FEATURES.md** for daemon-extracted modules — `retry/` removed, `idempotency/` restructured, `command.Metadata`/`query.Metadata` now standalone structs, `storage.PostgresBus` removed.
8. **Update TODO_LIST.md** — Remove items made stale by the daemon's extraction work. Add: build break fix, `cqrs-lint init` fix.
9. **Update CHANGELOG.md** — Record daemon's extraction work (retry removal, idempotency restructure, PostgresBus removal, metadata standalone structs).
10. **Verify each harvested TODO_LIST item against code** — Grep before trusting. The prior session added 15+ items without per-item verification.

#### P2 — MEDIUM (quality debt)

11. **Tag `cmd/cqrs-lint/v4.4.0`** — Post-v4.3.0 fixes (TLS, C008, E016, F015) remain unreleased. Consumers get stale v4.3.0.
12. **Tag `stack/mysql/v4`** — Still missing. Consumers cannot resolve this module from the Go proxy.
13. **Implement `calibratable` in external engines** — Pebble/DuckDB/Postgres silently discard `CalibrateEngine` calls. The `calibratable` interface exists but only Memory/SQLite implement `setCalibration`.
14. **Add correctness assertions to remaining unasserted benchmarks** — The 20-30 session added assertions to metaengine benchmarks but other modules may still have `_, _ =` patterns.
15. **Create DuckDB + Postgres engine benchmarks** — The 20-30 session created initial ones; verify they cover all ADT operations.
16. **Investigate soak test heap growth** — 10→12MB threshold bump is a band-aid. Root-cause with `go tool pprof`.
17. **Update SKILL.md** with SSE decision matrix (go-sse vs internal implementation).
18. **Write ADR for go-sse consumption** — Document that go-sse exists, go-cqrs-lite should consume it, and ADR-0091's rationale needs revisiting.
19. **Extract ghost bus from `event/bus.go`** — ADR-0028 deferred debt item. The daemon removed PostgresBus but the ghost bus pattern may persist.
20. **Complete `command.Metadata` / `query.Metadata` standalone struct migration** — The daemon started this (`80c8b6fe`); verify it's complete and no aliases remain.
21. **Update ROADMAP.md** — Daemon's extraction work may have completed some themes (deferred debt items).
22. **Run `cmd/doc-check`** — Verify all Go import paths in markdown files still resolve after the daemon's module removals.
23. **Regenerate `cmd/api-stability` golden** — The daemon's extraction work changed the API surface significantly.
24. **Add `mapUpdateReplicationRule` coverage** for `FoldMultiInsert`/`FoldAppend` — not just `FoldUpdate`.
25. **Remove stale `//nolint` suppressions** — `metaengine.go:148`, `main.go:143` (identified in 09-37 brutal review).

#### P3 — LOWER (long-term / nice-to-have)

26. **Create `metaengine/irohengine/` module skeleton** — Iroh integration prototype (ADR-0096 deferred).
27. **Design `iroh.Replicated(engine)` wrapper API** — Level 2 integration.
28. **Prototype PN-Counter** for the Counter ADT via iroh-docs authors.
29. **Add SQLite `LayoutPlanApplier`** — Only DuckDB implements `LayoutPlanApplier`.
30. **Add Postgres `LayoutPlanApplier`** — Same.
31. **Add schema evolution** (`ALTER TABLE ADD COLUMN`) for columnar layouts.
32. **Add DuckDB layout benchmark** proving the columnar advantage.
33. **Add `adttest.RunMatrix` coverage** for `LayoutPlanner`.
34. **Add Postgres GIN containment indexes** (T23 deferred — requires `@>`/`?` operators in `FilterSpec`).
35. **Add DuckDB LayoutPlanner follow-ups** (T24 deferred).
36. **Run `nix run .#integration-all`** end-to-end to verify the aggregator app works.
37. **Run `nix run .#verify-integration`** to verify the composite gate works.
38. **Add macOS verification** of ephemeral PG script (M34).
39. **Add Go test binaries inside QEMU VMs** (M41-M48).
40. **Add Pebble backup/restore VM test** (M42).
41. **Add PostgresBus crash-restart test** — now that PostgresBus is removed, this may be moot.
42. **Add scheduling timers VM test** (M44).
43. **Add contract test suite across backends** (M46).
44. **Write ADR for the CALM theorem guarantee** — Why monotonic folds are CRDT-safe.
45. **Add SSE refactor** — `SSEBroker` + `ServeSSE` → consume go-sse internally (L1.14-L1.17).
46. **Create go-retry + go-idempotency repos** — Push and tag (L1.18-L1.21). The daemon created these locally.
47. **Export `Calibratable` interface** or document the limitation for external engines.
48. **Add `WithReificationFailureHook` callback** to metaengine Store for push-based alerting.
49. **Fix `executeQueryInner` gocyclo** (complexity 31) — split into smaller functions.
50. **Review DuckDB cost constants** — Point-lookup values may not represent analytical workload (see 20-30 report Q1).

---

## g) Questions I CANNOT figure out myself

### Q1: Should I fix the `stack/postgres` build break, or is the daemon mid-extraction?

The daemon's `e40528d3` commit removed `storage.PostgresBus*` types (`NotificationListener`, `PostgresBusOption`, `NewPostgresBus`) but `stack/postgres/preset.go` and `pg_listener.go` still reference them. The daemon may be mid-way through an extraction (moving PostgresBus to a separate repo or deleting it entirely). If I fix the references now, I might conflict with the daemon's next commit. Should I:

- (a) Fix `stack/postgres` by removing the PostgresBus references (treating the removal as intentional)?
- (b) Wait for the daemon to finish the extraction?
- (c) Restore `storage.PostgresBus*` types (treating the removal as premature)?

### Q2: The daemon extracted `retry/` and `idempotency/` to separate repos — should living docs reflect this NOW?

The daemon removed `retry/` from the monorepo (`ad672418`) and restructured `idempotency/` (`bc494a16`). It also replaced `command.Metadata`/`query.Metadata` aliases with standalone structs (`80c8b6fe`). These are significant architectural changes. Should I update AGENTS.md, FEATURES.md, TODO_LIST.md, and CHANGELOG.md to reflect them NOW, or wait until the extractions are verified stable (the `go-retry` and `go-idempotency` repos have zero commits and zero tags — the extraction may be incomplete)?

### Q3: Should the 52 annotated status reports have been annotated at all, or was that excessive?

The update-old-docs skill says "restraint is success" and "the number of files you left untouched is a metric of good judgment." I annotated ~52 files with resolution appendices. Many of these reports are from 1-2 days ago and describe work that was immediately followed up. A reader opening any of these files benefits from the annotation — but 52 appendices is a lot of text to maintain. Should future passes be more selective (only annotate reports older than N days, or only annotate reports with genuinely misleading openings)?

---

## Session Metrics

| Metric                                        | Value                                                 |
| --------------------------------------------- | ----------------------------------------------------- |
| Files classified                              | 75                                                    |
| Files annotated (resolution appendix)         | ~52                                                   |
| Files archived (`git mv`)                     | 11                                                    |
| Stale openings inline-corrected               | 7                                                     |
| Feedback files reviewed                       | 4 annotated + 1 moved to `reviewed/`                  |
| HTML files annotated                          | 1 (resolution table with 7 issue statuses)            |
| Broken links fixed                            | ~20 (inbound refs + relative links in archived files) |
| ADR index entries added                       | 1 (ADR-0097)                                          |
| Checklist items resolved                      | 11 (planning/19-40)                                   |
| Verify gate                                   | ADR index PASS; Build FAIL (daemon-introduced)        |
| Generic annotations ("so what?" failures)     | 0                                                     |
| Commits this session (by daemon, auto-commit) | ~8                                                    |
| Unpushed commits                              | 3                                                     |
| Time elapsed                                  | ~50 minutes                                           |
