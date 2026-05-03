# Session 38 Execution Plan: Deep Cleanup + Type Safety + Deduplication

> **Date:** 2026-05-03
> **Status:** Executing
> **Parent:** Session 37 comprehensive status report

---

## Honest Self-Assessment

### What We Got Wrong
1. **69+ gopls errors from stale go.mod files** — `projection/go.mod` missing indirect deps
2. **No `.golangci.yml`** — linter config is purely default, missing useful linters
3. **Latent `:=` vs `=` compile error** in `repository.go:96` broke integration/ and example/
4. **Golden files stale** — trailing newline diffs causing test failures
5. **`storage/event_store.go` Save/AppendBatch** have identical insertion loops (duplication)
6. **`outbox_publisher.go` publishPending/PublishNow** have identical poll-publish-ack loops
7. **`memory/checkpoint.go` + `memory/outbox.go`** missing LifecycleMixin (inconsistent)
8. **`FakeStore.Load` returns nil instead of `ErrAggregateNotFound`** — inconsistent with real impls
9. **`FakeOutbox.Ack` ignores IDs parameter** — clears everything
10. **`EveryNEvents` no input validation** — n=0 causes division-by-zero panic

### Key Architectural Observations
- **CatalogMeta** duplicated 3× across event/command/query packages (HIGH duplication)
- **subscribesTo()** duplicated between `core/event/runner.go` and `projection/runner.go`
- **streamKey** duplicated between `memory/helpers.go` (1×) and `testhelpers/fake_store.go` (5×)
- **dispatcher.Typed** interface is dead code
- **NewEvent takes raw `int` for version** instead of branded `event.Version`
- **query.Handler returns `any`** — violates project "no any" rule
- **projection/HandlerRegistry.On takes `string`** instead of `event.Type`
- **asyncapi/exporter has mutable exported fields** — inconsistent with options pattern

---

## Execution Plan (Sorted: Impact × Effort)

### Phase 1: Hygiene (fix build, lint, deps) — LOW EFFORT, HIGH IMPACT

| # | Task | Effort | Impact |
|---|------|--------|--------|
| 1 | `go mod tidy` all modules | 5min | Fixes 69+ gopls errors |
| 2 | Add `.golangci.yml` | 10min | Consistent linting |
| 3 | Commit golden files (trailing newline fix) | 2min | Clean test runs |

### Phase 2: Bug Fixes — LOW EFFORT, HIGH IMPACT

| # | Task | Effort | Impact |
|---|------|--------|--------|
| 4 | `NewEvent`: accept `Version` not `int` | 5min | Type safety on public API |
| 5 | `FakeStore.Load`: return `ErrAggregateNotFound` | 5min | Consistent error semantics |
| 6 | `EveryNEvents`: validate `n > 0` | 3min | Prevent division-by-zero |
| 7 | Remove dead `dispatcher.Typed` interface | 2min | Reduce API surface |

### Phase 3: Deduplication — MEDIUM EFFORT, HIGH IMPACT

| # | Task | Effort | Impact |
|---|------|--------|--------|
| 8 | Extract `insertEvents` helper in storage/ | 10min | Remove identical loop |
| 9 | Extract `pollPublishAck` in outbox_publisher | 10min | Remove identical loop |
| 10 | Add LifecycleMixin to memory/checkpoint + outbox | 10min | Consistency |
| 11 | Share `streamKey` between memory/ and testhelpers/ | 10min | Remove 5× duplication |
| 12 | Deduplicate `subscribesTo()` — projection calls event's | 5min | Remove 2× duplication |

### Phase 4: API Quality — MEDIUM EFFORT, MEDIUM IMPACT

| # | Task | Effort | Impact |
|---|------|--------|--------|
| 13 | `projection/HandlerRegistry.On`: accept `event.Type` | 5min | Type safety |
| 14 | `FakeOutbox.Ack`: respect IDs parameter | 5min | Correct test fake |
| 15 | projection runner retry: add jitter + cap | 10min | Match middleware pattern |
| 16 | Unexport asyncapi/exporter fields | 5min | Immutability |
| 17 | Remove redundant nil check in `saveSnapshot` | 3min | Clean code |

### Phase 5: Documentation + Verification

| # | Task | Effort | Impact |
|---|------|--------|--------|
| 18 | Update AGENTS.md with Session 38 changes | 5min | Memory |
| 19 | Update status report | 5min | Documentation |
| 20 | Full test + lint verification | 5min | Confidence |

---

## What We're NOT Doing (Deferred)

| Task | Reason |
|------|--------|
| CatalogMeta consolidation (3× → 1×) | Breaking change across 3 packages; needs ADR |
| query.Handler `any` removal | Requires generics redesign; architectural decision |
| RecordEvent: remove unused context | Breaking API change; needs deprecation cycle |
| SchemaVersion branded type | Low impact; can do later |
| CatalogBuilder de-dup with Registry | Architectural refactor; needs design doc |
