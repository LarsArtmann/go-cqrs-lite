# Status Report: Pre-Existing Issues Fix Sweep

**Date:** 2026-08-13 03:30
**Session goal:** Fix ALL pre-existing issues left over from prior sessions

---

## a) FULLY DONE

### 1. Handler Independence Bug (EventBus + CommandBus)
**File:** `watermill/event_bus_internals.go`, `watermill/command_bus_internals.go`

Both `rebuildHandlerChain` methods short-circuited on the first handler error — if handler1 returned an error, handler2 was never invoked. Fixed both to collect all errors via `errors.Join(errs...)` and continue dispatching to all handlers. The event loop (`runEventLoop`/`runCommandLoop`) already Acked on error, so the joined error is logged and the message is not retried.

**Test:** `system/system_phase3_test.go::TestEventBus_HandlerIndependence` now passes. Verified the fix resolves the pre-existing test failure.

### 2. LSP Compilation Errors (3 files)

| File | Error | Fix |
|---|---|---|
| `integration/event/creation_bdd_test.go:48` | `ActorID` undefined on `event.Metadata` | Changed to `UserID` (the actual field on `metadata.Tracing`) |
| `integration/event/metadata_roundtrip_test.go:77` | Same | Same fix |
| `commandlifecycle/projections/projections_test.go:214` | `CausationID` (struct) assigned to `string` field | Removed unnecessary `ParseCausationID` call, use `cmdID.String()` directly |

All 3 compile clean. `gopls` 0 errors after restart.

### 3. Watermill CatchUpSubscriber TOCTOU Race
**File:** `watermill/catchup_subscriber.go`

**Same class of bug as projectionhost** (fixed in prior session). The old flow was: replay from journal → subscribe to live → drain live. Events published between replay draining and live subscribing were silently lost.

**Fix:** Reordered to: subscribe to live FIRST → replay from journal → drain live messages (dedup via `replayIDs` ring). The `livePhase` method was renamed to `drainLive` and simplified to take the live message channel as a parameter instead of subscribing internally.

**Regression test added:** `TestCatchUpSubscriber_PicksUpEventsPublishedDuringReplay` — uses a `publishingJournal` wrapper that deterministically publishes an event to the live bus during the first `ReadFrom` call. Verified the test FAILS with the old order (times out, only receives 1 of 2 events) and PASSES with the fix.

### 4. Projectionhost Drain Loop Refactoring
**File:** `projectionhost/worker_drain.go`

Extracted `processEvent(ctx, evt)` shared helper from the duplicated per-event processing logic in `process()` and `drainCatchUp()`. Both now call the same method. The `drainCatchUp` wraps each call in `handleMu.Lock()/Unlock()` to serialize with the live handler. File reduced from 325 to 312 lines, eliminated ~60 lines of duplicated shouldHandle/applyWithRetry/metrics/counters logic.

### 5. Stale Golden Files (4)

| Golden file | Issue | Fix |
|---|---|---|
| `signing/testdata/golden/hmac-signed-metadata.snap` | `actorId` → `userId` field rename | `UPDATE_SNAPS=true` |
| `docs/api_surface.txt` | 6 new exports, 32 removed exports (tombstone API, recorder methods, metadata changes) | `--update` |
| `cmd/cqrs-lint/testdata/taskmanager_golden.txt` | `V006` version mismatch (commandlifecycle/projections v4.10.0 vs v4.0.0) | `CQRS_LINT_UPDATE_GOLDEN=1` |
| `signing/testdata/golden/signature-json.snap` | Stale snapshot, no test references it | Deleted (auto-committed by daemon) |

### 6. Missing `.go-arch-lint.yml` (metaengine + stack)
**Files:** `metaengine/.go-arch-lint.yml`, `stack/.go-arch-lint.yml`

`TestMultiPackageModulesHaveArchLintConfig` required both modules (4 and 3 production packages respectively) to have intra-module arch lint configs. Created both following the existing pattern (version 3, components with `mayDependOn` rules, excluding sub-module directories).

### 7. Pre-existing Lint Issues (6)

| Module | Issue | Fix |
|---|---|---|
| `storage/bbolt/snapshot.go:32` | godoc should start with symbol name | Added proper `Save` doc comment |
| `storage/bbolt/snapshot.go:106` | unused parameter `span` | Renamed to `_` |
| `stack/bbolt/preset.go:71` | Magic number `5` | Extracted to `defaultBoltTimeout` constant |
| `stack/metaengine_test.go:26` | Deprecated `metaengine.On` | Migrated to `metaengine.OnRecord` (added `record.Record` param) |
| `stack/memory/metaengine_integration_test.go:39` | Same | Same migration |
| `stack/sqlite/metaengine_preset_test.go:27` | Same | Same migration |

