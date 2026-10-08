# W1 T07 Status — Dup Gate CLEARED (hoist + annotations); pg/mysql Landed & Live-Verified; Preflight 7/9 (api golden drift = mine; templ = foreign); HALTED for Instructions

Date: 2026-10-08 21:09
Supersedes: `2026-10-08_21-03_w1-t07-pg-mysql-landed-gates-red.md` (written mid-flow; this report continues it — its "halted" framing was premature, the directive said keep going, and I did)
Directive: standing "READ, UNDERSTAND, RESEARCH, REFLECT … keep going until everything works"

## TL;DR

T07 compound-cursor rollout is **functionally complete on all 9 engines (+turso by delegation)** — every emit guard mutation-pinned, pg/mysql verified against live servers. The RED duplication gate is now **GREEN** (hoisted the ties test into enginetest, extracted the harness setup helper, annotated 4 dialect/idiom regions; 0 new clone groups, baseline 186). **Preflight-composed is 7/9**: api-stability FAIL is MY miss (new export `enginetest.RunPushdownStandardTiesTest` added after the last golden regen — same-edit contract violated, one-command fix pending); templ FAIL looks foreign (docserver). **Session halted per your order — awaiting instructions.**

## a) What I set out to do + immediate next steps

Carried from the 20:38 halt: duckdb mutation checks → duckdb full suite → api golden → pg slice → mysql slice → f064 → preflight → verify chain.

**Highest-priority next steps, in order:**

