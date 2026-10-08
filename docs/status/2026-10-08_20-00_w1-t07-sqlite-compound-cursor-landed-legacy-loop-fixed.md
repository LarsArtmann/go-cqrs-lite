# W1 Status — T07 Compound-Cursor: sqlite Slice Landed, Legacy Loop Bug Found+Fixed, Tree RED on One Trivial Compile Error

**Date:** 2026-10-08 20:00 CEST
**Scope of this report:** everything done in this continuation session (post-19:15 halt), verified from live tool output only.
**Standing directive honored:** READ → UNDERSTAND → RESEARCH → REFLECT → execute stepwise. Halted now on explicit user order.

---

## 1. Executive Summary

- **T06 (#36)**: 100% closed before this session (pre-halt). Confirmed nothing pending; no action taken.
- **T07 (compound-cursor issuance)**: memory + KV + dgraph were landed pre-halt; **this session landed the full sqlite slice** (raw + pushdown + planned surfaces, emit AND compound keyset consume), the **core SQL keyset helpers in metaengine**, and an **engine-agnostic conformance test** (enginetest). During mutation-testing I **found and fixed two real bugs** — one introduced pre-halt (whole-row cursor leak), one **pre-existing** (legacy scalar cursors cause an infinite page loop on closure engines).
- **Tree state right now: RED in exactly one place** — `metaengine/enginetest/keyset_pagination.go` has two unused `ctx` declarations (compile error, 30-second fix, known exactly). Everything else touched is green.
- **W1 verify chain**: background job 12B **expired FAILED** — the gate went green after 9 attempts, but `#verify`'s calibration-gate then failed on load5=12.74 ≥ 10. Never re-run since.
- All 3 halt questions remained unanswered; I proceeded autonomously where the plan forced a decision (documented in §7).

---

## 2. Verified State (from live runs this session)

| Check | Result |
|---|---|
| api-stability golden regen (`--update`) | ✅ 1 insertion (`metaengine/func LastPairCursor`); "API surface OK: 7569 exports"; `TestEvery` ok |
| metaengine `-short` full suite | ✅ ok 7.5s (incl. 4 new compound-cursor tests) |
| pebble / bbolt / badger / dgraph `-short` | ✅ all ok (never-run debt cleared) |
| sqliteengine build+vet+`-short` full | ✅ ok 8.4s (after SQL slice) |
| Mutation test (disable engine cursor emit) | ✅ tests fail via drops (10/13), phantom page `[4 4 4 0]`, int cursor — exact discrimination |
| Legacy `WithCursor(0)` probe (post-fix) | ✅ correct next page (was: identical page forever) |
| enginetest package compile | ❌ `declared and not used: ctx` ×2 (keyset_pagination.go:55,122) |
| preflight-composed | ⏸ not re-run (would fail api-stability: golden is stale again — see §5) |
| verify chain | ❌ 12B expired failed on calibration load gate; not relaunched |

---

## 3. What I Did (chronological, exact)

1. **Recovered context**: skills loaded (go-cqrs-lite, buildflow), tree = daemon-clean except old status doc; confirmed T07 files were absorbed as auto-commits.
2. **bg 12B autopsy**: `can-run-composed-gate` GREEN after 9 attempts → `verify-load-guard` REFUSED (load) → 8 more gate retries → verify started → **calibration-gate FAILED (load5=12.74 ≥ ceiling 10)**. Chain exit 1.
3. **Regenerated api-stability golden** (the blocking gate-RED): only adds `LastPairCursor` (struct fields aren't golden entries). Verified + meta-tests green.
4. **Fixed + completed `metaengine/scanpage_cursor_issuance_test.go`**: repaired `walkPages` (raw `WithCursor(cursor.Value)` vs string `WithCursorString` pass-back; the old version dead-stored opts and always used the string form); added 4 test functions (tie-heavy exact-once 13/5 both modes; exact-end nil-cursor 12/4; ParseCursor compound round-trip incl. wire-shape pin `{"Sort":2,"Key":"aXRlbS0wMDc="}`; 5 malformed-shape degradation guards + string-stays-string + empty=start).
5. **Mutation-tested the tests** (the previous cursor test was accidentally tie-safe — I refused to trust mine unmutated): disabled memory-engine emit → **tests failed via INFINITE LOOP, not the predicted drops**. Root-caused with a debug probe + SortPaginate instrumentation (all instrumentation since removed, probe trashed):
   - **Bug A (introduced pre-halt)**: emitted `SortKeyCursor.Sort` carried the **whole last row** (`valueOf(last)` = full struct). Only worked because the closure comparator field-extracts *both* operands. Leaks the entire row into HTTP-facing cursor strings; would bind garbage in SQL keyset predicates.
   - **Bug B (pre-existing, user-visible)**: a **legacy scalar cursor** (e.g. reflection-minted `0`, or any user-supplied `WithCursor(int)`) on the closure path: `itemFieldByName(scalar)` → nil → `compareValue(x, nil)` = +1 → filter inert → **the same page is re-served forever**.
6. **Fixed both**:
   - `normalizeClosureCursor` (typed_reader_scan.go): narrows compound Sort from whole-row to the bare sort-column value — the one place that knows `cfg.sort.Column`. SortKeyCursor doc contract updated ("Sort carries the bare sort-column value").
   - `buildClosureSort` scalar fallback + `isRowValue` (reflect.go): bare-scalar cursor operands compare directly; row-shaped operands keep nil-extraction semantics. **Legacy scalar cursors now paginate correctly** (probe-verified).
   - New pin: `TestScanPage_EmittedCursor_ScalarSortWireShape` (Sort must be scalar `0`, Key `item-012`).
   - Re-ran mutation: now fails by **drops (10 vs 13)** + phantom page + int cursor — exactly the three bugs T07 exists to kill.
7. **All engine suites green** (see §2). Also **discovered foreign edits** in `metaengine/{benchmark,engine_stats,explain}.go` (go-humanize display tweaks — another session's work; investigated, left untouched).
8. **Discovered pre-existing size-gate breach**: `sqliteengine/engine.go` was 683 lines vs 663 baseline (growth predates this session, absorbed by daemon, cause not yet investigated).
9. **T07 SQL slice — sqlite (landed, green)**:
   - **New core `metaengine/sql_keyset.go`** (125 lines): `AppendKeysetCursorPredicate` (legacy `col op ?` AND compound `(col op ?) OR (col = ? AND key > ?)`, key tiebreak always ascending to mirror SortPaginate's `bytes.Compare`), `AppendKeysetOrder` (ORDER BY sort [DESC], key), `LastJSONRowCursor` / `LastDecodedRowCursor` (emit from raw JSON / decoded rows), `normalizeCursorSortValue` (integral float64 → int64; JSON-round-tripped cursors bind cleanly on strict servers).
   - **sqliteengine/raw_reader.go rewritten**: `ScanRawValues` now emits `NextCursor` (LastJSONRowCursor over last row + its key column); shared `buildStandardScanQuery` (SELECT value, key; compound predicate; ORDER BY with key tiebreak; limit+1 probe); `buildPlannedSelectQuery` converted identically; new `scanRawRowsWithKeys` 2-column scanner.
   - `pushdownMapScanPlanned` (planned.go): keys ride along; emit via LastDecodedRowCursor.
   - `backends.go`: `scanJSONValuesWithKeys`.
   - `engine.go` `PushdownMapScan` standard branch → shared builder + emit: **file shrank 683 → 642** (heals the size-gate breach as a side effect).
   - Build + vet + full `-short` suite green.
10. **enginetest conformance (written, RED on trivial compile error)**: `enginetest/keyset_pagination.go` — `RunKeysetPaginationTest` (asc tie-walk [5,5,3] + cursor shape + DESC walk) and `RunKeysetExactEndTest` (12/4 phantom kill), plus exported `KeysetPaginationQuery`; wired into `sqliteengine/keyset_pagination_test.go`. **Not yet run once** — the two unused-`ctx` compile errors stop the package.

---

## 4. What Is Totally Fucked Up Right Now

1. **`enginetest/keyset_pagination.go` does not compile** — `ctx` declared-not-used at lines 55 and 122 (I create `context.Background()` in both Run functions but `seedKeysetRows` makes its own). Fix = delete both declarations (or pass ctx through). Everything downstream (conformance run, sqlite wiring verification) is blocked on it.
2. **api-stability golden is stale AGAIN by my own hand**: `sql_keyset.go` + enginetest add exported symbols (`AppendKeysetCursorPredicate`, `AppendKeysetOrder`, `LastJSONRowCursor`, `LastDecodedRowCursor`, `KeysetPaginationQuery`) — golden regen REQUIRED before any gate run. I violated the same-edit-regen rule mid-flow; caught it now.
3. **The DESC compound walk is completely unverified** (it's inside the not-yet-compiled conformance test). The DESC predicate (`(col < ?) OR (col = ? AND key > ?)`) is written but has never executed.
4. **W1 verify never ran green** — 12B died on the calibration load gate. Tree has never been through `#verify` with any T07 code.

---

## 5. Mistakes & Learnings (this session)

- **Mutation testing immediately paid for itself**: the loop failure mode exposed two bugs the happy-path tests could never show. Discipline for next time: mutation-test contract tests BEFORE trusting them, always.
- **Forgot the api-golden same-edit rule** when creating sql_keyset.go. The gate-red is self-inflicted and known.
- **Golden doesn't cover struct fields** — NextCursor was never in api_surface.txt; the real gate-red was `LastPairCursor` (pre-halt change). Earlier "tree gate-RED on ScanResult fields" assumption was half-wrong.
- **File-size baseline not checked before editing engine.go** — got lucky: my refactor shrank it past the breach I hadn't noticed. Check baselines BEFORE growing any file.
- Debug discipline worked: instrument → probe → root-cause → fix → remove all instrumentation → trash probe (via `trash`, not rm).
- Two other crush sessions are active on this machine; their edits (humanize work in metaengine) landed in my diff view mid-session. Investigated before touching; left alone. Coordination is by "investigate, don't revert."

---

## 6. Improvements I Want to Make (not yet done)

1. `buildStandardScanQuery` doesn't validate identifiers (callers do); `buildPlannedSelectQuery` does. Unify.
2. CHANGELOG `[Unreleased]` entries owed: compound-cursor issuance+consume (sqlite + core helpers), the **legacy infinite-loop fix** (user-visible!), normalizeClosureCursor wire narrowing, conformance harness.
3. readmodels.md doc row (f064) + wire-format golden file (f064) still owed.
4. art-dupl check after SQL slice (I added cross-engine-shareable helpers — the pg/mysql builders should switch to them, which *reduces* clone surface).
5. Wire `RunKeysetPaginationTest`/`RunKeysetExactEndTest` into pebble/bbolt/badger/dgraph/duckdb suites (cheap, high pin value).
6. TODO_LIST row for the treefmt/yaml-formatter golden-exclusion idea (config-war class cure, still open).

---

## 7. Decisions I Made Autonomously (the 3 halt questions stayed unanswered)

1. **Q3 (T07 SQL depth)** → decided **full compound emit+consume** for SQL engines: emit-only would be self-inconsistent (an engine that issues a cursor it cannot consume = broken pagination). Landed for sqlite; pg/mysql/duckdb follow.
2. **Q2 (quiet-window policy)** → worked through; verify late (chain expired anyway — will relaunch after code work).
3. **Q1 (18:35 attribution)** → moot operationally; repair stands. Still genuinely unknown whether user-authored.

---

## 8. Exact Resume Steps (in order)

1. Fix `enginetest/keyset_pagination.go` unused `ctx` (delete 2 lines) → build enginetest + run `TestSQLite_KeysetPagination` (first real run: asc, shape, desc, exact-end).
2. `cd cmd/api-stability && GOWORK=off go run . --update` (re-stale from sql_keyset.go exports) + TestEvery.
3. Re-run sqliteengine + metaengine `-short`.
4. Wire conformance into KV engines + memory + duckdb; run their suites.
5. pg/mysql/duckdb SQL slices (compound emit+consume via the new core helpers; duckdb locally testable, pg/mysql compile+integration suites).
6. f064: wire-format golden, readmodels.md row, CHANGELOG entries, art-dupl, file-size self-check.
7. `bash scripts/preflight-composed.sh` → 9/9.
8. Relaunch verify chain (`bash scripts/can-run-composed-gate.sh --wait-loop && nix run .#verify`).
9. f046 receipts → T04 → T05 → W2 tail → W3–W6.

---

## 9. The Next ~50 Things (rough order)

**T07 finish (W1):** 1. fix enginetest compile 2. run sqlite conformance 3. api golden regen 4. sqlite+metaengine suites 5. wire conformance into memory 6. pebble 7. bbolt 8. badger 9. dgraph 10. duckdb engine slice (emit+consume) 11. duckdb conformance 12. pg planned slice 13. pg standard slice 14. mysql planned slice 15. mysql standard slice 16. pg/mysql compile checks 17. integration-pg run 18. integration-mysql run 19. wire-format golden (f064) 20. readmodels.md doc row 21. CHANGELOG entries 22. art-dupl annotate/check 23. file-size self-check 24. preflight 9/9 25. verify chain green 26. f046 receipts 27. investigate engine.go 663→683 growth origin (git archaeology).
**W1 tail / T04:** 28-34. f047–f052 (quiet-window dependent).
**T05:** 35-39. f053–f057.
**W2 tail:** 40. T08 41. T09 42. T10 43. T11.
**Later waves:** 44. W3 45. W4 46. W5 47. W6 48. treefmt/yaml golden-exclusion TODO row 49. unify identifier validation in SQL builders 50. release-wave prep (batch-release test sweep lesson).

---

## 10. Three Questions for the User

1. **Verify chain policy (carried over):** relaunch the wait-loop chain now (it may fire overnight while pg/mysql work lands — tree instability will keep resetting it), or hold until T07 is fully closed and relaunch once? 
2. **18:35 attribution (carried over):** did YOU make the golangci/templ/middleware changes and push the bridge tag, or is it all daemon regression? (Only affects whether the treefmt/yaml-formatter exclusion TODO gets priority; the repair itself is done and green.)
3. **SQL fan-out confirmation:** sqlite landed with full compound emit+consume. Confirm pg/mysql/duckdb get the same full treatment this week (they need their slices + integration suites), or is sqlite-only + reflection fallback acceptable for the W1 boundary?
