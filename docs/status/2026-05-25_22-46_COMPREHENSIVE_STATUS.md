# Comprehensive Status Report — 2026-05-25 22:46

**Branch:** `master` (12 commits ahead of origin, clean tree)
**Test Suite:** 22/22 packages pass, 0 failures
**Lint:** 2 pre-existing issues (noinlineerr in core/command, core/query dispatchers)
**Session Net Change:** +1,565 / -219 lines across 30 files (12 commits)
**Total Codebase:** 307 Go files, 46,474 lines across 10 modules + 2 examples

---

## A) FULLY DONE ✅

### 1. EventCatalog Auto-Generation (20% → ~80%)

Complete channel, data store, flow, team, and user support added to the EventCatalog exporter.

| Commit    | What                                                                     |
| --------- | ------------------------------------------------------------------------ |
| `31f3ade` | Core types, registry methods, builder integration, exporter, auto-derive |
| `9d4dbb0` | Deduplicate `MessagePointer`/`FlowStepRef` → single `Ref` type           |
| `3a960ff` | LLMs.txt generation extended to all resource types                       |
| `7974dfd` | BuildTestCatalog updated with channels and data stores                   |
| `b843740` | ~25 tests: registry, builder, auto-derive                                |
| `b3ad286` | LLMs.txt content verification test                                       |

**New types** (`catalog/types_resources.go`): `DataStore`, `Flow`, `FlowStep`, `FlowActor`, `FlowCustomNode`, `FlowEdge`, `Team`, `User`

**Shared types** (`catalog/types.go`): `Badge`, `Repository`, `Operation`, `Specification`, `Attachment`, `Ref`, `ChannelParam`, `ChannelRoute`

**3 fluent option APIs** (27 option functions total):

- `ServiceOption` (8): Badges, Repository, WritesTo, ReadsFrom, Entities, Specifications, Attachments, Owners
- `DomainOption` (6): Sends, Receives, Entities, Badges, Owners, Attachments
- `ChannelOption` (8): Address, Protocols, Messages, DeliveryGuarantee, Parameters, Routes, Owners, Badges

### 2. Branded ID Types — Full Catalog Overhaul

8 branded types replacing raw `string` across the entire catalog module:

| Type          | Backing         | Used In                                                                  |
| ------------- | --------------- | ------------------------------------------------------------------------ |
| `ServiceID`   | `string` (slug) | AddService, ConfigureService, Message.Producers/Consumers, all exporters |
| `DomainID`    | `string` (slug) | AddDomain, ConfigureDomain                                               |
| `MessageID`   | `string` (slug) | Command[T], Event[T], Query[T]                                           |
| `ChannelID`   | `string` (slug) | AddChannel, ConfigureChannel                                             |
| `DataStoreID` | `string` (slug) | DataStore.ID, Service.WritesTo/ReadsFrom, registry map keys              |
| `FlowID`      | `string` (slug) | Flow.ID, Service.Flows, Domain.Flows, registry map keys                  |
| `TeamID`      | `string` (slug) | Team.ID, registry map keys                                               |
| `UserID`      | `string` (slug) | User.ID, registry map keys                                               |

### 3. Storage & Middleware Branded IDs (`0845b03`)

- `storage/event_reconstruction.go`: `reconstructEvent()` uses `id.EventID` + `id.AggregateID`
- `storage/event_store_scan.go`: Parses SQL strings into branded IDs
- `storage/pebble_serialization.go`: Net -72 lines (eliminated `serializableMetadata`, `parseBrandedID`, `deserializeMetadata`)
- `storage/outbox_helpers.go`: `outboxEvent` uses branded types
- `middleware/logging.go`: `logContext.aggregateID` → `id.AggregateID`

### 4. Branded Slice Fields (`8179c07`)

- `Message.Producers []ServiceID` (was `[]string`)
- `Message.Consumers []ServiceID` (was `[]string`)
- `Service.WritesTo []DataStoreID` (was `[]string`)
- `Service.ReadsFrom []DataStoreID` (was `[]string`)
- `Service.Flows []FlowID` (was `[]string`)
- `Domain.Flows []FlowID` (was `[]string`)

### 5. Design Documentation

- `docs/planning/2026-05-25_AGGREGATE_ID_DESIGN_REVIEW.md` — PRO/CONTRA for 3 AggregateID decisions
- `docs/planning/2026-05-25_EVENTCATALOG_AUTOGENERATION_DESIGN.md` — Full EventCatalog design
- `docs/planning/2026-05-25_21-06_EVENTCATALOG_AUTOGENERATION_EXECUTION_PLAN.md` — 95-task execution plan

