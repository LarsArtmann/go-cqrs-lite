# Status Report: Metaengine Universal Storage Substrate — T13–T17 completion, gate repair, Go 1.27 cutover

**Date:** 2026-09-19 12:12 CEST
**Session scope:** Execute the remaining metaengine-universal-storage-substrate plan
(T13 remainder, T14–T17), land the in-flight `queue/mysql` work, and fix every gate
the tree was red on. Companion discovery: a **half-done Go 1.27 migration** left by the
previous session was breaking every workspace-mode compile; completed it.

**Plan:** [`docs/planning/2026-09-18_16-17_SUPERB-metaengine-universal-storage-substrate.md`](../planning/2026-09-18_16-17_SUPERB-metaengine-universal-storage-substrate.md)

---

## a) FULLY DONE (verified green this session)

### queue/mysql engine surface (family completion of plan T09)

- `queue/mysql/engine.go` + `register.go` + `engine_test.go`: `NewEngine(ctx, dsn)` /
  `NewEngineFromDB`, `"queue-mysql"` metaengine driver registration, DueClaimer +
  FactSink + DedupStore via the ONE claimkit runtime over the queue's own database —
  the exact `queue/sqlite` + `queue/postgres` pattern (incl. the `art-dupl:accept`
  registration directive, AGENTS contract 19). go.mod gains claiming + metaengine
  (sibling replaces, budget 2→4 matching siblings). `doc.go` stale scope-note removed
  (the deferred wiring is no longer deferred — claimkit's MySQL dialect landed).
- Engine conformance test (live-gated `MYSQL_TEST_DSN`) + an un-gated
  empty-DSN-rejection registry test.
- **Live validation vs QEMU MySQL (round 2/4):** EVERY deterministic subtest green —
  ClaimInsertIdempotent, NotBeforeGating/DueOrdering, LeaseFence/ExpiryReclaim,
  RenewLeaseOwnerFenced, ClaimDeleteIdempotent, DeleteIfDueEpochGuard, ClaimLimit,
  ConcurrentClaimersDisjoint, plus the full queue conformance, mysqlengine ADT
  matrix, layouts, planned ops, reset tests incl. my new claimkit-reset assertions.
  Unit + `-race` green standalone.
- queue/mysql added as a leg to BOTH `scripts/vm-mysql.sh` and
  `vm-mysql-nspawn.sh` (full-run mode: `cd module; GOWORK=off`).

### T13 remainder + T15: structured capability refusals (ADR-0142 "never silence")

- `metaengine.EngineProfile.RefusedADTs map[ADT]string` + `RefusesADT()` — refusal
  is now DATA rendered by Doctor + the capability audit, not a code comment.
- Capability audit extended: `ADTDueClaim`/`ADTDedup` join `adtContracts` (rules 1/2
  now fire for implemented-but-undeclared and declared-but-unimplemented engines) and
  new **rule 4**: refusals must be disjoint from `Supports`; every engine must
  support OR refuse the write-side ADTs — the universality rule (ADR-0123 §9) made
  mechanical. Audit table renders `REFUSED: <reason>` rows.
- Engines that genuinely cannot, now say so:
  - **dgraph** — DQL upserts cannot atomically fence concurrent claimers / no
    CAS-with-TTL;
  - **bigtable** — no atomic arbitrary-value RMW (MapUpdate) or collection scan yet
    (native paths exist: CheckAndMutate CAS + ReadRows — revisit note recorded);
    plus bigtableengine gained its missing `TestCapabilityConformance` gate;
  - **iroh `Replicated` wrapper** — Profile() strips inherited write-side
    Supports/Degraded entries and refuses them (lease/dedup invisible to peers until
    convergence = silent divergence); passthrough policy doc updated with the
    DueClaimer/DedupStore/FactSink not-forwarded decision.

### T16: ADR-0136 reset ladder covers the claimkit collections

- sqlite/postgres/mysql/duckdb `ResetEngine` now clear `meta_due_claims` +
  `meta_dedup` (timers/dedup are derived → reset + replay). Previously a reset left
  live leases behind — a replay could be FENCED by pre-reset claims (real bug).
- Pinned per engine by tests asserting claims/dedup are gone after reset while
  journal positions keep advancing (sqlite version reads `sqlite_sequence` directly).
- Turso inherits the fix through its sqliteengine embedding.

### Gate repair (all verified green)

- **api-stability golden** regenerated for queue/mysql engine surface +
  `RefusedADTs`/`RefusesADT` (7400 → 7415 exports, incl. the concurrent session's
  `adttest.AssertFactSink`); meta-tests `TestEvery*` green after fixing:
  - `scheduling/engine` missing from `check-module-layers.sh` LAYER+DEP_BUDGET maps
    (it silently skipped layer enforcement);
  - five modules' go.mod/go.sum not tidy (projectionadapter, sqliteengine,
    tursoengine, queue/sqlite, scheduling/engine);
  - `metaengine/tursoengine` standalone build broken — missing `claiming` sibling
    replace (replaces do NOT cascade through the sqliteengine replace; the file's own
    comment documents the rule).