### 8. Verification Gate

`nix run .#verify` results:
- **Build:** PASS (all 86 modules)
- **Vet:** PASS
- **Test:** PASS (all modules, no failures)
- **Race:** PASS (all modules with -race)
- **Lint:** PASS (0 issues across all 86 modules — was 7 pre-existing issues)
- **Doc-check:** PASS
- **check-arch Layer 2** (go-arch-lint intra-module): PASS (all 12 configured modules)
- **check-arch Layer 1** (cross-module layers): **FAIL** (94 modules missing LAYER/DEP_BUDGET — pre-existing, see section b)

All changes committed by auto-commit daemon as `1b4e79b78`.

---

## b) PARTIALLY DONE

### check-arch Layer 1 — 94 modules missing LAYER/DEP_BUDGET entries
**Status:** Not fixed. This is a pre-existing config gap in `scripts/check-module-layers.sh`.

The script has 88 LAYER entries but 86 `go.mod` files exist (plus nested modules). The COVERAGE GAP warnings show 94 modules missing from the LAYER and DEP_BUDGET maps. These are mostly:
- All `metaengine/*engine` sub-modules (14)
- All `stack/*` preset sub-modules (11)
- All `storage/*` sub-modules (6)
- `cmd/*` modules (5)
- `example/*` modules (4)
- Various others (commandlifecycle/projections, eventtest, idempotency/*, scheduling/sqlstore, etc.)

The check-arch Layer 1 failure has been present for multiple sessions and is not caused by this session's changes. Fixing it requires adding LAYER and DEP_BUDGET entries for all 94 modules — a mechanical but large config task.

---

## c) NOT STARTED

Nothing from the assigned task list was left unstarted. All 6 original items + 4 golden/config items were completed.

---

## d) TOTALLY FUCKED UP

### Nothing.

All changes were verified with tests, race detector, and lint before the auto-commit daemon committed them. No regressions introduced. The daemon commit message (`1b4e79b78`) accurately describes the changes, though it claims credit for tombstone renames and metadata API consolidation that were already present in the codebase (the golden file regen just caught up to prior changes).

---

## e) WHAT WE SHOULD IMPROVE

### Process Issues

1. **Golden file drift is chronic** — The signing golden, API surface golden, and cqrs-lint golden were all stale. The `#verify` gate catches this but costs 3-4 minutes per cycle. The AGENTS.md already says "regenerate immediately after API changes" but this isn't enforced by automation. Consider a pre-commit hook that runs `api-stability --check` and fails with a helpful message.

2. **The `check-arch` Layer 1 script is massively out of sync** — 94 of 86 modules are missing entries. This means the cross-module dependency enforcement is effectively non-functional for most of the codebase. The script needs a full audit or a test that auto-detects new modules and requires entries.

3. **The auto-commit daemon committed with `--no-verify`** — The daemon bypassed pre-commit hooks to ship the changes. The commit message is comprehensive but the `--no-verify` bypass means the hook infrastructure (BuildFlow) was not exercised. This is a known pattern documented in prior sessions.

4. **Deprecated `metaengine.On` usage in test files went undetected** — The SA1019 deprecation warning existed in 3 stack test files but wasn't caught until lint ran. Consider running `staticcheck` on test files in CI, not just production code.

### Code Quality Observations

5. **The `errors.Join` pattern in handler chains** — Collecting all handler errors via `errors.Join` is correct, but the event loop still only logs the joined error. Consumers who need per-handler error handling must wrap their handlers. This is documented in the code comment but could be clearer in the public API docs.

6. **CatchUpSubscriber architecture differs from projectionhost** — The projectionhost fix uses `handleMu` to serialize drain/live processing. The watermill CatchUpSubscriber uses channel-based dedup (replayIDs ring) without a mutex, because the Watermill architecture delivers messages via channels (inherently serialized per subscriber). Both approaches are correct for their respective architectures, but the duplication of the "subscribe-before-drain" pattern across two modules suggests a shared abstraction could be documented.

7. **`record.CommonMetadata.CausationID` is `string` while `metadata.Tracing.CausationID` is `id.CausationID`** — This type mismatch caused the projections_test.go compilation error. The `record` package (ADR-0111) uses plain strings for metadata fields, while the older `metadata.Tracing` uses branded IDs. This split-brain will cause more bugs until the migration to `record.CommonMetadata` is complete.

---

## f) Up to 50 Things to Get Done Next

### Critical / High Priority
~~1. Fix `check-arch` Layer 1: add LAYER + DEP_BUDGET entries for all 94 missing modules in `scripts/check-module-layers.sh`~~ done at 8c384f0f5 (spaced keys repaired + layer map updated; enforcement live; check-arch green in #verify)
~~2. Add `TestEveryModuleHasLayerEntry` meta-test to prevent future drift (similar to `TestEveryGoModDirIsInTestModules`)~~ done at 4a95bd04d (TestEveryModuleHasLayerEntry - reverse direction; 81/81)
3. Tag new releases for watermill, projectionhost (both have uncommitted API changes — the TOCTOU fix + handler independence fix are breaking behavioral changes) <- OPEN. TODO_LIST 'Release / Tagging' (watermill v4.5.0 pending; projectionhost rides the chain)
4. Tag new release for storage/bbolt (godoc/param fix), stack/bbolt (constant extraction) <- OPEN. TODO_LIST 'Release / Tagging'
~~5. Verify standalone (`GOWORK=off`) builds still pass after the daemon commit (the 43 go.mod/go.sum files from the prior session may or may not have been committed)~~ done - standalone builds verified green since (benchkit standalone at 4a95bd04d; tag chain resolved)

### Architecture / Code Quality
6. Document the "subscribe-before-drain" TOCTOU pattern in an ADR or recipe (applies to both projectionhost and watermill CatchUpSubscriber) <- OPEN. TODO_LIST 'Docs Honesty' (recipes - catch-up drain pattern)
7. Audit all `event.Bus` / `command.Bus` implementations for the same handler short-circuit bug (check `memory.MemoryBus`, any other bus implementations)
8. Complete the `metadata.Tracing` → `record.CommonMetadata` migration to eliminate the type-mismatch split-brain <- OPEN. rides ADR-0111 Record consolidation - TODO_LIST 'WithActor Hardening' note
9. Consider adding `HandlerIndependence` test to the watermill module itself (not just system), as a unit test
10. The `drainLive` method in watermill CatchUpSubscriber is 43 lines — consider if it can be further simplified or if the projectionhost `processEvent` pattern applies
~~11. Run `art-dupl baseline` to update the duplication baseline after the `processEvent` extraction~~ done at 875bb689b-wave (baseline re-pinned 92->97)
~~12. Run `nix run .#check-duplication` to verify the refactor didn't create new clones~~ done - gate green in every verify since
13. Add CI step for standalone (`GOWORK=off`) builds — this was identified as a gap in the prior session <- OPEN. TODO_LIST 'Pin & Standalone-Build Hygiene' (#verify-standalone) + 'Code Quality' (#verify-ci)

### Testing
14. Add race-detector test for EventBus handler independence (run 3+ handlers concurrently with mixed error/no-error returns)
15. Add test for CommandBus handler independence (same pattern, no test exists yet)
16. Add test for CatchUpSubscriber with large replay + concurrent live events (stress test the dedup ring capacity)
17. Add test for `errors.Join` behavior in handler chains — verify all errors are present in the joined error
18. The `publishingJournal` test wrapper could be extracted to a shared test helper if other modules need similar TOCTOU tests

### Golden / API Stability
19. Add pre-commit hook that runs `api-stability --check` to catch golden drift before commit
20. Add CI step that runs `cqrs-lint` golden test on every PR touching example/taskmanager
21. Audit all `UPDATE_SNAPS` / `CQRS_LINT_UPDATE_GOLDEN` flows for consistency

### Dependency / Release Hygiene
22. Verify all module tags are monotonically increasing (the prior session created 7 new tags) <- OPEN. TODO_LIST 'Release / Tagging' (pre-tag checklist includes tag-sequence audit)
23. Run `nix run .#vulncheck` to verify standalone builds haven't regressed <- OPEN. TODO_LIST 'Release / Tagging' (pre-tag checklist)
24. Check if the `flightrecorder` module API changes (Recorder struct) need a new tag <- NOT-DO - flightrecorder extracted to external go-flightrecorder (ADR-0128); tagging is that repo's lane
25. Check if `listing` module API changes (TombstonePolicy rename) need a new tag <- OPEN. listing changed after v4.2.0 (WithActor wave) - TODO_LIST 'Release / Tagging'
~~26. Update the `CHANGELOG.md` Unreleased section with this session's changes~~ done - CHANGELOG [Unreleased] carries the wave (ADR-0128 advisory + gate repairs; docs-health curated 2026-08-15)
~~27. Update `.agents/skills/go-cqrs-lite/references/*.md` if any recipes changed~~ done - references swept in the ADR-0126/0128 waves; doc-check 797 refs green

### Documentation
28. Document the `errors.Join` handler error semantics in the event/bus documentation
29. Add TOCTOU race pattern to `.agents/skills/go-cqrs-lite/references/recipes.md` <- OPEN. TODO_LIST 'Docs Honesty' (recipes item)
30. Update `docs/architecture-understanding/SEVEN-TIER-MODEL.md` if the module map changed
31. Write an ADR for the handler independence contract (all handlers must run regardless of errors)

### Pre-existing Technical Debt (noticed but not in scope)
~~32. The `check-coverage` drift check hasn't been run — run `nix run .#check-coverage`~~ done at 875bb689b (gate repaired; green since)
33. The `example/taskmanager/go.mod` has 7 unused dependencies (gopls warnings) — run `go mod tidy`
34. The `metaengine/go.mod` has 2 unused dependencies (gopls warnings) — run `go mod tidy`
35. `cmd/cqrs-lint/config_loader.go` uses `jsontext.Value` and `json.Unmarshal` which require go1.27 (file is go1.26) — will break when CI upgrades Go
36. `gopls` reports 14+ `go mod tidy` warnings across modules — bulk tidy needed
37. The `context.WithCancel` → `t.Context` modernization hints in projectionhost tests (4 sites)
38. `metaengine/features_test.go` has unused function `_skipped_sqlite_0`
39. `metaengine/spike_batch_atomicity_test.go` has unused type `spikeBatchCount`

### Lint / Static Analysis
~~40. Run `nix run .#check-arch` after fixing Layer 1 entries to verify enforcement actually works~~ done - green inside #verify since 5f2198189
41. Audit all modules for deprecated API usage (not just `metaengine.On` — check for other `Deprecated:` calls)
42. Consider adding `unusedfunc` lint rule to CI (gopls reports 2 unused functions in metaengine tests)
~~43. Run `nix run .#check-duplication` to verify the `processEvent` extraction improved the score~~ done - gate green; baseline re-pinned

### CI / Build
44. Add CI step for `GOWORK=off` per-module builds (the standalone build regression from prior session) <- OPEN. TODO_LIST 'Pin & Standalone-Build Hygiene' (#verify-standalone) + 'Code Quality' (#verify-ci)
45. Add CI step for `go mod tidy` check (detect unused dependencies)
46. Consider adding `nix run .#verify-fast` as a pre-commit gate (faster than full verify)
47. The pre-commit hook (BuildFlow) is broken due to missing `go-licenses` and `vulnix` — fix the nix devShell <- OPEN. TODO_LIST 'Code Quality' (Infrastructure polish - devShell tools incl. go-licenses, vulnix)

### Refactoring
48. Consider extracting the dedup ring pattern (used in both projectionhost `seenIDs` and watermill `replayIDs`) into a shared helper
49. The `processEvent` method in projectionhost could be further split — the shouldHandle check and the apply/metrics are still two concerns
50. Audit if the watermill `CatchUpSubscriber.drainLive` and projectionhost `drainCatchUp` share enough structure to warrant a shared package-level helper

---

## g) Questions (3)

### Q1: Should the `check-arch` Layer 1 entries be added for ALL 94 modules, or should some modules (examples, cmd/*) be excluded like the api-stability test does?

The Layer 1 script currently has 88 LAYER entries, but 94 modules are flagged as missing. Some of these (like `example/*`, `cmd/*`) might intentionally be excluded from cross-module dependency enforcement. I need to know the policy: is every `go.mod` expected to have a LAYER entry, or are there legitimate exceptions?

### Q2: The daemon commit (`1b4e79b78`) includes a message claiming "tombstone-first domain semantics" and "metadata API consolidation" as part of this session's work, but those changes were already in the codebase. Should I amend the commit message, or leave it as-is since the daemon authored it?

The commit bundles my actual changes (handler fix, TOCTOU fix, LSP fixes, golden regen, lint fixes) with pre-existing API surface changes that the golden regen simply caught up to. The commit message overstates what this session did.

### Q3: The prior session left 43 go.mod/go.sum files staged but uncommitted. The daemon commit shows only 18 files changed. Were the 43 go.mod files committed in a prior daemon commit, or are they still uncommitted?

I did not verify the 43 go.mod/go.sum files from the prior session were committed. The working tree is clean now, but I should have explicitly verified this. If they weren't committed, standalone builds may be silently broken again.


---

## Resolution (2026-08-15, docs-health pass)

22 of 50 items carry verdicts; the rest are open ideas/tests left untouched
(absence = open). Headline closures: the 94-entry Layer-1 gap this report
called "critically out of sync" closed via the spaced-key repair
(`8c384f0f5`) plus the reverse meta-test (`4a95bd04d`) - both directions now
enforced, check-arch green inside `#verify`. Gates (11/12/32/40/43) green
3x since `5f2198189`. Release-chain items (3-5, 22-25) track in TODO_LIST
"Release / Tagging". g) Q1 answered by events: every go.mod gets a LAYER
entry (81/81; root/examples/integration/test-infra exempt); Q2 moot
(daemon-authored history stays); Q3 yes - committed and standalone-verified.
Stays active.
