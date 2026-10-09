# W1/T07 Status — KV Cursor-Key Unification, Pebble Sort-Index Overhaul, DuckDB Compound Landed (HALT)

Date: 2026-10-08 20:38 CEST · Branch: master · Session: v5-GOAL W1 T07 continuation (resumed via standing directive)

## Verdict

T07 (compound-cursor issuance + consume across engines) advanced from "sqlite-only verified" to
**sqlite + memory + bbolt + badger + pebble + duckdb all behaviorally verified** — including one
architecture-level unification (all engine families now tiebreak on the BARE user key) and two
genuine pre-existing bugs found and fixed via the new conformance harness. Tree is GREEN on every
suite run this session. NOT yet done: duckdb mutation check + full duckdb suite, pg/mysql slices,
f064 docs, api golden regen (keycodec export), preflight, verify chain.

## What landed this session (all verified by tests, mutations reverted)

### 1. enginetest conformance harness completed and de-vacuated

- Fixed the 2 unused-`ctx` compile errors (`keyset_pagination.go:55,122` deleted).
- **Vacuity bug found + fixed**: seeded via `ScopedCollection("keysetRow")` — wrong half of the
  contract. `store.Apply` targets must be the RECORD TYPE NAME (`"keysetRow"` — what `OnRecord`
  matches on); the QUERY name is what gets scoping. Inverted usage → 0 rows → test vacuously
  failing. Fixed: `queryName := ScopedCollection("keyset_pagination")` + `Apply(..., "keysetRow", ...)`.
- Added `engine pushdown surface` subtest: drives `PushdownMapScan` DIRECTLY (bypasses TypedReader
  path choice) — pins SQL emit+consume for every PushdownScan engine; skips closure-only engines.
- Added `keysetEngineWalk` helper (HasMore-terminated engine-level walk, 50-page bound).
- Harness now has 3 subtests: asc tie-heavy exact-once [5,5,3], desc exact-once (both directions of
  the (sort DESC, key ASC) contract), engine pushdown surface (cursor shape Sort=0/Key=item-012).

### 2. Mutation-verified: sqlite's three surfaces

- Raw surface (`ScanRawValues`): emit-off → drops 10/13 + phantom [4,4,4,0] + DESC breakage. ✓
- Planned pushdown (`pushdownMapScanPlanned`): emit-off → "HasMore without cursor". ✓
- Standard pushdown (engine.go:483): NEW local test `TestSQLite_PushdownStandardTies`
  (MapSet-seeded unplanned collection, asc+desc walks, pages [4×7,2] over 30 rows); emit-off → fails. ✓
- Discovery along the way: `metaengine.Plan` AUTO-APPLIES a layout for declared sorts — the harness
  query rides the PLANNED path; the standard path was previously untested for cursors.

### 3. Architecture: KV cursor keys unified to the bare user key (wire contract)

- **Bug**: bbolt/badger/pebble closure cursors carried the FULL STORAGE KEY
  (`m\x00<col>\x00"item-012"` — prefix + JSON-quoted) — violating the wire contract
  (`{"Sort":<scalar>,"Key":"item-012"}`) and breaking cross-engine cursor portability.
- **Fix**: new `keycodec.UserKeyBytes(fullKey, prefix)` — strips collection prefix, JSON-unwraps to
  the bare rendered key (the same `fmt.Sprintf("%v")` form the memory engine uses). Adopted in
  bbolt/badger/pebble `MapScan` pair construction (and pebble raw plain path).
- **adrtest registry updated**: `PaginationProbe.CursorKey` for the 3 KV engines moved
  `CursorKeyKVMapKey` → `CursorKeyRaw`; the 2026-10-06 pinned finding doc updated to record the
  2026-10-08 unification (all families now CursorKeyRaw). `CursorKeyKVMapKey` kept (self-test uses
  the pattern).

### 4. Bug: pebble layout sort-index violated the compound contract three ways (fixed)

Root cause chain: pebble implements `RawScanReader`; `Plan` auto-applies a layout for declared
sorts → the walk served from `scanWithSortIndex` (an index-iterator path predating T07):

1. No compound consume — `encodeIndexValue(SortKeyCursor)` seeked garbage → tie-block drops.
2. DESC walked the index backward (`Last→Prev`) → **key-DESCENDING within ties** (contract: asc).
3. No cursor emission at all.

- **Rewrite** (`sort_index.go`): compound cursor → composite bound on the full
  `(encodedValue, primaryKey)` index entry; ASC resumes strictly after the entry; DESC ranges the
  cursor's whole value group and skips the served prefix IN-LOOP (byte ranges alone cannot express
  "later keys of this group, plus all lower groups" — first attempt used `upperBound=entry` and
  failed exactly as predicted, fixed to `nextKey(group)` + skip rule). DESC orders ties ascending
  via per-run buffering + reversal, bounded by targetCount (drop-oldest). Hits now carry
  (primaryKey, rawValue); emission via `LastDecodedRowCursor` + `UserKeyBytes` unwrap.
- Index keys carry the JSON-ENCODED pk form (`encodeKeyStr` at maintenance) — compound consume
  re-encodes the bare cursor key; Get uses the encoded form as-is (caught a double-encode mistake
  before it shipped).
- Pebble raw PLAIN path (no layout): keys → `UserKeyBytes`; emit added; SortPaginate consume was
  already compound-correct.
- Mutation-verified: sort-index emit-off → asc drops + desc drops + phantom page. ✓

### 5. duckdb slice LANDED (green on first run)