---

## B) PARTIALLY DONE ⚠️

### 1. EventCatalog Coverage: 85.8% (was ~91.3%)

Coverage dropped from ~91.3% to 85.8% due to new untested code paths added faster than tests. The new resource types (flows, teams, users) and fluent APIs have tests, but the exporter resource generation paths need more coverage.

**Gap:** ~15 new functions/paths in `exporter_resources.go`, `auto_derive.go` without full coverage.

### 2. cattest/builders.go — Branded Type Migration (Low Priority)

Test helpers currently accept raw `string` and convert internally to branded types. Works correctly but is inconsistent with the public API.

**Remaining:** Update function signatures to accept `catalog.ServiceID`, `catalog.DomainID`, `catalog.MessageID` etc. directly.

### 3. AggregateID Design Review — Pending User Decision

Three decisions documented at `docs/planning/2026-05-25_AGGREGATE_ID_DESIGN_REVIEW.md`:

1. Should `AggregateID` switch to ULID backing? → Recommendation: **NO** (breaks consumer data)
2. Should `DeriveAggregateID` be removed? → Recommendation: **YES** (YAGNI, zero callers)
3. Should `AggregateMarker` be unexported? → Recommendation: **YES** (decorative only, zero interop)

### 4. TODO_LIST.md / FEATURES.md — Stale

Both docs have open items noting they're stale (Session 72). Missing entries for openapi, docserver, sync module, storage Dialect, and newer features from Sessions 87-100.

---

## C) NOT STARTED 📐

1. **Catalog diff/breaking-change detection tool** — Would compare two `Catalog` structs and surface API-breaking changes
2. **Saga/orchestration pattern** — Design doc exists (`docs/planning/SAGA_DESIGN.md`), no implementation
3. **Query handler generics** — `query.Handler` still returns `any`; `DispatchTyped[T]` is the workaround. Design doc at `docs/planning/QUERY_HANDLER_GENERICS.md`
4. **High-level test utilities** — AggregateTester, ProjectionTester, BusTester fluent API
5. **Pebble optimistic concurrency fix** — concurrent writes silently overwrite
6. **Outbox transaction co-participation** — SQLOutbox.Append and SQLEventStore.Save run in separate transactions
7. **collectResults goroutine leak** — doesn't drain channel on cancellation (projection/runner.go)
8. **FuzzParse case-sensitivity** — ULID case-folding roundtrip mismatch
9. **storage/dialect.go `any` cleanup** — 3 methods violate "no any" rule
10. **OutboxPublisher split-brain** — cancel stays non-nil after Close()
11. **catalog/asyncapi missing CommandMessage case** — exporter.go gap
12. **Push 12 commits to origin**

---

## D) TOTALLY FUCKED UP 💥

### Nothing is truly fucked. But here's the honest assessment:

1. **eventcatalog coverage regression (91.3% → 85.8%)** — New code added faster than tests. Not broken, but sloppy quality discipline. Should be >90%.

2. **176 open TODO items in TODO_LIST.md** — Many are stale/duplicated, but the sheer number suggests accumulation without triage. Needs pruning.

3. **12 unpushed commits** — All work this session is local only. One `rm -rf` away from total loss.

4. **AGENTS.md coverage table is stale** — Says `catalog/eventcatalog` is 91.3% but actual is 85.8%. Other numbers may also be outdated.

5. **Per-module isolated tests FAIL** — `GOWORK=off go test` fails for most modules due to missing go.sum entries. Workspace works fine, but isolated module builds are broken. This means consumers who `go get` individual modules may hit issues.

6. **2 lint issues remain** — `noinlineerr` in core/command and core/query dispatchers. Pre-existing, but still present.

---

## E) WHAT WE SHOULD IMPROVE 🔧

### Process

1. **Test-first discipline** — Coverage should never drop below previous session's level. Add tests BEFORE or WITH new code, never "later."
2. **Push after every session** — 12 unpushed commits is reckless. Make `git push` the last step of every session.
3. **TODO_LIST.md triage** — 176 items is noise. Prune completed/stale, consolidate duplicates, prioritize the top 20.
4. **AGENTS.md freshness** — Coverage numbers, module descriptions, and known issues should be updated same-session.

### Technical

