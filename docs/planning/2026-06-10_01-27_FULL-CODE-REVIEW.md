# Full Code Review — go-cqrs-lite

> **Date:** 2026-06-10 · **Reviewer:** Automated Architect Analysis · **Scope:** All 22 library modules

## Pareto Analysis

### 1% → 51% Impact (Fix These First)

| # | Issue | Module(s) | Impact |
|---|-------|-----------|--------|
| 1 | **Circular deps: event↔command, event↔memory, memory↔snapshot** | event, command, memory, snapshot | Prevents independent versioning |
| 2 | **Storage event/command store duplication (~300 lines)** | storage | Maintenance burden, bug hiding |
| 3 | **Error taxonomy split-brain: event/errors.go ≈ command/errors.go** | event, command | Confusing API surface |

### 4% → 64% Impact

| # | Issue | Module(s) | Impact |
|---|-------|-----------|--------|
| 4 | **Test deps in production go.mod (12 modules)** | 12 of 22 | Binary bloat for consumers |
| 5 | **command re-exports event types (AggregateRef, Metadata)** | command | Split brain, error wrapping incompatibility |
| 6 | **Lifecycle exported field on Dispatcher** | dispatcher | Exposes internal mutex |
| 7 | **WithReplay has no IsReplay getter** | event | Write-only context value |
| 8 | **listRefsFromStatus duplicated across listing/storage** | listing, storage | Copy-paste bug risk |

### 20% → 80% Impact

| # | Issue | Module(s) | Impact |
|---|-------|-----------|--------|
| 9 | HTTP code (SSE, healthcheck) mixed into middleware | middleware | Cohesion violation |
| 10 | Retry logic reimplemented in projection | projection | Duplication |
| 11 | defaultClock mutable global in event | event | Data race in parallel tests |
| 12 | AggregateProjection uses hardcoded `?` placeholders | storage | Postgres incompatible |
| 13 | query.Handler has different signature than command/event Handler | query | Asymmetric API |
| 14 | Map/ScanState/Tap reactive wrappers are test-only dead API | event | API surface pollution |
| 15 | pebble locks sync.Map grows unbounded | pebble | Memory leak in long-running processes |

## Module-by-Module Findings

### event/ (Core)

| Severity | Finding | File:Line |
|----------|---------|-----------|
| CRITICAL | Error taxonomy fully re-exported (also in command) | errors.go:3-73 |
| HIGH | WithReplay writes context value with no exported getter | replay.go:10 |
| HIGH | defaultClock is mutable package-level var | types.go:16 |
| MEDIUM | StreamKey() free function duplicates AggregateRef.StreamKey() | stream.go:11 |
| MEDIUM | Map/ScanState/Tap are test-only thin wrappers | reactive.go:112-128 |
| MEDIUM | SaveFunc exported but only used in eventtest | store.go:12 |
| LOW | NewMetadata() returns zero-value (no-op) | metadata.go:22 |

### command/

| Severity | Finding | File:Line |
|----------|---------|-----------|
| HIGH | AggregateRef/AggregateType split-brain with event | aggregate_ref.go:1-35 |
| HIGH | errors.go duplicates event/errors.go (90% unused in prod) | errors.go:1-71 |
| MEDIUM | Metadata options support only 4 fields vs event's 12+ | metadata.go |
| LOW | Close() uses event.WrapInfrastructure instead of own | dispatcher.go:99 |

### query/

| Severity | Finding | File:Line |
|----------|---------|-----------|
| HIGH | Close() imports event.WrapInfrastructure directly | dispatcher.go:153 |
| MEDIUM | BasicQuery has no Metadata, no AggregateID | query.go:19 |
| MEDIUM | Handler is type alias, not named type (inconsistent with command) | dispatcher.go:27 |

### decider/

| Severity | Finding | File:Line |
|----------|---------|-----------|
| MEDIUM | opError double-wraps (fmt.Errorf + event.Wrap) | load.go:53 |
| MEDIUM | applyEnricher silently skips non-ImmutableEvent | enricher.go:19 |
| LOW | Errors correctly delegate to event's taxonomy (correct pattern) | errors.go |

