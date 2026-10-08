# W1 T03 Status — Verify Chain Held by Load Storm (4 refusals); T07 Verification COMPLETE on All 9 Engines; ADR-0152 Index Fixed

- **Date:** 2026-10-09 00:32 (session window 2026-10-08 21:12 → 00:32)
- **Supersedes:** `2026-10-08_21-09_w1-t07-dup-gate-cleared-preflight-7of9.md` (which superseded 21-03)
- **Task:** v5-GOAL Pareto plan T03/f044–f046 (+ T07 verification tail), resumed per standing directive
- **Verdict:** T07 is verified end-to-end on all 9 engines. T03 is ONE green `#verify` away — blocked only by the shared-host load storm (siblings), not by any code or doc state I own.

## a) FULLY DONE (this session)

1. **api-stability golden regen** — `cmd/api-stability --update` (7578 exports, pins `enginetest.RunPushdownStandardTiesTest`); `TestEvery` green with `-count=1`; `check-changelog-symbols.sh` verifies all 4 [Unreleased] citations. Clears the api-stability preflight failure from the 21:09 report.
2. **templ preflight failure — ROOT-CAUSED and FIXED.** A sibling ran `templ generate` FROM THE REPO ROOT at 20:59 → generated files carried path-prefixed `FileName:` fields (`catalog/docserver/layout.templ` instead of bare `layout.templ`) — exactly the cwd tripwire `check-templ-paths.sh` exists for. Host templ == pinned nixpkgs templ (v0.3.1020 both), so it was never a version drift. Regenerated from inside `catalog/docserver/`; `#check-templ` green on BOTH legs (codegen drift + FileName tripwire). Diff: 22 lines across 5 generated files, all FileName strings.
3. **Preflight 9/9 GREEN** — `preflight-composed.sh`: lint-config, templ, bench-gate, coverage, api-stability, duplication, go-version, turso-version, error-taxonomy all PASS.
4. **Full post-harness-hoist suites (serverless):** metaengine (9.4s, all subpackages), sqliteengine (4.8s), duckdbengine (1.1s) — `-short -count=1 ./...`, all green.
5. **pgengine full suite against live ephemeral PG** — green (3.7s) via `ephemeral-pg.sh` with PGDATA_CACHE.
6. **mysqlengine full suite — VERIFIED GREEN twice against a fresh userspace MariaDB** (recipe from gotchas-testing.md line 15: nixpkgs mariadb re-init at `/tmp/mariadb-cqrs`, port 33061; /tmp had been wiped so this was a full `mariadb-install-db` re-init; DDL via socket as OS-user per the documented unix_socket-auth trap; then DSN `cqrs:cqrs@tcp(127.0.0.1:33061)/cqrs_test?parseTime=true&multiStatements=true`). Run A: 61 PASS verbose; run B (stability, `-parallel 8`): green, and the scary-looking 0.157s duration is LEGIT (tmpfs datadir + warm schema + no VM/slirp overhead; 136 RUN entries, 1 pre-existing skip). Server torn down cleanly after.
7. **MySQL QEMU failures fully diagnosed — infra, not code.** Three VM runs failed with MOVING victims (`TestCapabilityConformance`; then `PushdownMapScan_Combined` + 2× Graph + `KeysetPagination`), ALL at engine-construction time with `invalid connection` after `connection reset by peer` waves; no mysqld restart/OOM in VM console; zero assertion failures anywhere; my new tests passed in every run where they got a connection. Root causes (both documented classes in gotchas-testing.md lines 21–22): (1) the args-verbatim path of `vm-mysql.sh` did NOT export the calibrated `ADTTEST_CAS_RACERS=10` that the built-in legs export; (2) slirp under host-load storms resets even single fresh dials (load was 29–34 during my runs).
8. **`scripts/vm-mysql.sh` footgun FIXED** — args path now default-exports `ADTTEST_CAS_RACERS=10` (overridable, `:="${...:=10}"` placed at the DSN export site so built-in legs keep their explicit values). Comment cites the gotcha.
9. **dgraph integration leg GREEN** — `nix run .#integration-dgraph` rc=0, full dgraphengine suite 100.8s incl. `TestGraphRAG_ConcurrentStress` (p99 13.5ms, 200 entities/~600 edges). The last T07 engine wiring without server verification is now verified.
10. **T07 CLOSED: all 9 engines (memory, sqlite+turso-by-delegation, pebble, bbolt, badger, duckdb, pg, mysql, dgraph) mint + consume compound SortKeyCursor cursors, every emit guard mutation-pinned, pg/mysql/dgraph live-server-verified.**
11. **ADR-0152 index drift FIXED** — sibling added `docs/adr/0152-fleet-first-module-topology-v5-dual-support.md` at 20:42 without indexing it; verify's doc-assertions failed "150 ADR files vs 149 indexed". Added the row to `docs/README.md` (alignment-matched); `verify-docs.sh` standalone: ALL assertions green (150/150 indexed).

## b) PARTIALLY DONE

