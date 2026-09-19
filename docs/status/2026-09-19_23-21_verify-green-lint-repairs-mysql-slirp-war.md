# Status Report: Verify GREEN (rc=0), Lint Repairs Under the Concurrent Sessions, and the MySQL-Leg Slirp War

**Date:** 2026-09-19 23:21 CEST
**Session scope:** Execute the 15:37 continuation plan's remaining tail — F-list lint debt, `#verify` green, M4 integration legs, T18b load-sweep — on a tree simultaneously reshaped by THREE other active sessions (lint-debt takeover, goal-shaped-app/G-T23, 92-tag release train) plus the auto-commit daemon.

---

## a) FULLY DONE (verified this session)

1. **Mission reconciliation** — the F01–F29 lint-debt list from my 16:53 baseline was executed by the 16:51→18:05 takeover session and committed as `777507821` ("clear repo-wide lint debt to zero"). I re-verified their claims rather than trusting them — which paid off (see a/2).
2. **Repaired the regressed `.golangci.yml`** — the working tree held a BuildFlow-auto-configure mangle (incident #9 of the documented series: jsonv2 tag re-added, depguard allow-list deleted, go 1.26.7) directly contradicting the curated HEAD state. Restored to HEAD; `#check-lint-config` green (allow-list verified, exhaustruct canaries pass). Snapshot kept at `/tmp/golangci-regressed-snapshot-18-39.yml`.
3. **Found and fixed 7 broken nolint placements the takeover session's sweep missed** — HEAD itself was lint-red: trailing `) //nolint:sqlclosecheck` on multi-line calls (findings report at the OPENING line; the closing-line directive suppresses nothing AND nolintlint flags it unused) in `queue/sqlite/{facts,reads}.go`, missing directives in `queue/mysql/{facts,reads}.go`, and signature-closing-line maintidx/tparallel directives in `metaengine/adttest/claim_conformance{,_test}.go`. All converted to the proven standalone-above / func-line forms. Also resolved the standing `queue/conformance rc=4` mystery: it is a PACKAGE inside the queue module (no go.mod) — my earlier baseline loop linted it as a module and typecheck-failed; the takeover sweep covered it correctly via queue/.
4. **Full lint gate re-baseline: 85/85 modules, 0 findings** (`nix run .#lint`, rc=0) — first honest repo-wide proof since the takeover logs proved unreliable.
5. **`nix run .#verify` rc=0 — FULLY GREEN** (verify7, ~20:35–21:00) after fixing five distinct blockers across six failed runs (details in d/1):
   - templ codegen drift: 5 `catalog/docserver/*_templ.go` regenerated from the WRONG cwd (FileName paths baked in); fixed by regenerating from `catalog/docserver/` per the `check-templ-paths.sh` tripwire contract.
   - `metaengine/probe_warn_test.go` DATA RACE: raw `bytes.Buffer` installed as the process-global slog sink while parallel tests log through it → mutex-locked `lockedBuffer` sink.
   - `TestREADMEClaim_ModuleCountFloor`: meta-test still pinned the "80+ modules" claim after README moved to "90+ independently-versioned modules" (95 go.mods) — updated pin + floor.
   - `benchkit TestCompare`: 60s deadline vs ~36s standalone race time (only 1.7× headroom; blew up under full-suite race load; one re-run took 86s) → 150s budget with rationale comment.
   - `TestSystem_Drain_ContextExpired`: 200ms drainer vs 50ms budget — under scheduler stalls >150ms BOTH select cases are ready at entry and Go's random pick returns nil ~50% of the time → margins widened to 2s (runtime unchanged; ctx still aborts at ~50ms).
6. **Post-verify gates all individually green** on the fixed tree: check-bench-gate, check-coverage, check-api-stability, check-error-taxonomy, doc-check (1421 refs), check-templ, check-lint-config, check-duplication, check-turso-version.
7. **M4 PostgreSQL leg: rc=0** (`#integration-pg`, ephemeral, 21:04–21:12).
8. **MySQL leg — partial but real wins** (full state in b):
   - Diagnosed and killed a **leaked orphan QEMU** (from a crashed nixos-test-driver whose QMP socket reset at startup) that was poisoning port 33070 for every subsequent leg run.
   - Tidied `stack/mysql` (+5 more modules) go.sums twice — the 92-tag release train's pin-sweeps landed mid-leg with incomplete go.sums (the documented "pins-without-tidy" gotcha, now observed live).
   - `vm-mysql.sh`: sqlstore block now stubs `POSTGRES_TEST_DSN` so the package's `pgtestcontainer.TestMain` stops unconditionally booting a postgres container (+ryuk, ~80s Docker churn) that raced the QEMU slirp forward.
   - `idempotency/sqlstore/integration_helpers_test.go`: AtomicClaim contenders now retry `errorfamily.IsRetryable` failures (bounded ×5) — the store classifies connection resets as Transient; the test boundary now honors that contract. sqlstore leg leg-module went red→green.
   - `metaengine/adttest/dedup_conformance.go`: CAS racer got the same bounded retry (driver-agnostic; invariant-preserving: committed-then-reset attempts surface as `seen=true` on retry). CAS test went red→green.
   - With those fixes + a clean QEMU slate: stack/mysql ✅, idempotency/sqlstore ✅, and mysqlengine's graph/CTE/dueclaim suites all passing — until the NEXT un-hardened suite hit the same slirp RST (see b).
9. **Ephemeral native MariaDB pivot started**: datadir initialized, `mysqld` alive on 127.0.0.1:13306 (nixpkgs mariadb, same package the VM/nspawn legs use) — the nspawn-leg approach without the root requirement; all mysql test suites key off `MYSQL_TEST_DSN` so the full leg can run against it.
10. All fixes verified: per-module lint re-runs (queue/*, metaengine, adttest = 0 issues), race-hammers of the fixed tests (probe_warn+catchup ×5, drain ×3, TestCompare), `go vet`/builds on every touched module. Daemon absorbed everything into `chore:` commits (no commit authorization this session).

## b) PARTIALLY DONE

1. **M4 MySQL leg: ~70% validated, blocked at the vehicle layer.** After the fixes in a/8, run 8 failed ONLY in `TestMySQLADTMatrix/{Vector,Map}` — the same single-RST-kills-the-suite slirp fragility in the next un-hardened suite (engine construction opens parallel fresh dials through one slirp forward). Patching every suite with retries is whack-a-mole (I stopped after two: the pattern is systemic, not test-specific). Evidence trail: `/tmp/int-mysql{2..8}.log`. The ephemeral-native MariaDB on 13306 is up but UNPROVISIONED (no cqrs_test DB/user yet) and NO test module has run against it — the provisioning command hit the shell's 50-background-jobs limit (accumulated sleep-poll jobs; MariaDB server itself is alive).
2. **T18b `#load-sweep`** — never attempted: load never dropped below the <~10 window for long (verify runs, tag-wave builds, my own legs; 23:21 load is 59–73).
3. **M4 dgraph + redis legs** — not started (sequenced behind mysql; exclusivity + the shell job limit).
4. **Post-wave go.sum completeness sweep** — I tidied the 6 mysql-leg modules only; the wave session's own post-wave step may cover the rest (unverified by me).

## c) NOT STARTED

- `#load-sweep` + `benchmark-regression.sh --save` re-baseline (T18b).
- dgraph/redis integration legs.
- Plan-doc staleness addendum for `docs/planning/2026-09-19_15-37_SUPERB-verify-green-tag-wave-crm-ports.md` (its F-tables are now doubly stale: takeover session + this session superseded them).
- CHANGELOG entries for THIS session's fixes (templ regen was absorbed silently; test hardening + slirp mitigations unrecorded).
- Second leaked QEMU cleanup: PID 1029459 (from run 8, started 22:31) still holds 33070 at report time — killing it is the first step of any next mysql attempt.
- Gotcha documentation: the leaked-QEMU-poisons-port-33070 failure mode and the slirp-RST-whack-a-mole diagnosis belong in `docs/agents/gotchas-testing.md`.

## d) TOTALLY FUCKED UP (own it)

1. **Six failed verify runs before the green one** — not because the tree was far from green, but because I kept re-running the full 25–30 min pipeline into a MOVING tree (three concurrent sessions + daemon landing changes mid-run: templ regen at 19:18–19:19 DURING verify2, README edit before verify4). Each failure was a different moving-target issue; verify2's templ failure was even caused by a regen that landed between my phase passes. I should have switched earlier to running the CHEAP tail phases individually (which I eventually did) and only then committed to one full run.
2. **I replicated a wrong-cwd templ regeneration before reading the tripwire script.** I saw "generated from the wrong cwd" evidence, assumed repo-root was canonical, regenerated from repo root — INVERTING the contract (bare filenames are canonical; the tripwire script even prints the correct command). One extra regen cycle wasted; the mistake was caught by re-running the gate, not by reading first.
3. **Let sleep-poll background jobs accumulate until the 50-job shell limit bit mid-provisioning** — the ephemeral MariaDB provisioning (CREATE DATABASE/USER) never ran because the shell refused the command. Sloppy resource hygiene on my side; the server is up but unusable until provisioned.
4. **Whack-a-mole temptation on the slirp resets**: I hardened two test sites (sqlstore helper, adttest CAS racer) before recognizing the systemic pattern (every parallel-dial suite through slirp is one RST away from red). The second hardening was already past the point where the right move was switching vehicles (native MariaDB). Cost: one full VM-leg run (~9 min) I could have skipped.
5. **No authored commits** — every fix this session lives in daemon `chore: auto-commit` history (73-file `2af20d19a` etc.). Same authorship-mangling complaint as the 18:05/18:16 sessions; I did not request commit authorization either.

## e) WHAT WE SHOULD IMPROVE

1. **`.golangci.yml` needs a mechanical drift guard** (plan M16 exists, unbuilt): this session alone the config was mangled once more (incident #9) — `#check-lint-config` only helps when someone RUNS it. A hash-golden in verify/pre-commit or a daemon-side schema check would end the war.
2. **Verify needs a quiet-tree precondition** (plan M15's load guard is adjacent but this is different): a "no uncommitted/foreign changes newer than N min" check or simply the documented habit "run the cheap tail phases individually during concurrent-session hours; reserve full runs for quiet windows."
3. **The QEMU mysql leg is structurally fragile on a busy host**: nixos-test-driver QMP crashes orphan QEMUs that then hold hostfwd port 33070 and poison every retry. The script should (a) pre-kill stale listeners on $HOST_PORT with a warning, (b) trap-kill the process GROUP, not just the driver PID.
4. **`pgtestcontainer.TestMain` boots Docker even when zero PG tests will run** (`-run` filters do not filter TestMain). It should start lazily on first `DSN()` use — that is a library fix in `testutil/pgtestcontainer` (just tagged v4.2.1; would need a new version) benefiting every consumer that mixes dialects.
5. **claimkit does not classify errors with errorfamily** (unlike the sql stores) — its Transient-class failures are indistinguishable from Rejection-class at the call boundary, which is why the adttest retry had to be retry-on-anything instead of retry-on-retryable. Candidate API hardening for the next claimkit tag.
6. **Stale-pin meta-tests fail confusingly**: `TestREADMEClaim_ModuleCountFloor`'s message ("update the meta-test") was right, but the pin-vs-README drift existed for hours before anyone ran verify. The cheap meta-tests should run in `#verify-fast` if they aren't (they cost 0.1s).
7. **My own process**: kill finished poll jobs; read the tripwire/gotcha file BEFORE acting on a gate failure whose message names a fix command; recognize systemic-vs-local failure earlier (two same-class flake fixes in different suites = systemic).

## f) Up to 50 things to do next (ordered)

1. Kill leaked QEMU PID 1029459 (holds 33070; `ss -tlnp` to confirm).
2. Free background-job slots; provision the ephemeral MariaDB (CREATE DATABASE cqrs_test; cqrs user) on 13306.
3. Run the full mysql module set against 13306 with `MYSQL_TEST_DSN=cqrs:cqrs@tcp(127.0.0.1:13306)/cqrs_test?parseTime=true&multiStatements=true` + `ADTTEST_CAS_RACERS=10` (stack/mysql, idempotency/sqlstore `-tags integration -run TestIntegration_MySQL`, metaengine/mysqlengine, scheduling/sqlstore, queue/mysql, claiming) — the M4 mysql completion.
4. Consider codifying step 3 as `scripts/ephemeral-mysql.sh` + `#integration-mysql-ephemeral` flake app (mirrors ephemeral-dgraph/pg pattern) — PR-able repo improvement.
5. Run `#integration-dgraph` leg.
6. Run `#integration-redis` leg.
7. First quiet window <~10 load: `nix run .#load-sweep` + `./scripts/benchmark-regression.sh --save benchmarks/benchmark-baseline.txt` (T18b).
8. Add slirp/leaked-QEMU gotchas to `docs/agents/gotchas-testing.md` (evidence: /tmp/int-mysql*.log).
9. CHANGELOG `[Unreleased]` Fixed entries: probe_warn race sink, benchkit budget, drain margins, README pin, templ regen, sqlstore/adttest retry hardening, vm-mysql.sh POSTGRES_TEST_DSN stub (verify cited symbols against api_surface first).
10. Staleness addendum for the 15:37 plan doc (F-tables superseded by takeover session + this session; M3 done, M4 partial).
11. vm-mysql.sh: pre-flight stale-port check + process-group cleanup (d/3 root cause).
12. nspawn leg parity: give `vm-mysql-nspawn.sh` the same POSTGRES_TEST_DSN stub (same Docker churn, harmless there but wasteful).
13. M16: `.golangci.yml` hash-golden drift guard wired into verify.
14. M15: verify load-threshold guard (refuse >N loadavg, retry message).
15. `pgtestcontainer` lazy TestMain (library change; needs verify-before-filing + a new tag).
16. claimkit errorfamily classification audit (e/5).
17. Meta-tests (TestREADMEClaim_*, TestEvery*) into verify-fast if absent.
18. Post-wave repo-wide `GOWORK=off go mod tidy -diff` sweep (verify the wave session's completion).
19. Re-run full `#verify` once on the post-wave tree (my rc=0 predates the 92-tag pin-sweeps).
20. `#verify-ci` (GOWORK=off matrix) on the post-wave tree.
21. `#vulncheck` + `#check-arch` + `#check-coverage` (pre-release set; post-wave re-run).
22. Confirm CI is green at origin/master post-wave; triage any new failures.
23. api-stability golden review post-wave (new modules? goal-shaped-app exclusions in place?).
24. The tracked binary `example/goal-shaped-app/goal-shaped-app` should be untracked + gitignored (G-T23 hygiene debt; build keeps dirtying it).
25. Owner-gated (carried): tag-wave follow-ups, upstream exhaustruct/go-types filings, v5 items — unchanged from 18:05 report.

(26–50 held in reserve: the above 25 are the honest, evidenced list; padding further would be filler.)

## g) Questions I CANNOT figure out myself

1. **MySQL leg vehicle decision:** the QEMU slirp leg needs either (a) retry-hardening EVERY parallel-dial suite (whack-a-mole, weakens nothing but spreads retries everywhere), (b) serializing dial bursts in the engine tests, or (c) blessing an ephemeral-native-mariaDB leg (script + flake app, nspawn-equivalent without root) as the canonical local vehicle and demoting the VM leg to CI-only. Which do you want? (I'd build (c) and keep the VM leg for CI.)
2. **The BuildFlow-vs-curated-config war:** `.golangci.yml` was auto-regressed AGAIN this session (incident #9, directly undoing your committed curation). Is BuildFlow's golangci-lint-auto-configure supposed to EVER touch this repo's config — and if not, do you want the M16 hash-golden drift guard in verify/pre-commit now, or a BuildFlow-side skip for this repo?
3. **Commit authorization:** every fix this session (7 nolint repairs, 5 verify-blocker fixes, 2 test hardenings, 1 script fix, templ regen) was absorbed into daemon `chore:` commits, interleaved with two other sessions. Do you want authored, pathspec-limited commits at phase boundaries from here on (several prior sessions have asked; the answer changes repo history quality immediately)?

---

*Point-in-time snapshot. Evidence artifacts: /tmp/verify{,2..7}.log, /tmp/lint-full.log, /tmp/int-{pg,mysql2..8}.log, /tmp/eph-mysql/, /tmp/golangci-regressed-snapshot-18-39.yml.*
