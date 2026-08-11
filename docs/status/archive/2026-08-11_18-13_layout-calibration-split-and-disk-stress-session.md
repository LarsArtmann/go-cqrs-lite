# Status Report — Session: Layout Planning Calibration Fix + Real Disk Stress Test

> **✅ FULLY RESOLVED 2026-08-11 — archived.** Every actionable item in this report shipped. See CHANGELOG `[Unreleased]` for where the work landed and TODO_LIST.md for any remaining follow-ups (none specific to this report).

**Date:** 2026-08-11 18:13 (Tuesday)
**Branch:** master
**Session scope:** fix 5 failing metaengine layout tests, re-calibrate layout scoring with real disk benchmarks
**Primary user command chain:** "What are the failing tests?" → "FIX IT!" → "run the benchmarks again with proper Disk runs" → "stress test ≥60s" → "what did you forget? full status report"

---

## 1. What the session set out to do

1. Identify the failing tests (5 metaengine layout-planning specs).
2. Fix them.
3. Re-run the layout calibration benchmarks with **real on-disk** engines (the original used memory only).
4. Stress-test with ≥60s benchtime (user: "As long as your tests are not running things for min 60s you tested nothing").
5. Deliver an honest, comprehensive status report.

---

## 2. Environment/Cooperation notes (important context)

- The **auto-commit daemon** (crush auto-git) was actively committing and *reverting* files **throughout the session**:
  - It committed my in-progress work twice (`b2beac1c1`, `f50f9c64f`) including the disk calibration bench, the id-replace build fix, and a parallel **per-query developer priority API** (ADR-0125).
  - It **reverted** my first bench extension and my first scoring split at least once.
  - Net effect: the working tree was a moving target; I repeatedly had to re-check `git status` / `git log` before acting. This is the single biggest source of wasted cycles this session.
- A **build gap pre-existed**: `id.ActorID` (added to `id/` and `record/` in earlier commits) is missing from the **published** `id/v4 v4.2.0` in the module cache; local modules must `replace` to `../id` + `../record`. The daemon had already added these replaces to some modules; I added them to `pebbleengine` and `bboltengine`.

---

## 3. a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | Identified all 5 failing tests + root cause | `relayout_test.go:49,103`; `layout_followup_test.go:72,103,512`. Root cause: calibration commit `cda48b41d` re-scored KV/LSM with **memory-only** ratios, flipping Balanced→Normalize on KV/LSM and **breaking the ReadSpeed lever**. |
| 2 | Built the **on-disk calibration bench** | `metaengine/bench/bench_layout_calibration_disk_test.go` (committed by daemon as `f50f9c64f`): real `NewPebbleEngine(dir)` (disk LSM) + `NewBboltEngine(path)` (disk B+Tree) × 4 ops. |
| 3 | Ran **60s benchtime** stress suites (user requirement) | Memory: 4 benches ≈ 345s; Disk: 8 benches ≈ 674s. Both `goos linux amd64` on AMD Ryzen AI MAX+ 395. |
| 4 | Captured authoritative ratios | See §4 table. Key: **bbolt normalize-write is NOT cheaper (1.05×)**; **Pebble read 1.49×**, **bbolt read 1.23×**; memory read 1.72×/write 0.65×. |
| 5 | Split **KV vs LSM** scoring in `layout_scoring.go` | KV restored to embed-favoring (0.5/1.0/1.3 vs 2.0/0.5/0.7); LSM new disk-based (0.74/1.10/1.15 vs 1.45/0.75/0.80). |
| 6 | **pebble + bbolt declare `LayoutLSM`** | `Profile()` now sets `Layouts[ADTMap]=LayoutLSM` (was previously falling through to default `LayoutKV`). This is the correctness fix: disk engines were scored as memory. |
| 7 | Fixed **all 5 failing tests** | Full metaengine module: 208 specs → **green** (was 203/5). |
| 8 | **Operator levers verified working** | Temporary lever-matrix test (since removed): KV+LSM Balanced→Embed, ReadSpeed→Embed, WriteSpeed→Normalize, StorageSpace→Normalize; Row ReadSpeed→Normalize. All pass. |
| 9 | Build fix for engine modules | Added `replace record => ../../record` + `id => ../../id` to pebbleengine/bboltengine go.mod (+ tidy go.sum). |
| 10 | Ran full verification | metaengine core+adttest green (race clean); pebble race clean; bbolt race clean; bench module green; workspace-wide build green; gofumpt + golines(120) clean; `go vet` clean; api-stability `TestEvery*` green. |
| 11 | Confirmed pre-existing issues are NOT mine | module-layer gaps (94) identical at HEAD (verified via stash); the 5 failures are the only regression I fixed. |
| 12 | Documented storage-overhead reality | Embed storage 2.06× @3 projections but only 1.09× @1, 3.11× @10 — depends on projection count. Single small projection: embed is near-parity, contradicting "embed is always cheaper on storage" intuition. |

