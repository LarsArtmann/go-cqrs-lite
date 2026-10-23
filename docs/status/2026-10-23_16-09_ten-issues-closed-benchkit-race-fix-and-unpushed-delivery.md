# Session status: ten issues closed, a real data race fixed, delivery still unpushed

**Timestamp:** 2026-10-23 16:09 CEST
**Session window covered:** 2026-10-23 ~14:35 → 16:09 (the continuation after the 14:35 report; that earlier report is a separate file and is not rewritten here)
**Branch:** `master`, **25 commits ahead of `origin/master`**, nothing pushed
**Host:** shared 32-core box, 47-49 concurrent users, load1 45-155 throughout
**Scope of this report:** only what this session did and noticed. No new research.

---

## 0. One-paragraph summary

All ten open GitHub issues are now closed with per-issue evidence comments (file:line, regression test names, consumer numbers). Beyond closing them, the race gate produced the session's most valuable find: a **genuine data race in `benchkit`** (a progress-heartbeat goroutine writing to the user's `io.Writer` after the run had returned) that normal `go test` cannot see. It is fixed, and pinned by two deterministic, mutation-verified tests. `#check-release-scripts` and `#check-coverage` are green, the touched modules are green under `-race`, and the consumer runs reproduce the expected zeros. What did **not** happen: the composed `nix run .#verify` (load never came close to the gate), a full `#test` sweep after the benchkit change, and the push.

---

## a) FULLY DONE

### a1. The ten issues, closed with evidence

| Issue | Result | Evidence cited in the closing comment |
|---|---|---|
| #70 `EventWithID.ID` display form | verified, no code change needed | `metaengine/projectionadapter/typed_decoder.go:73,99` (`evt.StreamID().Get()`), `typed_decoder_test.go` |
| #71 `TypedReader.Scan` silent truncation | shipped `WithExact` + `ErrScanTruncated` | `metaengine/scan_options.go:108`, `metaengine/errors.go:195`, `TestWithExact_MakesTruncationLoud` (`with_exact_test.go:14`) |
| #72 keep-newest-N cap | shipped `WithCap` / `EnforceCap` / `EnforceCaps` | `metaengine/query_config.go:46`, `metaengine/cap.go:144,228`, 5 tests in `cap_test.go` |
| #74 S001 schema names as secrets | fixed, narrower than proposed | `pkg/rules/security/rules.go:69,212`; `TestS001_NoFindingForSchemaVocabularyConst/Variable`; webphone's stale S001 suppression proves it stopped firing |
| #78 E005 cross-package selectors | fixed | `pkg/analyzer/scanner_generic.go:65-77`; `TestE005_NoFindingWhenRegisteredViaSystemCrossPackage` (`new_rules_test.go:319`) |
| #79 E004/F002 catalog builder invisible | fixed **properly this time** (3 root causes) | `scanner_calls.go:136-142`, `type_helpers.go:97`, `scanner_catalog_helpers.go:24,79`, `scanner_emit_helpers.go:118`; 3 new tests |
| #80 C010 checked-error idiom | fixed | `swallow_helpers.go:140,162`; `TestC010_NoFindingForCheckedIfInitDecode` + unchecked control |
| #81 E003 all findings at `go.mod:1:1` | fixed | `e003_e007.go:92-118`; `TestE003_EmitsDistinctIDsPerPackage`; CRM now emits 5 findings at 5 distinct positions |
| #82 F010 metaengine-native graph | fixed | `adoption/scan_in_metaengine.go:14`; `TestF010_NoFindingWithMetaengineGraphADT` + import-only control |
| #83 findings baseline | shipped | `cmd/cqrs-lint/baseline.go`, `main.go:130-131`, 8 tests |

`gh issue list --state open` → **empty**.

### a2. The benchkit data race (new, found by the race gate, fixed)

