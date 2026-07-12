# Status Report — 2026-07-09 07:41

## Session: Pareto Plan Execution — Full TODO List

---

## a) FULLY DONE ✅

### T01-T03: Build Unblocked (P0)

| Task | What                                                                                                        | Result                                                                  |
| ---- | ----------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| T01  | Fixed `schema/validator.go` json/v2 `Unmarshal` signature mismatch                                          | `decodeJSON` wrapper added — schema/catalog/integration build unblocked |
| T01b | Fixed `catalog/docserver/docserver.go` — `SetIndent` → `WithIndent` (json/v2 API change)                    | Build unblocked                                                         |
| T01c | Fixed `example/taskmanager/http.go` — `DisallowUnknownFields` → `RejectUnknownMembers` (json/v2 API change) | Build unblocked                                                         |
| T01d | Fixed `graph/schema.go` — `slices.Contains` → `slices.ContainsFunc` (Go 1.26 type constraint)               | Build unblocked                                                         |
| T02  | Fixed `flake.nix` build app — added `${tagFlags}` to `go build` command (line 191)                          | `nix run .#build` now applies goexperiment tags                         |
| T03  | Verified full workspace build green                                                                         | 52 modules build clean                                                  |

**Impact:** The entire workspace was broken from a prior session's json/v2 migration. 4 separate build blockers across 4 modules fixed. Build is now 100% green.

### T04-T06: Idempotency Tests (P3)

| Task | What                                                                                                                                        | Result                  |
| ---- | ------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------- |
| T04  | BDD tests for command idempotency (Ginkgo) — 3 Describe blocks: duplicate rejection, different commands, empty key passthrough              | 15/15 Ginkgo specs pass |
| T05  | BDD tests for query idempotency (Ginkgo) — 2 Describe blocks: dedup by key, nil keyExtractor panic                                          | Pass                    |
| T06  | Integration test in `integration/` — full pipeline: dispatcher → middleware → decider → store → duplicate rejection → event count assertion | Pass                    |

### T07-T08: Taskmanager Idempotency Demo (P4)

| Task | What                                                                                                                                                                                                                                     | Result                                     |
| ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------ |
| T07  | Wired `CommandIdempotency` middleware into `example/taskmanager` `setupFeatures()` with 10min TTL                                                                                                                                        | Idempotency active in the flagship example |
| T07b | **Fixed pre-existing bug:** handlers were registered BEFORE middleware was configured. The dispatcher applies middleware at Register time, so all middleware was silently bypassed. Reordered `setupFeatures` before `registerHandlers`. | All middleware now actually runs           |
| T08  | Added `TestIdempotencyDemo` test — dispatches same command twice, asserts ErrDuplicate + only 1 event persisted                                                                                                                          | Pass                                       |

### T09-T11: Four-Tier Documentation (P5)

| Task | What                                                                                                                      | Result   |
| ---- | ------------------------------------------------------------------------------------------------------------------------- | -------- |
| T09  | Wrote `docs/architecture-understanding/FOUR-TIER-MODEL.md` — 7-tier table (0-6), explains why old 7-layer was fake        | Complete |
| T10  | Rendered `FOUR-TIER-MODEL.d2` → SVG with D2 ELK layout (33KB)                                                             | Complete |
| T11  | Added ADR-0046 (`docs/adr/0046-four-tier-model.md`), updated AGENTS.md module graph, added to ADR index in docs/README.md | Complete |

### T12-T17: id/ + metadata/ Extraction (P6 v4)

| Task | What                                                                                                                                                                                              | Result                                            |
| ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------- |
| T12  | Created `id/aggregate_type.go` — `AggregateType`, `AggregateRef`, `ParseAggregateType`, `NewAggregateRef` now live in `id/` (their natural home)                                                  | Builds clean                                      |
| T13  | Created `metadata/` module — new `go.mod`, `metadata.go` with `Tracing`, `CustomData[K]`, `MergeCustomMaps` moved from event/                                                                     | Builds clean                                      |
| T14  | (Combined with T13 — types moved in same step)                                                                                                                                                    | Done                                              |
| T15  | event/ now has deprecated aliases: `Tracing = metadata.Tracing`, `CustomData = metadata.CustomData`, `AggregateRef = id.AggregateRef`, etc. — all with `// Deprecated:` comments (SA1019-visible) | No silent re-exports                              |
| T16  | command/ now imports `id/` and `metadata/` directly — **event/ is no longer a direct compile dependency** of command/                                                                             | `command/` builds without event/ in require block |
| T17  | Verified decoupled command/ build + test                                                                                                                                                          | All command tests pass                            |

