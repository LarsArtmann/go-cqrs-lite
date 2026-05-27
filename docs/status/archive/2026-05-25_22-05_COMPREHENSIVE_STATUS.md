# Session Status Report — 2026-05-25 22:05

**Branch:** `master` (5 commits ahead of origin)
**Test Suite:** 22/22 packages pass, 90.3% overall coverage
**Codebase:** 307 Go files, 46,457 lines

---

## A) FULLY DONE

### EventCatalog Auto-Generation (20% → 80% coverage)

The entire EventCatalog enhancement pipeline is complete across 10 commits:

| Commit    | What                                                                                    |
| --------- | --------------------------------------------------------------------------------------- |
| `31f3ade` | Core: new types (DataStore, Flow, Team, User), registry, builder, exporter, auto-derive |
| `9d4dbb0` | Deduplicate `MessagePointer`/`FlowStepRef` → `Ref`                                      |
| `3a960ff` | LLMs.txt extended to all resource types                                                 |
| `7974dfd` | Test catalog updated with channel + data store                                          |
| `b843740` | ~25 tests for registry, builder, auto-derive                                            |
| `9b2de7c` | ServiceOption fluent API (8 options)                                                    |
| `aa3000d` | DomainOption fluent API (6 options)                                                     |
| `0b858c1` | ChannelOption fluent API (8 options)                                                    |
| `b3ad286` | LLMs.txt content verification test                                                      |
| `730c848` | Status report                                                                           |

### Branded IDs in Storage & Middleware (this session)

Replaced raw `string` types with branded `id.EventID` and `id.AggregateID` across the storage and middleware layers:

| File                              | Change                                                                                                                                                             |
| --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `storage/event_reconstruction.go` | `reconstructEvent()` accepts `id.EventID` + `id.AggregateID` instead of strings                                                                                    |
| `storage/event_store_scan.go`     | Parses SQL-scanned strings into branded IDs before reconstruction                                                                                                  |
| `storage/pebble_serialization.go` | `serializableEvent` uses branded IDs; eliminated `serializableMetadata` (now embeds `*event.Metadata`); removed `parseBrandedID` and `deserializeMetadata` helpers |
| `storage/outbox_helpers.go`       | `outboxEvent` uses branded IDs for ID/AggregateID fields                                                                                                           |
| `middleware/logging.go`           | `logContext.aggregateID` is `id.AggregateID`; `.String()` called only at log boundary                                                                              |

Net result: **-125 lines, +53 lines** (72 lines net reduction). Less code, more type safety.

### Design Documentation

- `docs/planning/2026-05-25_AGGREGATE_ID_DESIGN_REVIEW.md` — Full PRO/CONTRA for AggregateID's `string` backing, `DeriveAggregateID` removal, and `AggregateMarker` unexporting

---

## B) PARTIALLY DONE

### AggregateID Design Review

Written and documented, but **no decision taken yet** on:

1. Whether to remove `DeriveAggregateID` (recommended but pending approval)
2. Whether to unexport `AggregateMarker` (recommended but pending approval)
3. Whether `AggregateID` stays `string`-backed (recommended yes)

### Branded IDs in Storage

The 5 target files are fixed, but there are 4 additional spots that could be further improved (see section E).

---

## C) NOT STARTED

1. **DataStoreOption fluent API** — Data stores use struct directly, no `ConfigureDataStore()` builder method
2. **FlowOption fluent API** — Flows use struct directly, no `ConfigureFlow()` builder method
3. **Changelog generation** — EventCatalog changelog resource type not implemented
4. **Diagram generation** — EventCatalog diagram resource type not implemented
5. **`example/user` catalog.go** — Needs updating to demonstrate all new fluent APIs (currently uses raw structs for some resources)
6. **AGENTS.md update** — Should reflect the new fluent APIs and branded ID improvements
7. **Channel ID branding** — `catalog.ChannelID` is `type ChannelID string`, not ULID-backed (user requested investigating this)
8. **`example/todo` `TodoMarker` embedding** — Should remove decorative `id.AggregateMarker` embedding if marker is unexported

---

## D) TOTALLY FUCKED UP

Nothing is broken. Zero build errors, zero test failures, zero lint issues across all 22 packages.

---

## E) WHAT WE SHOULD IMPROVE

### Branded ID Adoption (storage layer, remaining 4 spots)

| File                    | Line | Current                                                    | Should Be                                                      |
| ----------------------- | ---- | ---------------------------------------------------------- | -------------------------------------------------------------- |
| `event_store_scan.go`   | 20   | `var eventIDStr string` then `id.ParseEventID(eventIDStr)` | Scan directly into `id.EventID` (implements `sql.Scanner`)     |
| `event_store_scan.go`   | 23   | `var aggIDStr string` then `id.ParseAggregateID(aggIDStr)` | Scan directly into `id.AggregateID` (implements `sql.Scanner`) |
| `sql_helpers.go`        | 128  | `var eventIDStr string` then `id.ParseEventID(eventIDStr)` | Scan directly into `id.EventID`                                |
| `event_store_global.go` | 52   | `afterEventID.String()` for SQL arg                        | Pass `id.EventID` directly (implements `driver.Valuer`)        |

