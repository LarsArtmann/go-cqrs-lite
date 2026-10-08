# W1 T07 Status — pg + mysql Compound-Cursor Slices LANDED (live-verified, mutation-pinned); 2 Pre-Existing MySQL Bugs Fixed; dup gate RED on 4 new groups

Date: 2026-10-08 21:03
Session: resumed from 20:38 halt (`2026-10-08_20-38_w1-t07-kv-unification-duckdb-landed.md`)
Directive: standing "READ, UNDERSTAND, RESEARCH, REFLECT … keep going" (re-issued = resume order)

## TL;DR

T07 (compound-cursor issuance + consume, all engines) is now **functionally complete across all 9 engines + turso-by-delegation**. pg and mysql were converted, verified against LIVE servers (ephemeral PG + QEMU MariaDB), and mutation-pinned. The live runs exposed **two pre-existing mysqlengine/claimkit bugs** (parallel-construction index race; 64-char identifier overflow) — both fixed and proven by the previously-failing tests. f064 docs landed (wire-format byte golden, readmodels row, CHANGELOG). **The session is halted mid-f064: `#check-duplication` is RED with 4 new clone groups that still need `//art-dupl:accept` annotations (or consolidation).**

## a) What I set out to do + immediate highest-priority next steps

Planned (from the 20:38 halt report's Exact Next Steps): duckdb mutation checks → duckdb full suite → api golden regen → pg slice → mysql slice → f064 → preflight → verify chain.

**Highest-priority next steps, in order:**

1. **Clear the 4 dup-gate groups** (blocking f064 + verify):
   - `pushdown_standard_ties` ×3 (duckdb/mysql/pg, ~90 lines each) — either annotate as cross-module mirrors (`//art-dupl:accept cross-module conformance mirror — separate go.mod`, same doctrine as register.go clones) **or** (better engineering) hoist a generic `enginetest.RunPushdownStandardTiesTest(t, eng, collection)` into the harness and shrink all three locals to 3 lines. Recommend the hoist: the bodies differ ONLY in engine constructor + collection name.
   - duckdb/pg pushdown.go DESC ORDER-BY block (2-clone) — tiny dialect region; annotate.
   - `enginetest/keyset_pagination.go:44-57 vs 141-154` (same-file helper duplication from the earlier harness session) — refactor into one helper or annotate.
   - `sqliteengine/backends.go:119-124 vs raw_reader.go:252-257` — **I did NOT touch these files this session**; the group likely surfaced via sibling-session edits or detection drift. Investigate before annotating (check `git log -1` on both files).
2. **preflight-composed.sh → 9/9**, then quiet-window verify chain (`can-run-composed-gate.sh --wait-loop && nix run .#verify`).
3. **dgraph integration leg** (`nix run .#integration-dgraph`) — the dgraph conformance wiring still skips serverless; T07 claims "all engines", so run the leg once.
4. f046 receipts → T04 (f047–f052) → T05 (f053–f057) → W2 tail.

## b) What worked (all test-verified)

- **duckdb debt paid**: guard 158 (planned) pinned by existing suite (3 subtests fail under mutation). Guard 125 (standard) was **unpinned** — first mutation PASSED. Wrote `TestDuckDB_PushdownStandardTies` (cgo, unplanned MapSet-seeded collection, 30 rows, pages [4×7,2] asc+desc) → mutation now fails both subtests. Full duckdb `-short` suite GREEN (no explain-golden fallout).
- **api golden**: regen (7577 exports), TestEvery green forced `-count=1` (the cached result right after a golden write was not trusted).
- **pg slice complete**: planned path on core helpers (`AppendKeysetCursorPredicate`/`AppendKeysetOrder`, `$N`, literal LIMIT n+1, compound-aware Sort validation); standard path with inline `::jsonb` compound predicate (dialect territory — key bind must NOT carry the jsonb cast, so the generic placeholder func cannot express it); 2-col scanners + `LastDecodedRowCursor` emit on both paths. Unit pins updated + 3 new compound tests (predicate shape, mis-typed Sort → Rejection, float64→int64 bind normalization).
- **Live-PG verification**: `ephemeral-pg.sh` (nixpkgs PG, PGDATA_CACHE) — conformance + standard-ties GREEN first run; **both emit guards mutation-verified against the live server** (planned guard → 3 subtests fail; standard guard → both subtests fail); revert clean; re-green.
- **mysql slice complete**: same architecture; MariaDB twin-column semantics preserved by passing `skc.Sort` into `jsonCursorExpr` (numeric cursors keep the DECIMAL-cast/twin-column path).
- **Live-MariaDB verification**: QEMU VM (`vm-mysql.sh`) — run 1 FAILED on the two pre-existing bugs below; after fixes run 2 conformance + standard-ties GREEN; **both guards mutation-verified on live VM**; reverts clean; serverless suite green.
- **Turso claim verified**: tursoengine delegates ALL storage to `sqliteengine.NewSQLiteEngine` (register.go:5) — the CHANGELOG's "(sqlite, turso, …)" is delegation-true, not a doc lie.
- **f064 docs**: `TestCursorWireFormatGolden` (byte-exact pins: compound float/int/int64 + legacy scalar; frozen-contract doc comment). readmodels.md gained "Keyset pagination: compound cursors (all engines)" (wire shape, ordering rule, normalization, HasMore contract). CHANGELOG [Unreleased]: 4 entries (compound all engines; pebble sort-index; claimkit race; mysql ident overflow; KV bare keys). doc-check 1174 refs ✓; md-go 1477 blocks ✓.

## c) What didn't work / mistakes I made

1. **Edit-tool header clobbering ×2** (mysql `scanMySQLJSONValues`, claimkit `claimsDDL`): my multiedit replaced a function's header with new content without re-including the original header — orphaned bodies broke the parse. Caught immediately by vet/build; repaired. LESSON (now internalized): when inserting BEFORE a symbol, the new_string must END with the original header text.
2. **CHANGELOG content drop**: the [Unreleased] edit's old_string contained the Turso IVM entry; my new_string omitted it. Noticed within a minute and restored. Same failure class as (1) — destructive replace without carrying all prior content.
3. **readmodels snippet wrong twice**: invented a page-struct API, then a builder chain; the real surface is `ScanPage(ctx, ...ScanOption)` + `WithLimit`/`WithCursorString` package funcs. Should have read the API BEFORE writing the doc.
4. **keep-alive VM died with its background shell** — the `&`-detached script was killed when the tool shell completed; fell back to a fresh VM per mutation round (~40s overhead each, correctness unaffected).
5. `grep -c` returning exit 1 on zero matches silently broke a `&&` chain (re-ran the missing step).
6. **The duckdb standard-path gap existing for a day**: the prior session declared duckdb "conformance green" — true, but vacuously for the standard path. Only the enforced mutation discipline exposed it.

## d) Pre-existing bugs found + fixed (both proven by tests that failed before / pass after)

1. **claimkit MySQL TOCTOU race** (`metaengine/claimkit/dialect.go`): `ensureIndexes` probes `information_schema` then CREATEs — two engines constructed in parallel (the harness does exactly this via `t.Parallel`) both pass the probe; the loser died with Error 1061 at CONSTRUCTION. Fix: tolerate 1061/Duplicate-key-name (`isDuplicateIndexErr`). Previously-failing `TestMySQL_PushdownStandardTies` construction now passes.
2. **MySQL 64-char identifier overflow** (`metaengine/mysqlengine/planned.go` + `evolve.go`): `idx_meta_planned_<collection>_<field>` overflows for realistic collection names (the harness's scoped names hit it immediately) — layout application failed entirely. Fix: `mysqlSafeIdent` (deterministic truncate + FNV-1a hash suffix, no collisions) applied in `registerPlannedLayout` AND at the top of `EvolveLayoutPlan` (which previously bypassed and would have ALTERed a nonexistent un-truncated table).

## e) Current blocked state (why I stopped)

`nix run .#check-duplication` **RED**: 4 new clone groups (baseline 186 → would be 190). The dirty-tree guard also forbids re-pinning the baseline while uncommitted changes exist, and re-pinning is the wrong tool anyway (all four groups are either deliberate mirrors or refactorable). Everything downstream (f064 completion, preflight, verify, f046 receipts) is sequenced behind this.

## f) Open questions (max 3, need user input)

1. **Foreign-failure policy during `#verify`**: two sibling sessions are active and the daemon absorbs their edits; if the exclusive verify run fails on files I did NOT touch (e.g. the sqliteengine clone group may already be sibling fallout), do you want (a) report-only, I fix nothing foreign, or (b) I fix foreign failures too when the fix is mechanical?
2. **Ties-test consolidation vs annotation**: hoist `enginetest.RunPushdownStandardTiesTest` (my recommendation, deletes ~180 clone lines) vs keep engine-local mirrors with `//art-dupl:accept`? Hoisting touches the shared harness again; annotation is zero-risk.
3. **Release timing for the T07 batch**: ride the next tag train as soon as verify is green, or hold until the W2 tail (T08–T11) so the compound-cursor contract ships once with its docs/lint tail (AsyncAPI + cookbook land in T08)?

## g) Suite/gate snapshot

- metaengine, sqliteengine, pebbleengine, bboltengine, badgerengine, duckdbengine, pgengine (full `-short`), mysqlengine (full `-short`): **GREEN**
- pg: live integration (conformance + standard-ties) GREEN; mysql: live QEMU integration GREEN (after 2 fixes); dgraph: skips serverless (integration leg owed)
- api-stability: golden updated, TestEvery green
- doc-check ✓ (1174 refs) · md-go ✓ (1477 blocks) · **check-duplication ✗ (4 groups)** · check-file-size: not re-run this session (all touched/new files <350 lines; mysqlengine/planned.go ~205, pushdown.go ~290, pgengine/pushdown.go ~270 — none baselined)
- verify chain: NOT run (queued behind dup gate + quiet window)

## Foreign activity (not mine, untouched)

- `flake.nix` modified by a sibling session (left alone).
- sqliteengine backends.go/raw_reader.go clone group — authorship unverified.
- Prior-session rules honored: no `rm`/`git reset`/`git checkout`; `trash`/`git restore` unused (nothing needed reverting); systemctl untouched; BuildFlow-owned steps untouched (no manual golangci/format/tidy).
