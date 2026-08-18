# Status Report — system/v4 Follow-Ups Execution: Roles, Durability, Honesty Docs, Coverage

**Date:** 2026-08-18 15:05
**Session scope:** Continuing the "system/v4 Full-Code-Review Follow-Ups" backlog
(`TODO_LIST.md` §system/v4). Previous checkpoint:
`2026-08-18_14-21_system-follow-ups-execution-checkpoint.md`.
**Repo state:** master, working tree = 1 modified file (`system/roles.go`, the
exhaustive-lint fix — the auto-commit daemon committed everything else as
`6577f26a3`).

---

## a) FULLY DONE (verified GREEN this session)

1. **`system/roles_test.go` compile fixes** — `NewStreamRef(streamType, streamID)`
   arg order corrected; snapshot API replaced with the real one
   (`snapshot.Snapshot` struct + `Save(ctx, snap)` / `Load(ctx, ref)`).
   All 5 role-wiring tests GREEN; `-count=3 -race` GREEN; full system suite GREEN
   (first full-suite run of this work stream).
2. **api-stability golden regenerated** — two regen waves (role/named-bus symbols,
   then durability symbols), 19 new exported symbols total; gate + `TestEvery`
   meta-tests GREEN both times.
3. **Durability wiring — metaengine core** (NEW `metaengine/durability.go`):
   `DurabilityTier` (strict/normal/relaxed; empty = engine defaults),
   `ValidateDurabilityTier`, `RejectDurabilityTier` (fail-loudly helper for
   drivers without tiers), `ErrUnsupportedDurability`, `DriverConfig.Durability`
   field. Memory driver: rejects `strict` (in-process storage cannot fsync),
   accepts empty/normal/relaxed, rejects invalid. Full metaengine suite GREEN
   (27.4s).
4. **Durability wiring — sqliteengine** (NEW `durability.go`): tier →
   `PRAGMA synchronous` FULL/NORMAL/OFF table (lifted from the stack presets per
   proposal §8); operator-set `synchronous` pragma + explicit tier = construction
   error (two sources of truth for one knob is a config error). 4 tests GREEN.
5. **Durability wiring — 8 engine guards**: badger, bbolt, dgraph, duckdb (CGo),
   mysql, pebble, postgres, turso register.go factories now call
   `RejectDurabilityTier` first; per-module guard tests all GREEN (pg 13.6s — it
   retries connection).
6. **Durability wiring — system** (NEW `system/durability.go`):
   `resolveEngineDurability` resolves per-ENGINE tiers from per-INSTANCE tiers;
   disagreement → `ErrDurabilityConflict` (answers open question #1 from the last
   checkpoint: fail construction, not strictest-wins); invalid tier names the
   instance in the error. Wired through `createEngineFromDriver`.
   **Removed the config loader's silent `normal` defaulting** — unspecified means
   engine defaults; silently stamping "normal" would now push an explicit tier
   onto engines without tier support. 6 system tests GREEN, incl. sqlite strict
   construct + memory strict fail + agreeing/conflicting instances.
7. **Reserved-config honesty** (answers open question #3: document-reserved, not
   delete): `BusConfig.Mode` documented introspection-only (README `mode: sync`
   example removed); `InstanceConfig.Subscribe` + `CacheConfig.Engine`
   documented reserved/not-read (removal at v5); `InstanceConfig.Collections`
   documented introspection-only; **discovered `Evolve(Internal())` is also a
   dead flag** — documented as recorded-but-not-enforced. `system/v4` still
   compiles and vets clean.
8. **EventAdapter backend contract doc** (NEW `system/doc.go`): Save atomicity
   classified AtomicAppender (ALL 10 shipped engines — verified by grep) →
   Transactional → racy fallback, with crash-window guidance.
9. **Coverage: system 74.4% → 79.4%** — 3 new test files:
   `cache_extra_test.go` (CachedEventStore passthroughs, stats, capability
   fallbacks via a bare `event.Store` wrapper), `adapter_command_test.go`
   (CommandAdapter batch + time-filtered loads + journal reads with
   deterministic `WithReceivedAt`), `evolutions_options_test.go` (`EvolveKey`,
   `Internal`, and the `reifyTo` JSON branch driven through a SQLite engine).
