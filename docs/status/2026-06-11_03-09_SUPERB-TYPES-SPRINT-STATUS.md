# Superb Types Sprint — Comprehensive Status Report

**Date:** 2026-06-11 03:09  
**Session scope:** Phantom types, strong IDs, error context loss, anti-pattern renames, duplicate consolidation  
**Source analysis:** `branching-flow all . --no-emoji`  
**Plan document:** `docs/planning/2026-06-11_01-24_SUPERB-TYPES-PHANTOM-IDS-DATA-MODEL.md` (84 tasks)

---

## Executive Summary

**6 commits, 28 files changed, 124 insertions, 109 deletions.** All 41 test packages pass. Error context quality: 95.0/100.

The catalog phantom types (Phase 1) were completed in the previous session. This session focused on Phase 3 (error context, anti-patterns, consolidation) and identified/fixed several items missed in the prior pass.

---

## a) FULLY DONE ✓

### Phase 1: Catalog Phantom Types (Previous Session)

- ✅ Created `catalog/types_phantom.go` with 18 phantom types: `Name`, `Version`, `Summary`, `Title`, `Description`, `Address`, `Protocol`, `Host`, `Email`, `URL`, `ContentType`, `DeliveryGuarantee`, `Method`, `Icon`, `Color`, `Language`, `Role`, `Summary`
- ✅ Applied phantom types to `Message`, `Service`, `Domain`, `Channel`, `Catalog` structs in `catalog/types.go`
- ✅ Applied to helpers: `Change`, `Badge`, `Repository`, `Operation`, `Specification`, `Attachment`, `Ref`, `ChannelParam`
- ✅ Applied to resources: `DataStore`, `Flow`, `FlowStep`, `FlowActor`, `FlowCustomNode`, `FlowEdge`, `Team`, `User`
- ✅ Renamed option functions `Name()`→`WithName()`, `Summary()`→`WithSummary()`, `Version()`→`WithVersion()` to avoid collision
- ✅ Propagated through all builders, registry, exporters (asyncapi, openapi, d2, eventcatalog, docserver)
- ✅ All catalog tests pass

### Phase 3A: Error Context Loss (13 → 2 remaining, both false positives)

- ✅ `memory/checkpoint.go:35,52` — Added `projectionName` to Load/Save closed errors
- ✅ `pebble/journal.go` — Added `limit`, `afterEventID` to corrupt/iterator errors
- ✅ `storage/event_store_global.go` — Added `limit`, `afterEventID` to checkClosed error
- ✅ `storage/sql/query_engine.go:48,61,92,99` — Added `aggType`/`aggID` to ALL query/scan error sites
- ✅ `middleware/logging.go:42` — Wrapped handler errors with prefix + message type
- ✅ `integration/simulation/generator.go:65` — Enriched with aggregates/eventsPerAggregate counts
- ✅ `watermill/protocol.go:115` — Added `topic` context to parseOptionalFields error path

### Phase 3B: Panic Guards + Anti-Pattern Renames

- ✅ `pkg/gracefulshutdown/shutdown.go` — Added `select/default` guards on both `errCh` sends
- ✅ `storage/sql.Base` → `DBHandle` (behavior-focused name)
- ✅ `storage/sql.ClosableBase` → `OwnedDBHandle` (behavior-focused name)
- ✅ `example/todo/storage.PebbleBase` → `PebbleHandle` (behavior-focused name)
- ✅ All callers and tests updated for renames

### Phase 2C: Duplicate Type Consolidation

- ✅ `asyncapi.Info` ↔ `openapi.Info` → `catalog.DocumentInfo` with shared JSON+YAML tags

### Encryption: KeyID Phantom Type

- ✅ `encryption.KeyID` phantom type (`type KeyID string` with `String()`/`IsZero()`)
- ✅ `encryption.WithKeyID(KeyID)` — type-safe option function
- ✅ `encryption.WithMiddlewareKeyID(KeyID)` — type-safe middleware option
- ✅ `encryption.ExtractKeyID()` returns `KeyID` (was `string`)
- ✅ All test files updated with `KeyID("...")` casts

