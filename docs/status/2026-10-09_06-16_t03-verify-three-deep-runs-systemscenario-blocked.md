# W1 T03 — Three Deep Verify Runs: Gate Parity Root-Cause, Five Blockers Fixed, systemscenario Blocks Run #3

> **Session:** 2026-10-09 01:58 → 06:15 (resumed from the 00:33 halt under the standing
> READ/UNDERSTAND/RESEARCH/REFLECT directive). HALTED 06:15 per user order (status report +
> wait for instructions).
> **Scope:** T03/f045 (one green composed `#verify`) — the last open W1 blocker for the
> f046 receipts; T04/T05 remained queued (quiet-window-gated) and were only PREPPED, not run.

## a) What was done (all verified)

1. **Load-refusal root cause FIXED — `can-run-composed-gate` load5 parity bug**
   (`scripts/can-run-composed-gate.sh`). The gate passed at `load5 < ceiling*1.5` (15) while
   verify's guard (via `calibration-gate.sh` v2 rule) refuses at `load5 >= 10` — the exact
   mechanism behind the 5 burned refusals of the prior session (gate GREEN at load5=14,
   verify REFUSED at 14.63). Now exact parity (`load5 < c`), header comment corrected, new
   self-test case `draining burst (load5 over ceiling) refused` with tonight's actual
   signature (7.8/14.6) — mutation-pinned: reverting the awk to `*1.5` turns the case red,
   restore turns green. Self-test 6/6.
2. **Verify chain wrapper v2** (`/tmp/verify-chain.sh`, tree-clean): v1 had a bash rc bug —
   `rc=$?` after a no-else `if...fi` yields 0 (the if-statement's status, not verify's).
   v2 captures rc from each command directly; classification (load-refusal retry vs
   stop-for-inspection) unchanged and correct.
3. **Run #1 blockers (Test phase) — ALL FIXED:**
   - `recipes.md` §2.21b doc lie: sibling's issue-#36 seam work (10-08 16:16–19:56) changed
     `Bundle.MetaEngine()` to return the `MetaEngineStore` interface; the fence still passed
     it where `*metaengine.Store` is required. Rewrote the fence to the sanctioned typed
     helpers (`stackmeta.WithStore(store)` / `stackmeta.Store(bundle)`, import added, Key
     points bullets corrected) + swapped the catalog import (`stack/v4` → aliased
     `stack/metaengine/v4`). Recipe compile gate green (82/82 snippets).
   - cqrs-lint fixture rot ×3 (`scanfixture`, `busfixture`, `typedfixture`): go.mod/go.sum
     stale after the 10-08 15:49 42-tag train moved `id/v4` v4.7.1→v4.7.2 requirements —
     `go list ./...` demanded tidy, packages.Load returned zero packages, ~14 tests "must
     not report clean". First tidy round did not converge (daemon absorbed mid-tidy writes);
     second round with immediate per-fixture `go list` verification converged all three.
     All 4 red packages (adoption/performance/resilience/version) green after.
   - V007 drift gate: registered `stack.WithMetaEngine` (marked Deprecated-for-v5 by the
     sibling, unregistered in `v007_tables.go`) → table row added with dated comment.
4. **Run #2 blocker (Race phase) — FIXED:** cqrs-lint `os.Stdout` data race —
   `captureStdout`/`runCLI` hold `stdoutMu`, but `TestOutputFindingsJSON`/`TestOutputFindingsEmpty`
   called `outputFindings` (prints via `fmt.Println`) unlocked while `t.Parallel()`. Locked
   both sites with the established mutex. 3× `-race` green.
