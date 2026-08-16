# WithActor Hardening — Resume, Gates & Toolchain Block (2026-08-16 01:33)

Session resuming `2026-08-13 WithActor Hardening`. All planned feature/test/doc work is now done;
the session ended blocked on a **pre-existing repo-wide toolchain mismatch** (`Go 1.26.5 vs 1.26.6`)
that makes `nix run .#build` / `#verify-fast` fail. One working-tree edit (`go.work`) is currently
in the WRONG state and needs a decision before the gates can pass.

---

## a) FULLY DONE (this session)

1. **Repo state verified** — the auto-commit daemon already committed everything from the prior
   session as `1153c7d11` ("feat(actor): propagate actor identity…"). Working tree was clean at
   session start. Note: that commit ALSO contains the `metaengine/*` layout-calibration changes
   the prior session explicitly said to exclude — the daemon doesn't respect exclusion lists
   (see d).
2. **Integration build FIXED** — `integration/go.mod`:
   - `command/v4` bumped v4.3.0 → **v4.6.0** (published tag already ships `WithActor`)
   - `replace event/v4 => ../event` + `replace middleware/v4 => ../middleware` (needed for
     unpublished `ActorEnricher` / `CommandActorContext`; documented repo pattern)
   - `google.golang.org/genproto` pinned to post-split `v0.0.0-20260810153831-ec0a7760b754`
     to fix an "ambiguous import" caused by cockroachdb/errors pulling the monolithic genproto
   - `query/v4` auto-upgraded v4.2.0 → v4.5.0 by tidy
   - `GOWORK=off go mod tidy` clean; `go vet` clean
3. **`TestActorPropagationEndToEnd` GREEN** — verbose-verified (`=== RUN … --- PASS`).
   The 3-hop actor audit trail (dispatch middleware → enricher → store+projection) is now
   guarded at integration level.
4. **Full integration module suite GREEN** (all subpackages, EXIT=0).
5. **All 13 session-touched modules re-tested GREEN**: id, event, command, query, middleware,
   watermill, commandlifecycle, deriver, scenario, metadata, storage/sql, storage/pebble,
   storage/bbolt.
6. **api-stability golden regenerated** — 8 new exports captured exactly as predicted:
   `event.ActorEnricher`, `event.ActorFromContext`, `event.WithActorContext`,
   `id.MarshalBinary`, `id.UnmarshalBinary`, `id.Validate`, `middleware.CommandActorContext`,
   `scenario.ThenEvents`. Meta-tests (`TestEvery*`) PASS; checker PASS (4081 exports verified).
7. **`nix fmt` run** — 6 files reformatted (2 pre-existing drift: `storage/view/{count,query}.go`;
   post-format re-tests of affected modules green).
8. **Skill docs WRITTEN** (plan item 1, previously zero):
   - `core.md` §3.8: actor propagation block (WithActor → CommandActorContext → ActorEnricher),
     wire format `"kind:raw"`, `Validate()`, pointer to recipes §2.21; cheat-sheet entry added
   - `modules.md`: 5 rows updated (`id` ActorID, `event` WithActor/context API, `middleware`
     CommandActorContext, `watermill` actor_id key, `scenario` ThenEvents)
   - `recipes.md`: new **§2.21 Actor Propagation — "Who Did It" Audit Trail** (full chain,
     manual context control, scheduling DispatchFunc attribution with `NewSystemActor`,
     trust levels/Validate/ParseActorID, transport + CBOR/AsRecord notes) + TOC entry
   - Every symbol claim verified against source before writing (query.New signature corrected,
     invented Timer Actor field removed — no fabricated APIs)
9. **doc-check GREEN** — 862 references valid across 41 packages (covers the new docs).

## b) PARTIALLY DONE

1. **`nix run .#verify-fast`** — doc assertions all OK, but gate FAILED on
   `nix run .#build`: `go: go.work requires go >= 1.26.6 (running go 1.26.5; GOTOOLCHAIN=local)`.
