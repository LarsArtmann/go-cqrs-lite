> **RESOLVED-BY-ROUTING — docs-health 9th pass (2026-09-20):** Superseded same day: the typedfixture root cause was found and fixed (stale fixture pins — 09-40 §a2, not an env divergence), the composed `#verify` went GREEN with the S03 record (16-39 report), and the binary-cleanup wave was verified (09-40 §a2). The open tail (T18b) lives in [TODO_LIST.md](../../../TODO_LIST.md).

# Verify tail: edge-poll race FIXED; cqrs-lint typedfixture blocks composed green (3 attempts)

> Session start: took over the 18-05 lint-debt handoff (report since archived by the
> 20-08 docs-health pass). Directive: execute the pending todo list (verify → verify-ci
> → T18b → pre-tag gates). Owner interrupt 00:19: full status, then WAIT.
> Machine reality all session: a 92-tag release train shipped from a PARALLEL live
> session (`batch-release.sh --smoke-all` + `tag-release.sh --smoke` still running at
> 00:19, taskmanager smoke stuck >1h), a foreign Node/V8 nix build driving load
> 18–167, and the auto-commit daemon absorbing every wave mid-flight.

## a) FULLY DONE (this session)

1. **Verify attempt 1 (job from prior session) triaged and root-caused.**
   rc=1 with a single race-phase failure:
   `TestBundle_RunProjections_GraphProjection` — "expected at least 1 edge from
   alice, got 0". Confirmed load-transient (5/5 standalone `-race` runs green), then
   fixed the actual shape flaw instead of re-running until lucky.
