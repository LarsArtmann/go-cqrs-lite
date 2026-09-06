# Pareto Execution — Session 4 (2026-08-30, ~07:00 → late)

Ledger for the continuation session executing
`docs/planning/2026-08-29_20-33_pareto-v4-closeout-and-v5-train-plan.md`.
Session 3 left A1/A2 done and A3 blocked; this session finished A3, the whole
B tier, the whole C tier (with two items closed as already-implemented), and
the D-tier items that are not user-gated (D1, D2, D7, D8). All commits
per-task, all gates run per task.

## a. Commits (in order)

| Commit      | Task      | What                                                                                                                                                                                                                                |
| ----------- | --------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `3bcb7030e` | B1        | depguard restored (v2 object rules schema), 84-entry allow list, indentation-tolerant check-depguard.sh, all 119 requires covered, lint 76/76 clean                                                                                 |
| `eef6fa85d` | A3        | dgraph `@recurse(depth: N)` counts node levels, not hops — GraphNeighbors requests depth+1; two tests that had pinned the bug recalibrated; ephemeral-dgraph.sh SIGTERM-wedge fixed (stop_pid + timeout -k); live suite 89/89 green |
| `fd347183f` | B2.1      | nested RunInTx deadlock fixed via ctx-marker rejection (dgraph + identical latent bug in sqliteengine); read-your-writes / retry-after-abort / commit / rollback pins for both engines                                              |
| `97ad66f1a` | B2.2      | MariaDB DESC twin-column sort + flipped keyset cursor pinned live                                                                                                                                                                   |
| `218eb0c23` | B3        | storage's pgtestcontainer pin bumped v4.0.0→v4.1.0 (AfterRun existed only in the workspace; GOWORK=off was broken); full PG loop green                                                                                              |
| `86bcd7aff` | B4        | wave-1 CHANGELOG backfill (8 symbols verified in source first); symbol gate: 119 honest citations                                                                                                                                   |
| `3e134939e` | B5+B6     | AGENTS disk-cache env chain + TEST_ARGS quoting trap; TODO L49/L66/L437 closed after verification; recipes 2.24 (RunInTx) + 2.25 (VectorCounter); faq KeysetPositionQueryChecked; doc-check 938 refs                                |
| `5849c8ebb` | B8        | ErrWorkerFailed sentinel (Infrastructure, not Transient) for failed-worker staleness — consumer-visible reclassification, documented; boundedMap negative-dip comment; catalog multi-embed last-wins note; api golden +1            |
| `41e04c969` | C3+C9+C10 | one-tx-per-event TODO closed with evidence; v5 deletion-safety scans recorded; macOS claim made honest (static-review-only + CI-runner route)                                                                                       |
| `1fddcfbb5` | C5        | Calibration data race (mutex); routingSignature now covers ReadCosts + plan version (stale-cache fix); Replan oscillation killed by incumbent-aware hysteresis (complexity-class wins always pass); race-clean                      |
| `a6cefd34a` | C6+C7     | create-github-releases.sh (changelog-extracted bodies); CONTRIBUTING pre-tag checklist + retract-and-republish policy; C6 closed as already-shipped (#verify-ci + CI matrix exist)                                                  |
| `f063de4d1` | C1        | normalizeAny table tests; loopback dedup reset-window pins; 1K-op pooled-stream stress (streams==1); eviction→reopen pin; WithStreamPooling README                                                                                  |
| `d7fbb9b06` | D8        | ClaimingTimerStore: lease-based claiming, PG SKIP LOCKED + SQLite single-writer, MySQL rejected loudly; live two-claimer contention test                                                                                            |
| `6c7e08f4a` | D2        | mysqlengine planned tables (backtick quoting, split DDL, ON DUPLICATE KEY upserts); live MariaDB verified                                                                                                                           |
| `b8aa29d96` | D1        | pgengine planned tables (JSONB value, DOUBLE PRECISION/BIGINT/TEXT columns, LayoutPlanApplier, Map routing); live PG verified incl. mis-type fail-loud                                                                              |

## b. Findings worth remembering

1. **The session-3 audits were stale in two places** — re-verify before
   executing: C4's "6 silent-ignore durability engines" was already fixed
   (ADR-0130 + ffb1ae35: sqlite/pg/pebble/bbolt/badger map tiers for real,
   dgraph/duckdb/mysql/memory reject, iroh has no DriverConfig path at all);
   C6's "#verify-standalone" already exists as `#verify-ci` plus the CI
   module matrix.
2. **The dgraph live suite had never been run green**: it hung in
   TestRunInTx_NestedRejected (mutex deadlock — the rejection check sat
   behind the lock it was meant to justify), and the script cleanup wedged
   separately (dgraph alpha ignores SIGTERM; `kill; wait` hung 4h45m). Both
   fixed; suite now green end-to-end in ~52s.
3. **The MariaDB DESC test, the ASC-only suite, and the GraphRAG grace
   assertion were all calibrated to the off-by-one** — tests that pass on
   buggy behavior are worse than no tests; recalibrated to pin the correct
   contract (carol/grace now asserted PRESENT at 2 hops).
4. **Contention flakes in the dgraph suite are calibrated to the serial
   phase** — new write-heavy tests must be serial (no t.Parallel) or they
   abort under the parallel batch (MultiAdd roundtrip flaked once).
5. **`ephemeral-*.sh` env passthrough strips quotes** — TEST_ARGS with
   `'|'` alternation needs no inner quotes.
6. **Two D1/D2 development bugs caught by the new tests themselves** — the
   claim SQL's lease predicate compared against the lease deadline (making
   every claim instantly reclaimable; caught by the live contention test),
   and the PG claim used an unbound $2 (42P18). Tests first, honest green.