2. **go.work toolchain fix** — root cause fully diagnosed, fix attempt went the wrong
   direction once (see d), correct fix (Go 1.26.6 via flake `overrideAttrs`) is mid-flight:
   - ✔ Verified `go1.26.6` exists upstream (go.dev/dl API, linux-amd64 sha256 verified)
   - ✔ Read the pinned nixpkgs `go/1.26.nix` derivation — `version` is hardcoded in
     `finalAttrs`, so `override { version = … }` fails; needs `overrideAttrs` with full src
   - ✘ SRI-hash conversion attempt failed (`xxd` not on PATH) — session interrupted here
3. **Commit of this session's follow-up** (integration go.mod, golden, docs, fmt) — blocked
   on the gate being green (per repo rule: never claim GREEN / commit on RED).

## c) NOT STARTED

1. Full `nix run .#verify` (build+vet+test+race+lint+doc-check+doc-assertions) — blocked on build.
2. Ancillary gates: `#check-arch`, `#check-coverage`, `#check-duplication`, `#vulncheck`.
3. Release-follow-ups (prior session's item 8): strip sibling replaces from
   event/command/query/middleware/integration go.mod at next batch release; tag
   metadata/schema/event/middleware minors so replaces can be dropped.

## d) TOTALLY FUCKED UP (honest ledger)

1. **I edited `go.work` (1.26.6 → 1.26.5) BEFORE understanding the root cause.** The bump
   was legitimate — the sibling checkout `../go-codec` has an _uncommitted_ `go.mod` bump to
   `go 1.26.6`, so `go work` demands ≥1.26.6. My downgrade made the build fail _differently_
   ("module ../go-codec requires go >= 1.26.6"). **`go.work` is NOW in a state I believe is
   wrong and it is UNCOMMITTED** — needs restore to 1.26.6 (or a decision on the sibling,
   which is not my change to revert). Violated my own READ-ROOT-CAUSE-FIRST rule.
2. **Auto-commit daemon squashed the excluded `metaengine/*` work into the actor commit**
   (`1153c7d11`) — including 3 large bench files and `relayout.go`/`layout_scoring.go`
   changes. Not authored by me, not revertible (history rewriting forbidden), but the
   "exclude metaengine from the actor commit" instruction is now unsatisfiable. Flagged.
3. **Guessed a genproto version** (`4ff94f1adbff`) that didn't exist instead of searching
   the local module cache first — wasted one cycle. (The cache-first lookup then worked.)
4. **Attempted `override { version=… }` without reading the derivation** — failed with
   "unexpected argument". Should have read `1.26.nix` first (which I then did).

## e) WHAT WE SHOULD IMPROVE

1. **Toolchain drift is systemic**: `go.work` demands 1.26.6, nixpkgs pin ships 1.26.5,
   `GOTOOLCHAIN=local` forbids auto-download. Every gate (`#build`, `#verify`) has been RED
   since commit `7c0a62c98` (2026-08-15 02:49) — sessions since only ran per-module
   `GOWORK=off` commands and never noticed. Fix once in the flake; consider a CI check that
   `nix run .#build`'s Go satisfies `go.work`.
2. **Sibling-checkout go.mod bumps leak into this repo's workspace** (go-codec 1.26.6 is
   uncommitted there). Cross-repo working-tree coordination needs a rule: either commit the
   sibling bump or keep both at 1.26.5.
3. **The auto-commit daemon defeats exclusion instructions.** For work that must land in
   separate commits, changes need to be committed IMMEDIATELY by the agent (or stashed in a
   path the daemon ignores). "Don't include in your commit" is unenforceable otherwise.
4. **`#verify` gate status should be checked at session START**, not end — this session
   inherited a RED gate unknowingly.
5. Dependency-version lookups: module cache first (`ls /mnt/buildcache/go-mod/…`), never
   guess pseudo-versions.

## f) NEXT — up to 50 things, in order

**Unblock the build (critical path):**

1. Decide go.work direction (see questions) — likely restore `go 1.26.6`.
2. Compute SRI hash for go1.26.6 src tarball using `python3` (no xxd needed).
3. Add `go_1_26.overrideAttrs` (version + src hash) — or a local overlay — in flake.nix
   (2 sites: `mkCqrsLintSource`, devShell/test env use `pkgs.go_1_26`).
4. Verify `nix develop -c go version` reports 1.26.6.
5. Re-run `nix run .#build` → green.
6. Re-run `nix run .#verify-fast` (exclusive, nothing heavy concurrent).
7. Run full `nix run .#verify` (exclusive; ~long).