- Standard path (`pushdown.go`): `SELECT value, "key"`; dialect-local compound predicate
  (`::json` casts on sort binds only — key is VARCHAR; the shared helper takes one placeholder
  func, so this stays dialect territory): `(sort op $N::json OR sort = $N::json AND "key" > $N)`,
  key tiebreak ascending; `ORDER BY json_extract(...) [DESC], "key"`; emit via
  `LastDecodedRowCursor`.
- Planned path: `buildPlannedSelectQuery` rewritten on the CORE helpers
  (`AppendKeysetCursorPredicate` + `AppendKeysetOrder`, placeholder `$N+1`); `SELECT value, "key"`;
  `pushdownMapScanPlanned` moved from layout_planner.go into pushdown.go with emit added.
- `scanDuckDBJSONValuesWithKeys` (2-col scanner) replaces the 1-col variant.
- layout_planner.go SHRANK 358 → 330 (baseline 358 — shrink-only ratchet ✓).
- Conformance: asc/desc/engine-pushdown ALL PASS first execution. NOT yet mutation-verified
  (mutation pipeline mis-ran; both emit guards verified RESTORED — no `if false` remnants).

### 6. Engine-per-Run rewiring (double-close fix)

The harness's `t.Cleanup(store.Close)` closes the ENGINE (store owns engine lifecycle — same note
as the soak harness). Two Runs sharing one engine → pebble panic "pebble: closed". All 6 wiring
files (pebble/bbolt/badger/duckdb/dgraph/sqlite) now construct one engine per Run call.

### 7. Sibling-session ripple absorbed

The other session added `go-humanize` to metaengine core; 12 engine modules' go.mod/go.sum needed
tidy (api-stability's TestEveryModuleGoSumIsTidy gates it). buildflow `gomod-check --fix` passed
its own criterion but not the test's (sibling-replace resolution) — ran the per-module
`GOWORK=off go mod tidy` the test itself prescribes: 12 modules tidied, TestEvery GREEN.

## Suites status (all run this session, workspace mode)

- metaengine core -short: **GREEN**
- sqliteengine -short: **GREEN** (6.5s)
- pebble / bbolt / badger -short (FULL suites incl. adttest matrix): **GREEN**
- dgraph: conformance SKIP (no server) — expected
- duckdb: conformance **GREEN**; FULL suite NOT yet run this session (SELECT-shape change could
  affect explain goldens — check pending)

## Honest ledger (mistakes this session)

1. ScopedCollection inverted usage → vacuous harness (caught by 0-rows failure, fixed).
2. sed comment broke composite literal in first mutation attempt (compile error, not a valid
   mutation; redone guard-style).
3. multiedit on layout_planner.go created a dangling duplicated function header (overlapping
   old_strings across sequential edits); repaired immediately, structure verified.
4. Pebble DESC compound bound first design (`upperBound = entry`) wrong — excluded the byte-greater
   same-group keys; hand-simulation caught it, second design (group range + in-loop skip) correct.
5. duckdb mutation verification mis-ran (no test output captured); guards verified restored, but
   the emit-off→fail proof is still owed.
6. `git restore` used once on a file whose only uncommitted delta was my own bad mutation line
   (daemon had already absorbed the legit work) — safe, but noted for discipline.

## Current tree state

- Modified (daemon will absorb): duckdbengine/layout_planner.go, duckdbengine/pushdown.go,
  metaengine/sql_keyset.go (FOREIGN doc-comment tweak by sibling session — "bind args: sort, sort,
  key" → "sort, key"; investigated, left alone).
- No `if false` mutation remnants anywhere (verified by grep after each revert).
- api golden: **STALE AGAIN** — `keycodec.UserKeyBytes` is a new export (keycodec module) + the
  enginetest additions; regen owed.

## Remaining (ordered)

1. duckdb: mutation-verify the two emit guards; run FULL duckdb -short suite (explain goldens may
   pin the old single-column SELECT — fix goldens if the new shape is correct).
2. api golden regen + TestEvery (keycodec.UserKeyBytes et al).
3. pg + mysql slices: `planned_scan.go:119-123` still on legacy `AppendPlannedCursor`/
   `AppendPlannedOrderLimit` — convert to `AppendKeysetCursorPredicate`/`AppendKeysetOrder` + emit
   (mirror duckdb planned path; pg uses `$N` plain — core helper fits verbatim); compile-check;
   integration suites if servers available (`nix run .#integration-pg` / `#integration-mysql-vm`).
4. f064: wire-format golden note, readmodels.md doc row, CHANGELOG [Unreleased] entries (now 4:
   compound issuance+consume; KV bare-key unification; pebble sort-index DESC/tiebreak fix; legacy
   infinite-loop fix from prior session), art-dupl check (rewrites may have CHANGED baselined
   shapes — `#check-duplication` needs a run; annotate rather than re-pin), file-size self-check
   (layout_planner shrank ✓, pushdown.go ~200 lines ✓ under 350).
5. `bash scripts/preflight-composed.sh` → expect 9/9.
6. Verify chain in a quiet window (`can-run-composed-gate.sh --wait-loop && nix run .#verify`).
7. Then: f046 receipts → T04 f047–f052 → T05 f053–f057 → W2 tail.

## Standing questions (answered autonomously per plan policy; user may override)

- Verify-chain policy → work through, verify late (quiet window).
- 18:35 attribution → moot; daemon absorbs; don't fight it.
- SQL fan-out → full compound emit+consume on pg/mysql/duckdb (an engine issuing a cursor it
  cannot consume is broken pagination) — duckdb done, pg/mysql next.
