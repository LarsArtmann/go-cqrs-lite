# Status Report — Issues #84 + #85: eventtest global-read setters + scenario versioned decide

**Date:** 2026-10-25 08:52 CET
**Scope:** This session only — review + implementation of
[LarsArtmann/go-cqrs-lite#84](https://github.com/LarsArtmann/go-cqrs-lite/issues/84) and
[#85](https://github.com/LarsArtmann/go-cqrs-lite/issues/85), plus everything noticed along the way.
**Format note:** `.md` at explicit user path demand (status-report skill default is HTML; user instruction wins — override flagged, not propagated).

---

## a) FULLY DONE

All items below are implemented, tested, and gate-verified in the working tree. Caveat: history was absorbed by the auto-commit daemon into `chore:` commits (`a52cb26fa`, `79c8b1cbf`, `62bf3a7cc`) — no authored per-issue commits exist.

1. **Issue #84 claim verification** — Confirmed against source: `readAllFn`/`readFromFn` fields existed and were honored (`event/v4/eventtest/fake_store.go:26-27,152,188`) but had no public setters while the rest of the family (`SaveFn`…`AppendBatchFn`) did. Also verified the consumer pain is real: InboxClean keeps a local `errLoader` fake purely for the `ReadAll` failure seam (`~/projects/InboxClean/cmd/inboxclean/cqrs_commands_test.go:24`).
2. **Issue #84 implementation** — Added `ReadAllFn(fn func() ([]event.Event, error)) *FakeStore` and `ReadFromFn(fn func(afterEventID id.EventID, limit int) ([]event.Event, error)) *FakeStore` to `eventtest.FakeStore`, exact family shape (plain assignment, chainable). Evidence: api golden `docs/api_surface.txt:2338,2340`; tests `TestFakeStore_ReadAllFn_Setter`, `TestFakeStore_ReadFromFn_Setter` (override fires + chaining identity) green.
3. **Issue #84 docs** — eventtest README setter table gains `ReadAllFn` / `ReadFromFn` rows.
4. **Issue #85 claim verification** — Confirmed: `scenario.DecideFunc[Cmd, State]` had no version slot; InboxClean's `adaptDecide`/`adaptLabelDecide` wrappers hard-code the version (`~/projects/InboxClean/internal/domain/email/scenario_test.go:16-33`). Verified the correct derivation from the decider itself: current version = count of loaded history (`decider/load.go:223` → `event.Version(len(events))`), and InboxClean's hand-maintained versions exactly equal `len(given)` — so count-based derivation is right by both contracts.
5. **Issue #85 implementation** — Added `scenario.VersionedDecideFunc[State]` (`func(state State, currentVersion event.Version) ([]event.Event, error)` — the exact `decider.DecideFunc[State]` shape, so domain decides pass BY NAME, zero adapter, which is the issue's stated goal) + `DeciderScenario.WhenVersioned(decide)` method. All decide call sites (`decideEvents`, `ThenError`, `ThenState`) route through a new `runDecide` picker; `requireWhen` accepts either. Mutual exclusion: `When` and `WhenVersioned` nil each other (last call wins). Evidence: api golden `docs/api_surface.txt:4806,4811`.
6. **Issue #85 tests** — 7 new tests in `scenario/dsl_test.go`: version-receive table (0/1/3 given events → versions 0/1/3), next-version stamping (2 given → emitted v3, the drift-guard the issue asked for), `ThenError`, `ThenState`, and both last-call-wins directions. Package-level fixture `decideIncrementVersioned` doubles as compile-level proof of direct `decider.DecideFunc[State]` interop. Suite green incl. `-race`.
7. **Issue #85 docs** — scenario README (API table row + Design bullet explaining the no-decider-import shape match and the anti-drift derivation) + skill reference `.agents/skills/go-cqrs-lite/references/modules.md` scenario row.
8. **API-surface ritual** — Golden regenerated (`--update`, 6309 exports, 4 new symbols); `TestEvery` meta-test green (after fixing the go.sum blocker, see d).
9. **CHANGELOG** — Two `[Unreleased]` Added rows citing `eventtest.ReadAllFn`, `eventtest.ReadFromFn`, `scenario.WhenVersioned`, `scenario.VersionedDecideFunc`; `check-changelog-symbols.sh` green (11 citations honest).
10. **Gate sweep — ALL individually green:** `#lint` (88/88 modules, 0 issues; fixed one `wsl_v5` finding in my own first draft), `#check-arch`, `#check-duplication` (0 new clone groups, baseline 189), `#check-md-go`, `#check-api-stability` (incl. race), `#check-lint-config`, `#check-docserver-css`, `#check-templ`, `#check-bench-gate`, `#check-coverage` (all within ±2%), `#check-architecture-reference`, `#check-turso-version`, doc-check (1356 refs, 57 packages), changelog-symbols, `nix fmt` (0 changed), file-size for MY files (298/291/308 + 384-line test file — tests exempt).
11. **Repo-wide go.sum tidiness sweep** — 91 modules carried stale entries (deletion-only diffs); `TestEveryModuleGoSumIsTidy` mandates the tidy; swept all modules mechanically, gate green. (Root cause analysis: NOT STARTED, see f.)
12. **Blast radius** — Change is purely additive; internal consumers (`example/goal-shaped-app`, `example/taskmanager`, cqrs-lint test, stack/decider/event test files) covered by verify-fast's build+vet legs (passed before the load-noise abort) and module suites.

## b) PARTIALLY DONE

1. **`#verify-fast` as ONE chained green run** — Works: every leg individually verified green (see a-10). Missing: the chained run aborted at the Test leg because `systemscenario` + `systemtest` hit `context deadline exceeded` under machine load 57–77 (36 users; the documented oversubscription flake class). Both packages green in isolation (1.9s / 1.1s vs 13s / 65s timeouts). Effort to finish: M (one quiet-window rerun). Blocker: machine load, nothing else.
2. **Issue close-out on GitHub** — #84/#85 are still OPEN. The `github-voice` skill is loaded, closing-comment approach decided (terse maintainer register, no AI footer), drafts mentally composed but NOT posted. Effort: S. Blocker: waiting — see g-2.
3. **Authored commit history** — AGENTS says commit at phase boundaries when authored history is wanted; I did not, so the daemon absorbed everything into 3 `chore:` auto-commits. Consequence: close comments cannot cite clean per-issue hashes. Not fixable retroactively without rewriting daemon history (won't). Lesson recorded in e.
4. **Whole-repo race leg** — `-race` ran standalone for scenario, eventtest, systemscenario, systemtest, and api-stability only; the full-repo race sweep was part of the aborted chain. Effort: S under quiet load.

## c) NOT STARTED

1. **Tagging `scenario` (last tag v4.4.3) and `eventtest` (pinned v0.4.0 by InboxClean)** — CRITICAL: without new tags, consumers on pinned versions CANNOT see `WhenVersioned`/`ReadAllFn`; the work is tree-only until released. Note `batch-release.sh verify=ok` builds but does NOT test — per-module suites must run over the changed set before cutting (2026-10-06 lesson). Priority: Critical.
2. **InboxClean-side migration** — Library side done; consumer side untouched: delete `errLoader` (use `FakeStore.ReadAllFn`), delete `adaptDecide`/`adaptLabelDecide` (use `WhenVersioned`), then their ADR-027 `systemscenario` cutover (their plan, their repo).
3. **Fixing the 3 pre-existing file-size offenders** (see d-1) — not mine, deliberately not bundled.
4. **HARVEST of this report's section (f) into TODO_LIST.md** — user ordered "write report, then WAIT"; harvest deferred to explicit instruction.
5. **Root-cause analysis of the go.sum drift** — swept the symptom, did not investigate which dependency wave left 91 modules untidy.

## d) TOTALLY FUCKED UP

1. **`#check-file-size` is RED on master** (pre-existing, NOT from this session): 3 NEW offenders absent from `scripts/file-size-baseline.txt` — `scheduling/sqlstore/store.go` (382), `metaengine/layout.go` (360), `system/adapter_event.go` (352). Last touched by daemon auto-commits (`234acaade`, `b0c454b2e`) before this session; tree was clean at session start. Severity: blocks a clean `#verify` for everyone until split or baselined. Mitigation: none needed for THIS change (my files pass); gate stays red until someone fixes it. Decision needed: g-3.
2. **91-module go.sum drift shipped** (pre-existing): recent waves left stale entries everywhere; `TestEveryModuleGoSumIsTidy` was red on master before I touched anything. Fixed mechanically this session (deletion-only, daemon-committed), but it means release/tag waves are NOT running per-module tidy — process defect, unfixed at the source.
3. **Session self-inflicted waste (radical honesty):**
   - Hit the `rg -r` hook trap TWICE in one session (host already counted a "third strike" before me) — blocked tool calls, wasted round-trips.
   - First `WhenVersioned` draft failed `wsl_v5` (assignment-after-var whitespace) — caught by `#lint`, fixed by deleting the dead `cmd`-zeroing entirely (which was unobservable hygiene anyway — the plain decide path is unreachable once `decide` is nil).
   - One full `TestEvery` cycle (76s) wasted discovering the go.sum issue affects ALL modules, not just the first one flagged (benchkit) — the test reports one module per run.
   - Fumbled `cd` depth (`../../scenario` from `event/v4/eventtest` needs 3 levels) and git pathspec globs (`'**/go.sum'` doesn't match `go.sum` files) — minor, corrected immediately.

## e) WHAT WE SHOULD IMPROVE

1. **Load-aware verification strategy** — With sustained load 57+, the chained `#verify-fast` is noise, not signal (2 packages red under load, green in isolation). Improvement: reach for `#quiet-window-run` / `scripts/verify-chain.sh` FIRST for final chained proofs during busy periods, not after a noisy failure. Impact: one wasted ~8min verify run + triage time, every busy session.
2. **Authored commits at phase boundaries** — The daemon absorbs mid-phase edits; close-out comments and release notes then can't cite clean hashes. Improvement: `git commit` per completed issue (harness permitting explicit user instruction) before starting the next.
3. **Tidy discipline in release waves** — 91 modules drifting means the tag wave tooling skips `go mod tidy` (or runs it in an env that computes different sums). Improvement: add a tidy step to the wave scripts or a CI gate that fails FAST listing ALL untidy modules at once.
4. **`TestEveryModuleGoSumIsTidy` UX** — Reports one module per run. Improvement: report all untidy modules in one failure (the sweep loop already iterates all).
5. **wsl_v5 knowledge** — assignments following a `var` declaration need a blank line between. Trivial, but it cost a lint cycle; worth one line in `docs/agents/gotchas-language-footguns.md` if not already there.
6. **Daemon commits bypass gates invisibly** — The file-size offenders arrived via daemon auto-commits; nobody ran the gate until this session. Improvement: nothing new (CI exists); just remember the daemon's absorbs are unverified by definition and assume gate debt after daemon-heavy periods.

## f) Next tasks (ranked; feeds docs-health HARVEST — deferred per user instruction)

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Tag-release wave: `scenario` + `eventtest` (+ `versions.json` refresh); run per-module test suites over the changed set FIRST (batch-release verify=ok does not test) | Critical | M | Release |
| 2 | Post terse close-out comments + close #84/#85 | High | S | Process |
| 3 | InboxClean: delete `errLoader`, drive ReadAll-failure test via `eventtest.FakeStore.ReadAllFn` | High | S | Cleanup |
| 4 | InboxClean: delete `adaptDecide`/`adaptLabelDecide`, pass domain decides to `scenario.WhenVersioned` | High | S | Cleanup |
| 5 | Fix `#check-file-size`: split `metaengine/layout.go` (360 → under 350) | High | M | Quality |
| 6 | Fix `#check-file-size`: split `system/adapter_event.go` (352 → under 350) | High | M | Quality |
| 7 | Decide + handle `scheduling/sqlstore/store.go` (382): baseline entry (v5-doomed file) vs split — needs owner call (g-3) | Medium | S/M | Quality |
| 8 | One green chained `#verify-fast` under `#quiet-window-run` (close out b-1) | High | M | Verification |
| 9 | Full-repo `-race` sweep under quiet load (close out b-4) | Medium | L | Verification |
| 10 | HARVEST this section into TODO_LIST.md (docs-health) | High | S | Documentation |
| 11 | Root-cause the 91-module go.sum drift: which wave/tool skipped tidy; add fast-fail-all gate or wave-script step | Medium | M | Quality |
| 12 | Make `TestEveryModuleGoSumIsTidy` list ALL untidy modules in one failure | Low | S | Quality |
| 13 | Add `wsl_v5` blank-line rule note to gotchas-language-footguns.md (if missing) | Low | S | Documentation |
| 14 | Verify gotchas-testing.md documents the systemscenario/systemtest `context deadline exceeded`-under-load flake class; add cure note if missing | Low | S | Documentation |
| 15 | Consider a recipes.md or core.md §testing mention of `WhenVersioned` for version-stamping BDD scenarios (modules.md row exists; recipes currently silent) | Low | S | Documentation |
| 16 | Confirm eventtest v0.x tag strategy at release time (modules.md records the v4.x tags of the `event/v4/eventtest` path as DEAD/proxy-invisible; InboxClean pins v0.4.0 — the v0.x series is the live one) | Medium | S | Release |
| 17 | InboxClean ADR-027 cutover to `systemscenario` (their plan; spike already 12/12 green) | Medium | L | Feature |
| 18 | After tags: run `cqrs-upgrade` dry-run against InboxClean to validate the wave landed cleanly for a real consumer | Low | S | Verification |

## g) Questions I cannot answer myself

1. **Tag now or ride the wave?** Should I cut `scenario` + `eventtest` tags immediately so InboxClean can pin the new APIs (per-module tests run first), or hold for the next scheduled batch wave? (I cannot know your release cadence preference; consumers are blocked on pins until tagged.)
2. **Close #84/#85 now or after your review?** The work is gate-verified but lives in daemon `chore:` commits — close comments would cite the tree, not clean hashes. Want me to post the closing comments + close, or do you want to read the diff first?
3. **The 3 pre-existing file-size offenders:** split `metaengine/layout.go` + `system/adapter_event.go` now (they survive v5), and baseline the v5-doomed `scheduling/sqlstore/store.go` instead of splitting a file that dies at v5.0.0 — or baseline all three? (Judgment call on a doomed file; gate stays red until something happens.)

---

**Session verdict:** Both issues fully implemented and gate-verified; the library-side work is DONE pending tags. The pre-existing master debt (file-size gate red, go.sum drift source, load flake) is documented above and deliberately not bundled.

*Report format: Markdown per explicit user path demand (skill default HTML overridden; flagged, not propagated). HARVEST into TODO_LIST deferred — user instructed WAIT after writing.*