**Key architectural win:** `command/` → `event/` hard compile dependency is BROKEN. command/ now depends only on `id/`, `metadata/`, `dispatcher/`.

### T18-T20: kv/ context.Context (P7 v4)

| Task | What                                                                                                                                                                                                                                                   | Result            |
| ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------- |
| T18  | Added `context.Context` as first param to ALL 11 kv I/O methods across Reader/Writer/Batch/ConditionalWriter interfaces                                                                                                                                | Interface updated |
| T19  | Updated 3 implementations: `kv/mem.go` (MemStore + memBatch), `storage/pebble/adapter.go` (KVAdapter + pebbleBatch), `storage/kv_sql.go` (SQLKVStore + sqlKVBatch — now threads ctx to actual SQL queries instead of hardcoded `context.Background()`) | All impls updated |
| T20  | Updated `kv/typed_store.go` (TypedStore wrappers), `idempotency/kv_store.go` (7 call sites), `stack/materialize_test.go` (mock), all 7 test files across kv/ and storage/pebble/ — ~60 call sites total                                                | All tests pass    |

### T21: Storage Split Proposal (P8)

| Task | What                                                                                                                                 | Result                      |
| ---- | ------------------------------------------------------------------------------------------------------------------------------------ | --------------------------- |
| T21  | Wrote `docs/planning/2026-07-09_STORAGE-SPLIT-PROPOSAL.md` — 4-package proposal with file inventory, migration path, risk assessment | Proposal ready for approval |

### Additional Fixes (On-Sight)

- Added `metadata/` to `go.work` workspace
- Added `metadata/` dependency + replace directives to **27 go.mod files** across the entire workspace
- Fixed `kv_sql_test.go` missing `context` import
- Fixed `pebble/adapter_test.go` — raw DB Get call (not kv interface) and store.Set with fmt.Appendf

---

## b) PARTIALLY DONE 🟡

### Event/Catalog Golden Test Failures

**4 test failures remain — ALL pre-existing from json/v2 migration (NOT caused by this session):**

| Test                                  | Module                | Root Cause                                                         |
| ------------------------------------- | --------------------- | ------------------------------------------------------------------ |
| `TestDecodePayload_EncodingMatch`     | event/                | json/v2 encoding format change — golden file mismatch              |
| `TestGolden_AsyncAPIJSON`             | catalog/asyncapi/     | json/v2 output format differs from v1 (field ordering, whitespace) |
| `TestGolden_AsyncAPIYAML`             | catalog/asyncapi/     | Same — YAML derived from JSON output                               |
| `TestGolden_EventCatalog_PackageJSON` | catalog/eventcatalog/ | Same                                                               |
| `TestGolden_OpenAPIJSON`              | catalog/openapi/      | Same                                                               |

**What's needed:** Regenerate golden files with `-update` flag OR adjust tests to normalize json/v2 output differences. This is ~15 min of work but was explicitly out of scope ("pre-existing failures not from this session").

### metadata/ Module

The module is created and types are moved, but:

- No dedicated tests for `metadata/` package itself (the types are tested transitively through event/command/query tests)
- No `doc.go` with package-level documentation
- Module is at v0 — needs versioning strategy decision

### Deprecated Aliases

The `// Deprecated:` comments are in place but:

- No test verifying the deprecation warnings are correct
- No migration guide written for consumers

---

## c) NOT STARTED ⏸

| Item                            | Why                                                                                                                      | Priority      |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------ | ------------- |
| Storage/ split execution        | Proposal written (T21), execution deferred per user instruction ("tell me when you want to start")                       | When approved |
| Event/ god module decomposition | Identified in architecture review (9 concerns, 130+ exports) but no work started — the deprecated aliases lay groundwork | v4            |
| Pebble store multi-process      | Single-process 256-shard mutex identified as scalability gap                                                             | v4+           |
| Distributed event bus           | `MemoryBus` is synchronous; Watermill GoChannel is single-process. No Redis/NATS backend.                                | Future        |

---

## d) TOTALLY FUCKED UP 💥

### metadata/ go.mod Contamination