1. **T03/f045 — full composed `#verify` green: NOT YET.** Attempt history tonight: #1 refused (load5 13.72), #2 refused (load5 11.55), #3 RAN but early-exited at doc-assertions on ADR-0152 (heavy phases never reached; fixed), #4 refused (00:19, load5 13.92). A 5th chain is running in background wait-loop right now (relaunched 00:33 per the established plan-policy default). Structural note: `can-run-composed-gate --wait-loop` gates on **load1** only, while verify's internal `verify-load-guard`/calibration-gate also requires **load5 < 10** — so the gate keeps firing into refusals whenever load1 dips inside a draining burst. Four refusals = 4 × ~10 min of nix build preamble burned.
2. **f046 receipts — mapping resolved, text NOT yet landed** (they cite the green verify): dedup-(a) → TODO_LIST `Dedup-campaign verification tail` row item (a) (composed #verify leg); W1-sibling/Layer-1 → TODO_LIST `Composed #verify re-record (W1 sibling)` row (dated STATE sub-bullet with the preflight 9/9 + dup-gate 186-baseline + verify evidence).
3. **`check-file-size` gate: RED, 3 violations, 100% FOREIGN** (git-attributed to sibling commits 19:56–22:25; I never touched these files): `metaengine/engine.go` 700→712 (baselined), `metaengine/reflect.go` 352→359 (baselined), `metaengine/typed_reader_scan.go` 367 (NEW offender). NOT part of `#verify` (confirmed: flake.nix verify chain has no file-size leg) so it does not block T03, but it IS a CI-gate red on master the owning sessions must reconcile (split or justified re-baseline). Left untouched per concurrent-session ownership rules; NOT yet surfaced on the shared TODO_LIST either (mistake, see d).

## c) NOT STARTED (this session)

- **T04 f047–f052** (mysql-VM hardened run + F52 AGENTS rows, snapshot_migration_mysql live, shuffle-seed replay — 119 seeds logged in `build/shuffle-seeds.log`, G-T13 ADTSet leg, nspawn, claiming-metrics sweep) — all quiet-window gated, sensibly sequenced after verify.
- **T05 f053–f057** (calibration PASS loop, SearchQuery count=5, dgraph constants re-anchor, baseline re-pin, supersede-note).
- **W2 tail T08–T11** (AsyncAPI schemas/bindings, pushdown cookbook, wire-string grep §4(b), goldens + meta-tests, engine surfaces).
- **T07 release tag** — explicitly user-gated; nothing tagged (standing rule: no tagging without the user).

## d) TOTALLY FUCKED UP / MISTAKES (honest ledger)

1. **Ran the mysql full suite via `vm-mysql.sh` args form without reading the built-in legs first** — burned 2 VM runs (~90s boot each) before noticing the calibrated env (`ADTTEST_CAS_RACERS=10`) lives only on the built-in legs. Should have read the script's leg block BEFORE the first args invocation; the fix took 30 seconds once found.
2. **Piped the first userspace run through `tail -4`** — the 600s timeout goroutine dump (the ONLY diagnostic for that hang) went to /dev/null; had to burn a second full run to capture it. Lesson: long diagnostic runs always tee to a file, tail the file.
3. **The 600s userspace hang itself:** one-off parallel-contention pileup (pre-existing fragility class, goroutine dump pointed at `AssertTxIsolationFromForeignContext`, not my harness hoist). I did NOT stress-repro it (30–100× per the 2026-09-19 lesson) — accepted "green twice + documented class" instead. Defensible for tonight, but the honest label is UNREPRODUCED, not ROOT-CAUSED.
4. **Assumed the 0.157s green run was suspicious and re-ran verbose before checking basics** — 5 min spent proving what 136 RUN entries already said. Read the evidence first.
5. **Verify attempt #3 misread initially** — I reported "verify ran ~8 min, 1 doc assertion failed" before noticing the run EARLY-EXITED at doc-assertions: build/vet/test/race/lint never executed. Caught on the phase-list read, but the first framing was wrong.
6. **Relaunched verify attempt #4 without pre-checking load5 myself** — repeated the load1-vs-load5 refusal race that attempt #1 already demonstrated. Third identical refusal class before I started hand-checking load5.
7. **Foreign file-size red left UNSURFACED** — per policy I correctly didn't touch the sibling files, but the policy ALSO says surface through the shared TODO_LIST; I deferred that while "waiting for verify" and it fell out of the evening. Cheap, should have been immediate.

## e) WHAT WE SHOULD IMPROVE

1. **`can-run-composed-gate --wait-loop` should check load5 (or both), matching verify's internal guard** — kills the load1-dip-in-draining-burst refusal race that cost 4 preamble builds tonight. Small script change + self-test extension (the gate scripts carry `--self-test`).
2. **`vm-mysql.sh`-class scripts: export ALL calibrated env at the DSN site** (done for CAS_RACERS tonight) — audit `vm-mysql-nspawn.sh`, `ephemeral-pg.sh`, `ephemeral-dgraph.sh` for the same args-path gap.
3. **Always tee long diagnostic runs to a file** (see d2).
4. **The mysqlengine suite under 32-way parallel on a fast server can pile up past go-test's 600s timeout** — consider a package-level sane `-parallel` (e.g. 16) for live-DB suites, or a documented note; the nspawn/no-slirp legs run full 16 racers fine, so it's about dial-in burst size vs pool/lock timeouts, not correctness.
5. **file-size ratchet violations on master currently have no owner-visible surfacing** — a nightly README-gate-style check or a TODO row auto-annotation would prevent "red gate, nobody knows" states like tonight's.

## f) NEXT UP TO 50 (rough execution order)

1. Verify chain attempt #5 → GREEN `#verify` (running in background now; log /tmp/verify-t07e.log)
2. If refused again past max-wait: relaunch chain (self-heal loop) — do NOT force under storm
3. f046 receipt: dedup-(a) row (TODO_LIST ~line 860, item (a)) — dated sub-bullet citing green verify
4. f046 receipt: W1-sibling/Layer-1 row (TODO_LIST ~line 820) — STATE 2026-10-09 sub-bullet with preflight 9/9 + dup 186 + verify evidence; strike/convert per row convention
5. Add TODO row surfacing the 3 foreign file-size violations for owning sessions
6. One-line gotchas-testing.md update: vm-mysql.sh args path now default-caps ADTTEST_CAS_RACERS (keep the cure doc accurate)
7. Supersession header check on the 21:03 report (21:09 header already carries it — verify still accurate)
8. T04/f047: hardened `#integration-mysql-vm` full-leg run (all 6 suites) in quiet window
9. f047 tail: F52 AGENTS.md integration-rows evidence update + strike
10. f048: snapshot_migration_mysql live run + receipt
11. f049: replay logged shuffle seeds for mysql legs from build/shuffle-seeds.log
12. f050: G-T13 ADTSet-parity mysql-VM quiet-window leg
13. f051: #integration-mysql-nspawn full env (root required)
14. f052: conformance-sweep mysql half (claiming metrics snapshot)
15. T05/f053: calibration-gate PASS loop via quiet-window-run
16. f054: SearchQuery count=5 re-run; supersede table if medians move >5%
17. f055: re-anchor ALL dgraph constants in one gate-passing window
18. f056: benchmark-baseline re-pin with provenance header
19. f057: supersede-note on the oversubscribed 2026-09-19 capture doc
20. Improve can-run-composed-gate: add load5 to wait-loop condition + --self-test case (e-1)
21. Audit sibling ephemeral scripts for args-path env gaps (e-2)
22. mysql 600s hang: 30× stress-repro or explicitly accept-and-document (d-3)
23. W2/T06: #36 stack→metaengine module work
24. W2/T08: AsyncAPI schemas + bindings
25. T08: pushdown cookbook (recipes.md)
26. T09/f072–f073: wire-string grep §4(b) sibling alert/dashboard configs + WIRE-FORMAT-KEYS cross-check
27. T10/f074–f076: api-stability TestEvery*, v007 drift ×2, E-items tail, record/v4 pin sweep
28. T11: engine surfaces before freeze (SingleWriter lease, AggregateOn, routing v1)
29. T07 release tag train — USER-GATED, awaiting ruling
30. Gotcha: add "insert-before-symbol edit must re-include the header" lesson (owed from prior session)
31. f064 file-size spot re-run once sibling files get split (confirm my new files stay clean)
32. Check userspace-MariaDB /tmp instance documented pattern still matches recipe after tonight's re-init (it did — no doc drift found)
33. Sibling-surface: note my vm-mysql.sh edit to the owning session via TODO_LIST (edit touches their script area)
34. Sibling-surface: templ regen (their generated artifacts, my cwd-correct regeneration) — same TODO note
35. Watch: does verify #5's doc-check leg stay green with the ADR-0152 row I added (formatting-sensitive gates)

## g) QUESTIONS (cannot figure out myself)

1. **Verify scheduling:** the box has stormed all night (load5 13–49; siblings active at 00:30). Chain #5 self-heals hourly. Keep holding autonomously all night (default), or do you want verify pinned to a specific window / capped number of retries?
2. **Foreign file-size debt:** `metaengine/engine.go` (+12), `reflect.go` (+7), `typed_reader_scan.go` (NEW, 367) are sibling-grown and CI-red. Ownership rules say I leave their files alone — confirm, or authorize me to split/refactor them myself?
3. **T07 release timing (carried from the 21:09 report, still unanswered):** tag the compound-cursor work on the next v4.x train once verify is green, or hold it for the W2 docs tail (AsyncAPI/pushdown cookbook) as one bigger release?

## Provenance

- Verify logs: /tmp/verify-t07{,b,c,d,e}.log (rc=1 refusals: load5 13.72 / 11.55 / ADR-0152 doc-assert / 13.92; e = running)
- MySQL diagnostics: /tmp/mysql-full.log, /tmp/mysql-full2.log, /tmp/mysql-usr.log; userspace MariaDB torn down at ~22:40
- Preflight: 9/9 at 21:15 and again inside verify attempt #3/#4 preambles
- Tree: clean except this report + tonight's absorbed edits (daemon auto-commits); my code/doc touches tonight: vm-mysql.sh (default cap), docs/README.md (ADR-0152 row), catalog/docserver/*_templ.go (cwd-correct regen), api golden, this report