### Golden Test Maintenance

- ✅ `codec/testdata/golden/json_encode.json` refreshed

---

## b) PARTIALLY DONE

### Phantom Types in Exporter/Internal Types (catalog)

- ⚠️ `catalog/asyncapi/types.go` — 16 violations remaining (Server.Host, Server.Protocol, Channel.Address, Message.Name, Message.ContentType, etc.)
- ⚠️ `catalog/openapi/types.go` — 11 violations remaining (Tag.Name, Parameter.Name, Operation.OperationID, etc.)
- ⚠️ `catalog/docserver/` — 8 violations
- ⚠️ `catalog/eventcatalog/` — 5 violations in writer_frontmatter.go
- ⚠️ `catalog/internal/cattest/builders.go` — 18 violations (highest single file)
- ⚠️ `catalog/types_resources.go` — 9 violations (Email, URL fields on Team/User)

### Strong IDs (21 violations remaining)

- ⚠️ `example/saga-pattern/main.go` — 7 `OrderID` fields (all `string`, should be branded)
- ⚠️ `middleware/healthcheck.go` — `ReleaseID`, `ComponentID` as `string`
- ⚠️ `middleware/sse.go` — `AddClient(id string)`, `RemoveClient(id string)`
- ⚠️ Catalog display IDs (`ServiceDisplayID`, `EventDisplayID`) — 4 violations

---

## c) NOT STARTED

### Phase 2A: Struct Splits

- Message has 17 fields → could extract `MessageMeta` (Owners, Labels, Badges, Changelog, Repository, Deprecated)
- Service has 16 fields → could extract `ServiceMeta` (Badges, Repository, Specifications, Attachments)
- These are structural improvements that don't change API surface (embedded for backward compat)

### Phase 3C: Remaining Phantom Types (Low ROI)

- `otel/attributes.go` — 5 violations (component constants — OTel semantic conventions, not our types)
- `storage/sql/` — 15 violations across query_engine, helpers, reconstruction (internal plumbing)
- `watermill/` — 4 `topic` violations
- `turso/` — 3 `dbPath` violations
- `pebble/` — internal `dbPath`, `prefix` params
- `event/reconstruct.go` — 5 violations

### Example Module Phantom Types (~68 violations)

- `example/saga-pattern/main.go` — 8 violations
- `example/todo/` — 19 violations across domain, commands, aggregate
- `example/user/`, `example/projection/`, `example/storage/`, `example/catalog-server/` — various

### Bool → Enum Conversions

- `Deprecated` (bool) → `DeprecatedStatus` enum in catalog Message
- `Required`, `Nullable` in schema types
- `Completed` in todo example

### Mixin Extraction (19 opportunities)

- All deferred — Go embedding works fine, low ROI

---

## d) TOTALLY FUCKED UP ⚠️

### LSP Stale Diagnostics

- The LSP shows 108+ stale errors across catalog files that **actually compile and test fine**. This has been consistent throughout all sessions. The gopls cache is poisoned by the phantom type changes and doesn't recover. Only `go build`/`go test` are reliable.

### branching-flow False Positives

- `gracefulshutdown/shutdown.go` still flagged for panic (lines 61, 70) **despite having `select/default` guards**. The linter doesn't understand the pattern.
- `middleware/recovery.go:34` flagged for missing `msgKind`/`typeName` in error — but `panicError()` already includes both. Linter doesn't trace through helper functions.
- `memory/store_load.go:35` flagged for missing `op` — but `getEvents()` already wraps with `op`. Same issue.

### Nothing Is Actually Broken

- All 41 test packages pass
- Clean build with `go build ./...`
- No race conditions, no dead code

---

## e) WHAT WE SHOULD IMPROVE

### Architecture & Type Safety

1. **asyncapi/openapi types still use raw `string`** — These mirror external spec formats (AsyncAPI 3.0, OpenAPI 3.0) and need JSON/YAML tags, making phantom types awkward. But catalog already has `Title`, `Version`, `Summary` phantom types — applying them would cascade through ~30 violations.

