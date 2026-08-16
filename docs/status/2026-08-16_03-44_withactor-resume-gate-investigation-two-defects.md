# WithActor Hardening Resume — Gate Investigation, Two Committed Defects Found, Toolchain Resolved (2026-08-16 03:44)

Follow-up session to `2026-08-16_01-33_withactor-hardening-resume-gates-and-toolchain-block.md`.
Mandate: answer the 3 open questions, unblock + run the gates (verify-fast, full #verify), commit the WithActor follow-up.

**Headline:** All 3 questions answered from evidence (no user input needed). Toolchain unblocked.
WithActor workstream fully committed and verified clean. But the full `#verify` gate is RED for **two
committed, repo-wide defects found and root-caused this session** — neither caused by the actor work.

**Critical context:** a parallel agent session ("ecosystem-execution session 5") was active in this repo
the ENTIRE session (still is — ~20 crush processes live). It landed 6 commits mid-session (`ea8fa5072`
toolchain, `842741cab`, `1840e5967`, `626f7426c`, `dba6f007b`, `b1e3b13aa`, `313d14b02` DemoteEngine,
`cdc525fd5` perf, `f836c7f1c` docs). All shared-tree work was sequenced around it; several collisions
occurred (see d).

---

## a) FULLY DONE (this session)

1. **Q1 (go-codec 1.26.6 bump) — ANSWERED: intentional.** Sibling has a coordinated three-way bump
   (go.mod + `.go-version` + new `scripts/check-go-version.sh` tripwire enforcing agreement). The
   parallel session adopted 1.26.6 here as `ea8fa5072` (flake `goToolchain` overrideAttrs). I
   independently computed the same SRI hash (`sha256-oHIcV…/LLE=`) — cross-validated. `nix run .#build`
   verified green at committed HEAD, multiple runs.
2. **Q2 (AsRecord `"user:<ulid>"` fallback) — ANSWERED: no behavior change.** Pre-actor code
   (`1153c7d11^`) already used the identical `brandedString(tracing.UserID)`; `id.UserID.String()`
   has rendered `"user:<ulid>"` since the branded-ID migration. WithActor only ADDS the
   `ActorID.PrefixedString()` path. Locked by `event/asrecord_test.go:69`. Nothing to decide.
3. **Q3 (daemon-mixed commit) — ANSWERED: leave history.** Rewrite off the table; status report is
   the documentation of the split.
4. **WithActor follow-up confirmed committed** — daemon captured everything (skill docs, integration
   go.mod fix, api_surface golden, fmt files) in `842741cab`. My 2 stragglers (watermill
   `message-metadata.snap` golden — deterministic, never committed; 4085-count api_surface) rode
   along in `dba6f007b`.
5. **api-stability golden drift FIXED at master**: committed `a298ea388` recording the 2 unrecorded
   sqliteengine exports (`NewSQLiteEngineFromDSN`, `OwnDB`). Checker green at committed master:
   "API surface OK: 4087 exports verified".
6. **Full verify-fast + full #verify runs executed** (in isolated worktrees at pinned commits, on a
   quiet machine, disk TMPDIR) — **both RED, root causes found** (see b).