1. **Fix per-module isolated builds** — `GOWORK=off` should work for every module. Run `go mod tidy` in each.
2. **eventcatalog coverage recovery** — Target >90% (was 91.3% before this session).
3. **Catalog Builder ergonomics** — `cattest/builders.go` should use branded types for consistency.
4. **Registry deterministic Build()** — Non-deterministic map iteration can cause test flakiness (noted in TODO_LIST).
5. **Pebble concurrency** — Known data-loss bug in Save(). Not production-safe.
6. **Outbox transactionality** — Two separate transactions for store+outbox means potential inconsistency.

### Architecture

1. **Remove DeriveAggregateID** — Zero callers, YAGNI, adds confusion.
2. **Unexport AggregateMarker** — Decorative embedding in example/todo provides no real interop.
3. **Query handler generics** — The `any` return type is the #1 type safety gap in the library.
4. **Owners []string polymorphism** — TeamID or UserID; consider `OwnerRef` union type or `[]OwnerID` with `OwnerKind()` discriminator.

---

## F) Top #25 Things We Should Get Done Next

| #   | Priority    | Task                                                                  | Impact                                      | Effort   |
| --- | ----------- | --------------------------------------------------------------------- | ------------------------------------------- | -------- |
| 1   | 🔴 CRITICAL | **Push 12 commits to origin** (`git push`)                            | Prevents data loss                          | 1 min    |
| 2   | 🔴 HIGH     | Fix per-module isolated builds (`go mod tidy` each module)            | Consumers can't `go get` individual modules | 30 min   |
| 3   | 🔴 HIGH     | Recover eventcatalog coverage to >90%                                 | Quality gate                                | 1-2 hrs  |
| 4   | 🟡 MEDIUM   | Remove `DeriveAggregateID` (pending user approval)                    | YAGNI cleanup                               | 15 min   |
| 5   | 🟡 MEDIUM   | Unexport `AggregateMarker`, update example/todo                       | API surface reduction                       | 15 min   |
| 6   | 🟡 MEDIUM   | Update `cattest/builders.go` to accept branded types                  | API consistency                             | 30 min   |
| 7   | 🟡 MEDIUM   | Update AGENTS.md with current coverage numbers                        | Documentation freshness                     | 30 min   |
| 8   | 🟡 MEDIUM   | Triage TODO_LIST.md — prune stale, deduplicate, prioritize top 20     | Noise reduction                             | 1 hr     |
| 9   | 🟡 MEDIUM   | Fix Pebble Store optimistic concurrency in Save                       | Data safety                                 | 1-2 hrs  |
| 10  | 🟡 MEDIUM   | Fix Outbox transaction co-participation                               | Data consistency                            | 2-3 hrs  |
| 11  | 🟡 MEDIUM   | Fix `collectResults` goroutine leak in projection/runner.go           | Resource leak                               | 1 hr     |
| 12  | 🟡 MEDIUM   | Fix OutboxPublisher split-brain (cancel non-nil after Close)          | Correctness                                 | 30 min   |
| 13  | 🟡 MEDIUM   | Fix storage/dialect.go `any` usage (3 methods)                        | Type safety                                 | 30 min   |
| 14  | 🟡 MEDIUM   | Fix asyncapi exporter missing CommandMessage case                     | Feature completeness                        | 30 min   |
| 15  | 🟢 LOW      | Registry deterministic Build() (sort map iteration)                   | Test reliability                            | 30 min   |
| 16  | 🟢 LOW      | Fix FuzzParse case-sensitivity roundtrip                              | Edge case correctness                       | 1 hr     |
| 17  | 🟢 LOW      | Add slog.Warn for corrupt IDs in Pebble deserialization               | Observability                               | 15 min   |
| 18  | 🟢 LOW      | Update FEATURES.md — add openapi, docserver, dialect, recent sessions | Documentation                               | 1 hr     |
| 19  | 🟢 LOW      | Design OwnerRef union type for Owners []string                        | Type safety                                 | 1 hr     |
| 20  | 🟢 LOW      | Fix 2 lint issues (noinlineerr in core/command, core/query)           | Zero lint                                   | 15 min   |
| 21  | 🟢 LOW      | Add catalog diff/breaking-change detection tool                       | API evolution safety                        | 3-4 hrs  |
| 22  | 🟢 LOW      | Query handler generics (TypedHandler[T] returning T, error)           | Type safety (breaking change)               | 4-8 hrs  |
| 23  | 🟢 LOW      | High-level test utilities (AggregateTester, ProjectionTester)         | Consumer DX                                 | 4-8 hrs  |
| 24  | 📐 PLANNED  | Saga/orchestration pattern implementation                             | Feature expansion                           | 1-2 days |
| 25  | 📐 PLANNED  | Publish go-composable-business-types as Go module                     | External adoption                           | 1-2 days |