When adding `metadata/v4` as indirect dependency to 27 go.mod files, the initial sed script produced **malformed go.mod files** in 12 modules — it inserted the require line outside a `require ( )` block, creating `unknown directive` errors. Required a second pass to fix all 12. This was a mechanical automation failure that should have been caught by a test build immediately after the batch edit.

**Lesson:** Never batch-edit 27 go.mod files with sed without running `go build` after every 2-3 files. The sed approach was fragile — should have used `go mod edit` instead.

### Near-Miss: TypedStore `_ context.Context` → `ctx context.Context`

The TypedStore wrapper already had `_ context.Context` in its method signatures (discarding ctx). When updating backend calls to pass ctx, I initially used sed to add `ctx` to the calls but forgot that the parameter was named `_` — the calls referenced an undefined `ctx`. Required reading the actual file and renaming `_` → `ctx` before the backend calls could work. This should have been caught by the first compile, not discovered after.

### Pre-Existing Bug Found & Fixed: Middleware Ordering

The taskmanager registered handlers BEFORE configuring middleware. Since the dispatcher applies middleware at registration time (not dispatch time), **NO middleware was ever running** in the taskmanager example — not just idempotency, but recovery, logging, retry, and OTel tracing were ALL silently bypassed. Fixed by reordering `setupFeatures()` before `registerHandlers()`. This was a significant correctness bug that nobody noticed because the tests didn't verify middleware execution.

---

## e) WHAT WE SHOULD IMPROVE 🔧

### Process Improvements

1. **Stop trusting session summaries blindly** — The prior session's summary listed "error taxonomy extraction" as the #1 TODO, but it was ALREADY DONE. I verified this before executing, saving ~50 files of wasted work. Always audit claims against actual code state.

2. **Check ADRs before planning** — The Outbox pattern (4 tasks, 48 min) was already declined (ADR-0016). I didn't catch this until the user pointed it out. ADR status should be the first check for any architecture TODO.

3. **Batch go.mod edits need validation gates** — The 27-file sed edit was reckless. Should use `go mod edit -require` and `go mod edit -replace` which parse correctly.

4. **Test middleware execution, not just handler results** — The taskmanager middleware ordering bug was invisible because tests checked "did the command succeed" not "did middleware run". Middleware tests should assert side effects.

### Code Improvements

5. **event/ still has 9 concerns** — Even with AggregateRef → id/ and CustomData → metadata/, event/ still owns: tombstone detection, command causality, replay mode, codec utilities, event construction/validation, store interfaces, bus interfaces, checkpoint tracking, and metadata (the rich event-specific one with Tombstone/Causation/Source fields). Further decomposition needed.

6. **metadata/ has no tests** — The extracted types are tested only transitively. Should have dedicated unit tests for Tracing.Merge, CustomData.Clone, MergeCustomMaps edge cases.

