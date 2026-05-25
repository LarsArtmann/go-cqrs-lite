# Comprehensive Status Report — Session 100 — 2026-05-25 23:40

**Branch:** `master` (pushed to origin, clean tree)
**Test Suite:** 22/22 packages pass, 0 failures
**Lint:** 2 pre-existing issues (noinlineerr in core/command, core/query)
**Branching-Flow:** 7 violations remaining (all intentional skips)
**Session Stats:** 28 commits, +5,955 / -547 lines across 57 files

---

## A) FULLY DONE ✅

### 1. EventCatalog Auto-Generation (20% → ~80%)

Complete resource model: channels, data stores, flows, teams, users.

- **New types** in `catalog/types_resources.go`: `DataStore`, `Flow`, `FlowStep`, `FlowActor`, `FlowCustomNode`, `FlowEdge`, `Team`, `User`
- **Shared types** in `catalog/types.go`: `Badge`, `Repository`, `Operation`, `Specification`, `Attachment`, `Ref`, `ChannelParam`, `ChannelRoute`
- **Auto-derivation**: `eventcatalog/auto_derive.go` derives producers/consumers from service topology using `map[catalog.MessageID][]catalog.ServiceID`
- **LLMs.txt** generation for all resource types
- **~60 new tests** across registry, builder, auto-derive, exporter resources
- **3 fluent option APIs** with 27 total option functions:
  - `ServiceOption` (8): Badges, Repository, WritesTo, ReadsFrom, Entities, Specifications, Attachments, Owners
  - `DomainOption` (6): Sends, Receives, Entities, Badges, Owners, Attachments
  - `ChannelOption` (8): Address, Protocols, Messages, DeliveryGuarantee, Parameters, Routes, Owners, Badges

### 2. Branded ID Type Safety — Complete Sweep

**8 catalog branded types** (`type X string` with `String()` methods):

| Type | Used In |
|------|---------|
| `ServiceID` | AddService, ConfigureService, Message.Producers/Consumers, all exporters |
| `DomainID` | AddDomain, ConfigureDomain |
| `MessageID` | Command[T], Event[T], Query[T], auto_derive map keys |
| `ChannelID` | AddChannel, ConfigureChannel, ChannelRoute.ID, ChannelRoute.To |
| `DataStoreID` | DataStore.ID, Service.WritesTo/ReadsFrom |
| `FlowID` | Flow.ID, Service.Flows, Domain.Flows |
| `TeamID` | Team.ID |
| `UserID` | User.ID |

**Additional branded types:**
- `asyncapi.URI` — AsyncAPI `Document.ID` (was bare `string`)
- `event.OutboxID` — pre-existing, unchanged
- Core IDs (`id.AggregateID`, `id.EventID`, etc.) — pre-existing, unchanged

**Storage & Middleware** (`0845b03`):
- `reconstructEvent()` uses `id.EventID` + `id.AggregateID`
- `pebble_serialization.go`: net -72 lines (eliminated `serializableMetadata`, `parseBrandedID`, `deserializeMetadata`)
- `middleware/logging.go`: `logContext.aggregateID` → `id.AggregateID`

**Naming consistency** (all exporters + test helpers):
- `msgID` → `messageID` across all files
- `svcID` → `serviceID` across all files
- `id` → `messageID` where appropriate
- `nextID` → `entryCounter` in `memory/outbox.go`
- `cattest/builders.go` params: all accept branded types directly
- `cattest/assertions.go`: `AssertServiceFrontmatter`/`AssertMessageFrontmatter` accept branded types
- All test callers updated (55+ call sites across 6 test files)

### 3. Generic Type Improvements

- `addObjectIDsListField` → generic `[S ~string]` (was `[]string`)
- `collectMessageIDs` returns `[]catalog.MessageID` (was `[]string`)
- `auto_derive.go` uses `map[catalog.MessageID][]catalog.ServiceID` (was `map[string][]catalog.ServiceID`)

### 4. Design Documentation

- `docs/planning/2026-05-25_AGGREGATE_ID_DESIGN_REVIEW.md` — PRO/CONTRA for 3 AggregateID decisions
- `docs/planning/2026-05-25_EVENTCATALOG_AUTOGENERATION_DESIGN.md` — Full EventCatalog design
- `docs/planning/2026-05-25_21-06_EVENTCATALOG_AUTOGENERATION_EXECUTION_PLAN.md` — 95-task plan

### 5. Error Context Enrichment

- `eventcatalog/writer.go`: examples marshal error includes dir
- `example/todo/commands/create_todo.go`: error includes description + priority
- `example/todo/commands/update_todo.go`: error includes description