- **Symptom:** sweep 2 at load1 ~312 produced 10 `race detected during execution of test` failures, all in `benchkit`.
- **Mechanism (from the race report, not guessed):** `(*progressReporter).beat()` runs on the heartbeat goroutine and writes to `ProgressWriter` (`benchkit/progress.go:77`); `stop()` only **closed** the done channel (`:112`) without joining, so `RunRepeated` printed the next `==> repeat N/M` header (`benchkit/repeat.go:83`) while the previous run's `beat()` was still inside `bytes.Buffer.Write` — two writers growing one buffer.
- **Second, independent hole:** `endPhase()` wrote to the same writer with no synchronization against a live `beat()`; the state mutex only guarded `num`/`phase`/`start`.
- **Fix:** (1) `stop()` closes `done` and then `wg.Wait()`s, so "stopped" means the goroutine exited; (2) every progress write goes through one `printf` helper guarded by `writeMu`.
- **Tests:** `benchkit/progress_test.go` — `TestProgressReporter_StopWaitsForInFlightBeat` and `TestProgressReporter_ConcurrentBeatAndEndPhaseSerialized`, both built on a writer that parks inside `Write`.
- **Mutation-verified, both directions:** removing `wg.Wait()` → test 1 fails in 0.00s; removing `writeMu` → test 2 fails.
- **Verification:** whole package `-race` → `ok … 161.509s`; consumers green (`cmd/cqrs-bench` 212s, `metaengine/bench` 21.8s, `stack/bench` build+no tests).

### a3. Gates run green on this tree

```text
nix run .#check-release-scripts      # all release-tooling self-tests, incl. check-nix-hashes
nix run .#check-coverage             # 11/11 modules, 0.0% delta each
nix run .#check-api-stability        # export golden unchanged (6269)
bash scripts/check-md-go.sh          # 1514 blocks valid, 105 baselined, 0 new
bash scripts/verify-docs.sh          # all documentation assertions passed
bash scripts/check-changelog-symbols.sh   # 27 citations honest
nix fmt benchkit/progress.go benchkit/progress_test.go   # 0 files changed
source scripts/go-env.sh; GOWORK=off go test -race -count=1 ./...  in:
  cmd/cqrs-lint, metaengine, metaengine/projectionadapter, projectionhost, benchkit
```

`#check-coverage` deserves emphasis: all eleven re-baselined modules measured **exactly** their documented number (command 90.1, decider 95.8, dispatcher 87.7, event 89.5, id 86.6, kv 73.7, metaengine 83.9, query 90.2, schema 85.3, snapshot 91.6, storage/memory 95.7). That converts "someone lowered the floors" from a suspicion into a measurement.

### a4. Consumer re-runs (binary built from this tree)

```text
crm      61 findings — 0 E004, 0 F002, 0 E005, 0 C010, 0 F010, 0 S001; E003 = 5 findings at 5 distinct positions
webphone 13 findings — 0 E004, 0 F002, 0 E005, 0 C010, 0 F010, 0 S001
         + "stale suppression at enroll.go:25 — rule S001 does not fire here; safe to remove"
```

The stale-suppression line is the strongest available proof for #74: the consumer had been forced to suppress a finding that the rule no longer produces.

### a5. `check-nix-hashes` made observable

An hour earlier I wired the tripwire into `#verify`/`#verify-fast`/`#check-release-scripts`, but its self-test printed **nothing** on success — so a CI log could not prove the legs ran. It now prints one line per fault injection (clean passes, zero-hash caught, `lib.fakeHash` caught, missing file loud, empty hash caught — the last one added this pass).

### a6. Documentation and bookkeeping