2. **`cattest/builders.go` is the worst offender** (18 violations) — Test helpers pass raw strings for service names, message names, IDs that should use catalog phantom types. This is the "test code quality" gap.

3. **Encryption `KeyID` is a phantom type, not a branded ULID** — For a security-critical type, `id.Of[keyMarker]` (branded ULID) would be stronger. But `KeyID` is often a human-readable string like "key-v1", not a ULID, so phantom is correct.

### Process & Quality

4. **LSP cache poisoning** — Every major phantom type change leaves the LSP reporting hundreds of false errors for hours. Need to restart gopls or accept `go build` as ground truth.

5. **Self-review before committing** — In the prior session, I missed watermill error context, the second query engine error site, and the Info duplicate. A systematic re-read of ALL linter output before declaring "done" would have caught these.

6. **Encryption go.mod has stale signing dependency** — The LSP reports `signing/v2` not in go.mod for `encryption/middleware.go`. This was from a previous session's refactor. May need `go mod tidy`.

### Documentation

7. **Plan document is stale** — `docs/planning/2026-06-11_01-24_SUPERB-TYPES-PHANTOM-IDS-DATA-MODEL.md` still shows all 84 tasks as TODO. Should be updated with completion status.

---

## f) Top 25 Things to Do Next (Sorted by Impact/Effort)

| #   | Task                                                                             | Impact | Effort | File(s)                                                            |
| --- | -------------------------------------------------------------------------------- | ------ | ------ | ------------------------------------------------------------------ |
| 1   | Apply phantom types to `cattest/builders.go` (18 violations)                     | HIGH   | 15min  | `catalog/internal/cattest/builders.go`                             |
| 2   | Apply phantom types to `asyncapi/types.go` (16 violations)                       | HIGH   | 15min  | `catalog/asyncapi/types.go`                                        |
| 3   | Apply phantom types to `openapi/types.go` (11 violations)                        | MEDIUM | 10min  | `catalog/openapi/types.go`                                         |
| 4   | Apply phantom types to `asyncapi/exporter.go` (7 violations)                     | MEDIUM | 10min  | `catalog/asyncapi/exporter.go`                                     |
| 5   | Apply phantom types to `asyncapi/builder.go` (5 violations)                      | MEDIUM | 10min  | `catalog/asyncapi/builder.go`                                      |
| 6   | Apply phantom types to `docserver/` (8 violations)                               | MEDIUM | 10min  | `catalog/docserver/*.go`                                           |
| 7   | Apply phantom types to `eventcatalog/writer_frontmatter.go` (5 violations)       | MEDIUM | 8min   | `catalog/eventcatalog/writer_frontmatter.go`                       |
| 8   | Apply phantom types to `d2/exporter.go` (6 violations)                           | MEDIUM | 8min   | `catalog/d2/exporter.go`                                           |
| 9   | Apply phantom types to `catalog/registry.go` (5 violations)                      | MEDIUM | 8min   | `catalog/registry.go`                                              |
| 10  | Apply phantom types to `catalog/message_config.go` (5 violations)                | MEDIUM | 8min   | `catalog/message_config.go`                                        |
| 11  | Apply phantom types to `catalog/types_resources.go` (9 violations — Email, URL)  | MEDIUM | 10min  | `catalog/types_resources.go`                                       |
| 12  | Apply phantom types to `catalog/docserver/html.go` (5 violations)                | LOW    | 8min   | `catalog/docserver/html.go`                                        |
| 13  | Apply phantom types to `catalog/openapi/exporter.go` (6 violations)              | LOW    | 8min   | `catalog/openapi/exporter.go`                                      |
| 14  | Add branded `OrderID` to `example/saga-pattern` (7 violations)                   | MEDIUM | 10min  | `example/saga-pattern/main.go`                                     |
| 15  | Add branded `ReleaseID`/`ComponentID` to healthcheck (2 violations)              | MEDIUM | 10min  | `middleware/healthcheck.go`                                        |
| 16  | Run `go mod tidy` in encryption module (stale signing dep)                       | LOW    | 2min   | `encryption/go.mod`                                                |
| 17  | Split `catalog.Message` → `Message` + `MessageMeta` (17 fields)                  | HIGH   | 15min  | `catalog/types.go`                                                 |
| 18  | Split `catalog.Service` → `Service` + `ServiceMeta` (16 fields)                  | HIGH   | 15min  | `catalog/types.go`                                                 |
| 19  | Update plan doc with completion status                                           | LOW    | 5min   | `docs/planning/2026-06-11_...`                                     |
| 20  | Add `bool` → enum for `Deprecated` in catalog Message                            | LOW    | 5min   | `catalog/types.go`                                                 |
| 21  | Add phantom types to `storage/sql/helpers.go` (7 violations)                     | LOW    | 10min  | `storage/sql/helpers.go`                                           |
| 22  | Add phantom types to `event/reconstruct.go` (5 violations)                       | LOW    | 8min   | `event/reconstruct.go`                                             |
| 23  | Add `Topic` phantom type to watermill (4 violations)                             | LOW    | 8min   | `watermill/protocol.go`, `publisher.go`, `subscriber.go`           |
| 24  | Add `DbPath` phantom type to turso/storage (5 violations)                        | LOW    | 8min   | `turso/connector.go`, `turso/sync.go`, `storage/sqlite_helpers.go` |
| 25  | Run full `branching-flow all .` and verify violation drop from 389 → target <200 | HIGH   | 5min   | all                                                                |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `asyncapi/types.go` and `openapi/types.go` use catalog phantom types (`Title`, `Version`, `Summary`) or remain as raw `string`?**

