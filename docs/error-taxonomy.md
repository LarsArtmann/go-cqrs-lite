# Error Taxonomy

> The go-cqrs-lite library uses [go-error-family](https://github.com/LarsArtmann/go-error-family) v0.10.0 for structured, classified error handling.

## The 6 Error Families

Every error produced by the library belongs to exactly one family. This enables consumers to make **programmable decisions** — retry, reject, escalate — without string matching or type assertions.

| Family             | Semantic                                    | Retryable?            | Example                                      |
| ------------------ | ------------------------------------------- | --------------------- | -------------------------------------------- |
| **Rejection**      | Business rule violation or invalid input    | No                    | `email is required`, `handler not found`     |
| **Conflict**       | Optimistic concurrency or state collision   | No (consumer decides) | `version conflict`, `user already exists`    |
| **Transient**      | Temporary failure likely to resolve         | Yes                   | `connection reset`, `timeout`                |
| **Infrastructure** | System-level failure requiring intervention | Maybe                 | `dispatcher closed`, `database unavailable`  |
| **Orchestration**  | Internal coordination failure (a bug)       | No                    | `projection worker setup`, `lifecycle error` |
| **Corruption**     | Data integrity violation                    | No                    | `type assertion failed`, `checksum mismatch` |

## Usage from Consumer Code

### Creating Classified Errors

```go
import errorfamily "github.com/larsartmann/go-error-family"

// Business rule violation — consumer's input is invalid
err := errorfamily.NewRejection("order.create.negative_total", "total must be positive")

// Optimistic concurrency — another write beat us
err := errorfamily.NewConflict("order.version_conflict", "order was modified by another request")

// Transient — network blip, safe to retry
err := errorfamily.NewTransient("order.publish.timeout", "failed to publish event within deadline")
```

### Classifying Errors

```go
family := errorfamily.Classify(err)
switch family {
case errorfamily.Rejection:
    // Return 400 to the client
case errorfamily.Conflict:
    // Return 409, maybe ask user to refresh
case errorfamily.Transient:
    // Retry with backoff
case errorfamily.Infrastructure:
    // Alert on-call, return 500
case errorfamily.Orchestration:
    // Internal bug — log diagnostic context, return 500
case errorfamily.Corruption:
    // Page on-call, investigate data integrity
}

// Or use the boolean helper
if errorfamily.IsRetryable(err) {
    // Exponential backoff and retry
}
```

### Wrapping Errors

```go
// Preserve classification through fmt.Errorf wrapping
err := fmt.Errorf("save order %s: %w", orderID, errorfamily.NewConflict("order.version_conflict", "version mismatch"))

// Classification still works
errorfamily.Classify(err) // => Conflict
```

## Error Families by Module

### core/event

| Error                     | Family    | Code                           |
| ------------------------- | --------- | ------------------------------ |
| `ErrEmptyEventType`       | Rejection | `event.empty_event_type`       |
| `ErrNilStreamID`          | Rejection | `event.nil_stream_id`         |
| `ErrEmptyStreamType`      | Rejection | `event.empty_stream_type`     |
| `ErrVersionNotPositive`   | Rejection | `event.version_not_positive`   |
| `ErrNilPayload`           | Rejection | `event.nil_payload`            |
| `ErrMismatchedEventCount` | Rejection | `event.mismatched_event_count` |
| `ErrVersionConflict`      | Conflict  | `event.version_conflict`       |
| `ErrStreamNotFound`       | Rejection | `event.stream_not_found`      |

### core/command

| Error                 | Family         | Code                         |
| --------------------- | -------------- | ---------------------------- |
| `ErrHandlerNotFound`  | Rejection      | `command.handler_not_found`  |
| `ErrDispatcherClosed` | Infrastructure | `command.dispatcher_closed`  |
| `ErrEmptyCommandType` | Rejection      | `command.empty_command_type` |
| `ErrNilStreamID`      | Rejection      | `command.nil_stream_id`     |
| `ErrTypeAssertion`    | Corruption     | `command.type_assertion`     |

### core/query

| Error                  | Family         | Code                      |
| ---------------------- | -------------- | ------------------------- |
| `ErrQueryNotSupported` | Rejection      | `query.not_supported`     |
| `ErrDispatcherClosed`  | Infrastructure | `query.dispatcher_closed` |
| `ErrEmptyQueryType`    | Rejection      | `query.empty_query_type`  |

### middleware

| Context            | Family         | Code                           |
| ------------------ | -------------- | ------------------------------ |
| Validation failure | Rejection      | `middleware.validation_failed` |
| Retry exhausted    | Infrastructure | `middleware.retry_exhausted`   |
| Panic recovery     | Corruption     | `middleware.panic_recovered`   |
| Meter required     | Rejection      | `middleware.meter_required`    |

### graph

All graph sentinels (schema validation, sink enforcement, read API) are **Rejection**.

| Error             | Family    | Code                            |
| ----------------- | --------- | ------------------------------- |
| `ErrPathNotFound` | Rejection | `graph.read.path_not_found`     |
| Schema violations | Rejection | `graph.schema.*` (12 sentinels) |
| Sink violations   | Rejection | `graph.sink.*` (5 sentinels)    |

### storage/relational

All relational schema and sink sentinels are **Rejection**.

| Error             | Family    | Code                                |
| ----------------- | --------- | ----------------------------------- |
| Schema violations | Rejection | `relational.schema.*` (8 sentinels) |
| Sink violations   | Rejection | `relational.sink.*` (4 sentinels)   |

### storage/view

All view-store mapper validation sentinels are **Rejection**.

| Error             | Family    | Code                                  |
| ----------------- | --------- | ------------------------------------- |
| Mapper violations | Rejection | `storage.view.mapper.*` (7 sentinels) |
| Nil view value    | Rejection | `storage.view.nil_value`              |

### stack

All bundle misconfiguration sentinels are **Rejection**.

| Error                  | Family    | Code                        |
| ---------------------- | --------- | --------------------------- |
| `ErrEmpty`             | Rejection | `stack.bundle_empty`        |
| `ErrMissingEventStore` | Rejection | `stack.missing_event_store` |
| `ErrMissingReadModels` | Rejection | `stack.missing_read_models` |
| `ErrMissingJournal`    | Rejection | `stack.missing_journal`     |

### projectionhost

| Error                  | Family         | Code                              |
| ---------------------- | -------------- | --------------------------------- |
| Constructor violations | Rejection      | `projectionhost.*` (6 sentinels)  |
| Shutdown timeout       | Infrastructure | `projectionhost.shutdown_timeout` |

### transport/grpc

| Error              | Family         | Code                      |
| ------------------ | -------------- | ------------------------- |
| Dispatch failure   | Infrastructure | `grpc.dispatch_failed`    |
| Query failure      | Infrastructure | `grpc.query_failed`       |
| Unmarshal result   | Corruption     | `grpc.unmarshal_result`   |
| Missing command ID | Rejection      | `grpc.missing_command_id` |

### deriver

| Error              | Family    | Code                     |
| ------------------ | --------- | ------------------------ |
| `ErrNilDispatcher` | Rejection | `deriver.nil_dispatcher` |

### storage (SQL facade)

| Error      | Family         | Code             |
| ---------- | -------------- | ---------------- |
| `ErrNilDB` | Infrastructure | `storage.nil_db` |

Operational wrap codes (not sentinels): row-scan/reconstruct failures are
**Corruption** (`storage.scan_*`, `storage.reconstruct_*`,
`storage.parse_stream_*`); DDL/execution failures are **Infrastructure**
(`storage.exec_ddl`, `storage.set_synchronous`); timer helpers are
**Infrastructure** (`storage.schedule_timer`, `storage.due_timers`).

### storage/pebble

| Error                   | Family    | Code                          |
| ----------------------- | --------- | ----------------------------- |
| `ErrNilDatabase`        | Rejection | `pebble.nil_database`         |
| `ErrStreamTypeMismatch` | Conflict  | `pebble.stream_type_mismatch` |
| `ErrStreamIDMismatch`   | Conflict  | `pebble.stream_id_mismatch`   |
| `ErrVersionMismatch`    | Conflict  | `pebble.version_mismatch`     |

The deprecated `ErrAggregateTypeMismatch`/`ErrAggregateIDMismatch` aliases
forward to the Stream sentinels (removed at v5). Operational wrap codes
split the same way: corruption detection is **Corruption**
(`pebble.corrupt_event`, `pebble.command_corrupt`, serialization failures);
iterator/batch/IO failures are **Infrastructure** (`pebble.commit_batch`,
`pebble.create_iterator`); concurrency checks are **Conflict**
(`pebble.concurrency_check`, `pebble.check_version`).

### watermill

| Error                       | Family         | Code                               |
| --------------------------- | -------------- | ---------------------------------- |
| `ErrMissingMetadata`        | Rejection      | `watermill.missing_metadata`       |
| Replay consumer nack        | Orchestration  | `watermill.catchup.replay_nacked`  |
| Metadata parse fails        | Rejection      | `watermill.parse_*`                |
| Malformed metadata payloads | Corruption     | `watermill.corrupt_metadata`, `watermill.create_event_failed`, `watermill.convert_message_failed` |
| Catch-up checkpoint/replay  | Infrastructure | `watermill.catchup.load_checkpoint`, `watermill.catchup.replay_read` |
| Bus publish fails           | Infrastructure | `watermill.event_bus_publish`, `watermill.command_bus_publish` |
| Subscribe/publish/lifecycle | Infrastructure | `watermill.subscribe_failed`, `watermill.publish_event_failed`, `watermill.publish_command_failed`, `watermill.catchup_subscriber_closed`, `watermill.topic_closed`, `watermill.topic_cancelled` |

Note the nack semantics: `watermill.catchup.replay_nacked` fires ONLY on a
real consumer Nack — a `Close()` or ctx cancellation during the ack wait
shuts the replay down silently instead of reporting a nack that never
happened.

## Default Classification

Errors that are not constructed via the taxonomy constructors (plain `errors.New`,
`fmt.Errorf` without `%w` into a family error) are classified as **Transient**
by default. This is a fail-open design: unknown infrastructure errors get
retried. However, it means **business-rule errors returned as plain errors will
be retried** — always use `errorfamily.NewRejection` for non-retryable errors.

## Design Principles

1. **Sentinel errors** in `errors.go` files — every module's errors are centralized
2. **Contextual wrapping** — `fmt.Errorf("operation %s: %w", name, err)` preserves classification
3. **No panics** — all errors are returned as values
4. **Consumer decides** — the library classifies, the consumer chooses the response strategy
5. **Codes are namespaced** — `module.subdomain.specific_error` format for uniqueness