- **Layer lattice** honest again: `claiming` L4→L2 (zero-runtime statement library,
  consumed downward by metaengine claimkit per ADR-0142 — old L4 predated the ADR);
  `idempotency/sqlstore` L2→L4 (SQL stores; `NewFromEngine` legitimately depends on
  metaengine). Dep budgets updated for ADR-0142-sanctioned deps: metaengine 6,
  queue/sqlite + queue/postgres + queue/mysql 4, catalog 7 (icons submodule).
- **file-size ratchet** back to green (was red on 7 files): my engine.go growth
  fixed by moving `RefusesADT` + `String()` to types.go (700→676, shrinking allowed);
  pre-existing growths fixed by extracting `Profile()` to `profile.go` in
  duckdb/pebble/pg engines and condensing/extracting in system/config_types.go
  (425→409) + constructor.go (369→357, checkpoint resolution → checkpoint.go);
  480-line NEW offender `adttest/claim_conformance.go` split → claim + dedup files.
- **`nix fmt`**: 0 changes needed. **doc-check**: 1154 references valid. **changelog
  symbols gate**: 140 citations honest.

### Go 1.27 migration completed (the big one)

- Discovered: root `go.mod` + `go.work` said `go 1.27.1`, nix toolchain was
  `go_1_27`, but all 94 modules still said `go 1.26.x` — a HALF-DONE migration
  (flake bumped 2026-09-19 00:54 by the previous session). Consequence (reproduced
  with clean caches): **every workspace-mode compile under the 1.27 toolchain
  failed** — 55+ `json.Marshal/Unmarshal requires go1.27 or later` errors, because
  graduated `encoding/json/v2` gates those symbols to language ≥ 1.27.
- Fix: swept all 94 modules' go directives to `go 1.27.1`. Verified: full
  workspace build + workspace vet across metaengine/queue/scheduling/system/
  event/command/query/decider/storage/claiming/stack/middleware/... trees green
  under go 1.27.1; per-module suites re-run green (metaengine core 42s, adttest,
  claimkit, sqliteengine, iroh, dgraph, bigtable, scheduling/engine,
  idempotency/sqlstore, system, queue/mysql `-race`, duckdb cgo reset, pgengine
  95s full). Host go 1.26.7 works via `GOTOOLCHAIN=auto`.
- AGENTS.md updated (Language → 1.27.1; contract 10 rewritten for post-graduation
  reality).

### Docs (T17 rows)

- modules.md: `scheduling/engine` row added (API verified against source);
  module-map + FEATURES rows for `scheduling/engine` and `queue/mysql` (with
  driver-registration note); queue/mysql row in the uncommitted diff completed.
- CHANGELOG: four new Unreleased sections (ADR-0142 tail, reset-ladder fix,
  gates/budgets/1.27 sweep). TODO_LIST: T01–T17 marked done with evidence; open
  remainders itemized.

## b) PARTIALLY DONE

- **T17 recipes §2.x**: module rows shipped, but the recipes.md engine-backed
  timers/queue/dedup recipe block + recipes_catalog classification + doc-check fence
  NOT written (left as explicit TODO_LIST item). Deliberate: recipes are
  compile-gated and I prioritized gates + the 1.27 cutover.
- **Live MySQL validation**: deterministic semantics fully green; the
  `ConcurrentCASExactlyOneWinner` subtest (16 racing goroutines) hits the documented
  QEMU slirp connection-reset instability in EVERY round (2/3/4) — always
  `invalid connection`, never a semantic failure. Needs the nspawn leg (root) or a
  racer cap like `MYSQL_TEST_CONCURRENCY=10`.
- **`nix run .#verify` not run end-to-end** (exclusivity rule: never concurrent
  with the MySQL VM runs; the VM was running for most of the session). All its
  constituent gates were run individually and are green.

## c) NOT STARTED (this session; tracked in TODO_LIST)

- T18–T23 (benchmarks/load-sweep regate, v5 fold of capabilities into Engine,
  duplicate-stack deletion, release train, example/taskmanager, go-taskqueue
  semantic-diff probe) — all v5-gated by design.
- Dropping the now-noop `-tags "goexperiment.jsonv2"` from scripts/CI.
- Re-baselining load-sweep benchmarks under the 1.27 toolchain.
- Wiring `adttest.AssertFactSink` into engine test suites (the concurrent session
  authored the suite; sqliteengine does not call it yet).