1. **Regen api golden** (`cd cmd/api-stability && GOWORK=off go run . --update && GOWORK=off go test -run TestEvery .`) — pins `enginetest.RunPushdownStandardTiesTest`. Also re-run `scripts/check-changelog-symbols.sh` (my [Unreleased] entries cite pkg symbols against that golden).
2. **Triage templ preflight failure** (`/tmp/preflight-templ.log`): if it is stale generated artifacts → `(cd catalog/docserver && templ generate)`; if it is a sibling's live edit → leave + report. Ownership check via `git log -1 catalog/docserver` first.
3. **Re-run preflight → 9/9**, then full suites for metaengine/sqliteengine/pgengine/mysqlengine/duckdbengine after the harness refactor (only targeted runs so far).
4. **Quiet-window verify chain**: `bash scripts/can-run-composed-gate.sh --wait-loop && nix run .#verify` (background; load<10 + 60s tree stability; tree is CLEAN right now — daemon just committed everything).
5. **dgraph integration leg** (`nix run .#integration-dgraph`) — the only engine whose T07 wiring is not server-verified yet.
6. f046 receipts on TODO rows → T04 (f047–f052) → T05 (f053–f057) → W2 tail (T08–T11).
7. Cheap hygiene: annotate the 21:03 report as superseded (done via this file's header); add the "insert-before-symbol edit" lesson to docs/agents gotchas.

## b) What worked (this session, all test-verified)

- **duckdb debt**: guard 158 pinned by suite; guard 125 was UNPINNED (first mutation passed!) → wrote `TestDuckDB_PushdownStandardTies` → pinned. Full duckdb suite green (no explain-golden fallout).
- **pg slice**: planned path on core keyset helpers + compound-aware Sort validation; standard path with inline ::jsonb compound predicate (key bind must differ → generic placeholder func cannot express it); 2-col scanners + emit both paths. **Live ephemeral PG**: conformance + standard-ties GREEN first run; both guards mutation-pinned against the live server.
- **mysql slice**: same architecture; MariaDB twin-column semantics preserved (`jsonCursorExpr(sort.Column, skc.Sort)`). **Live QEMU MariaDB**: green after fixing the two pre-existing bugs below; both guards mutation-pinned on live VM.
- **2 pre-existing bugs found & fixed** (each proven by a test that failed before / passes after):
  1. claimkit MySQL TOCTOU race (`information_schema` probe → CREATE INDEX) — parallel engine construction died with Error 1061; now tolerated.
  2. MySQL 64-char identifier overflow — `mysqlSafeIdent` (truncate + FNV-1a suffix) in registerPlannedLayout + EvolveLayoutPlan (which previously would have ALTERed a nonexistent un-truncated table).
- **Turso claim verified**: tursoengine delegates ALL storage to sqliteengine.NewSQLiteEngine (register.go:5) — CHANGELOG "(sqlite, turso, …)" is delegation-true.
- **f064 docs**: `TestCursorWireFormatGolden` (byte-exact compound/legacy wire pins, frozen-contract doc); readmodels.md "Keyset pagination: compound cursors" section; CHANGELOG [Unreleased] ×4 entries + Turso preserved. doc-check 1174 refs ✓; md-go 1477 blocks ✓.
- **Dup gate RED → GREEN**: hoisted `enginetest.RunPushdownStandardTiesTest` (deleted ~270 clone lines across 4 test files — duckdb/mysql/pg became 10-line wrappers, sqlite converted onto the harness too); extracted `setupKeysetStore` (killed the harness-internal setup duplication); annotated duckdb/pg ORDER-BY regions (dialect, separate go.mod) + sqlite scanner preambles (idiom; foreign file — comment-only change). Gate: **0 new clone groups**.
- Preflight phases PASS: lint-config, bench-gate, coverage, duplication, go-version, turso-version, error-taxonomy (7/9).

## c) What didn't work / mistakes I made

1. **API-surface contract violation (the current blocker)**: the hoist added exported `enginetest.RunPushdownStandardTiesTest` and I did NOT regen the api golden in the same edit — preflight caught the drift. Fix is one command; the miss was real (AGENTS contract #5).
2. **Edit-tool header clobbering ×2** (mysql `scanMySQLJSONValues`, claimkit `claimsDDL`): multiedit replaced a function header with new content without re-including the original — orphaned bodies broke the parse; caught by vet/build instantly. LESSON: when inserting BEFORE a symbol, new_string must END with the original header.
3. **CHANGELOG content drop**: my [Unreleased] edit's old_string contained the Turso IVM entry; new_string omitted it. Noticed within a minute, restored. Same destructive-replace class as (2).
4. **Doc-before-API-read**: readmodels snippet invented the API twice (page struct → builder chain) before reading the real `ScanPage(ctx, ...ScanOption)` surface.
5. **Premature "halted" report** at 21:03 while the directive said keep going — I continued (this report supersedes), but the framing error is noted.
6. **duckdb standard-path gap existed a full day** — prior session's "conformance green" was vacuously true for the unplanned path; only enforced mutation exposed it.
7. Minor: `grep -c` exit-1 broke a `&&` chain; one wrong relative `cd` after an earlier `cd`; keep-alive VM died with its background shell (fell back to ~40s fresh VM boots per mutation round).

## d) Key metrics/files

- Changed (production): pgengine planned_scan.go+pushdown.go; mysqlengine planned_scan.go+pushdown.go+planned.go+evolve.go; claimkit/dialect.go; duckdbengine+pgengine+sqliteengine (annotations); enginetest keyset_pagination.go (setupKeysetStore) + NEW pushdown_ties.go.
- Changed (tests/docs): 4 standard-ties wrappers (duckdb/mysql/pg/sqlite); 2 conformance wirings (pg/mysql); 2 compound unit-test batteries; cursor_wire_golden_test.go; readmodels.md; CHANGELOG.md.
- All touched/new files <350 lines; only baselined file anywhere near (pgengine/engine.go 416) untouched.
- Tree state: CLEAN (auto-commit daemon absorbed everything — attribution lost as expected).

## e) Current blocked state

preflight-composed 7/9 → `#verify` must NOT launch (exclusivity + preflight remedy rule). Two blockers: api golden (mine, one command) + templ (ownership untriaged).

## f) Open questions (max 3, need your input)

1. **templ failure ownership**: `/tmp/preflight-templ.log` points at catalog/docserver templ artifacts I never touched. If it is a sibling session's live work: leave-and-report, or regenerate anyway?
2. **Verify scheduling**: two sibling sessions are active on this shared 32-core box. Run the wait-loop verify chain NOW in the background (it self-gates on load<10 + tree stability), or defer to a specific window you prefer?
3. **Release timing for the T07 batch**: ride the next tag train once verify is green, or hold until the W2 docs/lint tail (T08 AsyncAPI + cookbook) so the compound-cursor contract ships once with its full documentation?

## g) Suite snapshot

- metaengine, sqliteengine, pebbleengine, bboltengine, badgerengine, duckdbengine, pgengine, mysqlengine `-short`: **GREEN** (targeted re-runs after the refactor; full-module re-runs pending in next steps).
- Live: PG conformance+standard-ties ✓ (mutation-pinned); MariaDB conformance+standard-ties ✓ (mutation-pinned, after 2 fixes); dgraph leg OWED.
- Gates: doc-check ✓ · md-go ✓ · duplication ✓ (186 baseline, 0 new) · preflight 7/9 (api-stability ✗ mine · templ ✗ foreign?) · verify NOT run · file-size not re-run (spot-checks pass).
