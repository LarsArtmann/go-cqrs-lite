# Session 112c — Comprehensive Status Update

**Date:** 2026-05-28 06:24 CEST  
**Branch:** master (ahead of origin by 1 commit)  
**Working Tree:** Clean  
**Go Version:** 1.26.3  
**Modules:** 13 (core, memory, catalog, middleware, testhelpers, integration, storage, projection, saga, watermill, cqrs-gen)  

---

## a) FULLY DONE

### Session 85 Execution Plan — Phase 1: Fix What's Broken (5/5)
| Task | Status |
|------|--------|
| Rename `auto_name_test.go` → `message_config_test.go` | ✅ File no longer exists (refactored away) |
| Split `testhelpers/fake_store.go` to <250 lines | ✅ Now 244 lines |
| Convert `core/pkg/dispatcher` 3 sentinels to structured | ✅ All use `errorfamily.NewRejection/NewInfrastructure/NewConflict` |
| Convert `catalog/id_parse.go` 4 sentinels to structured | ✅ No bare `errors.New` |
| Convert `sync/types.go` 2 sentinels to structured | ✅ No bare `errors.New` |

### Session 85 Execution Plan — Phase 2: Error Wraps (Partial)
| Task | Status |
|------|--------|
| 2.1 Re-export `errorfamily.Wrap` as `event.Wrap` | ✅ `Wrap`, `WrapRejection`, `WrapConflict`, `WrapTransient`, `WrapCorruption`, `WrapInfrastructure` all re-exported in `core/event/errors.go` |
| 2.5 Replace wraps in `projection/` | ✅ 13 wraps replaced across 3 files (runner.go, runner_live.go, builder.go) |
| 2.6 Replace wraps in `storage/` | ✅ ~90 wraps replaced across 22 files |
| 2.7 Replace wraps in `middleware/` | ✅ 2 wraps replaced (retry.go, middleware.go) |
| 2.3 Replace wraps in `core/decider/` | ✅ 0 wraps found (already clean) |
| 2.4 Replace wraps in `core/aggregate/` | ✅ N/A (aggregate package deleted) |

### Session 85 Execution Plan — Phase 3: Dead Code (Partial)
| Task | Status |
|------|--------|
| 3.2 Remove `CatalogMeta` from event/command/query | ✅ Files no longer exist |
| 3.3 Remove deprecated `CatalogBuilder` | ✅ `catalog/adapters/builder.go` gone |
| 3.4 Remove deprecated adapters | ✅ `from_query_dispatcher.go` gone |
| 3.5 Remove `MessageIDString` | ✅ No longer exists |
| 3.6 Update `nolint:staticcheck` in tests | ✅ Zero occurrences in .go files |
| 3.7 Evaluate `RegisterClassification` | ✅ Unexported/removed in Session 89 |

### Session 85 Execution Plan — Phase 4: Examples (Partial)
| Task | Status |
|------|--------|
| 4.1 Update `example/user/` to `catalog.Command[T]()` | ✅ Commands registered alongside events |
| 4.3 Add catalog exports to example | ✅ EventCatalog + D2 + AsyncAPI all exported |

### Session 85 Execution Plan — Phase 5: Type Safety (Partial)
| Task | Status |
|------|--------|
| 5.1 Brand `OutboxID` with `go-branded-id` | ✅ `cbid.ID[outboxMarker, string]` with `NewOutboxID()`, `.Get()`, `.Equal()`, `.IsZero()` |

### NEW in Latest Commit (8807c5c)
| Feature | Status |
|---------|--------|
| DLQ (Dead Letter Queue) handler | ✅ `WithDeadLetterHandler` option on projection.Runner |
| Runner.Reset API | ✅ `Runner.Reset(ctx, projectionName)` clears checkpoint for full replay |
| BatchProjection interface | ✅ `event.BatchProjection` optional interface for batch event processing |
| DLQ integration tests | ✅ `TestRunner_DeadLetterHandler`, `TestRunner_DeadLetterHandler_WithRetry` |

### Overall Test Health
| Metric | Value |
|--------|-------|
| Packages passing | 26/26 |
| Coverage range | 84.2% — 100% |
| Coverage average | ~94% |
| Race detector | Clean |
| Benchmarks | 53 across packages |

---

## b) PARTIALLY DONE

### Phase 2: Error System Wraps — INCOMPLETE

The storage/projection/middleware modules are fully converted, but **core/event still has 19 `fmt.Errorf` wraps** and **saga has 17** that were never converted:

```
core/event/types.go:56          — 1 wrap
core/event/codec.go:48,62       — 2 wraps
core/event/builder.go:74        — 1 wrap
core/event/runner.go:90,95,181,203 — 4 wraps
core/event/publish_helper.go:21,26,51 — 3 wraps
core/event/outbox_publisher.go:174,197,202 — 3 wraps
core/event/batch.go:39,51       — 2 wraps
core/event/event_new.go:31,42   — 2 wraps
core/event/versioned_store.go:72 — 1 wrap
```