### 6. Branching-Flow: 30 → 7 Violations

All 7 remaining are intentional skips (see section D).

---

## B) PARTIALLY DONE ⚠️

### 1. EventCatalog Coverage: 85.7% (was 91.3%)

Coverage dropped due to new code paths added faster than tests. New resource types (flows, teams, users) and fluent APIs have tests, but exporter resource generation paths need more coverage.

**Gap:** ~15 functions/paths in `exporter_resources.go`, `writer.go` without full coverage.

### 2. AGENTS.md — Partially Updated

Updated: coverage numbers, catalog module structure, session 100 entry.
Still stale: some known issues may be resolved, TODO_LIST.md cross-references.

### 3. TODO_LIST.md — 176 Open Items, Needs Triage

Many items are stale/duplicated from sessions 1-50. Needs a full prune-and-prioritize pass.

---

## C) NOT STARTED 📐

1. **Catalog diff/breaking-change detection tool** — Compare two `Catalog` structs, surface API-breaking changes
2. **Saga/orchestration pattern** — Design doc exists (`SAGA_DESIGN.md`), no implementation
3. **Query handler generics** — `query.Handler` still returns `any`; `DispatchTyped[T]` workaround. Design doc at `QUERY_HANDLER_GENERICS.md`
4. **High-level test utilities** — AggregateTester, ProjectionTester, BusTester fluent API
5. **Pebble optimistic concurrency fix** — concurrent writes silently overwrite
6. **Outbox transaction co-participation** — SQLOutbox.Append and SQLEventStore.Save in separate transactions
7. **collectResults goroutine leak** — projection/runner.go doesn't drain channel on cancellation
8. **FuzzParse case-sensitivity** — ULID case-folding roundtrip mismatch
9. **storage/dialect.go `any` cleanup** — 3 methods violate "no any" rule
10. **OutboxPublisher split-brain** — cancel stays non-nil after Close()
11. **catalog/asyncapi missing CommandMessage case** — exporter.go gap
12. **Registry deterministic Build()** — non-deterministic map iteration
13. **Per-module isolated builds** — `GOWORK=off go test` fails for most modules (missing go.sum entries)
14. **Deprecate DeriveAggregateID** — add `// Deprecated:` comment
15. **Fix example/todo TodoMarker embedding** — misleading (wrong backing type)

---

## D) TOTALLY FUCKED UP 💥

Nothing truly broken. Honest assessment:

1. **eventcatalog coverage regression (91.3% → 85.7%)** — New code added faster than tests. Not broken, but sloppy discipline. Should be >90%.

2. **176 open TODO items** — Accumulated over 100 sessions without systematic triage. Noise level is high.

3. **Per-module isolated builds broken** — `GOWORK=off go test` fails for most modules. Workspace builds work fine, but this means individual module consumers could hit issues. **This is the #1 real quality problem.**

4. **AGENTS.md still has stale content** — Some known issues reference resolved items, coverage table doesn't include catalog/docserver in the summary.

5. **2 pre-existing lint issues** — `noinlineerr` in core/command and core/query dispatchers. Not from this session but still present.

---

## E) WHAT WE SHOULD IMPROVE 🔧

### Process

1. **Test-first discipline** — Coverage should never drop below previous session's level. Add tests WITH new code.
2. **Per-module isolated builds** — Run `GOWORK=off go test` in CI for each module. This is the #1 consumer-facing quality issue.
3. **TODO_LIST.md triage** — 176 items is noise. Prune to top 25, archive the rest.
4. **AGENTS.md freshness** — Update known issues table and coverage every session.
5. **Pre-commit hook false positive** — `todo-check` in BuildFlow matches `ToDotAddress` as `TODO`. Should use word boundary.

### Technical

1. **Recover eventcatalog coverage to >90%** — Target specific untested paths in exporter_resources.go and writer.go.
2. **Pebble concurrency** — Known data-loss bug in Save(). Not production-safe.
3. **Outbox transactionality** — Two separate transactions for store+outbox = potential inconsistency.
4. **Registry deterministic Build()** — Non-deterministic map iteration can cause test flakiness.
5. **Deprecate DeriveAggregateID** — Zero callers, YAGNI, but we can't remove it (SDK semver). Deprecation is the right path.
6. **Fix example/todo TodoMarker** — `TodoMarker` embeds `AggregateMarker` but uses ULID backing while `AggregateID` uses string backing. Misleading type relationship.

### Architecture

