# Testing Gates — Second Pass: Proofs, Real Defects, and the Tail (2026-10-27 04:20 CET)

**Session:** 2026-10-26 23:50 → 2026-10-27 04:20 CET (continuation of the 23-50 report)
**Branch:** `master` (auto-commit daemon active; this pass produced one authored commit — `verify: measure coverage once instead of running the suite twice`)
**Scope:** finish what the previous pass left unverified, then work the remaining plan tail (T10–T27).
**One-line verdict:** the previous pass's major gap — "the composed gate was never run" — is **closed for `#verify-fast`** (green, 866 s, every new gate wired in), the two "foreign reds" are **gone** (`check-modsums` fixed for all 9 modules, `check-depguard` green), and running the legs for real exposed **five defects that only execution could find**. Full `#verify` remains **blocked by host load from other sessions** (load5 67–456 across the evening, ceiling is 10), not by anything in the tree.

---

## §0 The five defects that only running it could find

Every one of these was green in the previous report's own framing, and every one had a plausible story. None of them survived contact with an actual run.

| # | Defect | How it was found | Cost if shipped |
| --- | --- | --- | --- |
| 1 | **`#test-repeat` never worked.** Ginkgo v2 rejects BOTH `-count>1` ("does not support using go test -count") and multi-value `-cpu` ("RunSpecs more than once"). Five of the twenty repeat modules are Ginkgo suites, so the nightly leg was red from the day it was added. | First real `#test-repeat` execution: five packages failed in 0.01 s each with Ginkgo's own explanation. | A permanently-red nightly job that everyone learns to ignore — the exact failure mode the plan exists to prevent. |
| 2 | **`#test` ran a 27-minute soak inside the suite.** The bbolt AutoCRUD soak (`SOAK_SKIP_BOLT` honored by `#verify`, not by `#test`) took **1620 s** in one package while sitting in the same `go test` invocation as everything else. | `#test` with `-json`: `metaengine/bboltengine` reported `Elapsed: 1620.96`. | Two *false* failures elsewhere in the same run (benchkit's journey budget, a lease-expiry probe) and a 30-minute test command nobody wants to run. |
| 3 | **Uncapped package parallelism.** `go test` defaults to `NumCPU`; duckdbengine (231 s of CGo) plus tursoengine (173 s) starved the rest and produced four `systemscenario` await timeouts and a benchkit budget failure **with no code change behind them**. | Same run, failure messages all read `context deadline exceeded`. | Flaky main-branch CI attributed to "the tests are just flaky" — the least fixable-looking class of red. |
| 4 | **benchkit's Compare budgets were fixed numbers.** Four `Compare`-based tests used literal 10 s / 30 s / 30 s / 150 s deadlines while every other timing ceiling in that package already used `loadScaledCeiling`. | `TestCompare_ThreeBackends` failed at 151.2 s in two separate runs, then passed in 24.8 s standalone. | A test that measures the machine, not the code. |
| 5 | **The API golden was missing 32 exports.** Adding `testutil` exports without regenerating `docs/api_surface.txt`. | `#test` failed on `TestAPISurfaceCheck`: "6334 expected, 6366 actual". | Caught by design — this one is in the table as evidence the gate works, not as a defect. |

Two smaller ones from the same runs: a lease-expiry conformance case that asserted "before expiry" with no guard against being descheduled past its own 500 ms lease, and a `nix fmt` run whose `golines` formatter was **killed by load** (`signal: terminated`) — the formatter was fine, the machine was not.

---

## §a FULLY DONE (this pass)

| # | Work package | Evidence |
| --- | --- | --- |
| 1 | **`--profile` reuse in `#verify`** — the Test step emits the merged coverprofile, `check-coverage` consumes it | One invocation produced `build/coverage/merged.out` (4.7 MB); `nix run .#check-coverage -- --profile …` then printed "✓ All 89 module and 168 package floors hold" — no second suite run. `#verify-fast` deliberately keeps its own measurement (short-mode coverage is a different number) with that reason written into the flake. |
| 2 | **T05-route: MySQL suites have a CI route** — new `mysql-integration` job runs `.#integration-mysql-vm` (QEMU VM, MySQL port-forwarded, Go suites on the host) | The route is now *enforced*: `check-test-skips.sh` resolves every skip-gating env var through three hops (workflow → script → flake app), fails on `NONE`, and prints `MYSQL_TEST_DSN … app ci.yml via .#integration-mysql-vm`. `MYSQL_TEST_DSN` stopped being `NONE`. |
| 3 | **Route enforcement hardened twice by real bugs in my own gate** — a comment mentioning a variable manufactured a fake route; then the gate script's own doc comment did it again | `code_mentions` strips comment lines in BOTH hops. Self-test grew from 6 to **13 cases** (6 reason + 7 route). |
| 4 | **T12 tail: per-package runtime budgets** | `scripts/check-test-runtime.sh` + `scripts/test-runtime-baseline.txt` (**168 packages**, generated from a real 445 s run: duckdbengine 231.6 s, benchkit 180.5 s, cqrs-bench 123.3 s …). 6/6 self-test; an empty-rows pipeline bug found and fixed in the same pass (grep's rc=1 under `pipefail` aborted the report). |
| 5 | **T26: matrix completeness + weekly soaks** | `scripts/check-test-matrix.sh` (6/6 self-test) pins `GT_PATHS=${modulePaths}`, `modulePaths ← testModules`, `lintModules == testModules`, and every `repeatModules` entry. `weekly-soaks.yml` (Sundays 05:00 UTC) runs `#soak-weekly` (bbolt + DuckDB) and the Dgraph soak via the ephemeral harness. |
| 6 | **T21: fault-injecting capability wrapper** | `testutil.NewFaultInjector` + `Capabilities()` + typed `ErrCapabilityUnavailable`; 7 contract tests including a 13-name vocabulary sweep and a behavioural (not merely asserted) forwarding proof. Lint-clean after four rounds (exhaustruct, gochecknoglobals, godoclint, ireturn, wrapcheck, S1021-vs-ST1023). |
| 7 | **T27: property/fuzz expansion** | `record` sweeps all 256 `Encoding` values + `FuzzParseEncodingTotality` (registered — fuzz inventory went 115 → **116**) with **no new dependency** (record stays zero-dep; native fuzzing instead of rapid). `kv` got two rapid properties at the blind-store boundary: codec independence both directions and pre-envelope raw-byte rows readable under both readers. |
| 8 | **T16 ratchet half** | `scripts/check-error-taxonomy-tests.sh`: **540 of 546** documented codes frozen, new code without a test fails, dangling row fails, STALE reported. 6/6 self-test. |
| 9 | **T23 verified already covered** | Quarantine → reroute → reactivate, "rejection never quarantines", loud no-candidate, auto-reprobe, threshold override (`engine_health_test.go`) + `CatchUpEngine` rebuild/requires-quarantine/`ErrCatchUpUnsupported` (`engine_failover_test.go`). No new code; recorded so it is not re-dispatched. |
| 10 | **T24 partial: both suites** | `metaengine/context_cancel_test.go` (read path surfaces ctx errors; write path never panics and leaves the store usable; cancellation inside `InTransaction` reaches the callback) and `signing/tamper_matrix_test.go` (ed25519 + HMAC tamper → `ErrInvalidSignature`, never panic). |
| 11 | **Evidence is kept now** | `go-test-shuffled.sh` tees every run to `build/test-logs/<label>-<ts>-i<N>-<seed>.log`. This exists because a failure in the previous pass could not even be named after the fact: the exact-seed replay passed, so the log would have been the only record. |
| 12 | **Both foreign reds closed** | `check-modsums`: 9 modules tidied (`integration` + 8 examples) → "102 modules tidy". `check-depguard`: green, 149 deps covered. |
| 13 | **Composed `#verify-fast`: GREEN in 866 s** | Includes Test Matrix, Test Skips (routes), Untested Exports, Test Sleeps, coverage floors, md-go, API stability, duplication, arch, lint. `#verify-fast` also gained the `-p 4` cap it was missing. |
| 14 | **T03.5 closing proof** | Two consecutive green shuffled runs on the final tree: seed `181394667509926` (525 s) and `61496612585225` (429 s), zero FAIL, logs retained. |

---

## §b The one thing still NOT done, and why

**Composed `nix run .#verify` has still never completed in this session.** It refuses at its own load guard:

```
calibration-gate: LOAD GATE FAILED — load1=133.13 >= ceiling 10
verify-load-guard: REFUSED — host load ≥ 10. Do not burn the attempt
```

The load is **not from this work**: `/proc/loadavg` showed 4 × `whisper-cli` at ~500 % CPU each, `cc1plus` Nix builds, and a `rustc` build from other sessions on the host, peaking at **load1 798 / load5 603**. A `can-run-composed-gate.sh --wait-loop --max-wait 7200` ran for over an hour (26 attempts) and never saw a window.

What stands in for it, per verify step: every cheap gate was run **individually and green** (arch, lint-config, duplication, templ, docserver-css, architecture-reference, turso-version, modules, md-go, error-taxonomy, error-taxonomy-tests, release-scripts, untested-exports, test-skips, test-matrix, file-size, package-surface, cycles); the Test step's shape is proven by two full green shuffled runs plus one coverage-producing run; the coverage step is proven with `--profile`; lint is green over all modules **except** `testutil`/`metaengine` first time around, which are now 0 issues; API stability, md-go and the doc corpus all ran inside the green `#verify-fast`. The one step never witnessed end to end is the Race leg and the bench gate inside the same composed invocation.

**Recommendation:** run `nix run .#can-run-composed-gate -- --wait-loop && nix run .#verify` from a quiet window. Do **not** `VERIFY_FORCE=1` here — the bench gate would measure a machine at load 50+, which is the exact failure mode the guard exists to prevent.

---

## §c Still open in the plan tail (T10, T13–T20, T22, T24-family decision, T25)

| Task | State | What actually blocks it |
| --- | --- | --- |
| T10 mutation gate | not started | Needs a tool decision (gremlin vs mutants vs AST operator swap) and PR-scoped time-boxing. Still owns the zero-assertion idea, correctly. |
| T13 `synctest` wave 1 | not started | L-sized; `scheduling` ticker/deadline tests must move without touching `scheduling.WithClock` (ADR-0153 D4 keeps those for system-level scenarios). |
| T14 planner model-based test | not started | Needs a reference model for the cost planner; L. |
| T15 differential engine harness | not started | Needs one operation script + a reviewed divergence allowlist; L. |
| T17 executable examples | not started | Absorbs the `GOWORK=off examplePaths` TODO row. |
| T18 tail-latency matrix | not started | Nightly + quiet window, i.e. the resource this session did not have. |
| T19 heap/goroutine/disk budgets | not started | Needs a stable measurement harness first (same quiet-window dependency). |
| T20 crash consistency (`kill -9`) | not started | L; needs subprocess orchestration per engine. |
| T22 broker crash matrix | not started | Redis/NATS disconnect + DLQ + commit-publish window; T21's injector now makes the in-process half feasible, the broker half needs live brokers. |
| T24 family decision | **owner** | A tamper currently classifies as `Rejection`, not `Corruption`. Changing a public error family changes consumer branching. |
| T25 HTTP/span/auth pins | not started | SSE CRLF, docserver traversal, span shape, cqrs-htmx auth E2E (companion repo). |

---

## §d Could have done better

1. **The first `#test` of this pass was not `-json`.** One flag would have named the failing package outright instead of costing a full 550 s replay that passed and taught nothing.
2. **I fixed benchkit's budget one call site at a time.** Two `150 s` literals existed; I fixed the first, ran, watched the second fail 40 minutes later. `grep 'parallelTimeoutCtx(t, '` before the first fix would have taken ten seconds.
3. **`#test-repeat` was added in the previous pass without ever being run.** That is the same class of mistake the whole plan targets ("a gate that cannot go red"), applied to a job instead of a check.
4. **The load gate was right and I kept trying anyway.** Three attempts at `#verify` were refused before the wait-loop; the preflight (`can-run-composed-gate`) is published precisely so this is one call, not three.
5. **`nix fmt` failures were dismissed too quickly as "golines under load"** — true, but it took three 8-minute runs to establish that instead of checking `formatted 0 files` + running `gofmt -l` in the first instance.

## §e Owner questions (3)

1. **T12 baseline provenance.** `scripts/test-runtime-baseline.txt` was measured on this host during a **loaded** evening (load1 up to 800 from other sessions) with `-p 4`. CI runs it at 3×/30 s for that reason. Should the committed baseline be re-measured in a quiet window (tightening CI's multiplier to 2×), or is the current 3× CI slack the intended permanent shape?
2. **T24 family decision.** A tampered signature is documented in code as `Rejection`; `docs/error-taxonomy.md` defines `Corruption` as "data integrity violation". Which is the contract — reclassify the error (consumer-visible) or amend the taxonomy?
3. **Does the write path need to honor cancellation?** `metaengine.Apply`/`ApplyIdempotent`/`ApplyBatch`/`InTransaction` succeed under an already-canceled context while reads fail. Making writes abort is a semantic change with a "already committed must not be undone" edge; the matrix test pins today's behaviour either way.

---

## §f Numbers, for the record

- Fuzz inventory: 115 → **116** targets.
- Exported API: 6,334 → **6,366** symbols (32 new, all `testutil`), all named by a test except the frozen tail (1,086 rows now, prune-able to 1,084).
- Skip sites: 238 with reasons; **0** unrouted env gates.
- Runtime budgets: **168** packages; slowest duckdbengine 231.6 s, benchkit 180.5 s, cqrs-bench 123.3 s.
- Coverage: 89 module floors + 168 package floors hold; verified against a **reused** profile.
- `#test` full shuffled run: 429–525 s (was 27 min when the bbolt soak rode along).
- `#verify-fast`: **866 s**, green.
- `nix fmt --fail-on-change`: clean (0 changed) once the host was quiet.
