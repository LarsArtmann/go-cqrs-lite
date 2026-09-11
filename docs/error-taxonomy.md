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

> The `middleware`, `graph`, `storage/relational`, `projectionhost`, and
> `transport/grpc` sections are mechanically drift-gated: every code below is
> extracted from the module's `errorfamily.*` call sites by
> `scripts/check-error-taxonomy.sh` (CI + `nix run .#verify`) and diffed
> against this document in both directions — a code missing here, a stale
> entry, or a wrong family label fails the build. Extend the gate by adding
> modules to `GATED_MODULES` in the script.

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

Retry/backoff and circuit-breaker misconfiguration is **Rejection** (bad
config); runtime exhaustion is **Infrastructure**; a recovered panic is
**Corruption** (state after a panic is suspect). Note the two breaker codes:
`middleware.circuit_open` (Transient — the breaker half-opens and retries)
vs `middleware.circuit_breaker_open` (Infrastructure — breaker wrap).
Dead-letter-store rows are Transient for reads (may succeed next poll) and
Infrastructure for DDL/maintenance; malformed stored timestamps are
**Corruption**.

| Context                    | Family         | Code |
| -------------------------- | -------------- | ---- |
| Validation failure         | Rejection      | `middleware.validation_failed` |
| Meter required             | Rejection      | `middleware.meter_required` |
| Retry config invalid       | Rejection      | `middleware.invalid_initial_delay`, `middleware.invalid_max_attempts`, `middleware.invalid_multiplier` |
| Breaker config invalid     | Rejection      | `middleware.cb_invalid_failure_threshold`, `middleware.cb_invalid_success_threshold`, `middleware.cb_invalid_timeout` |
| Retry exhausted/canceled   | Infrastructure | `middleware.retry_exhausted`, `middleware.retry_canceled` |
| Breaker open (retryable)  | Transient      | `middleware.circuit_open` |
| Breaker open (wrap)       | Infrastructure | `middleware.circuit_breaker_open` |
| OTel recorder init         | Infrastructure | `middleware.otel_recorder_init` |
| Panic recovery             | Corruption     | `middleware.panic_recovered` |
| Dead-letter malformed data | Corruption     | `middleware.deadletter_sql.parse_time`, `middleware.deadletter_sql.unexpected_string_type`, `middleware.deadletter_sql.unexpected_time_type`, `middleware.deadletter.unexpected_time_type`, `deadletter.scan` |
| Dead-letter reads          | Transient      | `deadletter.count`, `deadletter.query`, `deadletter.rows_err` |
| Dead-letter DDL/maintenance | Infrastructure | `deadletter.clear`, `deadletter.create_table`, `deadletter.migrate` |

### graph

All graph validation sentinels (schema, sink, node/edge refs, read API,
query decode) are **Rejection**; only projection lifecycle failures are
**Infrastructure**.

| Error                    | Family         | Code |
| ------------------------ | -------------- | ---- |
| `ErrPathNotFound`        | Rejection      | `graph.read.path_not_found` |
| Schema violations        | Rejection      | `graph.schema.*` (16 sentinels) |
| Sink violations          | Rejection      | `graph.sink.*` (5 sentinels) |
| Node/Edge ref validation | Rejection      | `graph.noderef.*`, `graph.edgeref.*` |
| Query decode             | Rejection      | `graph.edge_from`, `graph.edge_to`, `graph.shortest_path_from`, `graph.shortest_path_to` |
| Projection constructor   | Rejection      | `graph.projection.driver_required`, `graph.projection.handler_required`, `graph.projection.name_required` |
| Projection close         | Infrastructure | `graph.projection.close` |

### storage/relational

Validation sentinels (schema, sink enforcement, nil-guard) are
**Rejection** — both the dotted (`relational.schema.*`) and legacy
underscore (`relational.schema_*`) spellings. Operational wraps split:
row-scan/reconstruction is **Corruption**, DDL/queries/writes are
**Transient** (the SQL tier treats per-statement failures as retryable —
the caller decides via `errorfamily.IsRetryable`).