1. **Ref.ID polymorphism** — Currently `string` because it can be ServiceID, MessageID, ChannelID, etc. Consider an `IDRef` interface or `RefKind` discriminator.
2. **Owners []string** — Could be TeamID or UserID. Consider `OwnerRef` with `Kind()` method.
3. **FlowStep.ID / FlowEdge.ID** — User-defined identifiers in flow diagrams. Could be branded as `FlowStepLabel` to clarify they're not entity IDs.

---

## F) Top #25 Things We Should Get Done Next

| # | Priority | Task | Impact | Effort |
|---|----------|------|--------|--------|
| 1 | 🔴 CRITICAL | Fix per-module isolated builds (`go mod tidy` each module) | Consumers can't `go get` individually | 30 min |
| 2 | 🟡 MEDIUM | Recover eventcatalog coverage to >90% | Quality gate | 1-2 hrs |
| 3 | 🟡 MEDIUM | Deprecate `DeriveAggregateID` with `// Deprecated:` comment | API hygiene (semver-safe) | 5 min |
| 4 | 🟡 MEDIUM | Fix example/todo `TodoMarker` embedding (misleading backing type) | Type honesty | 15 min |
| 5 | 🟡 MEDIUM | Triage TODO_LIST.md — prune stale, deduplicate, prioritize top 25 | Noise reduction | 1 hr |
| 6 | 🟡 MEDIUM | Fix Pebble Store optimistic concurrency in Save | Data safety | 1-2 hrs |
| 7 | 🟡 MEDIUM | Fix Outbox transaction co-participation | Data consistency | 2-3 hrs |
| 8 | 🟡 MEDIUM | Fix `collectResults` goroutine leak in projection/runner.go | Resource leak | 1 hr |
| 9 | 🟡 MEDIUM | Fix OutboxPublisher split-brain (cancel non-nil after Close) | Correctness | 30 min |
| 10 | 🟡 MEDIUM | Fix storage/dialect.go `any` usage (3 methods) | Type safety | 30 min |
| 11 | 🟡 MEDIUM | Fix asyncapi exporter missing CommandMessage case | Feature completeness | 30 min |
| 12 | 🟢 LOW | Registry deterministic Build() (sort map iteration) | Test reliability | 30 min |
| 13 | 🟢 LOW | Fix FuzzParse case-sensitivity roundtrip | Edge case correctness | 1 hr |
| 14 | 🟢 LOW | Add slog.Warn for corrupt IDs in Pebble deserialization | Observability | 15 min |
| 15 | 🟢 LOW | Update FEATURES.md — add openapi, docserver, dialect, recent sessions | Documentation | 1 hr |
| 16 | 🟢 LOW | Design `OwnerRef` union type for Owners []string | Type safety | 1 hr |
| 17 | 🟢 LOW | Design `RefKind` discriminator for Ref.ID polymorphism | Type safety | 1 hr |
| 18 | 🟢 LOW | Fix 2 lint issues (noinlineerr in core/command, core/query) | Zero lint | 15 min |
| 19 | 🟢 LOW | Fix BuildFlow pre-commit hook false positive (ToDotAddress matched as TODO) | Developer experience | 15 min |
| 20 | 🟢 LOW | Brand FlowStep.ID as FlowStepLabel (clarify it's not an entity ID) | Naming clarity | 10 min |
| 21 | 🟢 LOW | Add catalog diff/breaking-change detection tool | API evolution safety | 3-4 hrs |
| 22 | 🟢 LOW | Query handler generics (TypedHandler[T] returning T, error) | Type safety (breaking) | 4-8 hrs |
| 23 | 🟢 LOW | High-level test utilities (AggregateTester, ProjectionTester) | Consumer DX | 4-8 hrs |
| 24 | 📐 PLANNED | Saga/orchestration pattern implementation | Feature expansion | 1-2 days |
| 25 | 📐 PLANNED | Publish go-composable-business-types as Go module | External adoption | 1-2 days |

---

## G) Top #1 Question 🤔

**Should `Ref.ID` stay as `string`, or should we introduce a discriminator?**

`Ref` is used in 4 semantically different ways:
- `FlowStep.Service` — references a `ServiceID`
- `FlowStep.Message` — references a `MessageID`
- `FlowStep.Channel` — references a `ChannelID`
- `Attachment.URL` — not an ID at all

The branching-flow tool flags `Ref.ID string` as a violation. But Go has no union type. Options:

1. **Keep `string`** — honest about polymorphism, document the valid types
2. **`RefKind` enum** — `Ref { Kind RefKind; ID string }` with runtime validation
3. **Split into typed refs** — `ServiceRef`, `MessageRef`, `ChannelRef` (duplicative but type-safe)
4. **Generic `Ref[T]`** — `Ref[ServiceID]`, `Ref[MessageID]` (cleanest but breaks JSON)

My recommendation: **Option 1 (keep `string`)** — the field name `Ref.ID` is already semantically clear, and adding runtime validation is overhead for what's a data-transfer type. Document the valid types in a comment.

---

## Coverage Summary

| Package | Coverage | Change |
|---------|----------|--------|
| `core/pkg/dispatcher` | 100.0% | — |
| `core/pkg/id` | 100.0% | — |
| `middleware` | 100.0% | — |
| `catalog/internal/caseutil` | 100.0% | — |
| `memory` | 99.6% | — |
| `core/query` | 98.4% | — |
| `catalog` | 96.3% | ↓ (was 96.8%) |
| `catalog/d2` | 95.0% | — |
| `catalog/openapi` | 94.4% | — |
| `projection` | 94.4% | — |
| `core/event` | 93.8% | — |
| `catalog/asyncapi` | 93.7% | — |
| `core/decider` | 93.6% | — |
| `core/command` | 92.3% | — |
| `testhelpers` | 91.3% | — |
| `catalog/docserver` | 90.1% | — |
| `storage` | 89.3% | — |
| `catalog/internal/schemautil` | 84.2% | — |
| `catalog/eventcatalog` | 85.7% | ↓ (was 91.3%) |

**Overall: ~90% weighted average**

---

## Branching-Flow: 7 Remaining Violations (All Intentional)

| Violation | File | Reason |
|-----------|------|--------|
| `serviceDisplayID` (×3) | d2/connections.go | Sanitized D2 display strings, not catalog identifiers |
| `OperationID` | openapi/types.go:49 | OpenAPI spec field, not our type to brand |
| `Ref.ID` | types.go:268 | Polymorphic (ServiceID, MessageID, ChannelID — no Go union type) |
| `FlowStep.ID` | types_resources.go:31 | User-defined step label in flow diagrams, not a catalog entity |
| `FlowEdge.ID` | types_resources.go:66 | User-defined edge label in flow diagrams, not a catalog entity |

---

## Session Commit History (28 commits)

```
48912a0 fix(catalog,event/todo): enrich error messages with more context
5bc35d3 docs(AGENTS.md): update coverage, catalog structure, session 100
613f989 style(catalog): rename svcID params to serviceID in asyncapi and openapi
3910f4f refactor(catalog,memory): use catalog.MessageID in maps, rename short vars
880bce5 style(catalog): adopt typed catalog.MessageID across d2, eventcatalog, cattest, types
51e1d53 style(catalog): use typed catalog.MessageID instead of string in asyncapi builder
8ccf3b3 style(catalog): normalize formatting in golden test files and docs
5401c86 style(catalog): align struct field tags and format multi-line function calls
3ade44d docs(status): add comprehensive session status reports
8179c07 refactor(catalog): branded types for Producers/Consumers, WritesTo/ReadsFrom, Flows
5ab50b6 refactor(catalog): AddService accepts ServiceID, AddDomain accepts DomainID
ed64448 refactor(catalog): Command[T]/Event[T]/Query[T] accept MessageID instead of string
ac0f2cb docs: add comprehensive session status at 22:26
c3b489a refactor(catalog): use branded ID types for DataStore, Flow, Team, User
0ae04a5 docs: add AggregateID design review and comprehensive session status
0845b03 refactor(storage,middleware): use branded IDs throughout serialization and logging
730c848 docs: add EventCatalog auto-generation session status report
b3ad286 test(catalog): add LLMs.txt content verification test for all resource types
0b858c1 feat(catalog): add ChannelOption fluent API for channel-level metadata
aa3000d feat(catalog): add DomainOption fluent API for domain-level metadata
9b2de7c feat(catalog): add ServiceOption fluent API for service-level metadata
b843740 test(catalog): add comprehensive tests for new registry, builder, and auto-derive
7974dfd test(catalog): update BuildTestCatalog with channels and data stores
3a960ff feat(catalog): extend LLMs.txt generation to include all resource types
9d4dbb0 refactor(catalog): deduplicate MessagePointer and FlowStepRef into single Ref type
31f3ade feat(catalog): comprehensive EventCatalog auto-generation (20% → 80% coverage)
d56608e docs: add EventCatalog auto-generation execution plan with 95 granular tasks
1ea8b14 docs: add comprehensive EventCatalog auto-generation design document
```

---

*Report generated: 2026-05-25 23:40 · Session 100*