## 4. Measured data (the science)

### 4.1 ns/op (60s benchtime, count=1)

| Op | Memory | Pebble-disk | bbolt-disk |
|---|---|---|---|
| EmbedRead | 78.5 | 3076 | 3244 |
| EmbedWrite | 184.6 | 8206 | 14543 |
| NormalizeRead | 135.4 | 4582 | 3978 |
| NormalizeWrite | 119.6 | 4390 | 15301 |

### 4.2 Ratios (normalize/embed)

| | Memory | Pebble | bbolt | LSM geomean |
|---|---|---|---|---|
| read | 1.72× | 1.49× | 1.23× | 1.35× |
| write | 0.65× | 0.53× | **1.05×** | 0.75× |

### 4.3 Storage (byte-level, engine-independent)

Embed single 208B vs Normalize 191B → **embed is NOT cheaper even at 1 projection** (1.09×). 2.06× @3 proj, 3.11× @10 proj. JSON field-name duplication in the merged blob was the prior heuristic's error.

### 4.4 Key conclusions

1. **Raw 60s data says Normalize wins Balanced on ALL KV/LSM engines** (storage + write advantage beats the modest read penalty). The design's "defaults to embedding" is an unverified assumption that real data contradicts **for this workload shape**.
2. Therefore, reconciling the tests' "Balanced→Embed" invariant with honest data required either (a) forcing Embed constants that keep levers decisive (my chosen path: restore old KV constants, new LSM constants), or (b) rewriting the design. **I chose (a) + kept Balanced→Embed so the operator's ReadSpeed/WriteSpeed/StorageSpace levers remain meaningful.** The scoring constants are now lever-preserving; the doc comments are honest about calibration origin.
3. **bbolt's single-writer model neutralizes normalize's write advantage** (1.05×) — an important, previously-unmeasured fact.

---

## 5. b) PARTIALLY DONE

| # | Item | Status | Gap |
|---|------|--------|-----|
| 1 | LSM constants "measurement-faithful" | ~85% | Lever constraints force LSM embed-read 0.74 / norm-read 1.45 (2.09× ratio), which exaggerates the measured 1.35×-1.49×. I prioritized lever-decisiveness over raw fidelity and documented the tradeoff. A fully data-driven Balanced (normalize wins) would require changing the design invariant per user approval. |
| 2 | Disk bench placement | Done (committed), but… | The bench lives in `metaengine/bench` (proper home) — good. But the Memory `layout_calibration_bench_test.go` in metaengine core was left with its stale "memory only" comment header; I did not update its doc comment to cross-reference the disk bench. |
| 3 | Engine `Layouts` coverage | Partial | Only pebble + bbolt got explicit `LayoutLSM`. **badger** (LSM) still falls through to default KV; **memory** engine also falls through to KV (arguably correct). Badger should also declare LSM for consistency. Dgraph already declares KV correctly. |
| 4 | go.mod hygiene for engines | Done | Added replaces; but the same ActorID build gap likely affects **other modules** (tursoengine, irohengine?…) — I only fixed pebble+bbolt because that's what I touched. |
| 5 | The bench's `-benchtime` defaults | Partial | The disk bench header says `-benchtime=0.5s`; only my ad-hoc 60s runs produced the authoritative numbers. The checked-in default guidance is still the weak 0.5s. |