| Error                     | Family    | Code |
| ------------------------- | --------- | ---- |
| Schema validation (dotted) | Rejection | `relational.schema.column_name_required`, `relational.schema.column_type_required`, `relational.schema.columns_required`, `relational.schema.duplicate_column`, `relational.schema.duplicate_table`, `relational.schema.index_no_name`, `relational.schema.no_tables`, `relational.schema.table_name_required`, `relational.schema.unique_no_name`, `relational.schema.unknown_index_column`, `relational.schema.unknown_pk_column`, `relational.schema.unknown_unique_column` |
| Schema validation (legacy) | Rejection | `relational.schema_column_no_name`, `relational.schema_column_no_type`, `relational.schema_duplicate_column`, `relational.schema_duplicate_table`, `relational.schema_index_no_name`, `relational.schema_table`, `relational.schema_unknown_index_col`, `relational.schema_unknown_pk`, `relational.schema_unique_no_name`, `relational.schema_unknown_unique_col` |
| Sink validation            | Rejection | `relational.sink.counter_in_key`, `relational.sink.empty_row`, `relational.sink.key_missing_pk`, `relational.sink.no_rows`, `relational.sink.unknown_column`, `relational.sink.unknown_table`, `relational.sink_counter_in_key`, `relational.sink_key_missing_pk`, `relational.sink_no_rows`, `relational.sink_unknown_column`, `relational.sink_unknown_table` |
| Nil guards / operator      | Rejection | `relational.nil_db`, `relational.nil_dialect`, `relational.nil_handler`, `relational.no_name`, `relational.conditions`, `relational.query_no_columns`, `relational.unknown_column`, `relational.unknown_table`, `relational.unsupported_operator` |
| Row scan / reconstruct     | Corruption | `relational.scan_row`, `relational.sink_query` |
| DDL / query / write wraps  | Transient | `relational.count`, `relational.migrate`, `relational.query`, `relational.rows_err`, `relational.projection_begin_tx`, `relational.projection_commit`, `relational.projection_reset`, `relational.sink_delete`, `relational.sink_ensure`, `relational.sink_increment`, `relational.sink_update`, `relational.sink_upsert`, `relational.sink_upsert_cols`, `relational.sink_upsert_expr` |

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

Constructor/config violations are **Rejection**; DLQ and reset operations
are **Infrastructure** (they wrap store failures); reconstructing a stored
dead letter is **Corruption**; staleness checks are **Transient** (may
pass on the next tick).

| Error                    | Family         | Code |
| ------------------------ | -------------- | ---- |
| Constructor violations   | Rejection      | `projectionhost.already_started`, `projectionhost.checkpoint_store_required`, `projectionhost.duplicate_name`, `projectionhost.empty_projection_name`, `projectionhost.journal_required`, `projectionhost.nil_db`, `projectionhost.no_dead_letter_store`, `projectionhost.register_after_start`, `projectionhost.reset_while_running`, `projectionhost.unknown_projection` |
| Shutdown timeout         | Infrastructure | `projectionhost.shutdown_timeout` |
| Checkpoint / worker ops  | Infrastructure | `projectionhost.reset_checkpoint`, `projectionhost.reset_projection`, `projectionhost.save_checkpoint_live`, `projectionhost.worker_failed` |
| DLQ operations           | Infrastructure | `projectionhost.dlq_count`, `projectionhost.dlq_delete`, `projectionhost.dlq_list`, `projectionhost.dlq_list_paged`, `projectionhost.dlq_purge`, `projectionhost.dlq_purge_before`, `projectionhost.dlq_scan`, `projectionhost.dlq_schema`, `projectionhost.dlq_store`, `projectionhost.list_dead_letters`, `projectionhost.reset_dlq_purge` |
| DLQ reconstruct          | Corruption     | `projectionhost.dlq_reconstruct` |
| Staleness check          | Transient      | `projectionhost.stale` |

### transport/grpc

Transport wraps are **Infrastructure**; decoding a corrupt wire payload is
**Corruption**; request validation is **Rejection**.

| Error                    | Family         | Code |
| ------------------------ | -------------- | ---- |
| Dispatch failure         | Infrastructure | `grpc.dispatch_failed`, `grpc.dispatch` |
| Query failure            | Infrastructure | `grpc.query_failed`, `grpc.ask` |
| Event client streaming   | Infrastructure | `grpc.event_client.open_stream`, `grpc.event_client.receive` |
| Event server streaming   | Infrastructure | `grpc.event_server.send`, `grpc.event_server.subscribe` |
| Unmarshal/marshal result | Corruption     | `grpc.unmarshal_result`, `grpc.query.marshal_result`, `grpc.event_client.decode`, `grpc.event_client.reconstruct` |
| Missing command ID       | Rejection      | `grpc.missing_command_id`, `grpc.dispatch_missing_id` |
| Command/stream ID parse  | Rejection      | `grpc.command.create`, `grpc.command.parse_stream_id`, `grpc.event_client.parse_stream_id` |

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
