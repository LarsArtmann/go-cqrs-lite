# Status Report: Metaengine Phase 3 Universal ADT + Verification Foundation

> **Date:** 2026-08-03 07:01
> **Session scope:** Execute the 27-task Pareto plan from `docs/planning/2026-08-03_04-18_METAENGINE-PHASE3-UNIVERSAL-ADT-AND-VERIFICATION-FOUNDATION.md`
> **Starting state:** Plan committed and pushed, 0 tasks executed. Verify gate GREEN (but missing 3 checks).

---

## a) FULLY DONE (verified, committed, tests passing)

### T1: Extend `nix run .#verify` with check-layers + check-duplication + check-coverage

- **What:** Added three quality gates to the `verify` app in `flake.nix` (lines ~833-835). These now run automatically between Lint and API Stability.
- **Bug found and fixed:** `check-module-layers.sh` was missing 21 of 64 modules from its LAYER and DEP_BUDGET maps. `stack` budget was 14 but actual production deps were 17. Added all missing modules (metaengine, pebbleengine, duckdbengine, pgengine, projectionadapter, flightrecorder, retry, transport/grpc, idempotency/kvstore, idempotency/sqlstore, benchkit, stack/duckdb, stack/mysql, stack/turso). Removed `testutil` from LAYER (it's test-only infrastructure, creates false layer violations from lower-tier modules).
- **Verified:** All three checks now exit 0 individually AND as part of `nix run .#verify`.
- **Commit:** `d4dbebbd`

### T2: Fix design doc lie — MapUpdate "emits" → "should emit"

- **What:** Changed `docs/planning/meta-engine-eventual-consistency-and-iroh.md:279` from "The planner emits a WARN diagnostic" to "Recommended: the planner should emit a WARN diagnostic."
- **Commit:** auto-committed by daemon

### T3: Push unpushed commits

- **What:** All commits pushed to `origin/master`.

### T4: Tag `metaengine/v4.3.0`

- **What:** Created annotated tag `metaengine/v4.3.0` with message describing replication model work. Tag pushed to remote.
- **Commit tagged:** `6f7c8838`

### T5: Add `DegradedADTs` field to `EngineProfile`

- **What:** Added `DegradedADTs map[ADT]bool` field to `EngineProfile` struct in `metaengine/engine.go`. Added `IsDegraded(adt ADT) bool` method. Documented semantics: Supports = "can I do this?", DegradedADTs = "am I good at this?"

### T6: Extend SQLite engine to 10/10 ADTs

- **What:** Added ADTVector, ADTSearch, ADTSpatial to SQLite `Supports` map as `ComplexityON`. Marked all three in `DegradedADTs`.

### T7: Extend Pebble engine to 10/10 ADTs

- **What:** Same as SQLite — added Vector/Search/Spatial as O(N) degraded.

### T8: Extend DuckDB engine to 10/10 ADTs

- **What:** Added 7 missing ADTs (Set, Graph, Log, Multimap, Vector, Search, Spatial) all as O(N) degraded.

### T9: Extend Postgres engine to 10/10 ADTs

- **What:** Added same 7 missing ADTs as DuckDB, all as O(N) degraded.

### T10: Implement `degradedADTRule` with SCREAM diagnostics

- **What:** Created `metaengine/rule_degraded_adt.go`. Rule emits `DEGRADED` diagnostic: `"DEGRADED: {adt} routed to {engine} via {complexity} fallback — native engine recommended for production"`. Also adds RuleTrace entry.
- **Registered** in `defaultRules()` in `rules.go`.

### T11: Improve `errADTNotSupported` error message

- **What:** Changed error message in `planner.go:184` to include actionable guidance: `"add a Memory engine (supports all ADTs) or declare it as degraded on an existing engine via DegradedADTs"`.

### T12: Integration tests for universal ADT (8 tests, all pass)

- **What:** Created `metaengine/universal_adt_test.go` with 8 tests:
  - `TestUniversalADT_MemoryEngineHasAllTenADTs` — verifies Memory has all 10 ADTs, none degraded
  - `TestUniversalADT_DegradedDiagnosticEmitted` — verifies SCREAM diagnostic fires for degraded routing
  - `TestUniversalADT_NoDegradedDiagnosticForNative` — verifies no diagnostic for native ADT
  - `TestUniversalADT_PrefersNativeOverDegraded` — verifies cost ranker picks native engine over degraded
  - `TestUniversalADT_DegradedRuleInRuleTrace` — verifies RuleTrace entry
  - `TestUniversalADT_NoDegradedWhenDegradedADTsNil` — verifies nil DegradedADTs never triggers
  - `TestUniversalADT_IsDegradedMethod` — verifies method behavior for set/unset cases
- **All pass** under `-count=1`.

### T13: Write ADR-0094: Universal ADT Support

- **What:** Created `docs/adr/0094-metaengine-universal-adt-support.md`. Full ADR with context (fragmentation problem), decision (DegradedADTs + SCREAM), coverage matrix (before/after), consequences.
- **docs/README.md** ADR index updated: count 91→92, added ADR-0094 entry.

### Coverage matrix after T5-T13 (ALL engines at 10/10)

| ADT       | Memory | SQLite  | Pebble  | DuckDB  | Postgres |
| --------- | ------ | ------- | ------- | ------- | -------- |
| Map       | O(1)   | O(logN) | O(1)    | O(logN) | O(logN)  |
| Set       | O(1)   | O(logN) | O(1)    | O(N)*   | O(N)*    |
| Counter   | O(1)   | O(1)    | O(N)    | O(1)    | O(1)     |
| Graph     | O(deg) | O(N)    | O(N)    | O(N)*   | O(N)*    |
| SortedMap | O(N)   | O(logN) | O(N)    | O(logN) | O(logN)  |
| Log       | O(N)   | O(logN) | O(logN) | O(N)*   | O(N)*    |
| Multimap  | O(1)   | O(logN) | O(logN) | O(N)*   | O(N)*    |
| Vector    | O(N)   | O(N)*   | O(N)*   | O(N)*   | O(N)*    |
| Search    | O(N)   | O(N)*   | O(N)*   | O(N)*   | O(N)*    |
| Spatial   | O(N)   | O(N)*   | O(N)*   | O(N)*   | O(N)*    |

`*` = degraded (non-native fallback)

---

## b) PARTIALLY DONE (started but interrupted)

### T14-T18: Replication Polish — NOT STARTED (0%)

The plan to add `WithReplication()`, `WithNetworkRTT()`, SerializablePlan replication fields, `ReplicationMode()` accessor, and MapUpdate WARN diagnostic was **not started**. The `planConfig` struct in `planner.go` was about to be edited (read the struct, understood the pattern) but no code was written. No partial changes exist in the working tree.

---

## c) NOT STARTED

### T19-T24: TODO_LIST Backlog (0%)

- T19: 10M soak test hardening — not started
- T20: Watcher typed-channel — not started
- T21: SSE+SQLite reconnect test — not started
- T22: Boundary key-type validation — not started
- T23: Postgres GIN containment indexes — not started
- T24: DuckDB LayoutPlanner follow-ups — not started

### T25-T27: Other (0%)

- T25: Iroh bridge evaluation ADR — not started
- T26: gopls hint cleanup (6 infertypeargs + 1 writestring) — not started
- T27: Run cqrs-lint against real consumer projects — not started

---

## d) TOTALLY FUCKED UP

### 1. `nixos.qcow2` (43MB binary) committed to git

A 43MB NixOS VM disk image (`nixos.qcow2`) was committed to the repository root. It is NOT in `.gitignore`. This bloats the repo and should be removed from tracking immediately. **Root cause:** a prior session's NixOS VM testing left the artifact, and the auto-commit daemon committed it.

### 2. API surface golden NOT regenerated for new symbols

`docs/api_surface.txt` has `metaengine/method IsDegraded` but is **missing**:

- `metaengine/field DegradedADTs` (the struct field on EngineProfile)
- `metaengine/method (EngineProfile).IsDegraded` is present but the field is missing

The AGENTS.md lint convention says: "API-surface changes require golden regen in the same edit." This was violated. The `nix run .#check-api-stability` gate will catch this, but it was not run in this session.

### 3. `nix run .#verify` was NOT run after all changes

Despite T1 being "extend the verify gate," the full verify gate was **never run end-to-end** after the T5-T13 changes. Only individual checks (check-layers, check-duplication, check-coverage, metaengine tests) were run. This is the exact "STALE GREEN" anti-pattern documented in AGENTS.md.

### 4. No race-detector run on new tests

The 8 universal ADT tests were verified with `-count=1` but NOT with `-race`. The plan explicitly calls for `go test -race ./metaengine/...` (S3.38).

### 5. Metaengine v4.3.0 tag does NOT include the Universal ADT work

The tag `metaengine/v4.3.0` was cut at commit `6f7c8838`, which is **before** the Universal ADT changes (commits `d4dbebbd` through `8b41f658`). The DegradedADTs field, degradedADTRule, universal_adt_test.go, and ADR-0094 are all **unreleased**. Consumers resolving `metaengine/v4` get v4.3.0 which has none of this work. A v4.4.0 tag is needed after verify passes.

### 6. Commit message corruption

Commit `8b41f658` has message `"ore(metaengine): add ADR 0094..."` — the `ch` prefix of `chore` was truncated by the auto-commit daemon.

---

## e) WHAT WE SHOULD IMPROVE

1. **ALWAYS run `nix run .#verify` before claiming done** — This session repeated the exact anti-pattern it was supposed to fix. The verify gate was extended (T1) but never run after the actual code changes.
2. **Regenerate api_surface.txt immediately** — Every exported symbol addition requires `cd cmd/api-stability && GOWORK=off go run main.go -update`. The DegradedADTs field is missing from the golden.
3. **Add `nixos.qcow2` to `.gitignore`** — And remove from git tracking (`git rm --cached`).
4. **Tag AFTER verify passes, not before** — The v4.3.0 tag was cut prematurely.
5. **Run `-race` on all new tests** — The plan called for it explicitly.
6. **The auto-commit daemon interleaves unrelated changes** — Multiple dependency bumps (go-output, flake.lock) got mixed into feature commits. This is expected but makes history harder to read.
7. **check-module-layers.sh had 21/64 modules missing** — Three sessions claimed GREEN without catching this. The script needs a meta-test that asserts every go.mod directory is in the LAYER/DEP_BUDGET maps.

---

## f) Up to 50 Things to Get Done Next

### Critical (do first)

1. Run `nix run .#verify` end-to-end after T5-T13 changes
2. Regenerate `docs/api_surface.txt` (`cd cmd/api-stability && GOWORK=off go run main.go -update`)
3. Add `nixos.qcow2` to `.gitignore` and `git rm --cached nixos.qcow2`
4. Run `go test -race ./metaengine/... -count=1` on new tests
5. Tag `metaengine/v4.4.0` after verify passes (includes Universal ADT work)

### T14-T18: Replication Polish

6. Add `WithReplication(r Replication)` planOption to planner.go
7. Add `WithNetworkRTT(d time.Duration)` planOption
8. Wire overrides into EngineProfile during Plan()
9. Add replication fields to SerializablePlan + SerializableQuery
10. Add `ReplicationMode(queryName string) Replication` accessor on Store
11. Implement MapUpdate WARN diagnostic (detect updateFold on replicated engine)
12. Write tests for WithReplication override
13. Write tests for WithNetworkRTT override
14. Write test for SerializablePlan replication inclusion
15. Write test for ReplicationMode accessor
16. Write test for MapUpdate WARN diagnostic

### T19-T24: TODO_LIST Backlog

17. T19: Add 100K-event smoke variant (SOAK_SKIP_10M=1)
18. T19: Add runtime.MemStats.TotalAlloc delta measurement
19. T19: Run soak test 3× with -race, document variance
20. T20: Design typed Watcher[V] interface
21. T20: Implement WatchTyped[V](<>) on Store (new method, keep chan any)
22. T20: Update dx.go reifyWatcherValue for typed path
23. T20: Write TestWatcher_TypedChannel_NoAssertion
24. T21: Write TestSSE_ReconnectWithSQLite end-to-end
25. T21: Verify Last-Event-ID replay with WatchWithSeq
26. T22: Add keyType check at Store.Execute entry
27. T22: Return ErrKeyTypeMismatch on mismatch
28. T22: Write TestStore_BoundaryKeyTypeValidation
29. T23: Add @> operator support to pgengine pushdown
30. T23: Write TestPgEngine_GINContainment
31. T24: Add explainScan for DuckDB planned + standard paths
32. T24: Centralize planned-table helpers
33. T24: Add DuckDB layout benchmark
34. T24: Add adttest matrix coverage for LayoutPlanner

### T25-T27: Other

35. T25: Research Iroh C bindings (iroh-ffi, iroh-go) availability
36. T25: Evaluate CGo FFI vs sidecar tradeoffs
37. T25: Write ADR-0095: Iroh bridge decision
38. T26: Remove 6 unnecessary type arguments in cqrs-lint
39. T26: Fix writestring hint in commands.go:78
40. T27: Run cqrs-lint against example/taskmanager
41. T27: Run cqrs-lint against example/readme-quickstart
42. T27: Document false-positive findings

### Meta/Quality

43. Add meta-test: every go.mod dir must be in check-module-layers.sh LAYER map
44. Add meta-test: every go.mod dir must be in check-module-layers.sh DEP_BUDGET map
45. Update AGENTS.md with DegradedADTs pattern + ADR-0094 reference
46. Update AGENTS.md verify gate description (now includes layers/dup/coverage)
47. Update TODO_LIST.md with completed Phase 3 items
48. Update SKILL.md with DegradedADTs + universal ADT documentation
49. Run `cmd/doc-check` to verify all import paths in new ADR-0094 are valid
50. Full `nix run .#verify` as final gate before session ends

---

## g) Questions I Cannot Answer Myself

1. **Should `nixos.qcow2` be removed from git history (git filter-branch/BFG) or just `git rm --cached` + `.gitignore`?** The 43MB is already in the commit history. A simple `git rm --cached` stops tracking but doesn't shrink the repo. BFG/filter-branch rewrites history but requires force-push. Your call on whether the history rewrite is worth it for 43MB.

2. **Should the `metaengine/v4.4.0` tag wait until T14-T18 (replication polish) is also done, or cut immediately after verify passes?** Cutting now gives consumers DegradedADTs sooner. Waiting gives a more complete replication story. The plan assumed separate tags.

3. **For T20 (Watcher typed-channel), should WatchTyped[V] be a generic method on Store or a separate TypedReader-like type?** The plan says "add as NEW method, keep chan any for backward compat" but doesn't specify the receiver type. Store is not generic, so a generic method on a non-generic type is valid Go but unusual in this codebase's style.

---

## Resolution (2026-08-03)

T1 (extended verify gate: `d4dbebbd`), T5-T13 (Universal ADT: `DegradedADTs`, all 5 engines 10/10, `degradedADTRule`, ADR-0094 `8b41f658`), T3 (push), T4 (`metaengine/v4.3.0` tag) all shipped. `nixos.qcow2` untracked (`502bc338`). API golden regenerated. `metaengine/v4.4.0` force-moved to `c45b39c8` in `09-35`.