10. **CHANGELOG** — `[Unreleased]` Added (follow-ups) + Changed (reserved
    honesty, Save contract) sections; `check-changelog-symbols.sh` GREEN (59
    citations verified).
11. **TODO_LIST backlog ticked** — 9 of 10 items `[x]` with one-line DONE notes;
    only Release coordination remains open.
12. **Skill refs updated** — `references/modules.md` metaengine + system rows
    (named dispatch, durability tiers, role wiring, PublisherFor, Save
    contract); `cmd/doc-check` GREEN (921 refs, 42 packages).
13. **`nix fmt`** GREEN — also reformatted 3 pre-existing-unformatted idempotency
    test files (committed unformatted by an earlier session; formatting-only
    diff, kept).
14. **Verify lint failure FIXED** — `nix run .#verify` failed at the final lint
    step: 2 `exhaustive` findings in `system/roles.go` (the switches written
    earlier this session). Fixed with explicit case labels + 1 scoped nolint
    (loop over exactly 3 roles). `nix run .#lint` standalone now **76/76 modules
    clean**; role tests re-verified GREEN.

## b) PARTIALLY DONE

1. **Full verify single-run GREEN** — the one full `#verify` run passed
   build/vet/test/race for all modules (240 `ok` lines) and failed ONLY at lint;
   lint is now fixed and standalone-green. A fresh end-to-end `#verify` run for
   a clean single-pass record has not been re-executed (≈10 min). This is the
   "stale GREEN" caveat: composite green, not single-run green.
