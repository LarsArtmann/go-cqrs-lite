# Status: ADR-0142 Universal Storage Substrate — T13 completion, T14/T15/T16 closed, duplication gate mid-flight

**Date:** 2026-09-19 12:12 CEST
**Session scope:** resumed at the T13 MySQL pause point (2 known bugs); executed through T13 close-out + T14c + T16 gap + T15 verification; stopped mid-`#check-duplication` on owner order.
**Plan:** [`docs/planning/2026-09-18_16-17_SUPERB-metaengine-universal-storage-substrate.md`](../planning/2026-09-18_16-17_SUPERB-metaengine-universal-storage-substrate.md)
**Prior report:** [`2026-09-19_10-04_metaengine-universal-storage-substrate-t11-t13.md`](2026-09-19_10-04_metaengine-universal-storage-substrate-t11-t13.md)

---

## a) FULLY DONE (this session, all verified green)

1. **`claiming.StampLeaseMySQLStmt` arg-order bug FIXED** — args were built ids-first while the statement places SET columns first; every placeholder misaligned by two (MySQL Error 1292: key string into `lease_until` DATETIME). Reordered to match placeholder order + **`TestStampLeaseMySQLStmtArgsAlign`** pin test (placeholder count AND per-position order — the misalignment class can't recur silently). `claiming` suite green.
2. **`metaengine/claimkit/dedup.go:126` bare `key` FIXED** — the raced-out `idColumn(d.dialect)` fix re-applied, grep-verified (only the SQLite-only test harness keeps a bare `key`, which is legal there).
3. **THIRD MySQL bug found + FIXED (not in the resume notes):** the `SELECT..FOR UPDATE` two-step dedup deadlocks under concurrency (Error 1213) — InnoDB gap locks on the absent row + concurrent INSERTs. Rewrote `checkAndRecordMySQL` lock-free with identical single-winner semantics: plain-read fast path (live window → seen, no locks), `INSERT IGNORE` + RowsAffected for the absent-row race (exactly one racer's insert lands), conditional `UPDATE .. WHERE expires_at <= now` CAS for lapsed-window takeover. Struct doc updated. **mysqlengine due-claims conformance ×3 under `-race` vs live MariaDB: green. Full mysqlengine suite under `-race`: green.**
4. **Consumer regression sweep green:** claiming, metaengine core+claimkit, sqliteengine, mysqlengine (live), pgengine, duckdbengine, bboltengine, badgerengine, pebbleengine, scheduling. (bbolt "failure" was the documented 509–1145s soak vs my 300s timeout — `SOAK_SKIP_BOLT=1` run green.)
5. **sqliteengine flake FIXED:** `TestSQLiteEngine_ConcurrentStreamReadVsAppendExpected` recorded a 5s-deadline landing mid-`StreamRead`/`StreamAppendExpected` as a failure (writers legitimately outlive the budget under `-race`). In-flight `ctx.Err()` shutdowns are no longer failures. ×3 + full suite under `-race` green.
6. **T13e turso:** NEW `metaengine/tursoengine/dueclaim_test.go` — full `AssertDueClaimer` + `AssertDedupStore` over a **local libSQL FILE DSN** (the real tursogo driver path, previously never exercised) + capability-surface test. Green.
7. **T13f refusal notes:** capability-refusal README sections for **dgraphengine, irohengine, bigtableengine** + **"Realized capability matrix (2026-09-19, T13 complete)"** section in ADR-0142 (engine × DueClaim/Dedup = native/degraded/REFUSED with reasons, pinned by `capability_audit`).
8. **T14c journal-never-disagrees conformance:** NEW exported `claimkit.Claims.ClaimFactsList` (fact-journal read side, seq-ordered) + NEW `adttest/factsink_conformance.go` `AssertFactSink` — facts ride `ClaimDueFacts` exactly for claimed items (NotBefore-gated item gets zero facts); matching-epoch `ClaimDeleteFacts` lands delete+fact in one tx; stale-epoch records NOTHING. Wired into sqlite + postgres + mysql + duckdb due-claim suites. Green live on sqlite/mysql/duckdb (pg leg compiles, DSN-gated skip locally).
9. **T16 gap pinned:** NEW `TestResetEngine_FactsSurviveReset` (sqliteengine) — reset clears claims/dedup but NEVER the fact journal, and fact positions keep advancing (ADR-0142 §5 journal rung).
10. **T15 verified COMPLETE by evidence** (no new code needed): `capabilityDoctorSection` renders per-engine conformance incl. refusals in Doctor, `explainCapabilityWarnings` renders plan-time WARNs, `degradedADTRule` is ADT-generic, capability audit enforces support-or-refuse (rule 4) with `TestCapabilityAudit_RefusalRules`. The `RefusedADTs` API survived the concurrent engine.go refactor.
11. **Gates green:** `nix fmt` (26 files), api golden regen (7413 → **7415** exports: `ClaimFactsList`, `AssertFactSink`), api-stability `TestEvery`, symbols gate (**142** citations honest), `#check-file-size` (58 baselined, no growth), `#check-arch`. CHANGELOG: two new Unreleased sections (FactSink conformance + turso pin; MySQL live-concurrency fixes).
12. **Two REAL deduplications (from the 18 new clone groups):** shared `duckWriteLock` embedded struct in claimkit (was twin `lockWriter` methods on Claims/Dedup) and shared `mapRuntimeBackends` capability-assertion helper in metaengine (was twin constructors). claimkit + metaengine tests green after.

## b) PARTIALLY DONE

- **`#check-duplication` — STOPPED MID-FLIGHT (exact pause point).** 18 → ~13 new clone groups remain after my 2 real dedups. Remaining classes, all candidates for `//art-dupl:accept`: queue engine mirrors (mysql↔sqlite↔postgres: cancel/enqueue/lifecycle/open/engine/claim/details, ~9 groups), queue/conformance scenario boilerplate (`openEnv` + Complete-with-token, 4 small groups), engine dueclaim wiring mirrors (duckdb/mysql/pg/sqlite `if err != nil` + duckdb/pg DDL loop, 2 groups). Annotating is ITERATIVE (each round unmasks more).
- **T13 docs tail:** code + ADR matrix done; the per-module doc rows (modules.md, module-map) are T17 scope, not started.

## c) NOT STARTED

- **T17 docs:** SKILL recipes §2.x (engine-backed timers/queue/dedup) + `recipes_catalog` classification (compile-gated!), modules.md rows (claimkit, scheduling/engine, queue engines, duckdb/mysql/turso claim status), FEATURES, module-map, FAQ v5 note, doc-check zero-warning run.
- **Gates not yet run:** `#check-error-taxonomy` (no new error codes this session — expected green), doc-check, `#verify` (exclusive), `#verify-ci`, `#load-sweep`, integration legs (pg for the live FactSink leg, `#test-integration`).
- **TODO_LIST harvest:** mark T01–T13 done with CHANGELOG cross-refs.
- **T18** benches (claim/dedup micro vs direct-SQL baseline) + regression-gate baseline.
- **T22** example/taskmanager on engine-backed queue; **T23** go-taskqueue semantic-diff memo.
- **T19–T21** (v5 fold / delete dup stacks / release train) — v5-gated by plan.

## d) TOTALLY FUCKED UP

- **Nothing irreversible.** Three self-caught transient mistakes, all fixed within minutes: (1) ADR-0142 matrix insert briefly clobbered the `## Consequences` heading (re-anchored + restored; heading structure verified); (2) my new FactSink suite assumed `FactSink` embeds `DueClaimer` (it doesn't — asserted both) and initially asserted a "claimed" fact that a plain `ClaimDue` never records; (3) used interface _assignment_ instead of _assertion_ for the `ClaimFactsList` reader in the reset test (compile error, fixed).
- **Process miss (not code):** I made **zero explicit commits** this session — everything is being absorbed by the daemon into `chore: auto-commit` blobs, losing authored per-task history. The CHANGELOG sections are the only narrative record.

## e) WHAT WE SHOULD IMPROVE

1. **Commit discipline vs daemon:** the plan said "commit fast vs the auto-commit daemon"; I didn't. Every task slice should end with an explicit authored commit.
2. **Concurrent-session coordination:** another actor bumped ALL go.mod/go.work to go 1.27.1 (fixes the old toolchain split — good) and mid-edited `metaengine/engine.go` (transient unused-import break that cost a rebuild cycle). Before long test runs, check tree stability; before editing shared files, re-read at edit time.
3. **The summary under-reported reality:** the CHANGELOG showed T13/T15/T16 machinery (RefusedADTs, reset ladder, queue/mysql) largely done by absorbed prior work — I spent a few cycles verifying instead of re-doing, which is correct, but the status-report-to-truth drift cost one full evidence sweep. Keep reports current or mark them stale.
4. **Live-PG FactSink leg:** invariant is live-pinned on sqlite/mysql/duckdb only; run `#integration-pg` before calling T14 shipped.
5. **Clone annotation economics:** 4+ iterative art-dupl rounds for ~13 groups; consider annotating whole mirror classes per file in one pass (directive on first line of each cloned region).

## f) NEXT — up to 50, in priority order

1. Finish `#check-duplication`: `//art-dupl:accept` the queue engine mirrors (cross-module dialect isolation, contract 19 class).
2. …same for queue/conformance scenario boilerplate groups.
3. …same for engine dueclaim.go wiring mirrors (duckdb/mysql/pg/sqlite).
4. …same for duckdb/pg DDL-loop group.
5. Re-run `#check-duplication` to 0 new groups (iterate as needed).
6. `nix fmt` + rebuild affected modules after annotations.
7. Commit the T13/T14/T16 slice with an authored message.
8. Run `#check-error-taxonomy`.
9. Run doc-check (`cmd/doc-check`) zero-warning.
10. T17a: recipes.md §2.x — engine-backed timers via `scheduling/engine`.
11. T17a: recipes — engine-backed queue (`queue/sqlite|postgres|mysql` as drivers).
12. T17a: recipes — engine-backed dedup (idempotency facades).
13. T17b: classify every new recipes fence in `recipes_catalog*.go` (compile-gated).
14. T17: modules.md rows for claimkit + scheduling/engine + queue engines + duckdb/mysql/turso claim status.
15. T17: module-map.md internal rows.
16. T17: FEATURES.md maturity-matrix update.
17. T17: FAQ — "which engines host timers/dedup/tasks?" + refusal routing answer.
18. SKILL.md touch-up if recipes cross-reference it.
19. TODO_LIST harvest: T01–T13 done with CHANGELOG cross-refs.
20. `#integration-pg` — live Postgres run incl. the FactSink suite.
21. Re-run full mysqlengine + queue/mysql conformance vs MariaDB once more after all refactors (last green was before the duckWriteLock refactor — mysql uses claimkit).
22. Kill the ephemeral MariaDB (or decide to keep — see question 3).
23. T18a: claim micro-bench (ClaimInsert/ClaimDue/RenewLease) vs direct-SQL baseline, sqlite.
24. T18a: dedup micro-bench (CheckAndRecord) vs direct-SQL baseline, sqlite.
25. T18a: same pair on pg (integration leg).
26. T18b: `#load-sweep` (`-run 'Latency|Timer|Deadline'` under soakers).
27. T18: benchmark-regression gate baseline regen if warranted.
28. T22a: example/taskmanager switched onto engine-backed queue.
29. T22b: taskmanager README + end-to-end demo run.
30. T23a: go-taskqueue semantic-diff memo (ADT vs production contract, P5 input).
31. Decide + document v5 items (T19–T21) as plan annotations ("gated on v5 train") so the plan closes honestly.
32. Full `#verify` (exclusive run, no concurrent suites).
33. `#verify-ci` (GOWORK=off per-module matrix).
34. `#check-coverage` drift gate.
35. `#vulncheck`.
36. Review + land the still-uncommitted queue/mysql local-RTT-constants change (not authored by me — see question 3/coordination).
37. Re-verify api_surface.txt is committed-current after annotations (daemon races).
38. Sweep for stray `M` working-tree files older than this session and reconcile.
39. Confirm the plan-doc verification checklist items per tier (integration legs lines) are all ticked.
40. Status-report the T17–T18 slice when done (same discipline).

## g) QUESTIONS (cannot figure out myself)

1. **Commit strategy for this session's work:** everything so far landed as daemon `chore:` commits. Do you want me to write authored commits per remaining task slice from here on (and optionally a consolidated authored commit for today's already-absorbed work by re-stating it in one message-level commit on top), or is daemon-absorbed history acceptable and the CHANGELOG is the record?
2. **Clone-gate disposition:** remaining ~13 groups are the sanctioned "intentional cross-module mirror" class (queue dialect trio + engine wiring). Annotate each region with `//art-dupl:accept` (contract 14 preference, ~4 iterative rounds), or re-pin the baseline once as a structural shift (a whole engine family landed)? Both are sanctioned; they leave different review trails.
3. **Ephemeral MariaDB (port 13306, /tmp/mariadb-cqrs.wTOOUm):** kill now, or keep alive for the pending queue/mysql + final mysqlengine re-runs (restart costs ~30s)?

## Environment notes carried forward

- **Go 1.27.1 everywhere now** (external fix-forward landed mid-session; `GOTOOLCHAIN=auto` on host toolchains < 1.27).
- Per-module testing still via `cd <mod> && GOWORK=off go test -tags "goexperiment.jsonv2" [-race] -count=N ./...` + cache env chain (`GOCACHE=/home/lars/projects/.gocache-disk GOMODCACHE=/tmp/gomod-verify GOPATH=/tmp/gopath-verify GOTOOLCHAIN=auto GOTMPDIR/TMPDIR=/home/lars/projects/.gotmp`).
- MariaDB DSN: `MYSQL_TEST_DSN="root@tcp(127.0.0.1:13306)/cqrs_test?parseTime=true"` (pid 2179283).
- LSP diagnostics in this environment still show the go.work version error noise on file views — ignorable; CLI builds are the truth (now aligned at 1.27.1).