**Ship this session's work:**
8. `git status` review — confirm no unexpected daemon commits mid-flight.
9. Commit follow-up: integration go.mod fix (bump+replaces+genproto pin), api_surface.txt
golden, skill docs (core/modules/recipes), fmt-only fixes; note replaces-to-strip-at-release.
10. Re-run doc-check after any doc touch-ups.

**Ancillary gates (after verify green):**
11. `nix run .#check-arch` (dep budgets).
12. `nix run .#check-coverage` (coverage drift — new code added this feature).
13. `nix run .#check-duplication` (`actorString` helper now in 3 asrecord.go files — may
need baselining or dedup judgment).
14. `nix run .#vulncheck`.

**Release follow-ups (from prior session, still open):**
~~15. Tag `metadata` minor (Metadata[K] WAL-unification symbols are unpublished → command/query
standalone builds only work via replace today).~~ done — `metadata/v4.5.0` tagged 2026-08-16 04:15

~~16. Tag `schema` minor (`UpcastSourceTransform` unpublished; event's test dep).~~ done — `schema/v4.3.0` tagged 04:15

~~17. Tag `event` minor (ActorEnricher/WithActorContext/ActorFromContext).~~ done — `event/v4.7.0` tagged 04:16

~~18. Tag `middleware` minor (CommandActorContext).~~ done — `middleware/v4.5.0` tagged 04:16

~~19. Tag `command` patch if needed; then strip replaces from event/command/query/middleware/
integration go.mods in one sweep.~~ done for the tags (`command/v4.7.1`); repo-wide replace-strip sweep still open (TODO_LIST)

~~20. Confirm module version-sequence rule before tagging (`git tag -l '<module>/v4*' | sort -V`).

**Hardening leftovers worth considering:**~~ done — chain tagged with sequence checks; standalone-build gate added to `tag-release.sh` at `092b5e8a8` (cherry-pick onto master pending)

~~21. Decide/document AsRecord `"user:<ulid>"` legacy-UserID fallback (open question from prior
session — consumers previously saw bare ULID).~~ done — resolved in §h.2 (no behavior change; locked by `event/asrecord_test.go:69`)

~~22. `.golangci.yml` depguard: no new external deps added, but verify after go.mod edits.~~ done — no new external deps; depguard green

~~23. `TestStoreMetadataRoundtrip` actor assertion — confirm it ran against pebble AND bbolt in
this session's runs (it did via module tests; keep as regression canary).~~ done — ran via module tests; kept as regression canary

~~24. Consider golden/snapshot for watermill actor roundtrip already regenerated — verify the
`.snap` deletion didn't break `snaps.Clean` meta-tests (covered by watermill tests green).~~ done — watermill module green; `watermill/v4.5.0` tagged

~~25. gopls/LSP noise (`go.work requires go >= 1.26.6`) will disappear once toolchain matches.~~ done at `ea8fa5072` (Go 1.26.6 adopted)

~~26. Consider pinning `.go-version` file in go-cqrs-lite to 1.26.6 for non-nix users.~~ done at `ea8fa5072` (`.go-version` added)

27. Document the genproto split-pin in integration/go.mod with a comment (why it's required).
~~28. Consider extracting the actor-propagation e2e pattern into `example/` (optional, YAGNI-check).~~ Won't implement — YAGNI; the e2e pattern is documented in recipes.md §2.21

~~29. Update `CHANGELOG.md` [Unreleased] with the WithActor hardening entries if project
convention requires (verify — doc-assertion currently passes with 1 section).~~ done — CHANGELOG 2026-08-13 section + `[2026-08-16 module releases]` entries

30. Session-milestones entry for WithActor hardening completion.

## g) QUESTIONS (cannot be resolved from the repo alone)

1. **`../go-codec` sibling has an UNCOMMITTED `go.mod` bump to `go 1.26.6`** (committed
   state + tag v0.1.0 both say 1.26.5). Is that bump intentional (→ I should bring Go 1.26.6
   into this repo's flake via overrideAttrs and restore `go.work` to 1.26.6), or accidental
   (→ it should be reverted there, and `go.work` stays 1.26.5)? I won't touch the sibling's
   working tree either way without your call.
2. **AsRecord now emits `"user:<ulid>"` where legacy consumers saw the bare ULID** (UserID
   fallback is rendered with the `user:` prefix). Confirm this is acceptable/intended for
   the record layer, or tell me the exact wire format you want (bare ULID vs prefixed).
3. **The daemon already squashed the metaengine layout-calibration work into the actor
   commit.** Options: leave as-is (mixed-commit history), or I add a follow-up commit that
   only documents the split (status note) — history rewrite is off the table. Preference?

---

### Working-tree right now (uncommitted, all session-authored)

| File                                                                                                                                | Change                                                                     |
| ----------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `go.work`                                                                                                                           | ⚠ 1.26.6 → 1.26.5 (**wrong direction**, restore pending Q1)                |
| `integration/go.mod` + `go.sum`                                                                                                     | command bump v4.6.0, event+middleware replaces, genproto pin, query v4.5.0 |
| `docs/api_surface.txt`                                                                                                              | golden regen (+8 exports)                                                  |
| `.agents/skills/go-cqrs-lite/references/{core,modules,recipes}.md`                                                                  | actor docs                                                                 |
| `storage/view/{count,query}.go`, `metadata/metadata_test.go`, `integration/actor_propagation_test.go`, `metaengine/bench/*_test.go` | fmt-only                                                                   |

**Gate status: doc-assertions GREEN · per-module tests GREEN (13 modules + integration) · `#build`/`#verify-fast` RED (toolchain) · full `#verify` NOT RUN.**

---

## h) RESOLUTION (2026-08-16 ~02:00, follow-up session)

All three questions answered from evidence; no user input was required.

1. **Q1 (go-codec 1.26.6 bump) — INTENTIONAL.** The sibling has a coordinated three-way
   toolchain bump in flight: `go.mod` → 1.26.6, `.go-version` → 1.26.6, and a new
   `scripts/check-go-version.sh` tripwire enforcing all declarations agree (plus its own
   `2026-08-15_02-00 … toolchain-bump` status report). Resolution: adopt 1.26.6 here.
   The parallel ecosystem-execution session landed exactly that as `ea8fa5072`
   ("fix(toolchain): pin Go 1.26.6"): flake.nix `goToolchain` overrideAttrs (same
   SRI hash this session had computed: `sha256-oHIcV…/LLE=`), `go.work` restored to
   1.26.6 (fixing this session's wrong-direction flip), `.go-version` added, plus
   devShell tools (go-licenses, dprint, vulnix). Verified there: `go1.26.6`,
   `nix run .#build` exits 0, buildflow pre-commit passes.
2. **Q2 (AsRecord `"user:<ulid>"` legacy fallback) — NO BEHAVIOR CHANGE.** The pre-actor
   code (`1153c7d11^`) already used the identical `brandedString(tracing.UserID)`
   fallback; `id.UserID.String()` has rendered `"user:<ulid>"` since the branded-ID
   migration. WithActor only ADDS the `ActorID.PrefixedString()` path when a
   kind-discriminated actor exists. `event/asrecord_test.go:69` locks the fallback
   to `userID.String()`. Item f.21 closed — nothing to decide.
3. **Q3 (daemon-mixed commit) — LEAVE HISTORY.** Rewrite stays off the table; this
   report is the documentation of the split. Lesson recorded in (e.3) stands: commit
   immediately when separation matters.

Follow-up state at time of writing: f.1–f.5 done (via `ea8fa5072`); f.8–f.10 done —
the daemon committed the entire WithActor follow-up (skill docs, api_surface golden,
integration go.mod fix, fmt-only files) as `842741cab`; f.6–f.7, f.11–f.14 executed by
the follow-up session after the parallel session went idle (results below).
f.15–f.20 (release tags + replace-strip sweep) remain open for the release process.

## i) GATE INVESTIGATION RESULTS (2026-08-16 ~03:00, follow-up session)

A parallel agent session ("ecosystem-execution session 5") was active in this repo the
entire time; all shared-tree work below was sequenced around its activity.

1. **Toolchain fix verified independently** — `nix run .#build` green at committed HEAD
   (multiple runs). The flake's `goToolchain` overrideAttrs uses the same SRI hash this
   session computed independently (`sha256-oHIcV…/LLE=`) — cross-validated.
2. **`TestSystem_ResetProjection_RestartAndReplay` — TWO stacked causes (corrected
   after deeper bisect; the initial load-only theory was incomplete):**
   - **DETERMINISTIC REGRESSION at `313d14b02`** ("calibrate Row+Columnar layouts,
     ship DemoteEngine, converge replan" — the only code commit in 626f→a298).
     Full-suite reproduction: `cd system && GOWORK=off go test -count=1 ./...` —
     FAILS on a quiet machine; every commit 5127039da→626f7426c passes the same
     command. Phase-2 replay is DEAD, not slow: raising the test's 5s wait budget
     to 60s still yields `processed=0 errors=0`. Suspects: replan-convergence /
     DemoteEngine role defaults altering `system.New` phase-2 journal reads, or
     `replicator.applyJobFilter` skipping never-served collections. Fix belongs
     with the DemoteEngine owner.
   - **Pre-existing load sensitivity (≤ 626f7426c):** the same test also fails under
     heavy concurrent gate load (observed 02:14 in a full-suite run at 626f, which
     passes when quiet). Tight ~5s budget + t.Parallel overlap with
     `TestSystem_HealthCheck_FailedProjection`'s failing-decoder host. Recommendation:
     raise the `waitForProjectionProcessed` budget or serialize the projection-wait
     tests.
   - **The actor commit (`1153c7d11`) is verified clean** in both scopes (pair and
     full suite).
3. **`TestSoak_AutoCRUD_Bbolt` timed out at the verify gate's 8m per-package limit
   twice (loaded and quiet windows).** Duration measurement in flight; if it
   legitimately exceeds 8m the verify app needs a soak-specific budget (or
   SOAK_SKIP wiring), if it hangs it is a second committed regression in the same
   window.
4. **api-stability golden drift is the REAL repo-wide defect** (deterministic): the
   parallel session repeatedly commits new exports without regenerating
   `docs/api_surface.txt` (DemoteEngine: 4081→4085, sqliteengine DSN/OwnDB:
   4085→4087, then `event.ReconstructEventWithMetadata` +
   `storage/sql.MaxParametersForDialect`: 4087→4089). This session committed the first
   two regens (`a298ea388` and — riding along in `dba6f007b` — the 4085 file plus the
   missing watermill actor golden `message-metadata.snap`). Each fresh checkout of an
   affected revision fails the gate. The rule "regen golden in the same edit" needs a
   mechanical enforcer (e.g., pre-commit hook running the checker) — see TODO.
5. **Ops findings**: `/tmp` is a 48G tmpfs kept ~100% full by `/tmp/bigtest` (40G, not
   this session's data — needs owner decision); go temp dirs must be redirected
   (`TMPDIR`/`GOTMPDIR` to disk) or builds fail with ENOSPC at link time; the
   buildflow pre-commit hook must run inside `nix develop` (it invokes bare `go`,
   which is 1.26.5 on the system PATH).
6. **Worktree isolation pattern validated** for gating a shared moving master:
   `git worktree add` at a pinned SHA + symlinks for go.work's sibling `use` entries
   (or place the worktree in `~/projects/` where `../go-codec` etc. resolve naturally).

## j) RESOLUTION (2026-08-16 ~04:30, next session)

§i's two gate blockers are both closed — details in
`2026-08-16_03-44_withactor-resume-gate-investigation-two-defects.md` §h:

- The `313d14b02` "phase-2 replay dead" finding was a **test fixture bug**, not
  DemoteEngine logic: the in-memory shared-cache SQLite DSN was wiped when
  engines began closing self-opened `*sql.DB`. Fixed upstream by `5d66308c3`
  (`sqliteFileDSN`); verified green 3x at HEAD. The DemoteEngine suspects in §i.2
  were wrong.
- The bbolt soak gate timeout is resolved by policy: `SOAK_SKIP_BOLT=1` is
  exported by the full `#verify` app; `#test`/`#test-race` got explicit
  `-timeout=20m` so dedicated runs keep soak coverage.
- `/tmp/bigtest` was already gone (user approved trashing; tmpfs back to 1%).
