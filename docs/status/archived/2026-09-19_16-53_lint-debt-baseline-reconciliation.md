# Status Report: Lint-Debt Baseline Reconciliation (M1/M2 execution start)

**Date:** 2026-09-19 16:53

> **RESOLVED-BY-ROUTING (2026-09-19 docs-health 8th pass):** struck items above = verified shipped (CHANGELOG 2026-09-19 ADR-0142/ADR-0143 entries; TODO_LIST `[x]` rows). Open remainder tracked in TODO_LIST "Metaengine Universal Storage Substrate": T18b load-sweep + benchmark re-baseline (quiet-window gated), T19–T21 (v5-gated), tag waves, claim-metrics parity owner decision. ARCHIVED.
**Session:** Resumption of the metaengine-substrate-tail workstream (ADR-0142 tail, ADR-0143 fixed last session).
**Continuation plan:** `docs/planning/2026-09-19_15-37_SUPERB-verify-green-tag-wave-crm-ports.md` (M1–M32, F01–F113).
**Operating mode:** READ/UNDERSTAND/RESEARCH/REFLECT → execute-and-verify one step at a time.

---

## What this session did (before interruption)

1. Loaded the mandated buildflow skill; confirmed project policy: `nix run .#lint` / flake apps own quality gates, per-module `GOWORK=off` runs use the nix golangci-lint 2.13.2 binary with the root `.golangci.yml`.
2. Updated the todo list to true state: jsonv2-drop marked **completed** (verified last session); "Execute plan F01–F29 (lint debt)" added and set in_progress.
3. **Recovered a broken committed tree:** `stack/{duckdb,mysql,postgres}/preset.go` — last session's embedlit autofix restructured the `config` structs to embed `sqlopt.DSNConfig` (committed by the daemon), but the matching `defaultConfig()` literal updates were left **uncommitted in the working tree**. The committed tree could not compile those three modules. Verified the working-tree completion builds (`GOWORK=off go build ./...` green for all three), then `gofmt -w`'d them. Tests NOT yet run; commit NOT yet authored.
4. Read plan §M1/§M2/§M3 (F01–F32) and the lint wiring in `flake.nix:862-884` (per-module `golangci-lint run --config $PWD/.golangci.yml ./...`).
5. **Enumerated the real lint baseline** across 22 modules (GOWORK=off, nix golangci-lint 2.13.2, root config). Results below — the plan's F-list is **partially stale**: it was surveyed before intervening commits.

## Actual lint baseline (this session's enumeration)