2. **`integration/graph_projection_test.go` fixed at root.** The test polled until 2
   user NODES appeared, then immediately asserted the follow EDGE — but
   `user.followed` projects after the `user.created` events, so under race + load the
   edge legitimately lags. The edge now gets its own 3s deadline-bounded poll
   (mirrors the file's existing users-loop pattern). Verified: build green, 5/5
   `-race` runs green, `#lint-module integration` 0 issues, `nix fmt` clean.
3. **Docs folded:** CHANGELOG [Unreleased] Fixed section extended with the load-flake
   fix (test name + mechanism + 5/5 evidence); new gotchas-testing.md lesson
   "Poll the FINAL derived condition, not an upstream one" with the full story.
4. **The 20-08 docs-health session's unrun gate debt partially paid:** its §c1 left
   `check-doc-links.sh` + `check-changelog-symbols.sh` + doc-check unrun after moving
   70 files. I ran the two light ones GREEN (791 relative links / 372 files / 0
   broken; 6 symbol citations honest). doc-check rides inside `#verify`.
5. **Fiction killed on sight:** the 20-08 report cited `scripts/check-rows.py` —
   which does not exist anywhere in the repo. Dated correction appended to its §c1
   (committed file, its session ended; safe to annotate).
6. **`#check-lint-config` GREEN after three daemon auto-commit waves** — the
   depguard allow-list and exhaustruct canaries survived; no re-mangling.
7. **Forensics that unblocked the diagnosis:** identified that the 18-05 report was
   archived, that the 92-tag release train ALREADY shipped (tag-wave owner question
   from the handoff is moot), and that `scripts/verify-docs.sh` IS the `#verify`
   wrapper (S03's "verify-docs.sh" leg is covered by any composed green).

## b) PARTIALLY DONE

1. **Composed `#verify` GREEN — 3 attempts, 0 green.**
   - Attempt 1 (prior session, job 28F): race-phase edge flake → FIXED (a2).
   - Attempt 2 (22:16–22:52): failed on `cmd/cqrs-lint` P014/V007
     "typed fixture context should have typed confirmations available (auto mode)"
     — diagnosed as a MID-FLIGHT EDIT RACE: the release session's pin-sweep rewrote
     `cmd/cqrs-lint/testdata/typedfixture/go.mod`+`go.sum` (event v4.11.0→v4.11.1,
     id v4.6.0→v4.6.1) at 22:50:26, the exact window verify2's Test phase loaded the
     fixture. Transient: tags exist, and the tests passed standalone at 22:53.
   - Attempt 3 (23:16–~23:45, CLEAN committed tree, dirty=0 throughout): the SAME
     two packages failed the SAME way — so this failure is NOT churn; it is a real
     environment divergence: my standalone run (GOMODCACHE=/tmp/gomod-verify) passes,
     `#verify`'s Test phase (its own GOMODCACHE/proxy env, from flake.nix's mkApp)
     fails. Working hypothesis at interrupt time: the fixture's `GOWORK=off`
     `packages.Load` cannot resolve the freshly-bumped v4.11.1/v4.6.1 pins in the
     verify env's module cache and SILENTLY degrades to syntax-only (LoadErrors is
     empty — the harness's `TypedConfirmations()` gate is the only tell). Was reading
     flake.nix's verify app env (line ~1509) when interrupted. NOT yet reproduced
     under verify's exact env; NOT yet fixed.
2. **S03 acceptance record** (date + commit + durations for a quiet-window composed
   green) — blocked on b1.
3. **Stray-binary cleanup verification:** noticed the daemon had committed a 5.9MB
   `testdata/typedfixture/typedfixture` binary (plus `catalog/ec-fixture` and two
   20–27MB example binaries from `go build ./...` during the release train). A
   concurrent cleanup session staged deletions + a `.gitignore` edit; the daemon
   absorbed it all at 23:15 (commit 2af20d19a, 73 files). Did not yet verify the
   deletions landed complete or that `.gitignore` covers every pattern.

## c) NOT STARTED

1. `#verify-ci` per-module matrix — sequenced behind composed verify green.
2. **T18b:** `#load-sweep` + `benchmark-regression.sh --save` baseline regen — the
   load gate (<~10) never opened this session (observed 5.67 for one instant at
   start, then 18–167 the rest of the night; Node/V8 build + release smokes).
3. Pre-tag gates: `#vulncheck`, `#check-arch`, `#check-coverage`.
4. Standalone bbolt soak in a quiet window.

## d) TOTALLY FUCKED UP (honesty ledger)

1. **Launched verify2 without checking for live release processes.** The handoff
   discipline said re-check git/log — I did, tree looked fine — but `ps` would have
   shown `batch-release.sh --smoke-all` alive since 21:46. Its sweep mutated the
   fixture mid-run and burned a 36-minute verify. Lesson: the pre-composed-gate
   checklist must include `ps aux | grep -E 'batch-release|tag-release|go (mod|build|test)'`
   and a dirty-go.mod count, not just git state.
2. **Launched verify3 with the release session STILL alive (relprocs had grown 2→4).**
   I consciously bet that smokes only build in temp dirs; the bet failed on a
   different axis (env divergence, d1 above) — but two consecutive verify attempts
   against a live release session is a pattern, not bad luck. Should have either
   waited the smokes out or first reproduced the cqrs-lint failure under the
   verify app's exact environment (cheap, minutes) instead of paying 30 min per
   composed attempt to rediscover it.
3. **One wasted edit round on the 20-08 correction** — old_string typo
   ("repopreferences"); the tool's closest-match hint made the retry trivial, but
   the read-then-copy discipline slipped.
4. **Noticed the committed 5.9MB binary at 22:53 but under-escalated** — I reasoned
   "foreign session's file, daemon will absorb" and moved on; it took a DIFFERENT
   concurrent session to stage the cleanup 3 minutes later. Right call on ownership,
   but flagging it in the owner channel immediately would have been the honest
   move in a shared moving tree.

## e) WHAT WE SHOULD IMPROVE (structural, observed this session)

1. **The typedfixture harness fails SILENTLY.** `BuildContext` returned zero
   LoadErrors while delivering no type information; the ONLY symptom is downstream
   tests failing on `TypedConfirmations()` with a message that does not say WHY
   (unresolved module? missing sum? proxy miss?). The harness should surface the
   `go list` error output when typed info is unavailable — this exact ambiguity
   cost this session two 30-minute verify attempts to separate churn from env.
2. **Pin-sweeps touch `testdata/typedfixture/go.mod` but nothing re-runs the
   cqrs-lint typed tests before the daemon absorbs the bump.** The fixture is a
   load-bearing test dependency; the sweep (or its CI) should run
   `cd cmd/cqrs-lint && GOWORK=off go test ./pkg/rules/...` after bumping it.
3. **`#verify`'s Test phase env vs the documented interactive env chain diverge**
   (different GOMODCACHE at minimum) — the canonical "reproduce what verify saw"
   recipe should be written down (or #verify should print its env into the log
   header) so a standalone repro actually reproduces.
4. **Release-train hygiene:** `go build ./...` at module scope wrote main-package
   binaries into the tree and the daemon committed megabytes of them (tag-release.sh
   line 556 already knew the `-o throwaway-dir` cure; the sweep paths did not use
   it). The .gitignore addition patches the symptom; the sweep scripts should take
   the `-o` cure too.
5. **Pre-gate checklist as a script:** the TODO already carries `wait-for-quiet.sh`;
   extend the idea to a `can-run-composed-gate.sh` that asserts: no release scripts
   alive, dirty=0 (or stable across 2 samples 60s apart), load under threshold. This
   session is the Nth payer of the mid-gate-mutation tax.

## f) NEXT (ordered; up to 50, realistically the first 12 are the work)

1. Reproduce the cqrs-lint P014/V007 failure under `#verify`'s EXACT env (read
   flake.nix mkApp env first — was mid-read at interrupt).