5. **system wiring flake FIXED** (hit in run #1's Test phase): `TestSystem_WiringDeterministic`
   byte-compared `Explain()` including the process-global `Drivers:` line (mutated by the
   parallel leak tests' `RegisterDriver` calls between construct A and B snapshots) and a
   wall-clock `Time:` line. Stripped both non-wiring lines (engine lists, roles, routing,
   EngineNames comparison keep the determinism pin fully armed). 20× green + `-race` 3× green.
6. **Docs/TODO debt (owed from prior session):**
   - Foreign file-size debt surfaced on TODO_LIST — 10 sibling-grown offenders (not 3:
     cqrs-lint d005_version.go 406 NEW, resilience/helpers.go 507 NEW, suppression/stale.go
     489→533, doctor.go 400→433; system system.go 358 NEW, constructor.go 357→413;
     metaengine engine.go 700→712, reflect.go 352→359, typed_reader_scan.go 367 NEW,
     adttest/pagination_conformance.go 353 NEW). Gate `nix run .#check-file-size` red;
     NOT part of #verify.
   - `gotchas-testing.md`: vm-mysql.sh args-verbatim path now default-caps
     `ADTTEST_CAS_RACERS` (documented; overridable).

## b) Verify runs this session (each deeper than the last)

| Run | Window                        | Phases reached              | Failure                                                  | Fixed by     |
| --- | ----------------------------- | --------------------------- | -------------------------------------------------------- | ------------ |
| #1  | 02:02→03:24 (gate waited ~1h) | Build → Vet → Test          | recipe_l1492 + 3-fixture rot + V007 drift + wiring flake | me (session) |
| #2  | 04:59→05:13 (window held)     | Build → Vet → Test ✓ → Race | cqrs-lint os.Stdout race (2 printer sites)               | me (session) |
| #3  | 05:18→06:14 (window held)     | Build → Vet → Test          | **systemscenario ×3 — NOT fixed, sibling-active**        | — (halted)   |

## c) The systemscenario blockers (run #3, left for instructions)

All three surfaced in `cmd/api-stability` meta-tests + the systemscenario package itself:

1. **go.mod/go.sum not tidy** — `TestEveryModuleGoSumIsTidy`:
   `module systemscenario: go.mod/go.sum not tidy (cold-cache builds will fail); run GOWORK=off
   go mod tidy in that module` — same 42-tag-train rot class as the cqrs-lint fixtures.
2. **api golden drift** — `TestAPISurfaceCheck`: golden mismatch on
   `systemscenario/method Await` (sibling added the method; golden not regenerated —
   the "API-surface change ⇒ api golden regen in the SAME edit" contract was missed).
3. **600s test hang** — `panic: test timed out after 10m0s` in `systemscenario/v4`;
   goroutine dump anchors at `(*Scenario).captureMiddleware.func1.1` /
   `(*Scenario).When` (scenario.go:154) inside watermill gochannel plumbing. Same class as
   the prior session's unreproduced mysqlengine 600s pileup (32-way parallel, fast host).

1+2 are the same mechanical-repair pattern I applied to the cqrs-lint fixtures (tidy +
`cmd/api-stability --update`); 3 is a genuine hang needing diagnosis. systemscenario is
sibling-active territory (their recent commits), hence the halt for instructions rather
than a fourth repair on someone's in-flight module.

## d) T04/T05 (queued, prep only)

Prepped and ready to fire in the next quiet window(s): f047 VM-suite row + F52 AGENTS rows;
f048 `storage/snapshot_migration_mysql_integration_test.go` (build tag `integration`,
`MYSQL_TEST_DSN`, userspace MariaDB); f049 21 mysql-labeled seeds extracted to
`/tmp/mysql-shuffle-seeds.txt` (replay via `go test -shuffle=<seed>`); f050 G-T13 ADTSet
mysql leg; f051 nspawn BLOCKED on root presence; f052 `queue/mysql` conformance half;
f053–f057 calibration campaign (dgraph `DG_NetworkRTT` prior at
`metaengine/dgraphengine/probe.go:12`, baseline re-pin protocol in
`benchmarks/benchmark-baseline.txt` header, supersede command in the 09-19 capture banner).

## e) Mistakes this session (honest ledger)

1. Wrapper v1 rc bug (bash no-else if/fi status quirk) — mislabeled rc=0; stop-for-inspection
   behavior was accidentally correct. Fixed in v2.
2. First fixture-tidy round didn't converge and I verified only after all three — daemon
   absorbed mid-tidy writes. Lesson reaffirmed: verify per-fixture immediately.
3. multiedit failures ×2 (non-unique import line; whitespace mismatch) — re-viewed, fixed
   with tighter context; no damage.
4. Initially reported "3 foreign file-size offenders" from the prior session's summary;
   actual count 10 — corrected in the TODO row.

## f) Chain state at halt

Wrapper v2 STOPPED (correct non-load inspection stop) after run #3. No verify running.
Load at halt: 9.36 / 16.69 / 17.01. Tree clean except `scripts/batch-release.sh` (sibling,
untouched). All my edits absorbed by the daemon (last: 06:15).

## g) Questions pending with the user

1. **systemscenario blockers:** authorize the mechanical repair (tidy + api golden regen,
   same pattern as the cqrs-lint fixtures) and a hang diagnosis attempt, or leave for the
   owning sibling session?
2. **Verify scheduling:** relaunch the self-healing chain after systemscenario resolves
   (default), or pin a window/retry cap?
3. **T04/T05:** fire in the next quiet window(s) after verify greens (default), or reprioritize?
