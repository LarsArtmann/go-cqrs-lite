# Session 38 — Deep Cleanup, Type Safety, Deduplication

> **Date:** 2026-05-03
> **Status:** Complete — all 21 test packages pass, zero lint

---

## What Was Done

### Phase 1: Hygiene
| Task | Status | Detail |
|------|--------|--------|
| `go mod tidy` all modules | ✅ Done | Workspace handles module resolution |
| `.golangci.yml` | ✅ Already exists | Comprehensive config with 60+ linters |
| Golden files fixed | ✅ Done | TrimSpace comparison prevents trailing newline drift |

### Phase 2: Bug Fixes
| Task | Status | Detail |
|------|--------|--------|
| `:=` vs `=` in repository.go:96 | ✅ Done | `err :=` → `err =` (already committed in prior session) |
| `EveryNEvents` validate n > 0 | ✅ Done | Panics on zero/negative interval |
| Remove dead `dispatcher.Typed` | ✅ Done | No production code referenced it |
| `exhaustruct` on `New*Error()` constructors | ✅ Done | Explicit `cause: nil` in all 5 constructors |
| `noinlineerr` in repository.go | ✅ Done | Split inline error handling to plain assignment |

### Phase 3: Deduplication
| Task | Status | Detail |
|------|--------|--------|
| Extract `insertEvents` in storage/ | ✅ Done | Save/AppendBatch now share one insertion loop |
| Extract `pollPublishAck` in outbox_publisher | ✅ Done | publishPending/PublishNow now share one cycle |
| Export `event.SubscribesTo`, deduplicate from projection/ | ✅ Done | Identical logic was copy-pasted |
| Remove unused `customCmd` type in dispatcher_test | ✅ Done | Dead test type |

### Phase 4: API Quality
| Task | Status | Detail |
|------|--------|--------|
| `FakeOutbox.Ack` respects IDs | ✅ Done | Was clearing all entries; now filters by provided IDs |
| `HandlerRegistry.On` accepts `event.Type` | ✅ Done | Changed from `string` to branded type |
| 2 dynamic errors → sentinel errors | ✅ Done | `ErrEmptyQueryType`, `ErrNilSchema` |
| Golden test TrimSpace comparison | ✅ Done | Prevents chronic trailing-newline mismatches |

---

## What Was Deferred (With Reason)

| Task | Reason |
|------|--------|
| `NewEvent` accept `Version` not `int` | 100+ call sites, low ROI for the type safety gain |
| `FakeStore.Load` return `ErrAggregateNotFound` | Requires repository.Load to handle it first (behavior change) |
| LifecycleMixin for memory/checkpoint + outbox | Test utilities — over-engineering for closed-state protection |
| `query.Handler` returns `any` | Requires generics redesign; architectural decision |
| CatalogMeta 3× consolidation | Breaking change across 3 packages; needs ADR |
| `asyncapi/exporter` mutable fields | Consumers don't mutate them; low risk |
| Projection retry: add jitter + cap | Would change runtime behavior; needs careful testing |
| `RecordEvent` unused context parameter | Breaking API change; needs deprecation cycle |

---

## Test & Lint Status

```
21 packages: all pass
0 lint issues across all modules
```

---

## Commits This Session

| Hash | Message |
|------|---------|
| `16adf24` | fix: consistent error assignment, nil causes, and test helper refactor |
| `e34a50c` | docs: add Session 38 execution plan |
| `4f5f5be` | fix(aggregate): validate EveryNEvents interval is positive |
| `ab54f0c` | refactor(dispatcher): remove unused Typed interface |
| `765cfd4` | refactor(storage): extract insertEvents helper from Save/AppendBatch |
| `7415a1c` | fix: FakeOutbox.Ack respects IDs, HandlerRegistry.On accepts event.Type |
| `072ac58` | fix: golden tests use TrimSpace comparison, update golden files |
| `1bc221f` | style: fix lint issues in EveryNEvents panic test |
| `752c711` | chore(deps): run go mod tidy on all modules, fix projection go.mod |
| `ddcac96` | refactor(event): extract pollPublishAck from publishPending/PublishNow |
| `bdc15e8` | refactor: export event.SubscribesTo, deduplicate from projection/runner |
| `04045d7` | refactor: replace 2 dynamic errors with sentinel errors |
| `639ccc0` | docs: fix stale godoc comment on SubscribesTo |

---

## Known Issues (Updated)

| Issue | Severity | Detail |
|-------|----------|--------|
| `MemoryBus.Publish` holds RLock during handler execution | LOW | Acceptable for test utility |
| `query.Handler` returns `any` | LOW | Violates "no any" rule; `DispatchTyped[T]` is the workaround |
| `CatalogMeta` duplicated across 3 packages | LOW | Near-identical structs in event/command/query |
| `Root.LoadEvents` vs `Core.LoadFromHistory` boilerplate | LOW | Every aggregate must implement `LoadEvents` delegation |
| Cross-package sentinels not in `Classify()` | MEDIUM | Circular dependency prevents mapping aggregate/projection/storage errors |
| Projection runner retry lacks jitter + max-delay cap | MEDIUM | Diverges from middleware/retry.go implementation |
| `projection.Runner.Close()` is a no-op | MEDIUM | No way to stop a running Run() loop |
| `gochecknoglobals` on test-only variables | LOW | Golden test flag pattern; suppressed in .golangci.yml |