7. **Two committed defects root-caused and documented**:
   - `TestSystem_ResetProjection_RestartAndReplay`: deterministic regression introduced at
     `313d14b02` (DemoteEngine ship). Full-suite repro: `cd system && GOWORK=off go test -count=1 ./...`.
     All commits 5127039da→626f7426c pass; 313d14b02+ fail. Phase-2 replay is DEAD, not slow (wait
     budget 5s→60s still `processed=0 errors=0`). **Actor commit `1153c7d11` verified clean** in both
     scopes (pair + full suite).
   - `TestSoak_AutoCRUD_Bbolt` **passes but takes 509s** (at a298) / **708s** (at 1153c7d11) solo —
     the verify gate's per-package `-timeout=8m` (480s) kills a passing test, in Test AND Race phases.
   - Separate pre-existing issue: the same system test is ALSO load-flaky (tight 5s budget +
     t.Parallel overlap with `TestSystem_HealthCheck_FailedProjection`'s failing-decoder host).
8. **TODO_LIST updated** (3 new entries: system replay regression w/ evidence link, bbolt soak
   budget, mechanical api-stability golden enforcement) + **prior status report updated** with §h
   (question resolutions) and §i (investigation results, self-corrected).
9. **Worktree isolation pattern validated** for gating a moving master shared with another live
   session: `git worktree add` at pinned SHA in `~/projects/` (sibling `../go-codec` resolves
   naturally). Scratch worktrees cleaned up; `wt-head` kept pinned at `a298ea388` for continuation.

## b) PARTIALLY DONE

~~1. **Full `nix run .#verify`** — runs completed but RED: Build ✓ Vet ✓ Test ✗ (system replay
   regression + bbolt soak budget); lint/doc-check/api-stability phases never reached. Cannot go
   green until the two defects in (a.7) are fixed.~~ done — full `#verify` GREEN 2026-08-16 13:15 (run #4) after both defects closed + parallel wave landed

~~2. **Ancillary gates** (`#check-arch`, `#check-coverage`, `#check-duplication`, `#vulncheck`) — NOT
   run. Note: vulncheck + license check ran inside the buildflow pre-commit hook and passed.~~ partial — `#check-coverage` + `#check-duplication` EXIT=0 at 13-15; `#check-arch`/`#vulncheck` runs not recorded

~~3. **bbolt soak timing after `cdc525fd5`** (the other session's metadata-deserialize perf fix landed
   mid-session and may shorten it) — measurement killed to free CPU; one data point pending.~~ done — measured 1145.001s at `06e046c2f` (§h post-resolution); cdc525fd5 did NOT shorten it, spread is load-driven

## c) NOT STARTED

~~1. Fix for the `313d14b02` system phase-2 replay regression (suspects: replan-convergence / DemoteEngine
   role defaults breaking `system.New` phase-2 journal reads, or `replicator.applyJobFilter` skipping
   never-served collections).~~ done upstream — root cause was a test-fixture bug (shared-cache in-memory DSN), fixed by parallel session `5d66308c3` (§h.2)

~~2. Verify-gate budget fix for the soak (soak-specific timeout or SOAK_SKIP wiring in the verify app).~~ done — `SOAK_SKIP_BOLT=1` in the verify app + `-timeout=20m`→`30m` dedicated budgets (§h.3)

~~3. `#verify-fast` re-run on the final tree (last run failed only on the same two test defects).~~ done — `#verify` + `#verify-fast` GREEN (13-15 run #4)

~~4. Release follow-ups f.15–f.20 from the prior report (tag metadata/schema/event/middleware minors;
   strip sibling replaces from event/command/query/middleware/integration go.mods).~~ partial — metadata/schema/event/middleware minors tagged in the 22-tag chain (04:12–04:24); replace-strip sweep still open (TODO_LIST)

## d) TOTALLY FUCKED UP (honest ledger)

1. **Published a wrong root-cause verdict, then had to self-retract.** I concluded the system replay
   failure was "ENVIRONMENTAL, not a code regression" and wrote it into the status report — based on
   pair-only reproductions passing. The decisive variable (full-suite vs pair-only scope) wasn't
   controlled. Only after a full-suite bisect did the deterministic `313d14b02` regression appear.
   The report is now corrected, but for a window my documented conclusion was false. LESSON:
   bisect with the EXACT failing command (`./...`), never a narrowed `-run` filter.
2. **First bisect suppressed checkout errors (`2>/dev/null`)** — silent untracked-file checkout
   failures made commit attribution unreliable; wasted a debug cycle before I noticed and redid it
   with verified `dirty=` checks.
3. **Late ENOSPC diagnosis.** `/tmp` (48G tmpfs, 100% full via `/tmp/bigtest` 40G + go-build dirs)
   caused: a "transient" verify-fast build failure (actually link-time ENOSPC), an api-stability
   "golden mismatch" (compile corruption), and a system-test link failure — three misread symptoms,
   one cause. Also each bash call is a fresh shell: I set TMPDIR once and lost it repeatedly.
4. **Commit racing the daemon**: first commit died mid-hook (HEAD moved under it after 207s),
   second found my files already committed by the parallel session. Outcome fine, process ugly —
   in this repo, stage+commit FAST or pre-write the message file (which I did on retry).
5. **Killed a 25-min measurement** (626f soak timing) to write this report — incomplete data
   (had 2 of 3 planned points).

## e) WHAT WE SHOULD IMPROVE (repo-level)

1. **api-stability golden regen needs mechanical enforcement** — THREE consecutive feature commits
   shipped new exports without regenerating `docs/api_surface.txt` (4081→4085→4087→4089 drift);
   every fresh checkout of those revisions fails the gate. ~1s GOWORK=off checker run as a
   pre-commit step would have caught all three. (TODO entry added.)
2. **The last-green `#verify` is older than anyone thinks.** Gates have been RED since at least
   `7c0a62c98` (toolchain), and the soak budget + system regression mean even the toolchain fix
   alone doesn't restore green. CI green ≠ local-verify green; sessions were claiming per-module
   GREEN while the full gate rotted. The AGENTS.md "check gate status at session START" rule from
   the prior report is still not practiced.
3. **Two concurrent agent sessions in one repo double every investigation cost** — every failure
   had to be re-tested in isolated worktrees because the main tree and machine load were
   contaminated. If this is the new normal, we need a session-lock convention for gate-class runs.
4. **/tmp capacity is a standing landmine**: `/tmp/bigtest` (40G, not mine) holds the tmpfs at 100%;
   any go link can fail. Gate apps should set disk-backed GOTMPDIR themselves rather than relying
   on callers to remember.
5. **buildflow pre-commit hook invokes bare `go`** (system 1.26.5) — must run inside `nix develop`.
   The hook should resolve the flake's pinned toolchain itself.

## f) NEXT — in order

~~1. Re-measure `TestSoak_AutoCRUD_Bbolt` at current HEAD (`f836c7f1c`) — `cdc525fd5` (skip metadata
   JSON round-trip on deserialize) may have shortened it; under `-race` too.~~ done — 1145.001s at `06e046c2f` under load; cdc525fd5 not a shortener (§h)

~~2. Fix verify-gate soak budget: per-package `-timeout=8m` → soak-appropriate budget (or wire
   SOAK_SKIP_*=env into the verify app; -short already excludes soaks in verify-fast).~~ done — `SOAK_SKIP_BOLT=1` exported by the full verify app; bboltengine 9.5s in the pinned-`954cef1a4` checkpoint; `#test`/`#test-race` at `-timeout=30m` (§h)

~~3. Diagnose + fix the `313d14b02` system phase-2 replay regression (DemoteEngine owner's fresh
   feature; `replanWithTransition`, `applyJobFilter`, role defaults are the suspects).~~ done upstream — `5d66308c3` file-backed DSN fixture fix (§h.2)

~~4. Re-run full `#verify` (exclusive) on the fixed tree.~~ done — full `#verify` GREEN at 13-15 run #4 (golden regen landed with the parallel wave)

~~5. `#verify-fast`, then ancillary gates: `#check-arch`, `#check-coverage`, `#check-duplication`
   (note: `actorString` helper now in 3 asrecord.go files — baselining may be needed), `#vulncheck`.~~ partial — coverage + duplication EXIT=0 (13-15); `#check-arch`/`#vulncheck` runs not recorded

~~6. Add api-stability checker as pre-commit step (TODO entry exists).~~ open — tracked at TODO_LIST "Enforce api-stability golden regen mechanically" (pre-commit hook step)

~~7. Raise `waitForProjectionProcessed` budget (5s→15s+) or serialize projection-wait tests (load flake).~~ open — load-flake concern remains the only TODO_LIST leftover from §h.4

~~8. Release follow-ups: tag metadata/schema/event/middleware minors; strip sibling replaces from
   event/command/query/middleware/integration go.mods in one sweep.~~ partial — all four minors tagged in the 22-tag chain; replace-strip sweep still open (TODO_LIST Release batch)

~~9. `/tmp/bigtest` owner decision + gate-app GOTMPDIR hardening.~~ partial — /tmp/bigtest resolved (§h.1, already gone); gate-app GOTMPDIR hardening not wired (flake.nix has no GOTMPDIR; AGENTS.md documents the manual workaround)

~~10. Optional: worktree-with-pinned-SHA gate procedure into AGENTS.md (pattern proven this session).~~ partial — the never-checkout gotcha documents the worktree escape hatch (AGENTS.md); a full pinned-SHA gate procedure was not added

## g) QUESTIONS (cannot resolve from the repo alone)

> **[ALL THREE RESOLVED 2026-08-16 (next session) — see §h. Q1 already gone, Q2
> fixed upstream by `5d66308c3`, Q3 skip wired.]**

1. **`/tmp/bigtest` (40G) is pinning the 48G tmpfs at 100%** and it is not this session's data.
   May I trash it, or is it yours/another project's active work? (I only trashed stale >90min
   go-build dirs so far.)
   **[RESOLVED: user approved trashing; the file was already gone by then — tmpfs
   back to 1%.]**
2. **Who fixes the `313d14b02` phase-2 replay regression?** It's the parallel session's fresh
   DemoteEngine feature and that session is STILL ACTIVE in this repo. Do you want me to take the
   fix (deep-dive replan/replicator), or leave it for the author to avoid two agents editing
   `metaengine/` concurrently?
   **[RESOLVED: user parked it; the parallel session then fixed it themselves in
   `5d66308c3` — see §h. NOT a DemoteEngine-logic bug; test fixture.]**
3. **Should soaks (509–708s) run in the full `#verify` gate at all?** Options: raise per-package
   timeout to ~20m (full verify gets much slower), skip AutoCRUD soak via env in the verify app
   (faster gate, less coverage in the default gate, still available via `nix run .#test-integration`
   / module-local runs), or slim the soak's event count. Preference?
   **[RESOLVED: user chose skip — `SOAK_SKIP_BOLT=1` wired into the verify app;
   `#test`/`#test-race` got explicit `-timeout=20m` so dedicated runs keep soak
   coverage.]**

## Gate status at session end

`#build` GREEN (committed HEAD, multi-run) · api-stability checker GREEN at committed master
(4087) · doc-assertions GREEN · per-module actor workstream GREEN + committed · full `#verify` RED
(system replay regression @313d14b02 + bbolt soak >8m budget) · `#verify-fast` RED (same two) ·
ancillary gates NOT RUN · background: parallel session still active at `f836c7f1c`.

Useful artifacts: `/tmp/verify-final.log`, `/tmp/verify-wt2.log`, `/tmp/system-60s.log`,
`/tmp/bbolt-solo2.log`, `/tmp/bbolt-solo.log`; worktree `wt-head` pinned at `a298ea388`.

## h) RESOLUTION (2026-08-16, next session)

User answered all three questions; both defects closed. Full `#verify` unblocked.

1. **Q1 `/tmp/bigtest`** — user approved trashing; it was already gone when we
   looked (tmpfs back to 1%, 407M/48G used). No action needed.
2. **Q2 replay regression — FIXED UPSTREAM, not by this session.** The parallel
   session landed `5d66308c3` ("make projection replay test use a file-backed
   SQLite DSN") while this session was wiring the soak skip. Root cause was a
   TEST FIXTURE bug, not DemoteEngine logic: the restart/replay test used a
   shared-cache in-memory SQLite DSN that only survived Close/reopen because the
   engine leaked its `*sql.DB`. Once engines began owning/closing self-opened
   DBs (in the `313d14b02` window), the last connection close wiped the journal
   before phase 2 could replay it — exactly the observed `processed=0` (and why
   60s of waiting changed nothing: dead, not slow). Fix: `sqliteFileDSN(t)` under
   `t.TempDir()`. Verified independently by this session: full system suite
   green (`-count=1 ./...`, 2.1s) and the replay test green `-count=3`. §d's
   lesson stands: my suspect list (replan/roles/replicator) was wrong; the
   bisect commit attribution was right.
3. **Q3 soak policy — SKIP, wired.** `SOAK_SKIP_BOLT=1` env-var skip in
   `metaengine/bboltengine/soak_autocrud_test.go` (same pattern as
   `SOAK_SKIP_DUCKDB`), exported by the full `#verify` app in `flake.nix`
   before the Test/Race phases (realized app script verified to contain the
   export). `#test` and `#test-race` got explicit `-timeout=20m` (the implicit
   10m default would kill a 708s loaded soak there too) — dedicated runs keep
   full soak coverage. AGENTS.md soak-env-var line updated; the TODO_LIST
   soak-budget entry closed as done.
4. **TODO_LIST updated**: replay entry rewritten (fixture root cause, fixed by
   `5d66308c3`); only the separate load-flake concern remains open. Prior
   report (01-33) §j pointer added.

## Gate status (post-resolution)

Both blockers closed. Fresh HEAD soak measurement (post-`cdc525fd5`): **PASS at
1145.001s** (`06e046c2f`, ~20 concurrent agent processes; log
`/tmp/bbolt-head-measure.log`) — cdc525fd5 did NOT shorten it; the 509→1145s
spread is load-driven. Dedicated-run budgets (`#test`/`#test-race`) therefore
bumped to `-timeout=30m`. Remaining gate work: full `#verify` re-run
(exclusive window), then `#verify-fast`. Known risk: the parallel session's
in-flight WIP (storage/view batch exports, `docs/api_surface.txt` drift) may
trip check-api-stability before it lands its own regen.

**Checkpoint result (2026-08-16 ~05:20)**: full `#verify` at pinned `954cef1a4`
(worktree `wt-head`, log `/tmp/verify-954.log`) — Build ✓ Vet ✓ and the ENTIRE
Test phase green across 118 packages: `metaengine/bboltengine` **9.5s** (was
509-1145s → soak skip works), `system` **2.4s ok** (replay fix works). Sole
failure: the two api-stability meta-tests — the PRE-EXISTING golden drift
documented in §i.4 of the 01-33 report (4087 committed vs 4089 actual:
`event.ReconstructEventWithMetadata`, `storage/sql.MaxParametersForDialect`),
plus the parallel session's newer wave-3 exports. Additionally, master tip
`fde8f9444` transiently breaks `nix run .#build` (`storage/sql/keyset.go`:
`undefined: err` — committed mid-edit; their fix is staged in the working
tree). Both are the parallel session's in-flight state, not this session's
defects; a fully GREEN gate needs their wave to land (keyset fix + golden
regen, both already visible as working-tree WIP), then a re-pin + full
`#verify` at that tip (Race/Lint/Arch/Duplication/Coverage phases are what
remained unexecuted after the Test-phase abort).