These types are serialization structs that mirror external spec formats (AsyncAPI 3.0, OpenAPI 3.0). They need exact JSON/YAML struct tags matching the spec. Using phantom types would:

- Add type safety (can't pass a `Host` where `Protocol` is expected)
- But require `string()` casts at every serialization boundary
- And might confuse consumers who expect plain `string` fields

The catalog domain types (`Message`, `Service`) already use phantom types. The question is whether the **exporter-specific serialization types** should too. This is a judgment call that depends on whether you value type safety or simplicity more in serialization-layer code.

---

## Linter Scores Summary

| Linter                | Before            | After             | Delta                                        |
| --------------------- | ----------------- | ----------------- | -------------------------------------------- |
| Phantom violations    | 315               | 273               | -42 (-13%)                                   |
| Strong-ID violations  | 25                | 21                | -4 (KeyID added)                             |
| Panic detections      | 5                 | 2                 | -3 (select guards; 2 false positives remain) |
| Error context quality | ~90/100           | 95.0/100          | +5 pts                                       |
| Duplicate type groups | 16 → 6 actionable | 15 → 5 actionable | -1 group (Info consolidated)                 |

## Test Results

**41 packages pass. 0 failures.** Full suite covers: event, command, query, decider, id, dispatcher, schema, snapshot, memory, catalog (9 sub-packages), middleware, integration (7 sub-packages), projection, signing (2), storage (2), pebble, codec, listing, otel, cmd/cqrs-gen, pkg (2), encryption, watermill, turso.

## Commits This Session

| Hash       | Message                                                                                  |
| ---------- | ---------------------------------------------------------------------------------------- |
| `2e4274f1` | `refactor: error context enrichment, anti-pattern renames across storage/middleware`     |
| `7e0d72cd` | `fix(pkg/gracefulshutdown): add select guards on errCh sends to prevent panic`           |
| `0bd1e64f` | `fix(watermill): add topic context to parseOptionalFields error path`                    |
| `fc6c8a73` | `fix(storage/sql): add aggType and aggID to query/scan error messages`                   |
| `4756c7ec` | `refactor(encryption): add KeyID phantom type for type-safe key identification`          |
| `1bc86821` | `refactor(catalog): consolidate asyncapi.Info and openapi.Info into shared DocumentInfo` |