### id/

| Severity | Finding | File:Line |
|----------|---------|-----------|
| MEDIUM | AggregateID uses string (not ULID), creating type split | aggregate_id.go:26 |
| LOW | AggregateIDFrom bypasses empty-string validation | aggregate_id.go:75 |

### dispatcher/

| Severity | Finding | File:Line |
|----------|---------|-----------|
| HIGH | Lifecycle exported field exposes internal mutex | dispatcher.go:49 |
| MEDIUM | Close() always returns nil | lifecycle.go:12 |

### memory/

| Severity | Finding | File:Line |
|----------|---------|-----------|
| MEDIUM | MemoryCommandStore duplicates MemoryStore indexed-log pattern | command_store.go |
| MEDIUM | filterByTimestamp duplicated from event filtering | command_store.go:216 |
| LOW | 5 structs all reimplement checkClosed | All files |

### storage/

| Severity | Finding | File:Line |
|----------|---------|-----------|
| HIGH | loadParams struct duplicated between event/command stores | event_store_load.go:13, command_store_load.go:65 |
| HIGH | loadWithSpan() identical between event/command stores | event_store_load.go:22, command_store_load.go:74 |
| HIGH | queryEvents/queryCommands structurally identical | event_store_load.go:124, command_store_load.go:103 |
| MEDIUM | listRefsFromStatus duplicated from listing | sql_aggregate_reader.go:182 |
| MEDIUM | AggregateProjection uses hardcoded `?` placeholders | aggregate_projection.go:67 |
| MEDIUM | SQLBackend wraps only event store (incomplete facade) | sql_backend.go:10 |
| MEDIUM | checkClosed inconsistency (event vs command error sentinels) | event_store.go:64, command_store.go:61 |

### middleware/

| Severity | Finding | File:Line |
|----------|---------|-----------|
| MEDIUM | HTTP code (SSE, healthcheck, metrics HTTP) mixed with CQRS middleware | sse.go, healthcheck.go, metrics_http.go |
| MEDIUM | OTel metrics duplicates generic metrics pattern | metrics.go, metrics_otel.go |
| LOW | NewSSEBroker returns nil on error | sse.go:33 |

### projection/

| Severity | Finding | File:Line |
|----------|---------|-----------|
| MEDIUM | handleWithRetry reimplements middleware retry logic | runner_live.go:95 |
| LOW | Close() doesn't wait for goroutines | runner.go:232 |

### signing/, otel/, watermill/

| Severity | Finding | File:Line |
|----------|---------|-----------|
| ✅ | **All clean** — exemplary architecture, well-scoped, well-typed | — |

### listing/

| Severity | Finding | File:Line |
|----------|---------|-----------|
| MEDIUM | listRefsFromStatus duplicated in storage | aggregate_reader.go:25 |

### pebble/

| Severity | Finding | File:Line |
|----------|---------|-----------|
| MEDIUM | sync.Map locks grow unbounded (memory leak) | store.go:26 |

## Code Quality Metrics

| Metric | Value |
|--------|-------|
| Total production Go files | ~330 |
| Total test Go files | ~267 |
| Files over 350 lines | 6 (catalog builders, watermill protocol, decider, event/types) |
| Duplicate code groups (dupl) | 24 groups |
| Lint issues | 0 |
| Test pass rate | 100% (40 packages) |
| Circular module dependencies | 4 bidirectional |

## Strengths

1. **ISP-split interfaces** — Store = Sink + Source applied consistently across event, command, snapshot, checkpoint
2. **Pure-function Decider** — Decider[State] with Fold is textbook functional CQRS
3. **Branded IDs** — Phantom type parameters eliminate ID confusion at compile time
4. **5-family error taxonomy** — Rejection/Conflict/Transient/Infrastructure/Corruption with IsRetryable()
5. **Zero lint issues** — golangci-lint clean across all 22 modules
6. **Consistent naming** — New/Must/NewX, WithX options, ErrX sentinels throughout
7. **Tombstone soft-delete** — Proper event sourcing pattern, no destructive deletes
8. **signing/, otel/, watermill/** — Exemplary adapter modules with clean interfaces