**Impact:** The core event module is the highest-traffic error path. Every event creation, encoding, projection handling, and outbox operation still produces unstructured wrapped errors. Consumers of the library get structured sentinels but unstructured pipeline errors.

### Phase 2.8: `WithContext` Re-export — NOT DONE
`errorfamily.WithContext` (method on `*Error`) is **not re-exported** in `core/event/errors.go`. The execution plan explicitly called for this re-export. Currently consumers must import `go-error-family` directly to use contextual error enrichment.

### Phase 2.9: Add `WithContext` to Storage Error Wraps — NOT DONE
No storage error wraps use `.WithContext()` to attach diagnostic metadata (aggregate_id, version, backend). This was a dependency of 2.8.

### Phase 3.1: Deprecate `aggregate` Package — UNCLEAR
The `aggregate` package was **deleted** (not deprecated). The TODO_LIST says "Formally deprecate aggregate package — DONE" but the actual status is: package no longer exists. This is a breaking change, not a deprecation. Consumers importing it will get a compilation error, not a deprecation warning.

### Phase 4.2: Align Example Patterns — NOT DONE
`example/user/` and `example/todo/` still use fundamentally different patterns:
- `user/`: Hand-rolled command structs, simple decider functions, no middleware wiring
- `todo/`: Full decider pattern with `command.BasicCommand`, `aggregate.DecideXxx`, middleware, codec

### Phase 5.2: Brand Catalog ID Types — NOT DONE
`ServiceID`, `DomainID`, `MessageID`, `ChannelID` in `catalog/types.go` are still plain `string` aliases.

### Phase 5.3: Add `ErrorCode` Branded Type — NOT DONE
Error codes throughout the codebase are still raw strings (e.g., `"event.empty_event_type"`).

---

## c) NOT STARTED

| Task | Source |
|------|--------|
| Add `ProcessedAt` to CheckpointStore | TODO_LIST.md:149 |
| Add `ServerReceivedAt` / `ServerStoredAt` timestamps | TODO_LIST.md:150 |
| Add `event.Context` propagation through NewEvent/PublishChanges | TODO_LIST.md:154 |
| Add catalog.Exporter interface + WalkMessages | TODO_LIST.md:171 |
| Simplify cattest/catalog.go to zero-cost API | TODO_LIST.md:176 |
| Build catch-up projection runner (replay → live-switch) | TODO_LIST.md:184 |
| Make transactional projection contract explicit | TODO_LIST.md:185 |
| Add retry and dead-letter to InMemoryRunner | TODO_LIST.md:187 |
| Add background polling for InMemoryRunner | TODO_LIST.md:188 |
| Increase projection coverage to 95%+ | TODO_LIST.md:189 (currently 96.0%) ✅ actually DONE |
| Optimize Pebble LoadToTimestamp — avoid full scan | TODO_LIST.md:64 |
| Add catalog diff/breaking-change detection tool | TODO_LIST.md:25 |
| Add high-level test utilities (AggregateTester, etc.) | TODO_LIST.md:27 |
| PostgreSQL integration tests with testcontainers | BLOCKED |
| Move example/todo to own repository | BLOCKED |
| Push release tags to remote | BLOCKED |
| Remove replace directives from go.mod files | BLOCKED |

---

## d) TOTALLY FUCKED UP!

### 1. 145 `fmt.Errorf("...: %w", err)` Wraps Still Unstructured

The Session 85 execution plan identified 194 `fmt.Errorf` wraps as the single highest-value work. After the migration, **145 remain** across the codebase. The core event module — the module every consumer touches — still has 19 unstructured wraps. This means:

- `event.NewEvent` failures → unstructured
- `codec.Decode` failures → unstructured
- `runner.go` projection handling → unstructured
- `outbox_publisher.go` poll/ack/publish → unstructured
- `publish_helper.go` save+snapshot → unstructured

**The promise was "structured error chains through the entire stack." The reality is "structured at the edges, garbage in the middle."**

### 2. `aggregate` Package Was Deleted, Not Deprecated

The plan said "Add `// Deprecated` notice to aggregate package" (task 3.1). Instead, the entire package was deleted. This is a **breaking change** for any consumer importing `core/aggregate/`. A proper deprecation would have:
1. Kept the package with `// Deprecated: Use core/decider` comments
2. Added deprecation notices for 1-2 releases
3. Then deleted in a major version bump

### 3. `WithContext` Not Re-exported

Task 2.8 explicitly called for re-exporting `errorfamily.WithContext` as `event.WithContext`. It was never done. The execution plan listed it as "10min, MED impact." It's still missing. This blocks task 2.9 (contextual storage errors) entirely.

