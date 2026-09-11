# Status Report — Follow-up Verification, Repo-Wide go.sum Repair, and CI Triage

**Session:** 2026-09-11, ~02:25–03:43 CEST · **Repo:** go-cqrs-lite · **Branch:** master
**Scope discipline:** continuation of the 02:16 dgraph-shuffle report; this report covers
ONLY this session's work (executing that report's own §f follow-ups) plus issues observed
while doing it. Format is `.md` at the user's explicit request (skill default is HTML).

> Answering the three review questions directly: I forgot to run the lint gate before
> calling the new test file done, and I first reproduced the sqlstore failure with the
> wrong build tags — both fixed mid-session and now encoded as gotchas/process rules.
> The biggest remaining gap: ~14 CI jobs are red for reasons I classified but did not fix.

---

## TL;DR

This session executed the prior report's follow-ups and went one level deeper than
planned: the mysql-vm live verification surfaced a **repo-wide broken-go.sum wave**
(7 modules, missing `/go.mod` hashes after the recent pin bumps) that is red in CI's
Module matrix today — all 7 fixed and re-verified (84/84 modules green standalone).
The `-race` dgraph run is green, the contention retry logic is now unit-pinned, the
conflict-domain knowledge is documented, and §f was harvested into TODO_LIST/ROADMAP.
The big open front: **CI master has ~15+ failing jobs and no green run in the last 30**;
I classified the causes and fixed the go.sum class, the rest is triaged into a 🔥 TODO.

| Category            | Count |
| ------------------- | ----- |
| a) FULLY DONE       | 13    |
| b) PARTIALLY DONE   | 5     |
| c) NOT STARTED      | 15    |
| d) TOTALLY FUCKED UP| 4 mine + 3 attribution guards |
| e) Improvements     | 10    |
| f) Next actions     | 50    |
| g) Questions        | 3     |

---

## a) FULLY DONE

1. **CHANGELOG symbol gate executed and green** (prior §b4 closed).
   `scripts/check-changelog-symbols.sh`: 20 citations verified against the API golden.

2. **`check-file-size` root-cause completed — and it changes the picture.** The gate
   IS wired into CI (`file-size-gate` job, ci.yml:457, inline check — the flake app is
   a local mirror) but NOT into `#verify`. The prior session's "RED on master, maybe
   not enforced" is now precise: CI is red, locally reproducible, and the offenders
   number **58 repo-wide** (recounted), not 1. The split-waves program remains parked
   on the owner's policy decision (already-tracked 🔥 TODO item, refreshed with the
   new count).