2. Fix per root cause: harness fail-loud with the underlying `go list` error, and/or
   make the fixture resolve its deps deterministically (vendored? pinned cache?
   GOFLAGS=-mod=mod with explicit proxy?).
3. Decide whether the fixture should be swept at all (exclude from pin-sweeps vs
   sweep + test in the same change).
4. Composed `#verify` GREEN (attempt 4) once smokes are done AND tree stable.
5. Record S03 acceptance: date + commit + durations in TODO_LIST/plan.
6. `#verify-ci` per-module matrix.
7. T18b `#load-sweep` (needs load <~10).
8. `benchmark-regression.sh --save benchmarks/benchmark-baseline.txt` (same window).
9. `#vulncheck`, `#check-arch`, `#check-coverage` pre-tag gates.
10. Verify the binary-cleanup wave landed complete (no stray binaries left,
    `.gitignore` patterns cover all four names seen).
11. Standalone bbolt AutoCRUD soak in the same quiet window.
12. `wait-for-quiet.sh` + `can-run-composed-gate.sh` tooling (TODO row + e5).
13. Upstream filing: exhaustruct v5.0.3 `skippedNamed` panic (carried from handoff).
14. Upstream filing: go/types + x/tools race (carried from handoff).
15. The 20-08 §g2 owner ratifications still pending (annotation gate interpretation).
16. TODO_LIST `[x]`-row deletion sweep (the pre-tag-wave cleanup the 20-08 pass deferred).
17. Module-map + FEATURES census rows (73 of 95) from the 20-08 harvest.
18. If the fixture fix changes `analyzer.BuildContext` behavior: api-stability golden
    regen + doc-check in the same edit (repo procedure).
19. Re-check whether verify1's repaired `.golangci.yml` needs the `run.go` pin bumped
    again if any module moves past 1.27.1.
20. Post-green: confirm the archived 18-05 report's forward items are all struck or
    carried (docs-health pass-scoped interpretation).

## g) QUESTIONS FOR THE OWNER (cannot be figured out from here)

1. **The live release session:** `batch-release.sh --smoke-all /tmp/wave_all.txt`
   (21:46) is still running at 00:19, and its `tag-release.sh --smoke
   example/taskmanager v0.2.0` child has been alive >2h — far past the ~2min proxy
   poll bound, so it is likely wedged in install+probe. Is that session still
   yours/attended? Do I have your blessing to kill the wedged smoke (it is blocking
   nothing of mine directly, but its parent will keep the machine busy), or should
   it be left strictly alone?
2. **Fixture policy for cqrs-lint:** when the root cause is confirmed, do you want
   `testdata/typedfixture` (a) EXCLUDED from release pin-sweeps (stays on older
   published tags, stability over freshness), or (b) still swept but with the typed
   tests made part of the sweep's own verification, plus the harness made fail-loud?
   Both are defensible; (b) is more moving parts, (a) decouples your linter's test
   fixture from the release train's cadence.
3. **Quiet-window priority:** the remaining heavy gates (composed #verify, #verify-ci,
   T18b load-sweep + benchmark baseline) all want the same quiet window, and tonight
   the machine never gave one. When it opens (or on your signal), in which order —
   composed-verify-first (S03 record), benchmarks-first (they need the STRICTEST
   quiet), or sequential overnight verify→verify-ci→benchmarks in one unattended
   block?

— Session paused per owner instruction. No gates running; tree clean at 00:19;
verify3 log preserved at /tmp/verify3.log (attempts 1–2 at /tmp/verify.log,
/tmp/verify2.log).