## c. D-tier state

- **D1/D2 done as the planned-table core slice**: plan registration +
  LayoutPlanApplier + Map routing + live round-trip/conflict/mis-type tests.
- **D3 (full filter/sort pushdown through planned tables + EXPLAIN
  index-usage proofs + cross-engine matrices) is the next engineering
  slice** — the routing seam (planFor) is in place in both engines;
  PushdownMapScan/MapScan still target meta_map and need the planned-table
  branch, then the adttest/adt matrices run per engine.
- **D4–D6 (v5 deletion waves) are semver-gated, not merely effort-gated**:
  they break published API and must land at the v5.0.0 cut per the plan.
  Everything they need is staged: consumer-scan safety proofs
  (v5-deprecation-sweep §6), snapshot wire-tag design note (§4), execution
  rules, and `docs/V5-MIGRATION-GUIDE.md` (D7) with the cut checklist.
- **D9 remains user-blocked** (billing, root, macOS hardware, external tags)
  plus the two still-open §g questions (DLQ fail-loudly semantics; unattended
  long-job policy). B1's third question (depguard) is resolved by this
  session: restored, per the plan default.

## d. Gate status (end of session)

- Live dgraph suite: 89/89 PASS.
- Live PG loop (7 modules): green through scheduling/sqlstore; final
  pgengine run is part of the closing `#verify` sweep validation.
- Live MariaDB mysqlengine: green. SQLite claiming: green.
- metaengine race paths (calibration/routing): green.
- Lint 76/76 (with depguard re-enabled), vulncheck 0 findings across all
  modules (also proves the GOWORK=off standalone matrix).
- doc-check 938 references, zero warnings; changelog-symbol gate 119 honest.
- Closing `nix run .#verify`: **GREEN — EXIT=0, "All verification checks
  passed"** (build, vet, test, race, lint 76/76, check-arch, check-depguard,
  check-duplication 111-baseline/0-new, check-coverage, api-stability 4339
  exports, doc-check). One mid-run cycle failed honestly: the api-stability
  golden was missing D1/D2/D8's 10 new exports (regenerated in `4b7d5a440`)
  and lint flagged 5 new files (fixed in `0c0d795eb`); the duplication gate
  then caught 12 cross-engine clone groups from the planned-table ports —
  all annotated as intentional (`1955e6f21`) with the gate re-run to zero.