2. **Durability breadth** — only sqlite maps tiers natively; the other 8 engines
   fail loudly on explicit tiers (honest, per proposal, but pebble
   (`Sync`/`DisableWAL`), postgres (`synchronous_commit`), and bbolt (`NoSync`)
   have real knobs available and were deliberately deferred.

## c) NOT STARTED

1. **Release coordination** (backlog item 7): metaengine v4.12.0 → engine
   adapters → system/v4.5.0 with replace-stripping (go-release flow). All CODE
   prerequisites are now done; awaiting the release-window decision.
2. **v5 wave implementation choices** flowing from the honesty docs (implement
   `BusConfig.Mode` sync/async vs delete; enforce `Internal()` vs delete).

## d) TOTALLY FUCKED UP (session mistakes — all caught and fixed)

1. **Exhaustive lint findings shipped into the verify run** — I wrote two
   enum switches in `roles.go` and never linted the module before the 10-minute
   gate; the gate failed at its LAST step. First remediation attempt (default
   clauses) also failed — this repo's exhaustive config does not set
   `default-signifies-exhaustive`. Second attempt (explicit cases) passed.
   Two wasted cycles that a 5-second per-module `golangci-lint run` after
   editing would have prevented.
2. **Write-tool refusals ×8** — batched `write` calls on engine register.go
   files I had only `cat`-ed, not `view`-ed. The tool requires View-tool reads;
   re-viewed all 8, then the writes applied. Pure round-trip waste.
3. **First draft of `system/durability.go`** contained a leftover `maps` import
   kept alive by a dummy `var _ = maps.Copy[...]` line — deleted before commit.
4. **First draft of `metaengine/durability_test.go`** hand-rolled
   `contains`/`indexOf` helpers instead of `strings.Contains` — replaced.
5. **`system/doc.go` first draft** had a second package doc comment (would trip
   lint) — caught by self-review, restructured before build.
6. **Test-authoring API guesses** (again): missing key field panicked AutoInsert
   (fixed with `EvolveKey("Other")`); `SliceFromVersion` is index-based, not
   version-exclusive (fixed expectation). Pattern holds: grep the interface
   BEFORE writing tests.
7. _(Inherited, now resolved)_ the daemon-committed non-compiling `roles_test.go`
   from the prior session — this session's first action fixed it; the repo has
   been continuously compilable since.

## e) WHAT WE SHOULD IMPROVE

1. **Lint changed modules immediately** — `cd <module> && golangci-lint run`
   right after editing, not at the end of a 10-minute gate. The exhaustive
   failure was 3 hours stale by the time the gate caught it.
2. **Batch-write protocol** — when planning parallel `write` calls, View every
   target FIRST in one batch; the tool enforces it and mixed read/write batches
   just fail.
3. **The verify gate runs lint LAST** — a lint-only failure costs the full
   build+test+race runtime. Consider running `#lint` before `#verify` locally,
   or flipping gate order (flake observation, not changed this session).
4. **`Internal()` was a dead public option** found only by coverage-driven
   reading — a capability audit pass over exported options (do they change any
   behavior?) would catch this class mechanically.
5. **Test-API assumptions remain the #1 time sink** — the repo's SKILL.md
   references have exact signatures; consult them before writing tests against
   unfamiliar modules (snapshot/command/eventtest).
6. **Engine durability mapping is a copy-paste-shaped task** (pebble/pg/bbolt
   each need ~30 lines + tests); a follow-up session could clear all three in
   one pass using the sqliteengine file as the template.

## f) NEXT — ordered, up to 50

**Gates & wrap-up (do first):**

1. Re-run `nix run .#verify` end-to-end for a single-run GREEN record (~10 min).
2. `nix run .#check-coverage` — confirm the 79.4% system lift moved the drift
   gate the right way.
3. `nix run .#check-arch` + `#check-duplication` — durability code touched 10
   modules; verify no new clones tripped the baseline (engine guard tests carry
   `//art-dupl:accept` directives).
4. `nix run .#load-sweep` — timing paths untouched this session, but the sqlite
   pragma order changed (synchronous now applied before user pragmas); cheap to
   confirm.
5. Commit the `system/roles.go` exhaustive fix (daemon will likely do it —
   verify it lands).

**Release wave (blocked on user answer, then go-release flow):**
6. Decide release window (question g-1).
7. `git tag -l 'metaengine/v4*' | sort -V | tail -1` — confirm v4.12.0 is next.
8. Sweep local replaces: `grep -rn "=> \.\./\|=> /" --include=go.mod .` — drop
every replace whose target has a published tag (system has 6).
9. Cut metaengine/v4.12.0 via `scripts/tag-release.sh` (annotated tags only).
10. Re-tag engine adapters that consumed unpublished metaengine symbols.
11. Cut system/v4.5.0 with replaces stripped; `#vulncheck` per-module standalone
build to catch version-sequence breaks.
12. Verify `go get github.com/larsartmann/go-cqrs-lite/system/v4@v4.5.0` resolves
from a clean module cache.
13. Roll the CHANGELOG `[Unreleased]` sections into versioned entries at cut
time.

**Durability breadth (post-release or pre-release if user wants):**
14. pebbleengine: tier → `Sync`/`DisableWAL` write options mapping + tests.
15. pgengine: tier → `synchronous_commit` session var + tests.
16. bboltengine: tier → `NoSync`/`NoFreelistSync` + tests (mirror stack/bbolt
semantics: preset defaults to Strict).
17. badgerengine: tier → `SyncWrites` option + tests.
18. tursoengine: same PRAGMA path as sqlite (it delegates to sqliteengine) —
accept tiers instead of rejecting.
19. mysqlengine: `innodb_flush_log_at_trx_commit` mapping (1/2/0) if desired.
20. Surface resolved per-engine tiers in `Introspection()` output.
21. Add a scream rule: durability tier set on an engine the driver ignores →
already an error; add advisory for `relaxed` on non-SOT instances (none
exists).

**system/ polish:**
22. Push coverage further: `introspection_extended.go` (3 fns), `multi_bus.go`
(2), `runtime.go` (2), `projection_builder.go` (2) still have zero-cov funcs.
23. `query_constructors.go` — 6 of 20 functions at 0% (error paths mostly).
24. Consider a repo-wide minimum coverage threshold wired into
`check-coverage.sh` (none exists today).
25. Implement or delete `BusConfig.Mode` (v5 decision — question g-3).
26. Implement per-instance `Subscribe` (projectionhost per-bus consumption) or
delete at v5.
27. Enforce `Internal()` evolution marker (exclude from Lookup/Find builds) or
delete the option at v5.
28. Named dispatch for `system.Get`/`Find` (currently only GetCount by name —
Lookup/QuerySet share input types by construction, but explicit name
dispatch would be consistent).
29. `EventAdapter.Save` racy fallback: emit a scream-style WARN at construction
when an engine implements neither AtomicAppender nor Transactional.
30. Document durability tier → engine matrix in `system/README.md` (which
drivers accept which tiers).

**Docs & meta:**
31. Update `.agents/skills/go-cqrs-lite/references/recipes.md` with a
named-dispatch recipe (two counters example) — modules.md row updated, no
recipe yet.
32. Consider an ADR for the durability-tier contract (currently only proposal
§5 + godoc).
33. Mark the review HTML (`docs/reviews/2026-08-16_full-code-review-system.html`)
items as routed/done to match TODO_LIST.
34. Retire the `/tmp` cache env workaround from AGENTS.md buildcache gotcha
(repaired 2026-08-18, noted in TODO_LIST; AGENTS.md still tells agents to
redirect caches).
35. The exhaustive-vs-default lesson → add a line to AGENTS.md gotchas
(`default:` does NOT satisfy the exhaustive linter in this repo).

**v5 pre-cut wave (deferred, tracked in TODO_LIST §v5):**
36. Delete `CacheConfig.Engine` / `Subscribe` reserved fields at v5.
37. Delete `BusConfig.Mode` (if implement-not chosen in #25).
38. Stack preset deletion Phase 8 (ADR-0123) — durability tables now have a
metaengine home, making the stack lift complete.
39. `record.NewStreamRef` signature change (NOTE already in place).
40. Port scream-rule ACK key format doc into SKILL.md references (currently only
AGENTS.md).

**Nice-to-have:**
41. Engine guard tests could share one table via `enginetest` harness — blocked
by dep isolation; keep as clones with directives (already accepted).
42. `Introspection()` could report resolved durability per engine (overlaps #20).
43. Bench: measure sqlite strict-vs-relaxed write throughput delta to quantify
the tier trade-off in docs.
44. Consider `EXPLAIN`-style output for durability resolution (what tier each
engine got and why) in `Doctor`.
45. Cleanup: `docs/status/2026-08-18_14-21_*` checkpoint is superseded by this
report — annotate or leave as history (docs-health HARVEST could pull the 3
then-open questions; 2 are now answered).
46. Update the review HTML follow-up status footer if it lists the 10 items.
47. Meta-test idea: assert every `RegisterDriver` factory calls a tier check
(source-grep test in metaengine) so new engines cannot skip it.
48. `slices.Backward` tripwire check on `metaengine/pebbleengine` nextKey (known
daemon-revert hotspot) during the pebble durability work (#14).
49. Run `UPDATE_SNAPS=clean` pass if any golden snapshots referenced Count
dispatch output (none expected — named dispatch is additive).
50. Close out: mark this status report done in the next checkpoint once #1-#5
are green.

## g) Questions (cannot figure out myself)

1. **Release timing** (carried from the last checkpoint, now the only blocker):
   cut metaengine/v4.12.0 + system/v4.5.0 NOW with this session's work, or
   defer into the v5 pre-cut wave? All code prerequisites are done; published
   system/v4.4.0 still carries all 5 P1 bugs until this ships.
2. **Durability breadth before release?** Only sqlite maps tiers today; the
   other engines fail loudly on explicit tiers. Should pebble/pg/bbolt/badger
   mappings (#14-#17, maybe 1-2 hours total) land BEFORE the release so
   operators get real coverage, or is fail-loudly + sqlite enough for v4.5.0?
3. **`BusConfig.Mode` endgame:** implement real sync/async publish semantics at
   v5, or delete the field? (I documented it introspection-only; the proposal
   table says "implement or delete" — the choice is yours, and it determines
   whether the YAML `mode:` key ever does anything.)