---

## G) Top #1 Question I Cannot Figure Out Myself 🤔

**Should `DeriveAggregateID` and `AggregateMarker` be removed?**

The design review at `docs/planning/2026-05-25_AGGREGATE_ID_DESIGN_REVIEW.md` recommends:

- **Remove `DeriveAggregateID`** — Zero callers, SHA-256 hash-based ID derivation that nobody uses
- **Unexport `AggregateMarker`** — Only used by `example/todo` for decorative embedding with zero actual type interop

But both are technically **breaking changes** for anyone who imported them. The question is:

> Do you want to take these breaking changes now (before any external consumers exist), or defer to a v2 milestone?

My recommendation: **Remove both NOW.** Zero external consumers + the API is cleaner. The longer they stay, the harder they are to remove.

---

## Coverage Summary

| Package                       | Coverage  | Change            |
| ----------------------------- | --------- | ----------------- |
| `core/pkg/dispatcher`         | 100.0%    | —                 |
| `core/pkg/id`                 | 100.0%    | —                 |
| `middleware`                  | 100.0%    | —                 |
| `catalog/internal/caseutil`   | 100.0%    | —                 |
| `memory`                      | 99.6%     | —                 |
| `core/query`                  | 98.4%     | —                 |
| `catalog`                     | 96.3%     | ↓ (was 96.8%)     |
| `catalog/d2`                  | 95.0%     | —                 |
| `catalog/openapi`             | 94.4%     | —                 |
| `projection`                  | 94.4%     | —                 |
| `core/event`                  | 93.8%     | —                 |
| `catalog/asyncapi`            | 93.7%     | —                 |
| `core/decider`                | 93.6%     | —                 |
| `core/command`                | 92.3%     | —                 |
| `testhelpers`                 | 91.3%     | —                 |
| `catalog/docserver`           | 90.1%     | —                 |
| `storage`                     | 89.3%     | —                 |
| `catalog/internal/schemautil` | 84.2%     | —                 |
| **catalog/eventcatalog**      | **85.8%** | **↓ (was 91.3%)** |

**Overall: ~90.2%** (weighted average across all packages)

---

## Module Size

| Module      | Files   | Lines      |
| ----------- | ------- | ---------- |
| core        | 48      | 11,922     |
| catalog     | 39      | 11,784     |
| storage     | 23      | 7,345      |
| memory      | 9       | 2,611      |
| projection  | 6       | 2,459      |
| middleware  | 8       | 2,200      |
| testhelpers | 9       | 1,997      |
| integration | 0       | 1,758      |
| **Total**   | **307** | **46,474** |

---

_Report generated: 2026-05-25 22:46_

## Appendix A: Design Decision Corrections

### AggregateMarker — KEEP EXPORTED

**Original recommendation (section G):** Unexport `AggregateMarker`.

**Corrected recommendation:** **Keep `AggregateMarker` exported.**

**Reason:** This is an SDK. We have zero visibility into who imports our public types. Unexporting `AggregateMarker` is a breaking change for any consumer who references it — and we cannot claim "zero external consumers" because we are a library, not an application. The type exists for a valid reason: it's the phantom type for the `AggregateID` brand, and consumers should be able to reference it for domain-specific ID interop.

The `example/todo` embedding (`TodoMarker` embeds `AggregateMarker`) is misleading because `TodoID = id.Of[TodoMarker]` uses ULID backing while `AggregateID = cbid.ID[AggregateMarker, string]` uses string backing — they don't actually interoperate. That's the example's bug to fix, not a reason to unexport.

### DeriveAggregateID — DEPRECATE, DON'T REMOVE

**Original recommendation (section G):** Remove `DeriveAggregateID`.

**Corrected recommendation:** **Deprecate with `// Deprecated:` comment.** Keep the function, mark it deprecated, remove in a future major version. This follows semver-compatible deprecation practice for SDKs — we cannot know if an external consumer relies on it.

### SDK Awareness Principle

> **This is a library/SDK, not an application.** We cannot make claims about external consumer count. Every exported type and function is a semver contract. Breaking changes require a major version bump, period. The only safe "removal" is deprecation → major version.
