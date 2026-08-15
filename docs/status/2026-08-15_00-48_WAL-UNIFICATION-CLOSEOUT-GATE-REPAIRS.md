# Status Report: WAL Unification Close-Out + Gate Repairs

> **Scope:** This session only (2026-08-14 evening → 2026-08-15 00:48 CEST) — the final
> close-out steps of the WAL Unification plan plus on-sight gate repairs discovered en
> route. A concurrent session (transport deprecation, metaengine, cqrs-lint internals,
> event/v4 restructuring) was editing the same tree throughout; interactions are noted
> where they touched my work.
>
> **Format note:** written as `.md` per explicit user request (skill default is HTML).

---

## a) FULLY DONE

All items below are verified by passing gates in this session.

| # | Item | Evidence |
|---|------|----------|
| 1 | **`LoadStream` migration** — `EventAdapter.Load` and `CommandAdapter.Load` now delegate to `AdapterCore.LoadStream` (system/adapter_event.go:139, system/adapter_command.go:78). The last duplicated StreamRead+FromAny body in the system adapters is gone. WAL Unification Phase 6 is 100% complete. | system suite: 118 tests, `-count=3 -race` green; module lint 0 issues |
| 2 | **Harmful clone consolidated** — `verifyEventParam`/`verifyRecordEventParam` merged into `verifyTypedParam[E]` (metaengine/fold.go:430). Error strings preserved byte-for-byte (`metaengine.On(...)` / `metaengine.OnRecord(...)`). | metaengine suite green (16.5s), lint 0 issues |
| 3 | **Duplication baseline re-pinned** — 10 new clone groups judged one by one: 1 harmful (fixed, #2), 9 intentional accepted (cross-engine parity tests, test-setup boilerplate, idiomatic RLock pairs, Dgraph GraphQL strings, asrecord structural adapters, insert-vs-update + On/OnRecord parallel builders). Baseline 92 → 97 groups. | `nix run .#check-duplication` green ("No new clones") |
| 4 | **`check-coverage` gate repaired** — it had been broken since commit `baf2fb1f0` (2026-08-11): (i) the `[storage / memory]` display key crashed the `tr ' ' '\n'` iteration under `set -u`; (ii) path construction didn't strip spaces (`./storage / memory/...` → 0% coverage); (iii) the loop-piped-to-`sort` pattern ran in a subshell, silently discarding the `drifted` counter — **the gate was a false GREEN for 3 days**; (iv) stale `[codec]=69.2` entry (module extracted, no tests). All fixed; EXPECTED refreshed to actuals (metaengine 81.0→83.3, schema 89.9→92.2 — real improvements that the broken gate had hidden). Stale "verified 2027-07-27" date corrected to 2026-08-14. | `nix run .#check-coverage` green, 11/11 modules within tolerance, drift counting verified by observing correct DRIFT output before refreshing numbers |
| 5 | **benchkit hang-threshold fix** — the `raceEnabled` branch was mis-modeled: parallel verify load inflates wall-clock in the *non-race* phase too (observed 14s vs 5s ceiling → 2 false failures). Both tests (`TestRun_DurationAborts`, `TestRun_CancelledContext`) now use a flat 30s hang-detection ceiling with rationale comment; a true hang is still caught by the go test timeout. | benchkit `-count=3 -race` green (145s), lint 0 issues |
| 6 | **`.go-arch-lint.yml` de-dangled** — removed `codec` component + deps entries left dangling after the concurrent session deleted `codec/` (`1ff2b53d0`). | `TestGoArchLintConfigsAreValid` + full api-stability package green |
| 7 | **Plan doc annotated** — `docs/planning/2026-08-14_11-27_WAL-UNIFICATION.md` now carries an EXECUTED status banner, including the documented task-29 deviation (SQL event-store insert deliberately NOT moved onto `Inserter[T]` — it already owns cached templates + batched multi-VALUES inserts). | banner present; link targets verified to exist |
| 8 | **Prior status report closed out** — close-out addendum appended to `docs/status/2026-08-14_16-44_WAL-UNIFICATION-PHASES-5-7-EXECUTION.md` covering all four of its "Next Steps". | addendum present |
| 9 | **AGENTS.md gotcha generalized** — the `check-module-layers.sh` ` / ` bullet widened to cover all bash maps keyed by module (incl. check-coverage.sh), with the `tr`-tokenization anti-pattern and `"${mod// /}"` path fix spelled out. | edit landed; doc-check green |
| 10 | **FULL VERIFY GREEN** — `nix run .#verify`: 238 packages `ok`, 0 `FAIL`, lint 0 issues across modules, all doc assertions pass. This is an honest GREEN observed at the end of the session, after all fixes. | /tmp/verify-final.log: `grep -cE '^(FAIL|--- FAIL|ERROR)'` → 0 |

---

## b) PARTIALLY DONE

| Item | What's open | Risk / effort to finish |
|------|-------------|------------------------|
| **Art-dupl baseline hygiene** | The 97-group baseline was re-pinned while the concurrent session had uncommitted files in flight (dgraphengine counter_test.go, cqrs-lint doctor_*.go). Their clone signatures are baked into the baseline. | If their work is reworked/reverted, groups may re-flag or disappear; a re-pin at the next clean tree settles it (S, mechanical) |
| **metaengine test depth** | fold.go change validated with `-count=1` suite + full verify race pass — not the `-count=3 -race` discipline used for system/benchkit (suite is 135s; full verify already runs race). | Optional 3x run for parity of confidence (S) |
| **CHANGELOG for this session's fixes** | check-coverage gate repair, benchkit threshold fix, and arch-lint de-dangle are repo-hygiene commits; the auto-commit daemon committed them without curated CHANGELOG entries. | Decide whether gate repairs belong in [Unreleased] or are below the bar (S) |

---

## c) NOT STARTED

| Item | Status | Priority |
|------|--------|----------|
| `DecorateJournal` for `VersionedSeekableJournal` | Deferred (recorded in prior report) — schema upcasting path still lacks the DecorateStore-equivalent for journals | Medium, still wanted |
| `brandedString` extraction into `record/` | Deferred — asrecord clone pair is larger than the helper, so extraction alone won't clear the clone gate; needs a judgment call | Low |
| **docs-health HARVEST of this report's section (f)** | Not run — user instructed to report and wait | High (pending go-ahead) |
| **Wiring auxiliary gates into `#verify`/CI** | Identified this session (see e/1) but not ticketed or implemented | High |

---

## d) TOTALLY FUCKED UP

Radical honesty, including my own messes:

1. **[DISCOVERED, PRE-EXISTING, WORST FINDING] `check-coverage.sh` was a false GREEN for 3 days.**
   Since `baf2fb1f0` (2026-08-11) the gate both crashed on the `storage / memory` key AND
   (when it didn't crash) silently discarded DRIFT results via a subshell. Consequence: zero
   coverage enforcement for 3 days; two real coverage improvements went unnoticed; the
   `[codec]=69.2` entry referenced a module that no longer has tests. Root cause: bash
   associative-array iteration anti-pattern + pipe-to-sort scoping. Mitigated: fixed and
   green this session. Severity: process-level, not user-facing.
2. **[MINE, BRIEF] Inverted DRIFT/ok branches in my first check-coverage.sh edit.** While
   restructuring the loop I swapped the status arms so drift would have printed "ok". Caught
   within a minute by re-reading the diff before running; fixed immediately. Lesson: when
   transforming conditional logic, re-read the arms before executing — never trust a
   mechanical edit of inverted conditions.
3. **[MINE] Wasted a full verify cycle on truncated logs.** The second verify run (job 01C)
   piped to `tail` only; the FAILing package name was cut off and I got a bare `FAIL` line.
   Blind, I launched a third full run with `tee`. ~10+ minutes burned that a `tee` from the
   start would have saved. Lesson: always capture full gate logs to a file.
4. **[MINE] Self-inflicted false test failure.** I ran the cqrs-lint analyzer package with
   `GOWORK=<root>/go.work` from inside the package dir → "directory prefix . does not
   contain modules listed in go.work". I briefly treated it as a real regression before
   remembering verify runs per-module with `GOWORK=off`. Retested correctly: green.
   Diagnostic noise of my own making; the lesson (workspace env is positional) is already
   half-documented and worth finishing.
5. **[ENVIRONMENT] Concurrent-session verify races.** Two verify attempts hit transient
   breakage from the other session's in-flight `event/` → `event/v4/` go.work move
   (conflicting replacements) and cqrs-lint golden drift. I correctly diagnosed these as
   not-mine, did not touch their state, and re-ran after they settled. No damage — but
   parallel sessions sharing one working tree WILL keep producing ghost-red verifies; this
   is the third session in a row that has paid this tax.

---

## e) WHAT WE SHOULD IMPROVE

1. **Unwired gates rot — proven.** ~~check-coverage, check-arch, check-duplication, and
   vulncheck are separate nix apps that nothing runs automatically.~~ _[Correction,
   2026-08-15: wrong — the gates have run inside `#verify` since `6f7c88388` (2026-08-03).
   check-coverage rotted while WIRED: the script itself was broken (false GREEN), fixed at
   `875bb689b`. Only `#vulncheck` sits outside `#verify`.]_ check-coverage rotted
   for 3 days unnoticed. Fix: add them to `#verify` (accepting +5-10 min) or to CI on every
   PR. The current state is "gates that only fire when someone remembers".
2. **No meta-test guards script-owned maps.** `.go-arch-lint.yml` has
   `TestGoArchLintConfigsAreValid` (it caught the codec dangle — proof the pattern works).
   `check-coverage.sh`'s EXPECTED map has no equivalent existence check. Fix: a tiny
   meta-test asserting every EXPECTED key resolves to a real module dir.
3. **Log-capture discipline for long gates.** Every `nix run .#verify` should tee to a
   timestamped file by default (script change), so post-mortems never depend on terminal
   scrollback.
4. **Intentional clones should be marked in-place.** The 9 accepted clone groups exist only
   as baseline entries. `//art-dupl:accept` directives (the tool supports them — the
   baseline command output says so) would document the intent at the clone site and survive
   re-pins.
5. **Baseline re-pins should require a clean tree.** Re-pinning with another session's
   uncommitted work in flight bakes their transient state into the golden. A dirty-tree
   guard in the gate (or just discipline) prevents this.
6. **Dates in script comments go stale silently.** The EXPECTED map carried "verified
   2027-07-27" (wrong year, never noticed). `--update` should stamp the date itself.
7. **Parallel sessions on one tree keep colliding.** Three sessions in a row have paid the
   transient-red-verify tax. Either serialize heavy gates, or use `verify-parallel` +
   worktrees so sessions stop observing each other's half-finished states.

---

## f) Next tasks (ranked; session-derived)

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
~~| 1 | Wire `check-coverage` + `check-duplication` (+ `check-arch`) into `#verify` or CI | Critical | M | Quality |~~ done (existing wiring): gates have run inside #verify since 6f7c88388 (2026-08-03); this report's premise was wrong - check-coverage was BROKEN, not unwired; script fixed at 875bb689b
| 2 | Add meta-test: every `check-coverage.sh` EXPECTED key resolves to an existing module dir (codec-dangle class) | High | S | Quality | <- OPEN. TODO_LIST 'Code Quality' (check-coverage.sh hardening)
~~| 3 | Run docs-health HARVEST: pull this report's section (f) into TODO_LIST.md | High | S | Documentation |~~ done. TODO_LIST carries the harvest (Duplication-baseline hygiene, benchkit wall-clock audit, DecorateJournal, brandedString items) - docs-health audit 2026-08-15
~~| 4 | Confirm concurrent session's `event/v4` restructure + module deletions landed clean; run `#verify` once at a clean tree | High | S | Verification |~~ done at 5f2198189 (first fully-green verify gate after the wave; three GREENs since)
| 5 | Re-pin `.art-dupl-baseline.json` at the next clean tree (current pin includes in-flight foreign code) | Medium | S | Quality | <- OPEN. TODO_LIST 'Code Quality' (Duplication-baseline hygiene)
~~| 6 | Sweep other root configs/scripts for deleted-module references (go.work is clean; check flake.nix `testModules`, check-module-layers.sh, api-stability modules list) after the deletion wave settles | High | S | Cleanup |~~ done at 2e9a2fc28 (stale references to extracted modules cleaned; meta-tests enforce lists since)
| 7 | Audit benchkit's remaining wall-clock assertions (`raceEnabled` use at benchkit_test.go:823) for the same load-sensitivity mis-model | Medium | S | Quality | <- OPEN. TODO_LIST 'Code Quality' (benchkit wall-clock audit; raceEnabled still at benchkit_test.go:821)
| 8 | Make `check-coverage.sh --update` auto-stamp the "verified" date comment | Low | S | Quality | <- OPEN. TODO_LIST 'Code Quality' (check-coverage.sh hardening covers the --update date stamp)
| 9 | Add `//art-dupl:accept` directives at the 9 intentional clone sites (document intent in-place) | Low | S | Quality | <- OPEN. TODO_LIST 'Code Quality' (Duplication-baseline hygiene)
| 10 | Add dirty-tree guard to duplication baseline command (warn or refuse) | Medium | S | Quality | <- OPEN. TODO_LIST 'Code Quality' (Duplication-baseline hygiene)
| 11 | `DecorateJournal` for `VersionedSeekableJournal` (deferred from ADR-0126 work) | Medium | M | Feature | <- OPEN. TODO_LIST 'Code Quality' (DecorateJournal item)
| 12 | Decide + implement (or permanently drop) `brandedString` extraction into `record/` | Low | S | Cleanup | <- OPEN. TODO_LIST 'Code Quality' (brandedString item)
| 13 | Tee verify output to `docs/../.logs/` or `/tmp` by default inside the verify script | Medium | S | Quality |
| 14 | Investigate golangci-lint fact-cache warnings on `/mnt/buildcache` during parallel sessions (shared cache races) | Low | S | Quality |
~~| 15 | Curate CHANGELOG [Unreleased] entries for gate repairs (coverage false-GREEN fix, benchkit thresholds, arch-lint de-dangle) | Medium | S | Documentation |~~ done. CHANGELOG [Unreleased] 'repo gates' entry (docs-health session 2026-08-15)
| 16 | v5 deprecated-shell deletion sweep (transport/*, codec/retry shells, `metadata.CustomData`, `schema.VersionedStore`, `signing.Rejecting*`) — TODO_LIST Phase-8 entry exists | High | L | Cleanup | <- OPEN. TODO_LIST 'v5 Unification Phase 8' (codec/retry shells already deleted at 5127039da; transport/* + ADR-0126 compat shells wait for the v5 cut)
~~| 17 | check-arch 94-gap catalog issue (pre-existing, ticketed earlier) — still open | Medium | M | Quality |~~ done at 8c384f0f5 (layer-map key convention repaired, 87 plain-key LAYER entries, check-arch green inside #verify)
| 18 | Consider making `verify-parallel` the default `#verify` to cut wall-clock and reduce exposure to concurrent-session interference | Medium | M | Quality |
| 19 | Add `LoadStream` error-path tests (noun-wrapped backend failure) in system if coverage there is thin | Low | S | Quality |
~~| 20 | Document the GOWORK-positional gotcha (workspace env + package-dir CWD = false failures) in AGENTS.md alongside the existing GOWORK notes | Low | S | Documentation |~~ done. AGENTS.md gotcha 'GOWORK env is positional' (docs-health session 2026-08-15)

---

## g) Questions I cannot answer myself

1. **Gate placement:** Should `check-coverage` / `check-duplication` / `check-arch` run
   inside `nix run .#verify` (adds ~5-10 min to every run, but they can never silently rot
   again — and check-coverage just proved they DO rot), or stay separate and run only in
   CI (fast local iteration, but "CI-only" is how we got a 3-day false GREEN)? This is a
   workflow tradeoff only you can pick.
2. **Baseline policy:** May future sessions re-pin `.art-dupl-baseline.json` on dirty
   trees when a concurrent session is mid-flight (current state), or do you want re-pins
   gated on clean trees even if that means the duplication gate stays red until the other
   session lands?
3. **HARVEST now or later:** You told me to report and wait — should I run docs-health
   HARVEST to move section (f) into `TODO_LIST.md` before this session ends, or is another
   session going to own that?

---

*Report written 2026-08-15 00:48 CEST. WAL Unification is CLOSED; all gates GREEN at time
of writing (verify log: /tmp/verify-final.log, 238 ok / 0 FAIL).*


---

## Resolution (2026-08-15)

16 of 20 items resolved or routed; items 13 (tee verify), 14 (golangci
fact-cache), 18 (verify-parallel default), 19 (LoadStream error-path tests)
remain open and unrouted, so this report stays active. Notable finding
while annotating: item 1's premise was factually wrong - the gates have run
inside `#verify` since `6f7c88388` (2026-08-03); check-coverage was broken,
not unwired (script fixed at `875bb689b`). Item 17 closed by the layer-map
key-convention repair at `8c384f0f5`.