7. **kv/ context is propagated but not respected** — MemStore ignores ctx (fine, it's in-memory). SQLKVStore now passes ctx to SQL queries (good). But Pebble KVAdapter ignores ctx (Pebble's API doesn't take context). Should document this limitation.

8. **No consumer migration guide** — The deprecated aliases in event/ work but consumers need a written guide: "import id/ for AggregateRef, import metadata/ for Tracing/CustomData, stop importing event/ for these types."

### Architectural Improvements

9. **The dispatcher middleware-at-registration-time design is surprising** — Most middleware frameworks apply at dispatch time. This should either be documented loudly or changed. The taskmanager bug proves it's a footgun.

10. **27 go.mod files for 48 modules is a maintenance nightmare** — Every new module requires editing ~27 files. Consider whether go.work can reduce this, or whether a central deps management script would help.

---

## f) Up to 50 Things We Should Get Done Next

### Critical (blocks correctness/trust)

| #   | Task                                                                               | Impact                               | Effort |
| --- | ---------------------------------------------------------------------------------- | ------------------------------------ | ------ |
| 1   | Fix 5 golden test failures (regenerate with `-update` or normalize json/v2 output) | Unblocks CI green                    | 30m    |
| 2   | Add tests verifying dispatcher middleware actually runs after registration         | Prevents silent bypass bugs          | 20m    |
| 3   | Write consumer migration guide for id/ + metadata/ extraction                      | Prevents consumer breakage confusion | 30m    |
| 4   | Add unit tests for metadata/ package (Tracing.Merge, CustomData.Clone, edge cases) | Coverage for new module              | 20m    |

### High Value (architectural health)

| #   | Task                                                                                 | Impact                                | Effort |
| --- | ------------------------------------------------------------------------------------ | ------------------------------------- | ------ |
| 5   | Document dispatcher "middleware at registration time" behavior prominently           | Prevents reuse of the taskmanager bug | 15m    |
| 6   | Decompose event/ further — extract tombstone detection to `tombstone/` or `listing/` | Reduces god module                    | 2h     |
| 7   | Decompose event/ further — extract command causality to its own concern              | Reduces god module                    | 1h     |
| 8   | Execute storage/ split (proposal ready, awaiting approval)                           | 109 files → 4 focused packages        | 4h     |
| 9   | Add `context.Context` support to Pebble KVAdapter (or document the limitation)       | Honesty about ctx propagation         | 30m    |
| 10  | Add `doc.go` to metadata/ package                                                    | Package documentation                 | 10m    |

### Medium Value (quality/observability)

| #   | Task                                                                                                | Impact                    | Effort |
| --- | --------------------------------------------------------------------------------------------------- | ------------------------- | ------ |
| 11  | Add BDD tests for EventIdempotency middleware (only command+query have BDD)                         | Test parity               | 20m    |
| 12  | Add idempotency demo endpoint to taskmanager HTTP API (POST /api/tasks with Idempotency-Key header) | Showcase feature          | 30m    |
| 13  | Wire query idempotency into taskmanager (currently only command)                                    | Full demo coverage        | 15m    |
| 14  | Add `.gitignore` for `coverage.out` and other generated files                                       | Clean repo                | 5m     |
| 15  | Run `go mod tidy` across all 48 modules (many have stale indirect deps from this session's edits)   | Dependency hygiene        | 30m    |
| 16  | Add CI check that all modules have matching require + replace for metadata/                         | Prevents go.sum drift     | 20m    |
| 17  | Consider central go.mod dependency management script                                                | Reduces 27-file edit pain | 1h     |
| 18  | Add integration test for the full idempotency + retry + recovery middleware stack                   | Defense in depth          | 30m    |
| 19  | Benchmark kv/ with context.Context overhead vs without                                              | Performance verification  | 20m    |
| 20  | Add lint rule: no `context.Background()` in production code (only tests)                            | Code quality              | 15m    |

### Lower Value (polish/nice-to-have)

| #   | Task                                                                                     | Impact                    | Effort     |
| --- | ---------------------------------------------------------------------------------------- | ------------------------- | ---------- |
| 21  | Update SKILL.md with id/ + metadata/ module references                                   | AI consumer accuracy      | 20m        |
| 22  | Update all module READMEs to reference four-tier model                                   | Documentation consistency | 30m        |
| 23  | Add architecture decision record for command→event decoupling                            | Decision archaeology      | 20m        |
| 24  | Document the deprecated alias removal timeline (v3.x → v4)                               | Consumer planning         | 15m        |
| 25  | Add `nolint` comments where needed for unused ctx params (MemStore)                      | Lint clean                | 10m        |
| 26  | Consider versioning metadata/ at v1 instead of v3 (it's a new module)                    | Versioning honesty        | 5m         |
| 27  | Add example of custom idempotency key extractor to getting-started example               | Education                 | 15m        |
| 28  | Add chaos test: what happens when idempotency store is unavailable mid-dispatch?         | Resilience                | 30m        |
| 29  | Add projectionhost idempotency test (replay scenario)                                    | Replay safety             | 30m        |
| 30  | Consider whether `context.Context` should be on kv.Iterator methods (Next/Key/Value)     | Interface completeness    | 20m        |
| 31  | Add test that deprecated event/ aliases actually compile and delegate correctly          | Backward compat safety    | 15m        |
| 32  | Add `go vet` check across workspace for common issues                                    | Code quality              | 10m        |
| 33  | Update FEATURES.md with idempotency status (DONE) and metadata/ module                   | Feature inventory         | 15m        |
| 34  | Update TODO_LIST.md with completed items from this session                               | Task tracking             | 10m        |
| 35  | Consider adding `errgroup` support to projectionhost for parallel projection processing  | Performance               | 1h         |
| 36  | Add distributed tracing test for idempotency middleware (span propagation through dedup) | Observability             | 30m        |
| 37  | Document the kv/ context.Context migration path for third-party KV implementations       | Consumer guidance         | 15m        |
| 38  | Add test for TTL expiry in idempotency store (MemoryStore + KVStore)                     | Correctness               | 20m        |
| 39  | Consider whether `command.Bus` should support idempotency natively                       | Architecture              | Discussion |
| 40  | Add SSE broker idempotency test (Last-Event-ID replay dedup)                             | Transport reliability     | 30m        |

### Speculative (future/v4+)

| #   | Task                                                                                                  | Impact                | Effort     |
| --- | ----------------------------------------------------------------------------------------------------- | --------------------- | ---------- |
| 41  | Explore Redis-backed idempotency store                                                                | Multi-process support | 2h         |
| 42  | Explore distributed event bus (NATS/Redis pub-sub via Watermill backend)                              | Multi-process         | 4h         |
| 43  | Consider CQRS code generator (cqrs-gen) for idempotency middleware boilerplate                        | Developer experience  | 2h         |
| 44  | Explore event/ decomposition: extract replay mode to `replay/` package                                | God module reduction  | 2h         |
| 45  | Explore event/ decomposition: extract checkpoint to `checkpoint/` package                             | God module reduction  | 1h         |
| 46  | Consider GraphQL transport module (was previously a HARD NO, revisit?)                                | API surface           | Discussion |
| 47  | Add OpenTelemetry metrics for idempotency hit rate                                                    | Observability         | 30m        |
| 48  | Consider whether the dispatcher should apply middleware at dispatch time instead of registration time | Architecture          | Discussion |
| 49  | Explore Pebble multi-process via shared filesystem + flock                                            | Multi-process storage | 4h         |
| 50  | Write "go-cqrs-lite v4 migration guide" covering all breaking changes                                 | Consumer planning     | 2h         |

---

## g) Top 2 Questions

### Q1: Should the dispatcher apply middleware at dispatch time instead of registration time?

The current `dispatcher.Dispatcher[H, M]` applies middleware in `Register()` via `d.middleware.Apply(handler, wrap)`, baking the middleware chain into the handler closure at registration time. This means:

- Middleware added AFTER registration is silently ignored
- This caused a real bug in the taskmanager (all middleware was bypassed)

**Two options:**

1. **Change to dispatch-time application** — Store raw handler + middleware list, apply on each Dispatch call. Slightly more allocation per dispatch, but eliminates the ordering footgun. Breaking change to the generic dispatcher.
2. **Keep registration-time + document loudly** — Add a panic or error if `Use()` is called after any `Register()`. Less invasive, preserves performance.

I cannot decide this myself because it's a fundamental API contract change that affects all consumers.

### Q2: Should we regenerate golden files for json/v2 or normalize the output?

The 5 remaining test failures are all golden file mismatches caused by json/v2 producing slightly different output (field ordering, whitespace, number formatting). Two paths:

1. **Regenerate goldens** — Run tests with `-update` flag, accept json/v2 output as the new baseline. Quick but means golden files are now json/v2-specific.
2. **Normalize output** — Add canonical JSON marshaling in catalog/docserver tests so the output is stable regardless of json v1/v2. More work but more robust.

I cannot decide this because I don't know if the golden files are consumed by external tools or are purely test fixtures.

---

## Session Metrics

| Metric                  | Value                                                                               |
| ----------------------- | ----------------------------------------------------------------------------------- |
| Tasks planned           | 27 (after removing outbox + error taxonomy)                                         |
| Tasks completed         | 20 (T01-T21, T27 parked per user)                                                   |
| Build status            | ✅ Green (52 modules)                                                               |
| Test status             | 52 ok, 5 FAIL (all pre-existing golden file mismatches)                             |
| New files created       | 8 (metadata module, 4 docs, 3 tests)                                                |
| Files modified          | ~55 (kv interface + impls + consumers + go.mod files + taskmanager + event aliases) |
| New Go module           | `metadata/v4`                                                                       |
| Build blockers fixed    | 5 (schema, docserver, taskmanager, graph, flake.nix)                                |
| Architectural wins      | command/ → event/ compile dependency BROKEN; kv/ now context-aware                  |
| Pre-existing bugs fixed | 1 (taskmanager middleware ordering — ALL middleware was silently bypassed)          |