## d) TOTALLY FUCKED UP (found broken, now fixed)

1. **Half-done Go 1.27 migration** — broke ALL workspace-mode compiles under the
   nix toolchain (proven with clean-cache vet: 55+ errors in metaengine+queue alone).
   Completed the sweep; everything green.
2. **`go.work`/root `go.mod` bumped to 1.27.1 by the auto-commit daemon mid-session
   (10:39)** — I initially reverted to 1.26.7, then discovered root go.mod was ALSO
   bumped and that the intended direction was forward; restored 1.27.1 and completed
   the module sweep instead. Lesson: check WHO/WHAT moved a directive before
   reverting it (the daemon absorbs side-effect writes).
3. **Reset ladder gap (ADR-0136 violation)** — engine resets left live
   `meta_due_claims`/`meta_dedup` behind. Fixed + pinned per engine.
4. **Layer gate violations committed by the daemon** (metaengine→claiming,
   idempotency/sqlstore→metaengine) + 4 dep-budget breaches + 7 file-size ratchet
   violations — all from auto-committed previous-session work that never ran the
   gates. Fixed honestly (reclassification + budget reviews + file splits).
5. **tursoengine standalone build broken** (missing cascading replace) — caught by
   the go.sum-tidy meta-test; fixed.

## e) WHAT WE SHOULD IMPROVE (observations)

1. **The auto-commit daemon commits gate-red work.** Three separate sessions' worth
   of breakage (layer violations, budgets, file-size, half-1.27) landed as `chore:
   auto-commit` because the daemon doesn't run gates. The gates catch it LATER at
   higher cost. Improvement: daemon pre-commit hook that runs check-module-layers +
   check-file-size (fast, <5s) before committing.
2. **A concurrent session was active in this repo during mine** (factsink_conformance.go
   appeared mid-edit at 11:54, broke my VM compile once, then completed itself;
   api golden moved 7413→7415 under me). Coordinate: only one session should own a
   module tree at a time, or the daemon should note in-flight files.
3. **QEMU slirp cannot carry 16-way concurrent connection bursts** — the CAS race
   conformance test will forever flake there. Either cap racers via env (the
   idempotency precedent: `MYSQL_TEST_CONCURRENCY=10`) or make the nspawn leg
   runnable without root.
4. **Workspace-mode vs standalone drift**: `go test ./x/...` from repo root (manual
   VM mode) was broken while `cd x && GOWORK=off go test` was green — the gowork
   decision table should grow a row for "VM script manual mode" (it cd's to
   REPO_ROOT). I patched around it with `-p 1`, not the table.
5. **Doc debt follows shipped code**: scheduling/engine shipped 2026-09-19 with ZERO
   doc rows (modules.md/module-map/FEATURES) until this session. The
   "Change an Exported Symbol" procedure should include "new module ⇒ add rows to
   all three doc files".
6. `check-module-layers.sh` exit code was masked by `| tail` in my first run
   (pipeline masking gotcha — bit me exactly as documented).

## f) NEXT (up to 50, prioritized)

**P1 — close this session's loops**

1. Wire `adttest.AssertFactSink` into sqliteengine/pgengine/mysqlengine/duckdbengine/tursoengine + queue engines' test suites (T14c completion — coordinate with the concurrent session that authored it).
2. Run `nix run .#verify` end-to-end on a quiet tree (the exclusive full gate).
3. Cap `ConcurrentCASExactlyOneWinner` racers via `MYSQL_TEST_CONCURRENCY`-style env, or teach claim_conformance to skip the race under QEMU (detect via env the VM script sets).
4. Validate queue/mysql + mysqlengine against the nspawn leg (`nix run .#integration-mysql-nspawn`, needs root) — removes slirp from the equation.
5. Write the recipes.md §2.x "engine-backed timers/queue/dedup" recipe + recipes_catalog classification + doc-check green (T17 remainder).
6. Re-run `nix run .#test-integration` (SQLite+Pebble+bbolt+DuckDB+PG+MySQL+Dgraph) after the 1.27 sweep — the sweep changed every go.mod.
7. `nix run .#load-sweep` + benchmark-regression gate under 1.27; re-baseline if the toolchain shifted medians (documented TODO item).
8. Check CI (ci.yml) actually ran green post-sweep; fix any 1.27-vs-gocache interactions in Nix.