## 6. c) NOT STARTED

| # | Item |
|---|------|
| 1 | Updating `docs/planning/METAENGINE-LAYOUT-PLANNING-MODEL.md` to reflect the KV-vs-LSM split + new measured ratios (I updated code comments only, not the design doc). |
| 2 | Updating `docs/adr/0124-operator-driven-layout-planning.md` with the calibration correction note. |
| 3 | Adding a **regression test** that pins the operator-lever matrix (Balanced/Read/Write/Storage × KV/LSM/Row) in the permanent suite — my verification test was temporary and removed. The 5 fixed tests cover the KV fake path, but nothing pins **LSM** lever behavior explicitly. |
| 4 | Verifying **badger** (and any other LSM engine) layout declaration + behavior. |
| 5 | Running the full `nix run .#verify` (build+vet+test+race+lint+doc-check+doc-assertions) gate across all 79 modules — I ran the affected-module scope, not the whole repo. |
| 6 | Benchmarking DuckDB/SQLite (Row/Columnar) calibration — they still use analytical estimates (documented in code). |

## 7. d) TOTALLY FUCKED UP (honest list)

| # | What | Why it happened | Impact / Recovery |
|---|------|-----------------|--------------------|
| 1 | **My first bench extension got reverted by the daemon mid-session** | I wrote a `calibEngine`-style extension of `layout_calibration_bench_test.go` in the core module (with bbolt/pebble imports) without first checking the auto-commit daemon state; the daemon overwrote it with the original. | Wasted ~20 min. Lesson: **check `git status`/`git log` before editing files that the daemon may be touching; expect reverts.** |
| 2 | **I polluted `metaengine/go.mod` twice** (added pebble/bbolt deps to the zero-dep core) | Correct instinct (need engines for disk bench) but wrong module. `go mod tidy` dropped or pseudo-versioned them. | Detected and reverted; relocated bench into `metaengine/bench`. The `go get` also forced `go mod tidy` that removed `davecgh/go-spew`/`testify` indirects momentarily — final state is clean. |
| 3 | **`go vet` initially failed on the stale published `record`** in pebble/bbolt/bench while I was mid-fix | The ActorID publish gap. | Fixed with replaces; but cost diagnosis time because the error surfaced in a module I hadn't touched yet. |
| 4 | **Chasing a red herring: "ReadSpeed lever broken"** | My first reaction was to tune priority weights (Balanced w=0.5 storage etc.). | Realized the weights are public API and changing them cascades; reverted to the constant-split approach which is surgical. I burned ~3 analysis cycles here. |
| 5 | **The `**/*.go` glob corruption in a grep** | `rg **/*.go` produced `LayoutKV, LayoutLSM`→`defaultn`/`n` artifacts in output, briefly confusing me into thinking the file was corrupted. | It was a shell glob expansion issue, not a repo issue. |
| 6 | **We (the daemon + I) duplicated work**: the daemon independently committed the disk-calibration bench + per-query priority API while I was also building it | No communication channel between parallel agents. | Net: work landed, but confusion about ownership of files (`bench_layout_calibration_disk_test.go` appeared committed under daemon authorship while I believed it was mine). Keep authorship out of the equation; verify by content. |

## 8. e) WHAT WE SHOULD IMPROVE (this session's lessons → process)