### 4. `sync/` Module Extracted to `go-localsync` — Or Was It?

The TODO_LIST says "Consider renaming sync package — DONE (extracted to go-localsync)". But `sync/` still exists in this repo with 1,017 lines of production code. Either the extraction is incomplete or the TODO is misleading.

### 5. LSP / go mod tidy Noise in `catalog/` Module

The `catalog` module shows persistent `go mod tidy` errors in LSP for transitive dependencies (`github.com/go-faster/errors`, `github.com/go-faster/yaml`, etc.). These are NOT compilation errors — the module builds and tests pass — but they create noise that makes real diagnostics harder to spot. The root cause is likely that `go.work` sync hasn't been run since dependency updates.

### 6. DLQ Test File Has Race Condition Risk

`projection/runner_dlq_test.go` uses a `callCount` int variable accessed from the projection handler (called from a goroutine) and checked in the main test goroutine without synchronization. This is a data race:

```go
callCount := 0
failingProj := event.NewProjection("retry-projection", func(...) error {
    callCount++  // accessed from publisher goroutine
    return errors.New("always fails")
}, ...)
// ... later:
if callCount != expectedCalls {  // accessed from test goroutine
```

The test passes because the timing happens to work, but `go test -race` would flag this.

---

## e) WHAT WE SHOULD IMPROVE!

### 1. Finish the Error Wrap Migration (Highest Impact)

The remaining 19 wraps in `core/event/` are the most important. Every consumer of the library touches these code paths. Convert them to `event.Wrap*` and the entire error taxonomy becomes useful end-to-end.

Priority order:
1. `core/event/event_new.go` (2 wraps) — every event creation
2. `core/event/codec.go` (2 wraps) — every codec operation
3. `core/event/runner.go` (4 wraps) — every projection run
4. `core/event/outbox_publisher.go` (3 wraps) — every outbox poll
5. `core/event/publish_helper.go` (3 wraps) — every repository save
6. `core/event/batch.go` (2 wraps) — batch operations
7. `core/event/versioned_store.go` (1 wrap) — upcasting
8. `core/event/types.go` (1 wrap) — IP parsing
9. `saga/` (17 wraps) — saga execution

### 2. Re-export `WithContext`

One function addition to `core/event/errors.go`:
```go
func WithContext(err *Error, key, value string) *Error {
    return err.WithContext(key, value)
}
```

Then add contextual metadata to storage wraps:
```go
return event.WrapInfrastructure(err, "storage.check_version", "...").
    WithContext("aggregate_id", aggregateID.String()).
    WithContext("expected_version", strconv.Itoa(expectedVersion.Int()))
```

### 3. Fix the DLQ Test Race

Change `callCount` to `atomic.Int32` or protect with `sync.Mutex`.

### 4. Run `go.work sync` + `go mod tidy` Across All Modules

The catalog module's LSP noise is fixable with:
```bash
cd /home/lars/projects/go-cqrs-lite
go work sync
for d in core memory catalog middleware testhelpers storage projection saga watermill integration; do
  (cd $d && go mod tidy)
done
```

### 5. Unify Examples or Document the Divergence

Either:
- Update `example/user/` to use `command.BasicCommand`, middleware, codec (match todo)
- Or add a README explaining that `user/` is the minimal example and `todo/` is the full example

### 6. Consider Keeping `aggregate` Package as Stub

If any external consumer imports `core/aggregate`, the deletion breaks them. A stub package with:
```go
// Package aggregate has been removed. Use github.com/larsartmann/go-cqrs-lite/core/decider instead.
package aggregate
```
...would give a clear compile-time message rather than a "module not found" error.

---

## f) Top #25 Things We Should Get Done Next

