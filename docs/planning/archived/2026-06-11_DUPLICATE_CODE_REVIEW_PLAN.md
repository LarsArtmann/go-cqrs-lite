# Duplicate Code Review — 2026-06-11

**Source:** `branching-flow dupe . --format markdown`
**Result:** 15 groups, 5 actionable, 10 false positives.

## Finding: Group 13 Already Refactored

Group 13 (`AggregateProjection` / `SQLAggregateReader`) **was already extracted**
in commit `63fe9885` (HEAD):

- `storage/listing_table.go` — centralizes `validateListingTablePrefix` +
  table-name derivation in one constructor `newListingTable(prefix)`.
- `storage/aggregate_projection.go` — now stores `table listingTable`; no
  inline table name or validation.
- `storage/sql_aggregate_reader.go` — same.

The remaining structural similarity flagged by the tool (3 fields:
`db *sql.DB` + `dialect sqlpkg.Dialect` + `table listingTable`) is
**coincidental** — the two types implement different interfaces
(`event.Projection` vs `listing.AggregateReader`) and must each carry
their own DB and dialect handle. Further extraction would harm
readability for no benefit.

**Verdict: ACCEPT (remaining similarity is structural necessity).**

## Pareto-Sorted Decision Table

| Rank | Group                                            | Impact | Effort | Customer Value | Decision   | Rationale                                                                                                                                |
| ---- | ------------------------------------------------ | ------ | ------ | -------------- | ---------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| 1    | 13 (storage)                                     | LOW    | DONE   | LOW            | **ACCEPT** | Already refactored in commit 63fe9885. Remaining field overlap is structural necessity.                                                  |
| 2    | 14 (projection: `Builder` / `builtProjection`)   | LOW    | HIGH   | LOW            | **ACCEPT** | Builder pattern: builder is mutable config, builtProjection is immutable `event.Projection`. Different lifecycles, different invariants. |
| 3    | 2 (example: `CreateUserPayload` etc.)            | NONE   | N/A    | NONE           | **ACCEPT** | Example apps intentionally use distinct payload shapes to demo variation.                                                                |
| 4    | 9 (example: `ItemAdded` / `ItemRemoved`)         | NONE   | N/A    | NONE           | **ACCEPT** | Two semantically opposite event types in projection demo.                                                                                |
| 5    | 12 (example: `CreateUserCmd` / `RebirthUserCmd`) | NONE   | N/A    | NONE           | **ACCEPT** | Different commands for different use cases.                                                                                              |

## Accepted Duplications — One-Line Rationale

| Group | Members                                                                                         | Reason for acceptance                                                                                                                                                                                    |
| ----- | ----------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1     | Empty marker/brand types (`AggregateMarker`, `JSONCodec`, `*Marker`, etc.)                      | Distinct phantom types used for type branding. Shape similarity (all zero-size) is a Go language feature, not duplication.                                                                               |
| 2     | `CreateUserPayload`, `UserCreated`, `UserCreatedPayload`, `UserRebornPayload`, `ReadModel`      | Different event/command payload types across distinct example apps. Demo variation, not drift risk.                                                                                                      |
| 3     | `InventoryReleased`, `InventoryReserved`, `OrderConfirmed`, `PaymentCharged`, `PaymentRefunded` | Distinct domain events with intentionally similar scalar payloads — different aggregate contexts.                                                                                                        |
| 4     | `ChangeStatusHandler`, `CreateTodoHandler`, `DeleteTodoHandler`, `UpdateTodoHandler`            | Per-action handler types in a todo example. Local to one demo.                                                                                                                                           |
| 5     | `ChangeUserNamePayload`, `Tag`, `UserNameChangedPayload`                                        | Different payload types in different examples / different fields when inspected carefully.                                                                                                               |
| 6     | `CountTodosHandler`, `GetTodoHandler`, `ListTodosHandler`                                       | Per-query handler types in todo example. Local to one demo.                                                                                                                                              |
| 7     | `SQLCommandStore`, `SQLEventStore`                                                              | Different stores with intentionally different fields (commands have different schema than events).                                                                                                       |
| 8     | `SQLCheckpointStore`, `SQLSnapshotStore`                                                        | Different stores; different schemas.                                                                                                                                                                     |
| 9     | `ItemAdded`, `ItemRemoved`                                                                      | Two semantically opposite event types in a projection demo.                                                                                                                                              |
| 10    | `Ref`, `SchemaRef`                                                                              | Different references in different packages (catalog vs catalog/schema).                                                                                                                                  |
| 11    | `aes256gcm`, `xchacha20`                                                                        | Different algorithm parameter types in encryption module — different fields, different roles.                                                                                                            |
| 12    | `CreateUserCmd`, `RebirthUserCmd`                                                               | Two different commands for different use cases.                                                                                                                                                          |
| 13    | `AggregateProjection` / `SQLAggregateReader`                                                    | Already refactored (commit 63fe9885). Remaining field overlap is structural necessity.                                                                                                                   |
| 14    | `Builder` / `builtProjection` (projection/builder.go)                                           | Builder pattern: builder is mutable config that grows; builtProjection is the immutable `event.Projection` returned by `Build()`. Different lifecycles, different contracts. Sharing fields is required. |
| 15    | `Dispatcher` (multiple modules)                                                                 | Different dispatcher interfaces per module type (command, query, event). Coincidental name, distinct contracts.                                                                                          |

## Verification

- Storage tests pass: `cd storage && GOWORK=off go test ./... -count=1` → `ok`
- Lint: 0 new issues introduced by this review. Pre-existing issues in
  `storage/sql/coverage_test.go` (9 noctx) and
  `catalog/internal/cattest/builders.go` (1 unconvert) are unrelated to
  this review and were not authored by this session.