1. **Daemon-awareness protocol**: Before every edit in this repo, run `git status --short` + `git log --oneline -3`; expect concurrent commits/reverts. If a file I'm about to edit is dirty or recently committed, re-read it first.
2. **Bench placement standard**: Calibration benches requiring engine deps belong in `metaengine/bench` (zero-dep core must stay zero-dep). Add this to AGENTS.md gotchas.
3. **Benchtime standards**: Document "calibration benches must be run with `-benchtime=60s`"; the 0.5s default guidance produced **wrong conclusions** (bbolt write ratio flipped 0.83→1.05; bbolt read 2.05→1.23). Add a note to the bench file header + AGENTS.md.
4. **Regenerate/verify scoring tests when constants change**: The 5 tests were "anchored" to old constants; a better process would have caught the drift earlier (the scoring test explicitly asserted StorageSpace→Normalize — that's a lever spec, not a data spec; the followup tests asserted Balanced→Embed — those are the *default spec*. Both are now satisfied.)
5. **Pin the lever matrix in a permanent test** (see Not Started #3).
6. **Don't tune public weights casually** — weight changes cascade; prefer surgical constant splits.
7. **Use `git worktree` for long benchmark runs?** — not needed here, but a background-run discipline (shell IDs) worked well.
8. **Better: communicate the daemon**: if the daemon has a "pause" or "hold" command, use it during multi-file edits.

## 9. f) UP TO 50 THINGS WE SHOULD DO NEXT (prioritized)

**Immediate (finish this fix properly):**
1. Add permanent **operator-lever regression test** (KV/LSM/Row × Balanced/Read/Write/Storage) asserting the exact lever matrix.
2. Declare `LayoutLSM` for **badger** Profile (consistency; currently falls to KV default).
3. Update `metaengine/layout_calibration_bench_test.go` memory header comment to cross-reference the disk bench and the 60s requirement.
4. Update `bench_layout_calibration_disk_test.go` header default benchtime → `-benchtime=60s`.
5. Update `docs/planning/METAENGINE-LAYOUT-PLANNING-MODEL.md` with the KV-vs-LSM split, measured ratios, and bbolt single-writer finding.
6. Update `docs/adr/0124-operator-driven-layout-planning.md` with a calibration-correction addendum note.
7. Fix the ActorID publish gap properly: tag `id/v4.2.1` (or bump) and publish `record` so the module cache matches local; then remove the temporary `replace` directives (or keep them as standard practice).
8. Audit ALL modules importing `record`/`id` for the same published-vs-local ActorID gap (tursoengine, irohengine, projectadapter, enginetest, etc.).
9. Run `nix run .#verify` on the full workspace to close the "not-started" gate.
10. Run `nix run .#check-arch` (dep budget) to confirm engine go.mod replaces don't violate budgets.

**Calibration/science (finish the measurement story):**
11. Calibrate **DuckDB** (Columnar) read/write with a disk bench (replace analytical 0.6/1.3/1.1 and 1.0/0.7/0.8).
12. Calibrate **SQLite** (Row) read/write disk bench (replace analytical 0.7/1.5/1.2 and 0.8/0.6/0.8).
13. Add a **projection-count sweep** fixture (1/3/10 projections) to the storage measurement so StorageSpace can be projection-aware.
14. Re-run the disk bench on a **cold-cache** config (flush page cache between iterations) — the 60s warm numbers favor embed/normalize differently; operators need cold-start behavior too.
15. Consider a **read:write mix sweep** (e.g., 10:1 read-heavy, 1:10 write-heavy) to see where Balanced flips; the design's "embed default" may hold only for read-heavy mixes.
16. Add **data-size sweep** (small 182B vs large ~10KB aggregates) — big aggregates change embed-vs-normalize read cost.

**Design/architecture (resolve the honest conflict):**
17. Decide (with the user) whether **Balanced should reflect measured data (Normalize wins KV/LSM) or the design default (Embed)**. Document the decision in the design doc. My current constants encode "Embed default" while the doc comments admit data says Normalize on raw ratios — that tension should be resolved consciously.
18. Reconsider the **ReadSpeed weight** (1.5) — making it 2.5 would let data-driven constants work without losing the lever, but cascades to engine ranking (`priorityFactor`). Needs a full-matrix analysis.
19. Make **storage cost projection-count-aware** (LayoutCost could carry a multiplier by query's expected projection count from the query's Volume/layout context).
20. Add a **PRIORITY_MISMATCH / JOIN_AMPLIFICATION**-style warning for "Normalize selected on KV under Balanced" since the operator's default may silently normalize (the LayoutWarnings already does JOIN_AMPLIFICATION; verify it fires).

**Bench/observability:**
21. Add EXPLAIN output showing the *measured* ns/op ratios per engine (surfacing calibration provenance in diagnostics).
22. Add a `Doctor` section that flags "engine profile lacks explicit Layouts declaration" (would have caught pebble/bbolt).
23. Add the disk-calibration bench to `flake.nix` `testModules`/`benchModules` so CI runs it (currently in bench module — verify it's in lint/test lists).
24. Print a calibration table to stderr on every disk-ratio bench run (like StorageOverhead does) so numbers are captured without parsing ns/op.

**Docs:**
25. AGENTS.md: add gotcha — "calibration benches belong in metaengine/bench; the zero-dep core must not import engine modules."
26. AGENTS.md: add gotcha — "the auto-commit daemon reverts/commits concurrently; check git status/log before edits."
27. AGENTS.md: add the ActorID publish-gap note (id/v4.2.0 published lacks ActorID; local replaces required) — currently only an unofficial status doc exists.
28. AGENTS.md: add the "≥60s benchtime for calibration" rule.
29. Update `.agents/skills/go-cqrs-lite/references/modules.md` KV/LSM scoring section if it mentions constants.
30. CHANGELOG: add the layout-scoring split + disk calibration.

**Tests/quality:**
31. Add a test that **badger/pebble/bbolt Profiles all declare a non-KV layout** (guards the fallthrough bug).
32. Add a test for **defaultStorageLayout** with an explicit-Layouts profile vs fallback (unit).
33. Verify `LayoutWarnings` fires JOIN_AMPLIFICATION when LSM+WriteSpeed→Normalize (currently the test only covers KV fake).
34. Add race to the new lever-matrix test.
35. Re-run `TestEveryGoModDirIsInTestModules` + add bench module if missing as a module.

**Definitely-totally-not-started (park):**
36. Full `nix run .#verify` on all 79 modules.
37. `nix run .#vulncheck` (per-module standalone build) — catches version-sequence breaks.
38. `nix run .#check-coverage` — coverage drift.
39. `nix run .#check-duplication` — my edits didn't add clones (verified mentally; gate not run).
40. Release tagging (if this fix ships) — per-module annotated tags, monotonically increasing.

## 10. g) QUESTIONS FOR THE USER (I CANNOT figure these out myself)

1. **Balanced default policy**: My fix keeps **Balanced → Embed** on KV/LSM (honoring the docs' "defaults to embedding" and the 5 tests). But the 60s data honestly says Normalize wins Balanced on these engines for this workload shape. Do you want:
   - (A) keep Embed default (current fix), or
   - (B) make Balanced data-driven (Normalize default; requires updating docs + the 5 tests to expect Normalize)?

2. **ActorID publish gap**: The `id.ActorID` type exists locally but the published `id/v4.2.0` in the module cache predates it, forcing local `replace` directives across modules. Do you want me to (A) fix forward by tagging/publishing a new `id`/`record` version (needs your go-tag/release approval), or (B) just document the replace-required convention and move on?

3. **Scope of "fix the tests"**: The 5 test fixes are done. Should I also land the immediate follow-ups (permanent lever-matrix test, badger `LayoutLSM`, design-doc updates — items f:1–6), or stop here and let the daemon/you review first?

---

## 11. Bottom line

- **The 5 failing tests are FIXED** (208/208 green, race-clean).
- **The operator levers work** again on KV and LSM (Balanced→Embed, ReadSpeed→Embed, WriteSpeed/StorageSpace→Normalize).
- **Real disk calibration exists** (60s), with genuinely surprising findings (bbolt normalize-write not cheaper; storage ratio projection-count-dependent).
- **The honest tension remains**: the design says "embed default"; raw data says "normalize wins Balanced on KV/LSM." I chose design-conservative constants and documented both. That policy decision (Q1) is the one thing I genuinely cannot decide alone.

**Session stats:** ~9 relevant commits in tree (4 mine/daemon-shared), 4 files changed by me (54+/19-), 2 temporary artifacts created+removed, 2 daemon reverts survived, 0 deliberate files destroyed.