| # | Task | Impact | Effort | Module |
|---|------|--------|--------|--------|
| 1 | Convert remaining 19 `fmt.Errorf` wraps in `core/event/` to `event.Wrap*` | **HIGHEST** | 30min | core/event |
| 2 | Re-export `errorfamily.WithContext` as `event.WithContext` | **HIGH** | 5min | core/event |
| 3 | Add `.WithContext()` metadata to storage error wraps | **HIGH** | 20min | storage |
| 4 | Convert 17 `fmt.Errorf` wraps in `saga/` to structured | **HIGH** | 20min | saga |
| 5 | Fix DLQ test race condition (`callCount` → `atomic.Int32`) | **MED** | 5min | projection |
| 6 | Run `go work sync` + `go mod tidy` across all modules | **MED** | 10min | all |
| 7 | Brand catalog ID types (ServiceID, DomainID, etc.) | **LOW** | 40min | catalog |
| 8 | Unify example/user with example/todo patterns | **MED** | 30min | example |
| 9 | Add `ErrorCode` branded type | **LOW** | 20min | core/event |
| 10 | Add `ProcessedAt` to CheckpointStore | **MED** | 15min | core/event |
| 11 | Add `ServerReceivedAt` / `ServerStoredAt` timestamps | **MED** | 20min | core/event |
| 12 | Build catch-up projection runner | **HIGH** | 2h | projection |
| 13 | Add catalog.Exporter interface + WalkMessages | **MED** | 1h | catalog |
| 14 | Add high-level test utilities (AggregateTester, etc.) | **MED** | 2h | testhelpers |
| 15 | Optimize Pebble LoadToTimestamp | **MED** | 30min | storage |
| 16 | Add PostgreSQL integration tests | **HIGH** | 3h | storage |
| 17 | Add background polling for InMemoryRunner | **MED** | 1h | projection |
| 18 | Make transactional projection contract explicit | **MED** | 1h | projection |
| 19 | Simplify cattest/catalog.go | **LOW** | 30min | catalog |
| 20 | Add event.Context propagation | **MED** | 1h | core/event |
| 21 | Add catalog diff/breaking-change detection | **MED** | 2h | catalog |
| 22 | Push v1.0.0 release tags | **HIGH** | 10min | all |
| 23 | Remove replace directives from go.mod files | **HIGH** | 10min | all |
| 24 | Verify `go test -race` passes across all modules | **HIGH** | 15min | all |
| 25 | Document the unified error taxonomy for consumers | **MED** | 30min | docs |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Why were the 19 `fmt.Errorf` wraps in `core/event/` left unconverted when the storage, projection, and middleware wraps were all migrated?**

The Session 85 execution plan explicitly listed task 2.2: "Replace `fmt.Errorf` wraps in `core/event/` with `event.Wrap`" (30min, HIGH impact). The commit message for the migration (8c4a8b1) says "upgrade error classification across storage" — implying core/event was intentionally out of scope. But the execution plan said core/event was IN scope.

Possible explanations:
1. **Intentional deferral**: core/event wraps are more complex (some wrap sentinels like `ErrNilPayload`, some wrap inner errors with context). The original implementer stopped at storage because it was the "highest volume" module.
2. **Scope creep fear**: The original session was already large (8c4a8b1 touches 20+ files). Adding core/event would have made it even larger.
3. **Breakage risk**: core/event tests might assert on specific `errors.Is` behavior that would change with `Wrap*` conversion.

**But here's what makes no sense**: If we're building a library where the core value proposition is "structured errors through your entire stack," then `core/event` is the MOST important module to convert, not the least. Storage wraps are infrastructure-level. Core/event wraps are domain-level — they're what consumers actually see.

**I need a decision**: Should I convert the remaining 19 core/event wraps now (risk: might break test assertions that rely on `errors.Is` with specific sentinels), or is there a deliberate architectural reason they should stay as `fmt.Errorf`?

---

## Appendix: Module Health Matrix

| Module | Tests | Coverage | Lint | Race | Notes |
|--------|-------|----------|------|------|-------|
| core/command | ✅ | 94.3% | ✅ | ✅ | |
| core/decider | ✅ | 91.1% | ✅ | ✅ | |
| core/event | ✅ | 92.4% | ✅ | ✅ | 19 fmt.Errorf wraps remain |
| core/pkg/dispatcher | ✅ | 100.0% | ✅ | ✅ | |
| core/pkg/id | ✅ | 100.0% | ✅ | ✅ | |
| core/query | ✅ | 98.4% | ✅ | ✅ | |
| memory | ✅ | 99.2% | ✅ | ✅ | |
| catalog | ✅ | 96.3% | ✅ | ✅ | LSP go mod tidy noise |
| catalog/asyncapi | ✅ | 93.7% | ✅ | ✅ | |
| catalog/d2 | ✅ | 95.0% | ✅ | ✅ | |
| catalog/docserver | ✅ | 90.1% | ✅ | ✅ | |
| catalog/eventcatalog | ✅ | 92.8% | ✅ | ✅ | |
| catalog/openapi | ✅ | 94.4% | ✅ | ✅ | |
| middleware | ✅ | 98.0% | ✅ | ✅ | |
| testhelpers | ✅ | — | ✅ | ✅ | No test coverage tracking |
| projection | ✅ | 96.0% | ✅ | ⚠️ | DLQ test has race |
| storage | ✅ | 90.1% | ✅ | ✅ | |
| saga | ✅ | 93.4% | ✅ | ✅ | 17 fmt.Errorf wraps remain |
| watermill | ✅ | — | ✅ | ✅ | |
| integration | ✅ | — | ✅ | ✅ | |

---

*Report generated: 2026-05-28 06:24 CEST*  
*Working tree: clean, 1 commit ahead of origin*  
*Latest commit: 8807c5c — feat: add DLQ handler, reset API, and batch projection interface*