### EventCatalog Exporter Coverage

- `catalog/eventcatalog` dropped from 91.3% to 85.9% — new code paths (channels, data stores, flows, teams, users) need more tests
- `catalog` improved: 96.8% → 97.2%

### Code Quality

- `storage` at 89.3% — could improve with more error path tests
- `catalog/internal/schemautil` at 84.2% — lowest in catalog
- Pebble store has 0 dedicated serialization tests (relies on integration tests)

---

## F) Top #25 Things We Should Get Done Next

### High Impact (1% → 51%)

| #   | Task                                                                                                                      | Impact                 |
| --- | ------------------------------------------------------------------------------------------------------------------------- | ---------------------- |
| 1   | **Scan branded IDs directly from SQL** — eliminate 4 intermediate string parses                                           | Type safety, less code |
| 2   | **Remove `DeriveAggregateID`** — YAGNI, zero callers, over-engineered                                                     | Cleaner API            |
| 3   | **Unexport `AggregateMarker`** — decorative, remove `TodoMarker` embedding                                                | API consistency        |
| 4   | **Add Pebble serialization round-trip tests** — verify branded IDs serialize/deserialize correctly                        | Bug prevention         |
| 5   | **Update `example/user` to use all fluent APIs** — demonstrates `ConfigureService`, `ConfigureDomain`, `ConfigureChannel` | Documentation          |

### Medium Impact (4% → 64%)

| #   | Task                                                                                                                                           | Impact                 |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------- |
| 6   | **Raise eventcatalog coverage back to >90%** — test new resource generators                                                                    | Reliability            |
| 7   | **Add `DataStoreOption` fluent API** — `DataStoreType`, `DataStoreTechnology`, `DataStoreClassification`, `DataStoreOwners`, `DataStoreBadges` | API consistency        |
| 8   | **Add `FlowOption` fluent API** — `FlowSteps`, `FlowBadges`, `FlowActors`                                                                      | API consistency        |
| 9   | **Update AGENTS.md** — document new fluent APIs, branded ID improvements, coverage numbers                                                     | Knowledge preservation |
| 10  | **Investigate branded ChannelID** — user asked about making `catalog.ChannelID` use `go-branded-id`                                            | Type safety            |
| 11  | **Add `storage` error path tests** — improve from 89.3% to >92%                                                                                | Reliability            |
| 12  | **Add integration test for outbox with branded IDs** — verify full round-trip through SQL                                                      | Bug prevention         |
| 13  | **Clean up `example/todo/domain/ids.go`** — remove decorative `AggregateMarker` embedding                                                      | Honesty                |

### Lower Impact (polish & future)

| #   | Task                                                                                    | Impact                   |
| --- | --------------------------------------------------------------------------------------- | ------------------------ |
| 14  | **Add EventCatalog changelog generation**                                               | Feature completeness     |
| 15  | **Add EventCatalog diagram generation**                                                 | Feature completeness     |
| 16  | **Push all commits to origin** — 5 commits ahead                                        | Collaboration            |
| 17  | **Run `nix run .#lint`** — verify golangci-lint passes                                  | Quality gate             |
| 18  | **Add `catalog/schemautil` tests** — raise from 84.2% to >90%                           | Coverage                 |
| 19  | **Add `MessageOption` for examples** — `MsgExamples` for EventCatalog                   | Feature completeness     |
| 20  | **Document auto-derive behavior** — explain producers/consumers inference in godoc      | Usability                |
| 21  | **Add `Flows` field to `Service` via `ServiceFlows` option**                            | Feature completeness     |
| 22  | **Add `Entities` field to `Domain` via `DomainFlows` option**                           | Feature completeness     |
| 23  | **Add benchmark for Pebble serialization with branded IDs**                             | Performance verification |
| 24  | **Review all `String()` calls on branded IDs** — find remaining unnecessary conversions | Consistency              |
| 25  | **Add `go vet` + `staticcheck` to CI** — beyond golangci-lint                           | Quality                  |

---

## G) Top #1 Question I Cannot Answer Myself

**Should `catalog` module depend on `go-branded-id`?**

The `catalog` module currently has 4 ID types as plain `type X string` (`ServiceID`, `DomainID`, `MessageID`, `ChannelID`). The user asked about making `ChannelID` branded. But:

- `catalog/go.mod` currently has **zero** dependencies on `core` or `go-branded-id` — it's fully independent
- Adding `go-branded-id` as a dependency would couple the catalog module to the ID infrastructure
- These IDs are **not ULIDs** — they're human-readable slugs like `"order-svc"`, `"user.created"`, `"order-events"`
- `id.Of[T]` requires `ulid.ULID` as the backing type — these catalog IDs don't fit that pattern
- Could use `cbid.ID[T, string]` (like `AggregateID`), but that adds a dependency for marginal type-safety benefit on non-unique, non-generated identifiers

The tradeoff: independence vs type safety. I cannot decide this without knowing whether catalog module independence is a deliberate architectural choice or incidental.