- `TODO_LIST.md`: a dated `## cqrs-lint` harvest block (#79 fixture + fragment audit, catalog symmetry, #71 `ScanWithMeta`, #83 hardening ×3), a `## Metaengine — follow-ups` cap block (#72 engine coverage, trim benchmark, timer wiring, `WithCapOnWrite`), a `## CI / Infrastructure` block (coverage-floor-decrease guard, `flake.lock` coherence check, tree-wide `check-nix-hashes`, `#verify-serial`), the stale "cmd/* modules are not gated" row marked **RESOLVED** with the verifying commit, and a new row for the `projectionhost` cadence flake under the existing contention-stall section.
- `docs/status/README.md`: index rows for both 2026-10-23 reports; heading date moved 2026-10-17 → 2026-10-23.
- The 14:35 report gained a banner, §h (close-out, sweeps, §g resolutions, final gate table) and corrected §b rows.
- `CHANGELOG.md`: one `### Fixed` bullet for the benchkit race.

### a7. The three §g questions, settled

1. **Coverage floors** — resolved by measurement (a3). Keep; a guard row is queued.
2. **`flake.lock`** — the revert stands: `#check-md-go` (the gate the un-re-pinned bump broke) and every nix gate are green on the reverted lock.
3. **Delivery ordering** — re-answered below (§c, §g).

---

## b) PARTIALLY DONE

| Item | State | What is missing |
|---|---|---|
| `#72` cap primitive | memory engine only | SQL pushdown branch (`PushdownMapScan` + `DeleteRows`) and temporal branch (`MapDeleteAt` + `notifyLive`) untested; no trim-cost measurement at 10k/100k rows; no default `system.Timer` wiring (deliberate, but undocumented in a consumer-facing place) |
| `#83` baseline | semantics + 8 unit tests | no end-to-end run through the real binary on a fixture repo; no `--fail-on-stale-baseline`; no README CI recipe; `--update-baseline` exits 0 unconditionally; health score still counts baselined findings |
| `#71` `WithExact` | shipped + tested | `ScanWithMeta()` deliberately not shipped; `WithExact` + `WithCursor` (page 2 of an exact scan) untested |
| `#79` catalog detection | works on the real consumer | no typed-load fixture (see b-note below); helper matching is by **bare name**, so same-named helpers in different packages can cross-register; `simple.Command`/`simple.Query` ids not resolved (event-only symmetry) |
| Race verification | 3 workspace sweeps + 5 targeted `-race` modules | **no fully green workspace sweep exists**: sweep 1 had the pre-existing projectionhost flake, sweep 2 had the benchkit race, sweep 3 was environment-polluted. The touched set is green; the whole-workspace claim is not made. |
| Full `nix run .#test` | not re-run after the benchkit change | module + 3 consumers green instead (210s of real tests), but the repo's own full gate is unproven since the edit |
| Composed `nix run .#verify` | never launched | load1 78-155 vs the `< 10` guard, all session |
| Authored commit | impossible now | the auto-commit daemon absorbed every change into `chore:` commits; nothing of mine is separable without rewriting history |
| My own deliverable set | comments + closes done | the push (see §c, §g1) |

**b-note — why "typed-load fixture" is not nitpicking.** `BuildContextFromSource` builds ASTs with empty `types.Info`, and `IsQualifierFor` degrades to qualifier-name matching in that mode. That is precisely how the previous session's #79 fix passed its tests and failed in production. Until a fixture loads a real versioned module with type info, the same class can recur silently.

---

## c) NOT STARTED

1. **Push.** `master` is 25 commits ahead of `origin/master`.
2. **Composed `#verify`.**
3. **Full `#test` sweep** after the benchkit edit.
4. **All 20+ harvested TODO rows** (they are next work, by definition).
5. **Adoption proof for the new APIs.** Nothing in the fleet calls `WithCap`/`EnforceCap`/`EnforceCaps`, `WithExact`, or `--baseline`. They are unit-tested, not adoption-tested.
6. **Tag wave.** `cqrs-lint` fixes (#74, #78-#82), `metaengine` (#71, #72) and `benchkit` (race) are all unreleased.
7. **Skill-reference updates** for the new surfaces (`WithExact`, `WithCap`/`EnforceCap`, `--baseline`) — `recipes.md`/`core.md` do not mention them, and `cmd/doc-check` has not been asked about them.
8. **`mkApp` env-chain fix** (the class in §d2) — not attempted.
9. **Archive/banner sweep for the two 2026-10-23 reports** beyond the top banner.
10. **`flake.lock` re-bump done properly.**

---

## d) TOTALLY FUCKED UP

1. **I trusted a todo list over reality.** I marked "Composed `#verify`" as completed when it had never launched, and "Run `#test-race`" as completed while a race sweep was still running and failing. Both were self-inflicted bookkeeping lies that I had to correct in a second call. A status surface that asserts unverified facts is worse than no status surface.
2. **I did not source `scripts/go-env.sh` for `nix run` apps, and it cost two sweeps.** The repo's own TL;DR says to source that chain "before any go/buildflow command". I did it for per-module `go test` but not for `nix run .#test-race`, so the sweep inherited the ambient `/mnt/buildcache`. When that shared cache was evicted mid-run, ~30 `cmd/cqrs-lint` tests failed with `loading compiled Go files from cache … no such file or directory` — a failure mode indistinguishable, at a glance, from a code break. Re-running the same module with the env chain was green immediately.
3. **I spent ~30 minutes reasoning about the projectionhost cadence test's mechanism and ended with none.** I read the reporter, the runner, the subscriber, and the checkpoint store, and could not produce a mechanism consistent with the reported elapsed time. That time produced one useful negative result (it is not double-apply) and no diagnosis. The correct move was to cap the analysis and instrument, or to stop at "intermittent, controls done" — which is where it landed anyway.
4. **I wrote ten external comments that claimed "Fixed on master" before checking whether `master` == `origin/master`.** It is not: 25 commits are local. I had to append a delivery note to all ten. Checking push state costs one command and should precede any external "done" claim.
5. **I shipped a self-test that could not prove itself.** The `check-nix-hashes` self-test I added earlier printed only on failure, so its green CI line was indistinguishable from "never ran". Fixed this pass, but the original design was the defect.
6. **I never proved the tree formatter-clean.** I scoped `nix fmt` to my two files because a parallel session had the tree open, so the repo-wide format gate is unproven for this state. Defensible, but it means one more thing the next full gate run could surprise us with.
7. **Attempted edits on a file another session was actively writing.** `edit` rejected twice on `TODO_LIST.md` (external modification, stale read). No damage, but I was one careless step from clobbering their rows; the read-then-anchor discipline is what prevented it.

---

## e) WHAT WE SHOULD IMPROVE

1. **`mkApp` should source the env chain.** Every `nix run .#test*` app inherits whatever cache the shell carries. One line in the `mkApp` prelude (`source scripts/go-env.sh` when present) would retire an entire class of "cache entry not found" false reds. This is the highest-leverage fix the session surfaced.
2. **Make cache corruption diagnosable.** `srcfiles … cache entry not found` is presented as a package-load error. cqrs-lint already prints "the project likely does not compile"; it should say "the go build cache is corrupt/evicted — re-run with a private `GOCACHE`" when the error matches that shape. Two words of routing would have saved this session twenty minutes.
3. **Race sweeps should default to the touched set.** The workspace sweep is a release gate; the per-module sweep is the development gate. I ran the expensive one three times and the cheap one once.
4. **Success must print.** Generalize what `check-nix-hashes` now does: every `--self-test` prints one `✓ PASS` per leg, so a green CI log is evidence rather than silence.
5. **External claims need a state check first.** Before writing "fixed on master", verify `git log origin/master..HEAD`. Before writing "verified", name the command and its date.
6. **Flake analysis needs a time box.** Ten minutes of reasoning, then instrument or file it as observe-only with controls. Never let it silently consume a phase.
7. **A todo list is not a status report.** Mark items complete only after the verifying command's exit code is in hand.
8. **Don't claim an aggregate you have not run.** "Touched set green" and "workspace green" are different sentences; the report should always say which one it means (this report tries to).

---

## f) UP TO 50 THINGS TO GET DONE NEXT

**Delivery (blocking the value of everything above)**

1. Push `master` (25 commits) once the parallel session's `system/` work is at a commit point.
2. Re-read the ten closing comments after the push; add the push commit range if the owner wants a pointer.
3. Decide whether the two red gates (`system/adapter_event.go` 351 lines; the `resolveStreamKey` clone pair) belong to the parallel session before or after the push.

**Verification still owed**

4. One composed `nix run .#verify` in a genuine quiet window, log path recorded.
5. One full `nix run .#test` after the benchkit edit (146-package baseline).
6. One fully green `nix run .#test-race` (touched set + workspace) once the tree is quiet.
7. Re-run `#check-file-size` and `#check-duplication` after the parallel session's `system/` work lands, to confirm both return green with no action from me.
8. Reproduce-or-retire the projectionhost cadence flake: `-race -count=20` under a deliberate soaker, or accept observe-only and stop paying attention to it.

**Process / infra fixes this session proved necessary**

9. Make `mkApp` source `scripts/go-env.sh` so nix apps cannot inherit `/mnt/buildcache`.
10. Add a `gocache` sanity preflight to `#verify` (detect a cache that other processes are evicting; fail with the remedy, not with 30 test failures).
11. Teach cqrs-lint's package-load diagnostic to name the corrupt-cache class.
12. Add `✓ PASS` lines to every remaining `--self-test` that is silent on success.
13. Write the `#verify-serial` fallback tier (same gate list, `-p 1`, per-gate timeouts, no load precondition).
14. Document in `AGENTS.md` that `nix run` apps do not source the env chain (until item 9 lands).
15. Rowed already, still open: coverage-floor-decrease guard; `flake.lock` coherence check; tree-wide `check-nix-hashes`.

**The newly shipped APIs need adoption, not more unit tests**

16. Wire `WithCap`/`EnforceCap` into one real consumer (CRM's bounded read models are the obvious candidate) and record the trim cost.
17. Wire `WithExact` into one consumer where "is this the whole set?" matters (an export or an integrity check).
18. Adopt `--baseline` in one consumer with a checked-in `.cqrs-lint-baseline.json` and a CI step, which is also the missing end-to-end test for #83.
19. Add the `--baseline` CI recipe to `cmd/cqrs-lint/README.md`.
20. Add `--fail-on-stale-baseline`.
21. Guard `--update-baseline` against accidental CI use (require an explicit confirm flag, or refuse when `CI=true` without an opt-in).
22. Decide whether the health score should exclude baselined findings; document either way.

**Benchkit (newly touched, so finish the job)**

23. Consider a package-level note that `Config.ProgressWriter` must be goroutine-safe if shared across concurrent runs.
24. Check whether any other benchkit long-lived goroutine (resource sampler) has the same "signalled but not joined" shape — the sampler is the next candidate.
25. Add a benchkit-level test that runs two `RunRepeated` calls concurrently against one buffer, to pin the contract above.

**`#79` hardening**

26. Typed-load fixture: real versioned module on disk with full `types.Info`.
27. Audit every rule that still builds import fragments by hand.
28. Resolve `simple.Command`/`simple.Query` catalog ids (symmetry with events).
29. Document the bare-helper-name matching limitation in the rule's doc comment and in `RULES.md`.
30. Consider a collision guard: if two packages define the same helper name, refuse to replay rather than guess.

**`#71` / `#72` follow-ups**

31. `ScanWithMeta()` decision (or an explicit "no" recorded in the CHANGELOG).
32. `WithExact` + `WithCursor` page-2 test.
33. Cap test on a SQL engine.
34. Cap test on a temporal engine.
35. `EnforceCap` benchmark at 10k/100k rows, numbers appended to the COOKBOOK.
36. `system.Timer` wiring decision for `EnforceCaps`.
37. `WithCapOnWrite` decision for SQL engines.

**Docs / release**

38. Add `WithExact`, `WithCap`/`EnforceCap`, and `--baseline` to the skill references (`core.md`, `recipes.md`) and re-run `cmd/doc-check`.
39. Cut a tag wave covering cqrs-lint, metaengine and benchkit; run the per-module suites over the changed set first (the `batch-release.sh verify=ok` lesson).
40. After tagging, roll the linter binary to the fleet and re-measure CRM/webphone with the released build.
41. Archive or banner the two 2026-10-23 reports once the next report lands, so the newest is unambiguous.
42. Re-bump `flake.lock` properly (`nix flake update` + re-pin each derived vendor hash) if the owner wants the seven sibling inputs.
43. Add the "env chain for nix apps" incident to the crush-config `references/lessons.md` (cross-project lesson: a documented procedure that is not enforced by the tool will be skipped).
44. Reconcile `AGENTS.md`'s quick-reference rows with this session's new gate (`check-nix-hashes` is already there; confirm after any merge with the parallel session's edits).

**Hygiene**

45. Re-run `nix fmt` tree-wide once the parallel session commits, and confirm 0 changes.
46. Confirm `docs/api_surface.txt` needs no regeneration for this pass (it did not: all benchkit changes are unexported).
47. Re-run `#check-error-taxonomy` and `#check-modsums` on the next quiet window (both were green earlier in the session, neither was re-run after the benchkit edit, which adds no codes or deps).
48. Sanity-check that no test added this pass mutates `os.Stderr` under `t.Parallel()` (the new progress tests do not).
49. Status index row for THIS report is added (`docs/status/README.md`, top of the live table); confirm it renders after any parallel-session index reshuffle.
50. Keep the ten closed issues closed only if the push lands; otherwise reopen with the delivery note as the reason.

---

## g) THREE QUESTIONS I CANNOT ANSWER MYSELF

1. **Push now or hold?** `master` is 25 commits ahead. Most are this session's issue fixes, but the batch also contains a parallel session's `system/` sticky-stream-key work and catalog changes, and the daemon has already committed that work mid-flight (`system/adapter_event.go` is 351 lines and there is one `resolveStreamKey` clone pair, which is why two gates are red). Pushing publishes their committed state; waiting risks the daemon committing a half-finished change and the window closing anyway. I will not push without your word.

2. **Do you want `mkApp` to source `scripts/go-env.sh`?** It is a one-line change in the flake prelude that affects every `nix run` app in the repo, and it is what would have prevented three sweeps' worth of cache-noise reds. It changes behaviour for every consumer of those apps, so it is your call, not mine.

3. **Which consumer should adopt the new APIs first, and should that ride the tag wave or precede it?** `WithCap`/`EnforceCap`, `WithExact` and `--baseline` are shipped but have **zero** callers in the fleet. Their only evidence today is unit tests, and this session's central lesson is that unit tests can certify a dead path. If you name the consumer (CRM is the obvious one for `--baseline` and caps), I will wire it and use it as the release gate for those three surfaces; otherwise they ship on unit tests alone.

---

## Appendix: exact commands run this pass

```text
date                                                              # 2026-10-23 16:09 CEST
nix run .#check-release-scripts                                   # rc=0, all self-tests
nix run .#check-coverage                                          # rc=0, 11/11 at 0.0% delta
nix run .#check-api-stability                                     # rc=0
bash scripts/check-md-go.sh                                       # rc=0, 1514 blocks
bash scripts/verify-docs.sh                                       # rc=0
bash scripts/check-changelog-symbols.sh                           # rc=0, 27 citations
nix fmt benchkit/progress.go benchkit/progress_test.go            # 0 changed

# benchkit race fix verification
cd benchkit && GOWORK=off go test -race -count=1 .                # ok 161.509s
cd benchkit && GOWORK=off go test -race -run TestProgressReporter -count=3 .   # ok
# mutation 1: remove wg.Wait() -> TestProgressReporter_StopWaitsForInFlightBeat FAILS
# mutation 2: remove writeMu   -> TestProgressReporter_ConcurrentBeatAndEndPhaseSerialized FAILS

# targeted -race (with the env chain)
source scripts/go-env.sh && cd cmd/cqrs-lint              && GOWORK=off go test -race -count=1 ./...   # ok
source scripts/go-env.sh && cd metaengine                 && GOWORK=off go test -race -count=1 ./...   # ok
source scripts/go-env.sh && cd metaengine/projectionadapter && GOWORK=off go test -race -count=1 ./... # ok
source scripts/go-env.sh && cd projectionhost             && GOWORK=off go test -race -count=1 ./...   # ok
cd cmd/cqrs-bench   && GOWORK=off go test -count=1 ./...           # ok 212.737s
cd metaengine/bench && GOWORK=off go test -count=1 ./...           # ok 21.788s
cd stack/bench      && GOWORK=off go test -count=1 ./...           # build ok, no tests

# consumers (binary built from this tree)
/tmp/cqrs-lint-bin . --format json   # crm: 61 findings, zero target rules
/tmp/cqrs-lint-bin . --format json   # webphone: 13 findings, zero target rules + stale S001 suppression

# workspace race sweeps (three, none fully green; see b)
nix run .#test-race   # sweep 1: 145 ok / projectionhost cadence (pre-existing)
nix run .#test-race   # sweep 2: 145 ok / benchkit data race (fixed)
nix run .#test-race   # sweep 3: env-polluted (ambient /mnt/buildcache eviction + parallel-session WIP)

# issues
gh issue comment <n> --body-file /tmp/issue-<n>.md    # ×10
gh issue close <n> --reason completed                # ×10
gh issue list --state open                           # empty
```

State at report time: working tree clean, `master` 25 commits ahead of `origin/master`, 0 open issues, load1 45.