**P2 — 1.27 follow-through**
9. Drop `-tags "goexperiment.jsonv2"` everywhere (scripts, flake, AGENTS, cmd tools) in ONE coordinated sweep; keep GOEXPERIMENT env until scripts updated.
10. Sweep `GOEXPERIMENT=jsonv2` env exports from flake apps + vm scripts after #9.
11. Update `docs/agents/gowork-modes.md` for post-graduation reality (tag is a no-op; version floor is now 1.27.1; GOTOOLCHAIN=auto for old hosts).
12. gopls/LSP: go.work requires 1.27.1 while gopls ran a 1.26.7 toolchain — pin the LSP toolchain or accept the noise (115 errors all from this).
13. Consider `toolchain go1.27.1` directives vs bare `go 1.27.1` (uniformity review).
14. go-error-family / go-codec / sibling external repos: check they build under consumer 1.27 (GOTOOLCHAIN auto handles it, but pins may lag).

**P3 — ADR-0142 hardening**
15. Doctor: render a dedicated "--- Refused ADTs ---" section (today refusals appear inside the capability table only).
16. ExplainPlan/SCREAM plan-time diagnostic when a DeploymentConfig names a refused engine for timers/dedup (fail-loud at config load, not at first claim).
17. `system.TimerEngine()`: consider honoring a `RefusedADTs` check at wiring time with the reason in the error.
18. Capability audit rule 4 for `FactSink` (no ADT constant today — either mint `ADTFactSink` or document why it stays capability-only).
19. Conformance: lease-expiry reclaim under clock skew (fake-clock injection), cross-collection keyspace isolation test for claimkit MySQL (PG has one).
20. bigtable: implement MapUpdate (CheckAndMutate CAS loop) + ScanBackend (ReadRows prefix) → embed the Map runtimes, replace the refusal with DegradedADTs.
21. dgraph: re-evaluate refusal if Dgraph gains upsert-with-CAS primitives.
22. iroh: consider a leader-elected claim path (documented refusal reason points the way) — or leave refused forever, but write the ADR note.
23. `EngineProfile.String()`: include refused count in the rendered suffix.
24. claimkit: `ClaimFactsList` pagination/limits (unbounded list today per the factLister interface).
25. Benchmarks: claim/dedup micro-benches vs direct-SQL baseline (T18a).

**P4 — queue/mysql production readiness**
26. Tag wave: `claiming/v4.0.0`, `queue` family v4.0.0 (TODO_LIST item, blocked on clean tree — tree is 144 files dirty now, mostly the 1.27 sweep).
27. Strip sibling replaces at cut time (tag-release.sh) — verify it handles the new tursoengine claiming replace.
28. Deadlock-retry: add a metric/log line when `claimDeadlockRetries` fires (operators should see InnoDB deadlock rates).
29. queue/mysql `HealthCheck` implementation (engine interface parity with mysqlengine).
30. `example/taskmanager` on the engine-backed queue (T22) once tagged.

**P5 — repo hygiene**
31. Daemon gate hook (see e1).
32. Single-writer session protocol or daemon in-flight marker (see e2).
33. module-map: row the `metaengine/*engine/profile.go` split (no doc change needed, but the map cites file counts nowhere — skip if unchanged).
34. FEATURES: ADR-0142 capability matrix section (engines × DueClaim/Dedup/FactSink: native/degraded/refused).
35. FAQ: "why can't timers live on dgraph?" → point at RefusedADTs + Doctor.
36. docs/error-taxonomy.md: sync if the refusal messages minted new codes (they didn't — verify).
37. CONTRIBUTING: document the new-module checklist (go.work + flake testModules + api-stability modules slice + layer/budget maps + three doc rows + VM legs if SQL).
38. gowork-modes.md: add the VM-manual-mode row (see e4).
39. Consider making `check-module-layers.sh` self-test (`--self-test`) like calibration-gate.sh (gate-script convention).
40. api-stability: assert `TestEveryModuleGoSumIsTidy` in CI legs that absorb daemon commits (it caught 3 real breaks today).

## g) QUESTIONS (cannot figure out myself)

1. **Concurrent session policy:** another agent session was actively editing this
   repo during mine (factsink_conformance.go appeared/completed mid-run; api golden
   moved under me). Should I treat untracked in-flight files as owned-by-others and
   NEVER touch/complete them (I completed nothing of theirs — I only fixed compile
   order effects), or is completing half-written files fair game when they break
   shared builds?
2. **1.27 direction confirmation:** I completed the forward migration (94 modules →
   `go 1.27.1`) based on the flake/go.work/root bumps being deliberate. If those
   bumps were accidental, the alternative (pin everything back to 1.26.7 +
   go_1_26 flake) is still possible — say the word. Forward looked intended; say if not.
3. **MySQL live validation appetite:** the QEMU slirp flake blocks a 100% green
   live run for the 16-racer CAS test. Do you want (a) the racer-cap env approach
   (fast, keeps QEMU), (b) a one-time root-run of the nspawn leg for a clean
   signal, or (c) accept the current evidence (all deterministic semantics green,
   race flake is pure infra)?
