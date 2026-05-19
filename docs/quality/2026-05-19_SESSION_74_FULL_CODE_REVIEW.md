# Full Code Review — Session 74

**Date:** 2026-05-19 | **Reviewer:** Senior Software Architect | **Files reviewed:** 40 production files

## Critical Issues (Bugs)

| # | Severity | File | Issue | Fix |
|---|----------|------|-------|-----|
| 1 | 🔴 BUG | `storage/pebble_event_store.go` Save | No optimistic concurrency check — concurrent writes silently overwrite | Add version check like MemoryStore.Save |
| 2 | 🔴 BUG | `middleware/retry.go:104` | Timer leak — no `defer timer.Stop()` before select in normal path | Add defer |
| 3 | 🔴 BUG | `core/aggregate/load_helpers.go:93-122` | `trySnapshot` creates snapshot with nil state when codec is nil | Skip snapshot when codec is nil |
| 4 | 🟠 DATA | `storage/pebble_serialization.go:76-88` | Silently discards 4 ID parsing errors on deserialization | Log warnings for corrupt data |
| 5 | 🟠 DATA | `storage/pebble_event_store.go:120-123` | Corrupt events silently skipped, incomplete results | Return error or count skips |
| 6 | 🟠 PANIC | `sync/conflict.go:40` | `NewLWWResolver` allows nil `TimestampFunc` → nil pointer panic | Validate non-nil |
| 7 | 🟠 PANIC | `catalog/schema.go:25-29` | `SchemaFromType[T]()` panics when T is an interface type | Add nil type check |

## High-Impact Architecture Issues

| # | File | Issue | Impact |
|---|------|-------|--------|
| 8 | `core/event/event.go:226` | `time.Now()` not injectable | Non-deterministic tests |
| 9 | `projection/runner.go:211-237` | `filterEvents` is O(n) — scans entire stream | Performance cliff at scale |
| 10 | `core/decider/decider.go:110` | Runtime type assertion to `TransactionalStore` | Silent fallback to non-transactional |
| 11 | `core/decider/decider.go:113` | Dual `%w` in single `fmt.Errorf` | First error unreachable via errors.As |
| 12 | `core/event/outbox_publisher.go:221` | `publishPending` swallows errors with zero observability | Silent data loss in production |
| 13 | `core/event/runner.go:170` | `collectResults` doesn't drain channel on cancellation | Goroutine resource leak |
| 14 | `storage/helpers.go:163` | `saveWithOutboxTx` with 3 callback parameters | Architectural smell |

## Naming & Type Safety Issues

| # | File | Issue |
|---|------|-------|
| 15 | `core/event/types.go:66-68` | `ParseUserAgent` returns no error — inconsistent with other Parse* funcs |
| 16 | `core/event/event.go:39` | `Core` is ambiguous — sounds like the package, not an event implementation |
| 17 | `storage/sql_helpers.go:29` | Table name interpolated directly into SQL — potential injection vector |
| 18 | `sync/vectorclock.go:43-74` | `Compare` returns 0 for both "equal" and "concurrent" — caller can't distinguish |

## Duplication

| # | Files | What's duplicated |
|---|-------|-------------------|
| 19 | `core/decider` + `core/aggregate` | Snapshot loading, opError, save+publish logic |
| 20 | `catalog/openapi` + `catalog/asyncapi` | `objectSchema()` defined identically |
| 21 | `catalog/docserver` | 6 identical handler methods differ only in what they build |
| 22 | `middleware/retry.go` | 3 retry middleware functions with identical structure |
| 23 | `middleware/tracing.go` | 3 tracing functions with identical structure |
| 24 | `storage/outbox.go` + `transactional_store.go` | Nearly identical Append code |

## Missing Validations

| # | File | What's missing |
|---|------|----------------|
| 25 | `sync/vectorclock.go:17` | `Increment` allows empty nodeID |
| 26 | `sync/operation.go:40` | `NewOperation` allows empty ID |
| 27 | `catalog/docserver/docserver.go:58-79` | `NewDocsServer` allows empty ServiceName/Version |
| 28 | `catalog/registry.go:58-80` | `AddService` doesn't deduplicate messages |
| 29 | `core/pkg/dispatcher/dispatcher.go:213-215` | `RegisterCatalogEntry` not thread-safe |

## Functions Over 30 Lines

| # | File:Line | Function | Lines |
|---|-----------|----------|-------|
| 30 | `storage/pebble_serialization.go:47` | `deserializeEvent` | 71 |
| 31 | `storage/pebble_event_store.go:48` | `Save` | 55 |
| 32 | `catalog/asyncapi/builder.go:11` | `Export` | 55 |
| 33 | `catalog/openapi/exporter.go:137` | `addQuery` | 50 |
| 34 | `catalog/eventcatalog/writer.go:13` | `writeLLMsTxt` | 51 |
| 35 | `storage/pebble_helpers.go:14` | `Delete` | 47 |
| 36 | `core/decider/decider.go:90` | `Execute` | 43 |
| 37 | `catalog/openapi/exporter.go:58` | `Export` | 42 |

## Positive Findings

- Zero TODO/FIXME comments
- All modules build and pass tests
- Strong interface-first design
- Comprehensive error classification system
- Good use of branded types (Version, SchemaVersion, IDs)
- Consistent defensive copy patterns
- All middleware factories follow identical patterns (consistency)

## Recommended Priority

1. **Fix bugs** (issues 1-3) — correctness is non-negotiable
2. **Add missing validations** (issues 6-7) — prevent panics
3. **Add observability** (issue 12) — outbox publisher silent failures
4. **Extract duplication** (issues 19-24) — reduce maintenance burden
5. **Add concurrency guards** (issue 29) — prevent data races