3. **`storage/view/store.go` split** (prior report's named offender):
   358 → 276 lines + new `mapper.go` (83) holding `ViewColumn`/`ViewMapper`/`IndexSpec`
   verbatim. Same package, zero API change, zero imports needed. Build/vet/test green;
   lint 0 issues via the owning module (`storage` — view has no own go.mod).

4. **`stack/bundle.go` split** (the offender the gate surfaced next):
   363 → 253 lines + new `lifecycle.go` (117) holding the lifecycle cluster
   (`shutdownEdge`, `Close`, `GracefulClose`, `Drainer`, `registerCloser`).
   Behavior-preserving; unused imports dropped; build/vet/test green; lint 0.

5. **`-race` live dgraph run against the fixed code** (prior §b2 closed): PASS,
   100.5s, random seed `1789088901879882871`, full suite incl. soak — **zero race
   reports, zero contention errors**; only legitimate capability skips
   (Vector/Spatial unimplemented, `CALIB_DUMP` env-gated).

6. **Contention-retry unit tests** (§f7 closed): `metaengine/dgraphengine/transaction_retry_test.go`
   — `TestIsContentionError` (10 cases: verbatim abort/pending, wrapped variants,
   case-sensitivity pins, non-contention negatives) + 6 `TestRetryOnContention_*`
   behavior tests (succeed-after-transients, alter-path `txnScoped=false` retry,
   fail-fast on unrelated errors, immediate surface inside `RunInTx`, exhaustion at
   `contentionAttempts`, context-cancel during backoff). Verbatim Dgraph message
   capitalization is pinned with scoped `//nolint:staticcheck` directives; lint 0 issues.

7. **Shared-`dgraph.type` conflict domain documented** (§f8 + §f20 closed): new bullet
   in `docs/agents/gotchas-language-footguns.md` and a new "Concurrency & contention"
   section in `metaengine/dgraphengine/README.md` (why ALL parallel writers conflict
   on one Alpha, the retry schedule, RunInTx caller-retry semantics, pointer to the
   unit pins).

8. **mysql-vm live verification** (§f3 + prior §b1 closed): two full QEMU runs with
   the rolled-in shuffle flag. `stack/mysql`: PASS both runs (different random seeds).
   `idempotency/sqlstore`: 3/4 tests PASS both runs. The suite found real breakage
   (see #9) and one environment-only failure (see d1 / b3).

9. **Repo-wide broken-go.sum wave discovered and repaired.** The first VM run failed
   at MODULE SETUP (`missing go.sum entry for go.mod file`), not at any assertion.
   Root cause: the recent pin bumps (record v4.5.0 / id v4.6.0 / pgtestcontainer
   v4.2.0) landed without per-module tidies, leaving module `h1:` hashes present but
   `/go.mod` hashes missing. Workspace builds AND production `go build` stay green
   (test-only deps!) — only `GOWORK=off go test` sees it, which is why it shipped.
   Swept all **84 modules** with `GOWORK=off go vet -tags "goexperiment.jsonv2
   integration" ./...`: 7 broken (`metaengine/badgerengine`, `metaengine/mysqlengine`,
   `idempotency/sqlstore`, `projectionhost`, `stack/bench`, `stack/postgres`,
   `testutil/pgtestcontainer`); all repaired via `GOWORK=off GOFLAGS=-mod=mod go mod
   tidy` (badgerengine needed 3 added hashes + a toolchain-canonical retract
   reordering; mysqlengine cascaded pgx→record→id). Re-sweep: **84/84 green**.
   CI's `Module metaengine/mysqlengine` log shows the IDENTICAL error → the same-day
   fix addresses a real red CI class.

10. **HARVEST of the 02:16 report executed** (§f26 closed): 12 items routed into
    `TODO_LIST.md` (testing section, each with source citation + effort); the 3
    unresolved questions became ROADMAP **Open Questions #8/9/10**; 2 long-tail ideas
    → ROADMAP Raw Ideas. Deduped the stale "Wire `#check-file-size` into verify" TODO
    (superseded by the wired-in-CI reality; folding note added to the 🔥 item);
    refreshed the 🔥 350-line item with today's recount and both splits.

11. **New gotcha: pin bumps without per-module tidy** —
    `docs/agents/gotchas-module-management.md` gains the full class description:
    symptom (`missing go.sum entry for go.mod file` on tests only), why it ships
    (build+workspace blind to test-only deps), detection (per-module integration-
    tagged vet), repair (mod=mod tidy; iterate — fixing one hash unmasks the next),
    and the never-hand-edit rule.

12. **CI triage classification** (see b1 for the unfixed remainder): run 34548534824 —
    15+ failing jobs enumerated; **no green run in the last 30**; failures predate
    today (present on `82d5218fc`, 2026-09-11 00:55Z, before this session's work).
    Crucially: all four **NixOS VM Tests are GREEN in CI (incl. mysql)** — which
    reclassifies my local VM concurrency failure as host-environment-only (see b3).
    Everything folded into a new 🔥 TODO item with evidence trail.

13. **Final gates all green**: doc-check full corpus 1273 references valid (run 3×
    across doc edits); lint-module 0 issues for `storage`, `stack`,
    `metaengine/dgraphengine`; 84/84 standalone module vet sweep; `bash`-level checks
    unchanged. All session artifacts absorbed by the auto-commit daemon; nothing
    hand-committed per repo rule.

---

## b) PARTIALLY DONE

1. **CI repair is classified, not fixed.** The go.sum class (multiplied across the
   Module-matrix jobs) is fixed; ~14 other failing jobs are undiagnosed/unfixed:
   FlakeHub auth ERRORs in job logs despite `use-flakehub: false` everywhere in
   ci.yml (possibly fatal in the ephemeral dgraph/pg/redis integration jobs — all
   three pass locally), shellcheck SC2086 in `scripts/test-tag-release.sh`
   (`git $notag` is INTENTIONALLY unquoted — the "fix" shellcheck suggests changes
   semantics; needs a directive or restructure), Minimum Coverage, verify-fast,
   go.work sync check, Nix Flake Check, CGo build, Security Scan.

2. **check-duplication certification is half-true.** Verified: THIS session's changes
   introduce **zero** clone groups (the retry consolidation removed one). But the
   gate itself is RED: 2 new groups belong to the parallel session's cqrs-lint work
   (`typed_confirm.go` vs `lintutil/name_heuristics.go`; `c013_embedded_test.go`/
   `typed_confirm_test.go` test clones — daemon commit `d81a61746`, 02:07). Not my
   code, not my call to annotate or absorb into the baseline.

3. **mysql-vm shuffle adoption is verified to environment limits.** Everything that
   can pass locally passes (shuffled, twice). `TestIntegration_MySQLIdempotency_
   AtomicClaimUnderConcurrency` fails deterministically LOCALLY (2/2, slirp
   connection-reset bursts → "invalid connection") but is **GREEN in CI's VM job** —
   host-specific (nested KVM / kernel 6.18 slirp behavior). The standing evidence
   for that test is CI, not a local run.

4. **Full `#verify` gate still not run** — the parallel session keeps the tree dirty
   (now: `cmd/cqrs-lint` a020/a030/d011_test + `scheduling/sqlstore` claiming files);
   `nix fmt` fail-on-change + verify-exclusivity make it a bad moment. Per-module
   evidence stands in for everything I touched.

5. **vm-mysql-nspawn remains unverified** (root-blocked TODO unchanged).

---

## c) NOT STARTED

1. Fixes for the ~14 remaining red CI jobs ( FlakeHub config, shellcheck directive,
   coverage drift, verify-fast, go.work sync, flake check, CGo, security scan,
   ephemeral integration legs).
2. otel/prometheus counter for silent contention retries (`cqrs.dgraph.contention_retry`)
   — needs a `check-arch` dep-budget review for dgraphengine → otel/ first.
3. Skip-vs-fail policy for live conformance engine construction (ROADMAP OQ #10) +
   `newDgraphEngineOrSkip` tightening.
4. Shuffle eval + adoption for `scripts/test-integration.sh` / `test-all-backends.sh`
   (gated on OQ #9).
5. `go mod tidy` in `integration/` (gopls: unused genproto/rpc).
6. dgraphengine modernization sweep: 13× `b.Loop()` in bench_test.go + `atomic.Uint64`
   in helper_test.go.
7. Backport contention-retry review to turso/badger engines.
8. `doWrite` response-narrowing review (symmetry with `doMutate`).
9. `ensureEdgeSchema` in-tx Alter unit pin.
10. Unify ephemeral-script passthrough conventions (TEST_ARGS vs EXTRA_ARGS vs raw).
11. Persist shuffle seeds to a log file for post-hoc replay.
12. Watch dgraph + redis CI jobs (~10 shuffled runs) for order-induced flakes.
13. The 58-file 350-line split waves (owner policy pending).
14. `integration-mysql-nspawn` (needs root).
15. A `check-modsums` flake app automating the per-module go.sum sweep (my own §e
    proposal — would have caught this wave pre-push).

---

## d) TOTALLY FUCKED UP

**Mine:**

1. **Wrong-flag reproduction — nearly recorded a false "works locally".** My first
   local check of the failing sqlstore module used `go build` + untagged vet and
   PASSED; I was one step from writing the VM failure off as environmental noise.
   The failing leg compiles with `-tags "integration ..."` (vm-mysql.sh:115), which
   pulls the testcontainers test files — that's where the missing hash bites. Caught
   it by reading the script before concluding. Lesson encoded in the new gotcha:
   reproduce with the failing command's EXACT tags, never a weaker proxy.

2. **Scoped doc-check false alarm.** Ran doc-check on just TODO_LIST/ROADMAP → hard
   error "No Go references found … documents were NOT verified" → brief belief that
   my edits broke the gate. It is a methodology artifact: the full #verify corpus
   supplies the Go samples; a scoped file set without code blocks can never pass.
   One wasted cycle; the full-corpus run was green (1273 refs).

3. **Lint-last, not lint-first, on the new test file.** Shipped
   `transaction_retry_test.go` after gofmt+vet+test, then lint-module surfaced 2×
   ST1005 (capitalized error strings) and 1× golines (nolint comment made the line
   too long) → two extra fix rounds (same-line nolint → preceding-line directive
   restructure). Write → lint-module → done, in that order, next time.

4. **lint-module invoked on a package dir.** `nix run .#lint-module -- storage/view`
   → usage error; `storage/view` has no go.mod (it belongs to the `storage` module).
   Module-boundary blindness cost one cycle.

**Attribution guards (NOT mine — recorded to prevent misattribution):**

5. The 2 new duplication groups (cqrs-lint) and the broad CI redness predate this
   session (parallel session's cqrs-lint work, daemon `d81a61746` 02:07; CI red
   present on `82d5218fc` 00:55Z and for 30+ runs).

6. The broken-go.sum wave itself was CREATED by the earlier pin-bump session(s), not
   by me — this session only found and fixed it.

7. This session introduced **zero** red intermediate commits (every edit was a single
   atomic file write; the daemon absorbed cleanly). The red-commit class from the
   prior session did not recur.

---

## e) WHAT WE SHOULD IMPROVE

1. **Lint gates run immediately after file creation** — not after "tests pass". The
   ST1005/golines round-trip was pure ordering failure.
2. **Exact-flag reproduction rule** — environment/CI failures get reproduced with the
   failing command's exact tags/env/flags before any "works locally" verdict. Now in
   gotchas; should be in every verification checklist.
3. **Automate the go.sum sweep** — a `check-modsums` flake app (loop modules ×
   integration-tagged vet) wired into CI and `#verify` would have caught this wave
   before push and cheaply prevent recurrences.
4. **Pin-sweep waves should tidy per module mechanically** — the wave-mechanics gotcha
   already says "run the GOWORK=off matrix after a wave"; it plainly isn't enforced.
   A script beats a memo.
5. **CI red must page louder** — 30 consecutive red runs on master with no green
   since (unknown) is the F040 (no branch protection) failure mode again, now
   costing real repair effort.
6. **VM slirp resilience** — concurrency tests over QEMU slirp should classify
   transient resets (bounded retry) or declare CI-only execution; the script comment
   documents the sensitivity but the test still hard-fails on it locally.
7. **doc-check scoped-mode clarity** — "no Go samples in scope" should be a distinct,
   non-alarming outcome vs "broken references".
8. **Rollout-verification convention** — "executed at least once" per script (prior
   §e6) is now practiced but informal; a one-line status per integration script in a
   living doc would prevent syntax-only rollouts.
9. **art-dupl ownership** — group reports citing the introducing commit (blame
   integration) would make multi-session triage (like today's) one command instead
   of archaeology.
10. **FlakeHub half-state** — every job logs auth ERRORs for a cache backend ci.yml
    claims to have disabled; finish the removal (or fix credentials) so logs are
    trustworthy again.

---

## f) UP TO 50 THINGS TO GET DONE NEXT

> Brainstorm, not commitment — harvest routing applied to the high-value subset
> already (TODO_LIST/ROADMAP updated this session); the rest is ROADMAP fuel.

**Direct follow-ups on this session's work**

| #  | Action                                                                                                          | Effort |
| -- | --------------------------------------------------------------------------------------------------------------- | ------ |
| 1  | Verify the go.sum fixes turn the CI Module-matrix jobs green (next run; watch badgerengine/mysqlengine/pgtestcontainer) | XS |
| 2  | Build `check-modsums` flake app (84-module integration-tagged vet loop) and wire into CI + #verify               | S      |
| 3  | Fix remaining CI: decide FlakeHub (remove magic-nix-cache vs restore registration) per OQ/g1                     | S      |
| 4  | Fix shellcheck on `scripts/test-tag-release.sh` via disable directive (quoting `$notag` changes semantics)       | XS     |
| 5  | Diagnose Minimum Coverage / verify-fast / go.work sync / Nix Flake Check / CGo / Security Scan reds              | M      |
| 6  | Diagnose ephemeral dgraph/pg/redis CI legs (green locally 6+ runs today — suspect FlakeHub-fatal step)           | S      |
| 7  | Add slirp-reset classification (bounded retry) to `AtomicClaimUnderConcurrency`, or mark it CI-only              | S      |
| 8  | Re-run `nix run .#verify` once the tree is clean (gate this session's Go changes end-to-end)                     | M      |
| 9  | Re-run check-duplication after the cqrs-lint session lands; annotate or dedupe the 2 groups per g3              | XS     |
| 10 | Add `TestEveryModulePassesStandaloneVet`-style meta-test (repo-level guard for the go.sum class)                 | S      |

**Engine / suite quality (carry-overs, unchanged)**

| #  | Action                                                                                                          | Effort |
| -- | --------------------------------------------------------------------------------------------------------------- | ------ |
| 11 | otel counter `cqrs.dgraph.contention_retry` (dep-budget review first)                                            | S      |
| 12 | Skip-vs-fail policy implementation for `newDgraphEngineOrSkip` (OQ #10)                                          | S      |
| 13 | Shuffle eval + adoption: `test-integration.sh` (gated OQ #9)                                                     | S      |
| 14 | Shuffle eval + adoption: `test-all-backends.sh` (gated OQ #9)                                                    | S      |
| 15 | Backport contention-retry review to turso/badger                                                                 | M      |
| 16 | `ensureEdgeSchema` in-tx Alter unit pin                                                                          | S      |
| 17 | `doWrite` response-narrowing review                                                                              | XS     |
| 18 | Modernize dgraphengine bench_test (`b.Loop()`) + helper_test (`atomic.Uint64`)                                   | S      |
| 19 | `go mod tidy` in `integration/` (unused genproto/rpc)                                                            | XS     |
| 20 | Unify ephemeral-script passthrough conventions                                                                   | M      |
| 21 | Persist shuffle seeds to a log for post-hoc replay                                                               | XS     |
| 22 | Watch dgraph+redis CI ~10 shuffled runs; record failing seeds                                                    | XS     |
| 23 | Decide contention-retry knobs: internal forever vs exported (OQ #8; decides golden work)                         | S      |
| 24 | Bounded ctx for `init()`'s retry loop (v5 `New(ctx)` prerequisite)                                               | S      |
| 25 | Regression test reproducing seed 42's ordering without a live server (deterministic mock)                        | L      |
| 26 | enginetest contract: engines declare their contention model                                                      | L      |
| 27 | Evaluate read-only Dgraph txns for retry needs (document if N/A)                                                 | S      |
| 28 | Confirm no measurable p50 write-latency overhead from retry under low contention                                 | S      |
| 29 | Debug-gated "contention retries: N" summary at test exit                                                         | S      |
| 30 | Review `doWrite` error-wrap text parity with the deleted graph-only helper                                       | XS     |

**Docs / ledgers**

| #  | Action                                                                                                          | Effort |
| -- | --------------------------------------------------------------------------------------------------------------- | ------ |
| 31 | Annotate the 02:16 status report: §b1/b2/b4 + §f2/3/6/7/8/12/26 now DONE (ANNOTATE mode, inline)                 | XS     |
| 32 | Verify no archived doc still claims "dgraph not yet shuffle-evaluated" without annotation                        | XS     |
| 33 | CONTRIBUTING.md testing section: one-liner that ephemeral invocations ship `-shuffle=on` since 2026-09-11        | XS     |
| 34 | Document `ephemeral-nats.sh` no-default-suite design in the quick reference                                      | XS     |
| 35 | Record the host-env slirp sensitivity in vm-mysql.sh header (CI-green/local-flaky)                               | XS     |
| 36 | Sweep archived docs for `doWithAbortRetry` mentions; mark superseded                                             | XS     |
| 37 | Add the go.sum incident to the calibration/benchmarks docs only if latency numbers are ever wanted (else skip)   | XS     |

**Infrastructure / hygiene**

| #  | Action                                                                                                          | Effort |
| -- | --------------------------------------------------------------------------------------------------------------- | ------ |
| 38 | Root-cause or moot remaining 58 file-size offenders (policy decision gates the waves)                            | L      |
| 39 | F040: branch protection / required checks so 30-red-run stretches cannot recur unnoticed                         | M      |
| 40 | Daemon hook: build-check before sweep (prevents red intermediate commits class)                                  | M      |
| 41 | Modernize-sweep: silence remaining gopls modernize hints repo-wide (dgraphengine done last)                      | M       |
| 42 | `scheduling/sqlstore` claiming files: re-run mysql claiming suite after parallel session lands them              | XS     |
| 43 | Tag-wave note: projectionadapter sibling replace + metaengine pin bump ride next wave (existing item, verify)    | XS     |
| 44 | Evaluate `-shuffle=on` for verify-ci per-module matrix legs (after go.sum class verified green)                  | S       |
| 45 | Review TEST_TIMEOUT knob consistency (script 600s vs go -timeout vs timeout -k 15)                               | XS      |
| 46 | Record integration-suite duration drift thresholds (64–98s observed for dgraph; 100s+ under -race)               | S       |
| 47 | Confirm first green shuffled CI run for dgraph/redis explicitly; link it in the next report                      | XS      |
| 48 | Consider `-shuffle=on` for load-sweep.sh / verify-parallel.sh invocations                                        | XS      |
| 49 | Write the "shuffle program caught 2 real backend bugs" motivation line into the testing gotcha (§f50 prior)      | XS      |
| 50 | Celebrate: two sessions in a row, live verification (shuffle eval, VM run) caught what static checks never would | XS      |

---

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **FlakeHub: abandoned or coming back?** Every job logs `magic_nix_cache: FlakeHub:
   cache initialized failed: Unauthenticated` even though ci.yml sets
   `use-flakehub: false` with a comment saying to keep the GitHub cache backend.
   If FlakeHub is permanently gone, the right fix is removing the action entirely
   (and checking whether its absence is what kills the ephemeral integration jobs);
   if registration will return, the fix is credentials. I can't derive the intent,
   and the two fixes diverge.

2. **CI repair depth — all the way, or unblock-and-triage?** Master has ~15+ failing
   jobs and no green run in the last 30. Option A: a dedicated repair session that
   chases every job to green (Effort M-L). Option B: fix only verify-fast + the
   ephemeral integration legs (the ones that gate real merges), leave the rest
   triaged in the TODO. I can execute either; I can't tell how much CI-health is
   worth to you right now versus deferring while the parallel session lands work.

3. **The 2 new cqrs-lint duplication groups: accept or dedupe?** `typed_confirm.go`
   vs `lintutil/name_heuristics.go` (the `strings.ToUpper(structName)` cluster) and
   the `c013_embedded_test.go`/`typed_confirm_test.go` test clones trip
   check-duplication. They belong to the parallel session's in-flight lint-rule work.
   Intentional similarity → `//art-dupl:accept` annotations; accidental → dedupe into
   lintutil. I can't judge another session's intent, and the baseline must not
   silently absorb them either way.

---

## Verification receipts (commands + outcomes)

| Check | Result |
| ----- | ------ |
| `scripts/check-changelog-symbols.sh` | ✓ 20 citations honest |
| `nix run .#check-file-size` | 58 offenders (recount); store.go + bundle.go fixed |
| storage/view split: build/vet/test (GOWORK=off, jsonv2) | ok, 0.062s |
| stack split: build/vet/test | ok (contracttest/sqlopt included) |
| `nix run .#integration-dgraph` with `TEST_ARGS="-race -timeout 15m" CGO_ENABLED=1` | PASS 100.5s, seed 1789…871, 0 races |
| dgraphengine unit tests (`-run "TestIsContentionError\|TestRetryOnContention"`) | ok ×3 runs (0.75s) |
| `nix run .#lint-module -- storage / stack / metaengine/dgraphengine` | 0 issues / 0 issues / 0 issues (after nolint+golines fix) |
| `nix run .#integration-mysql-vm` (run 1) | RED: sqlstore module setup (go.sum) — legit catch |
| `nix run .#integration-mysql-vm` (run 2, post-fix) | stack/mysql PASS; sqlstore 3/4 PASS; AtomicClaim local-env fail (CI-green) |
| go.sum sweep (84 modules, integration-tagged vet) | 7 broken → tidied → 84/84 OK |
| CI run 34548534824 job scan | 15+ failing; NixOS VM Tests all SUCCESS incl. mysql; no green in last 30 runs |
| doc-check full corpus (×3) | ✓ 1273 references valid, 64 packages |
| `nix run .#check-duplication` | 2 new groups — parallel session's cqrs-lint files; 0 from this session |

*Nothing hand-committed per the never-commit rule; the auto-commit daemon has absorbed
all session artifacts (HEAD `526453f69` at report time). One file pending absorption at
write time: none — clean except the parallel session's five foreign dirty files.*