| Module                                                       | rc | Findings                                                                                                                                                                              |
| ------------------------------------------------------------ | -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| queue/sqlite                                                 | 1  | **7**: contextcheck `register.go:26`; err113 `register.go:21`; **sqlclosecheck ×5** (`cancel.go:134`, `enqueue.go:178`, `facts.go:137`, `reads.go:179`, `reads.go:216`) — NOT in plan |
| queue/postgres                                               | 0  | **clean** (plan expected embedded/err113/mnd — already fixed by intervening commits)                                                                                                  |
| queue/mysql                                                  | 1  | blocked: typecheck — cannot import `queue/conformance` (see broken area)                                                                                                              |
| queue/conformance                                            | 4  | **module fails to typecheck at all in GOWORK=off mode** (raw error text lost — see §e)                                                                                                |
| metaengine/claimkit                                          | 1  | **5**: embeddedstructfieldcheck ×3 (`claims.go:60`, `claims_test.go:24`, `dedup.go:24`); sqlclosecheck ×2 (`bench_test.go:165`, `facts.go:211`)                                       |
| metaengine/badgerengine                                      | 1  | 1: embedded `engine.go:65`                                                                                                                                                            |
| metaengine/bboltengine                                       | 1  | 1: embedded `engine.go:103`                                                                                                                                                           |
| metaengine/duckdbengine                                      | 1  | 1: embedded `engine.go:64`                                                                                                                                                            |
| metaengine/pebbleengine                                      | 1  | 1: embedded `engine.go:109`                                                                                                                                                           |
| metaengine/pgengine                                          | 1  | 1: embedded `engine.go:82`                                                                                                                                                            |
| metaengine/mysqlengine                                       | 1  | 1: embedded `engine.go:64`                                                                                                                                                            |
| metaengine/adttest                                           | 1  | **5**: maintidx `claim_conformance.go:34` (CC 31); revive unused-params `temporal_conformance.go:100` (`t1`), `:164` (`vw`); tparallel ×2 (`claim_conformance_test.go:44`, `:83`)     |
| metaengine                                                   | 1  | 11 = claimkit(5) + adttest(5) via replace fan-in + 1 own: godoclint `types.go:130` (`IsDegraded`)                                                                                     |
| metaengine/otelobserver                                      | 1  | 1: staticcheck SA1019 `observer_test.go:202` (`Emit` → `Value.String`)                                                                                                                |
| claiming                                                     | 1  | **6**: exhaustive ×3 (`migrate.go:16`, `rich.go:277`, `stmt.go:81` — missing `DialectDuckDB`); mnd ×3 (`rich.go:234`, `:298`, `:302`)                                                 |
| benchkit                                                     | 0  | **clean** (plan's varnamelen `lo` gone)                                                                                                                                               |
| cmd/cqrs-lint                                                | 1  | 1: gochecknoglobals `loader.go:17` (`loadMu`)                                                                                                                                         |
| stack/sqlite                                                 | 0  | **clean — the exhaustruct_v5 PANIC did NOT reproduce in GOWORK=off mode**                                                                                                             |
| stack/duckdb / stack/postgres / stack/turso / storage/pebble | 0  | clean                                                                                                                                                                                 |
| stack/mysql                                                  | —  | **NOT LINTED — my enumeration loop had a skip-guard bug** (see §d)                                                                                                                    |

Net: **~33 concrete findings across 13 modules**, 1 module-level typecheck breakage, 1 module skipped.

Also discovered: `.golangci.yml` still carries `build-tags: [goexperiment.jsonv2, …]` and `run.go: 1.26.7` — the jsonv2 sweep (marked complete) **missed the lint config**. Harmless no-op tags, but the `go:` pin now lags go.mod's 1.27.1.

---

## a) Fully done (verified)

- Todo list reconciled with reality (jsonv2 completed; F-task added).
- Broken committed tree in `stack/{duckdb,mysql,postgres}` repaired in working tree: builds green standalone, gofmt clean.
- True lint baseline enumerated per-module with the project's exact lint invocation (config + binary + GOWORK=off).
- Plan M1–M3 sections read and cross-checked against reality; staleness identified (not yet reconciled in the plan doc).

## b) Partially done

- **F01–F29 execution (in_progress):** enumeration phase complete; zero findings actually fixed yet.
- Preset repair: build-verified, **not tested, not committed** (daemon will absorb if I don't author a commit).
- exhaustruct panic (F28): failed to reproduce standalone for stack/sqlite; stack/mysql unverified; exclusion decision pending (Q3).

## c) Not started

- All lint code fixes (F01–F25 equivalents against the new baseline).
- F26/F27 (re-lint to zero; build+test touched modules).
- F30–F32 (`#verify` end-to-end, doc-check confirmation).
- M4 integration legs (pg / mysql-vm / dgraph / redis / composite).
- M5 gates (duplication / arch / file-size / api golden).
- `#load-sweep` re-run.
- Tag-wave assessment (M7/M8 — owner-gated).
- Plan-doc reconciliation addendum (F-list staleness).

## d) Totally fucked up (own it)

1. **Lost the raw error for `queue/conformance` rc=4:** my loop only printed grep'd `.go:NN:` lines, so the actual typecheck failure text was discarded — the exact pipeline-masking anti-pattern this repo's own AGENTS lessons warn about. Must re-run and capture full output to a file.
2. **Skipped `stack/mysql` in the baseline:** sloppy module list had it twice and I added a `continue` guard rather than deduplicating — so one of the two known exhaustruct-panic modules has NO fresh lint data.
3. **Build-only verification of the preset fix:** did not run the three stack modules' tests before moving on.
4. **Carried from last session (now surfaced):** the daemon split the embedlit change mid-flight, leaving master momentarily non-compiling for 3 modules. Nobody noticed between sessions — the "clean tree" claim in the last status report was about git status, not buildability. Lesson: after daemon-absorbed commits, build the touched modules.

## e) What to improve

- **Capture lint output to per-module files** (`/tmp/lint/<mod>.log`), report from files — never grep-only.
- **Re-baseline before executing a plan:** F-lists written hours ago rotted (queue/postgres clean, benchkit clean, sqlclosecheck ×5 new). Execute against fresh enumeration; reconcile the plan doc with an addendum instead of trusting its tables.
- **Sweep checklists must include config files:** jsonv2 residue in `.golangci.yml` survived a "verified complete" sweep.
- **Commit at phase boundaries** (per gotchas) so daemon races can't split struct+literal changes.
- Dedupe module lists mechanically (`sort -u`), not with ad-hoc guards.

## f) Up to 50 next tasks (ordered, re-baselined)

**Lint fixes (mechanical):**

~~1. claimkit `claims.go:60` embedded reorder~~ done 2026-09-19 — 18:05
~~2. claimkit `claims_test.go:24` embedded reorder~~ done 2026-09-19 — 18:05
~~3. claimkit `dedup.go:24` embedded reorder~~ done 2026-09-19 — 18:05
~~4. badgerengine `engine.go:65` embedded~~ done 2026-09-19 — 18:05
~~5. bboltengine `engine.go:103` embedded~~ done 2026-09-19 — 18:05
~~6. duckdbengine `engine.go:64` embedded~~ done 2026-09-19 — 18:05
~~7. pebbleengine `engine.go:109` embedded~~ done 2026-09-19 — 18:05
~~8. pgengine `engine.go:82` embedded~~ done 2026-09-19 — 18:05
~~9. mysqlengine `engine.go:64` embedded~~ done 2026-09-19 — 18:05
~~10. queue/sqlite `register.go:21` err113 → sentinel~~ done 2026-09-19 — 18:05
~~11. queue/sqlite `register.go:26` contextcheck~~ done 2026-09-19 — 18:05
~~12. queue/sqlite sqlclosecheck ×5 (`cancel.go:134`, `enqueue.go:178`, `facts.go:137`, `reads.go:179`, `reads.go:216`)~~ done 2026-09-19 — 18:05
~~13. claimkit `bench_test.go:165` sqlclosecheck → defer~~ done 2026-09-19 — 18:05
~~14. claimkit `facts.go:211` sqlclosecheck~~ done 2026-09-19 — 18:05
~~15. adttest `claim_conformance.go:34` maintidx (nolint w/ conformance justification, or split)~~ done 2026-09-19 — 18:05
~~16. adttest `temporal_conformance.go:100` `t1` → `_`~~ done 2026-09-19 — 18:05
~~17. adttest `temporal_conformance.go:164` `vw` → `_`~~ done 2026-09-19 — 18:05
~~18. adttest `claim_conformance_test.go:44` tparallel~~ done 2026-09-19 — 18:05
~~19. adttest `claim_conformance_test.go:83` tparallel~~ done 2026-09-19 — 18:05
~~20. metaengine `types.go:130` godoclint (`IsDegraded` doc start)~~ done 2026-09-19 — 18:05
~~21. otelobserver `observer_test.go:202` `Emit` → `Value.String`~~ done 2026-09-19 — 18:05
~~22. claiming exhaustive ×3 — add `DialectDuckDB` cases (`migrate.go:16`, `rich.go:277`, `stmt.go:81`)~~ done 2026-09-19 — 18:05
~~23. claiming mnd ×3 (`rich.go:234`, `:298`, `:302` → named consts)~~ done 2026-09-19 — 18:05
~~24. cqrs-lint `loader.go:17` `loadMu` nolint (mirrors engine register.go pattern)~~ done 2026-09-19 — 18:05

**Broken/skipped areas (investigate first):**
~~25. Re-run `queue/conformance` GOWORK=off lint with FULL output captured → fix the typecheck breakage~~ done 2026-09-19 — fixed, 18:05
~~26. Lint `stack/mysql` (missed in baseline)~~ done 2026-09-19 — 18:05
~~27. Re-lint `queue/mysql` after 25 lands~~ done 2026-09-19 — 18:05
~~28. Repro-check exhaustruct panic in WORKSPACE mode (stack/sqlite + stack/mysql) → decide F28 (Q3)~~ done 2026-09-19 — killed via rewrites

**Config hygiene:**
~~29. `.golangci.yml`: drop `goexperiment.jsonv2` tag; align `run.go` with 1.27; keep `gci` OUT~~ done 2026-09-19 — 18:05 §a8
~~30. `nix run .#check-lint-config` green~~ done 2026-09-19 — green

**Verify + integration:**
~~31. Test stack/{duckdb,mysql,postgres} modules (preset repair)~~ done 2026-09-19 — green
32. Author commit for preset repair + lint fixes (beat the daemon)
~~33. F26: per-module re-lint over all modules → 0~~ done 2026-09-19 — 18:05
~~34. F27: build+test touched modules (workspace mode)~~ done 2026-09-19 — 18:05
35. F30–31: quiet-window `nix run .#verify`; triage real vs load-transient
36. F32: doc-check phase green in-run
~~37. `#integration-pg` leg~~ done 2026-09-19 — 15:09 green
38. `#integration-mysql-vm` leg (`-p 1`, CAS cap live)
39. `#integration-dgraph` leg
40. `#integration-redis` leg
~~41. `#test-integration` composite local backends~~ done 2026-09-19 — 15:09 EXIT 0
42. `#load-sweep` re-run
~~43. `#check-duplication` (bbolt prefix-sweep rewrite)~~ done 2026-09-19 — 18:05
~~44. `#check-arch` + `#check-file-size`~~ done 2026-09-19 — 18:05
~~45. api-stability golden diff review (expect no-op)~~ done 2026-09-19 — 18:05
~~46. F42: review concurrent-session diffs (queue/mysql `SaveWatermark` GREATEST, pgengine reset keys)~~ done 2026-09-19 — 18:05
47. Reconcile the plan doc with a staleness addendum (this report is the evidence)
~~48. M17: re-pin `TestEngineHealth_CatchUpUnderConcurrentApplies` TODO~~ done 2026-09-19 — TODO_LIST L81 row
49. M14/M15 decisions after owner answers (upstream go/types race filing; verify load guard)
50. Tag-wave assessment (M7/M8) — BLOCKED on owner Q1

## g) Up to 3 questions for the owner

1. **Tag waves (existing Q1, still blocking M7/M8):** run the four-hard-mechanics tag waves now on this verify-green state, or hold for a release train after the lint debt + integration legs are green?
2. **Verify load-transient policy (existing Q3, shapes F30/M15):** should `#verify` refuse to start above a load threshold (with a retry message), or keep starting and classify failures as transient + auto-retry? I can implement either; refusing is more honest but annoying when co-sessions are active.
3. **F28 exhaustruct exclusion:** the exhaustruct_v5 v5.0.3 panic did NOT reproduce for stack/sqlite in GOWORK=off mode. Add the `.golangci.yml` exclusion anyway (dated upstream-bug comment, defensive against workspace-mode recurrence), or drop F28 once I confirm workspace-mode is also clean and keep the config minimal?
