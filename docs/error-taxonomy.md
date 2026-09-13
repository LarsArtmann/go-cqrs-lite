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

> The `middleware`, `graph`, `storage/relational`, `projectionhost`,
> `transport/grpc`, and `claiming` sections are mechanically drift-gated:
> every code below is
> extracted from the module's `errorfamily.*` call sites by
> `scripts/check-error-taxonomy.sh` (CI + `nix run .#verify`) and diffed
> against this document in both directions — a code missing here, a stale
> entry, or a wrong family label fails the build. Extend the gate by adding
> modules to `GATED_MODULES` in the script.

### core/event

| Error | Family         | Code                                 |
| ----- | -------------- | ------------------------------------ |
| —     | Corruption     | `event.build_failed`                 |
| —     | Infrastructure | `event.bus_closed`                   |
| —     | Rejection      | `event.date_cbor_decode`             |
| —     | Rejection      | `event.date_parse`                   |
| —     | Corruption     | `event.decode_custom_bytes`          |
| —     | Corruption     | `event.decode_payload_auto_no_codec` |
| —     | Corruption     | `event.decode_payload_failed`        |
| —     | Rejection      | `event.empty_event_type`             |
| —     | Rejection      | `event.empty_source`                 |
| —     | Rejection      | `event.empty_stream_type`            |
| —     | Rejection      | `event.event_not_found`              |
| —     | Rejection      | `event.inner_store_not_backwards`    |
| —     | Rejection      | `event.inner_store_not_journal`      |
| —     | Rejection      | `event.inner_store_not_multi_sink`   |
| —     | Rejection      | `event.inner_store_not_seekable`     |
| —     | Rejection      | `event.inner_store_not_streaming`    |
| —     | Rejection      | `event.instant_cbor_decode`          |
| —     | Rejection      | `event.instant_parse`                |
| —     | Rejection      | `event.invalid_date`                 |
| —     | Rejection      | `event.invalid_hour`                 |
| —     | Rejection      | `event.invalid_ip_address`           |
| —     | Rejection      | `event.invalid_minute`               |
| —     | Rejection      | `event.invalid_schema_version`       |
| —     | Corruption     | `event.marshal_payload_failed`       |
| —     | Rejection      | `event.mismatched_event_count`       |
| —     | Infrastructure | `event.nil_bus`                      |
| —     | Rejection      | `event.nil_event`                    |
| —     | Rejection      | `event.nil_payload`                  |
| —     | Rejection      | `event.nil_stream_id`                |
| —     | Rejection      | `event.schema_version_underflow`     |
| —     | Infrastructure | `event.store_closed`                 |
| —     | Rejection      | `event.stream_not_found`             |
| —     | Conflict       | `event.version_conflict`             |
| —     | Rejection      | `event.version_not_positive`         |
| —     | Rejection      | `event.version_underflow`            |
| —     | Rejection      | `event.walltime_cbor_decode`         |
| —     | Rejection      | `event.walltime_invalid_tz`          |
| —     | Rejection      | `eventtest.failing_handler`          |
| —     | Rejection      | `eventtest.failing_publisher`        |
| —     | Rejection      | `my.code`                            |

### core/command

| Error | Family         | Code                               |
| ----- | -------------- | ---------------------------------- |
| —     | Infrastructure | `command.dispatcher_closed`        |
| —     | Conflict       | `command.duplicate`                |
| —     | Rejection      | `command.empty_command_type`       |
| —     | Rejection      | `command.empty_stream_type`        |
| —     | Rejection      | `command.handler_not_found`        |
| —     | Rejection      | `command.memory_bus.subscribe`     |
| —     | Rejection      | `command.nil_handler`              |
| —     | Rejection      | `command.nil_stream_id`            |
| —     | Rejection      | `command.nil_subscribe_all`        |
| —     | Rejection      | `command.not_found`                |
| —     | Rejection      | `command.parse_stream_type`        |
| —     | Infrastructure | `command.store_closed`             |
| —     | Rejection      | `command.type_assertion`           |
| —     | Corruption     | `command.typed_store.decode`       |
| —     | Corruption     | `command.typed_store.encode`       |
| —     | Corruption     | `command.typed_store.encode_batch` |
| —     | Infrastructure | `command.typed_store.load`         |

### core/query

| Error | Family         | Code                       |
| ----- | -------------- | -------------------------- |
| —     | Infrastructure | `query.dispatcher_closed`  |
| —     | Conflict       | `query.duplicate`          |
| —     | Rejection      | `query.empty_query_type`   |
| —     | Rejection      | `query.handler_not_found`  |
| —     | Rejection      | `query.invalid_page`       |
| —     | Rejection      | `query.invalid_page_size`  |
| —     | Rejection      | `query.not_found`          |
| —     | Infrastructure | `query.store_closed`       |
| —     | Rejection      | `query.type_assertion`     |
| —     | Corruption     | `query.type_mismatch`      |
| —     | Corruption     | `query.typed_store.decode` |
| —     | Corruption     | `query.typed_store.encode` |
| —     | Infrastructure | `query.typed_store.load`   |

### middleware

Retry/backoff and circuit-breaker misconfiguration is **Rejection** (bad
config); runtime exhaustion is **Infrastructure**; a recovered panic is
**Corruption** (state after a panic is suspect). Note the two breaker codes:
`middleware.circuit_open` (Transient — the breaker half-opens and retries)
vs `middleware.circuit_breaker_open` (Infrastructure — breaker wrap).
Dead-letter-store rows are Transient for reads (may succeed next poll) and
Infrastructure for DDL/maintenance; malformed stored timestamps are
**Corruption**.

| Context                     | Family         | Code                                                                                                                                                                                                          |
| --------------------------- | -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Validation failure          | Rejection      | `middleware.validation_failed`                                                                                                                                                                                |
| Meter required              | Rejection      | `middleware.meter_required`                                                                                                                                                                                   |
| Retry config invalid        | Rejection      | `middleware.invalid_initial_delay`, `middleware.invalid_max_attempts`, `middleware.invalid_multiplier`                                                                                                        |
| Breaker config invalid      | Rejection      | `middleware.cb_invalid_failure_threshold`, `middleware.cb_invalid_success_threshold`, `middleware.cb_invalid_timeout`                                                                                         |
| Retry exhausted/canceled    | Infrastructure | `middleware.retry_exhausted`, `middleware.retry_canceled`                                                                                                                                                     |
| Breaker open (retryable)    | Transient      | `middleware.circuit_open`                                                                                                                                                                                     |
| Breaker open (wrap)         | Infrastructure | `middleware.circuit_breaker_open`                                                                                                                                                                             |
| OTel recorder init          | Infrastructure | `middleware.otel_recorder_init`                                                                                                                                                                               |
| Panic recovery              | Corruption     | `middleware.panic_recovered`                                                                                                                                                                                  |
| Dead-letter malformed data  | Corruption     | `middleware.deadletter_sql.parse_time`, `middleware.deadletter_sql.unexpected_string_type`, `middleware.deadletter_sql.unexpected_time_type`, `middleware.deadletter.unexpected_time_type`, `deadletter.scan` |
| Dead-letter reads           | Transient      | `deadletter.count`, `deadletter.query`, `deadletter.rows_err`                                                                                                                                                 |
| Dead-letter DDL/maintenance | Infrastructure | `deadletter.clear`, `deadletter.create_table`, `deadletter.migrate`                                                                                                                                           |

### graph

All graph validation sentinels (schema, sink, node/edge refs, read API,
query decode) are **Rejection**; only projection lifecycle failures are
**Infrastructure**.

| Error                    | Family         | Code                                                                                                      |
| ------------------------ | -------------- | --------------------------------------------------------------------------------------------------------- |
| `ErrPathNotFound`        | Rejection      | `graph.read.path_not_found`                                                                               |
| Schema violations        | Rejection      | `graph.schema.*` (16 sentinels)                                                                           |
| Sink violations          | Rejection      | `graph.sink.*` (5 sentinels)                                                                              |
| Node/Edge ref validation | Rejection      | `graph.noderef.*`, `graph.edgeref.*`                                                                      |
| Query decode             | Rejection      | `graph.edge_from`, `graph.edge_to`, `graph.shortest_path_from`, `graph.shortest_path_to`                  |
| Projection constructor   | Rejection      | `graph.projection.driver_required`, `graph.projection.handler_required`, `graph.projection.name_required` |
| Projection close         | Infrastructure | `graph.projection.close`                                                                                  |

### storage/relational

Validation sentinels (schema, sink enforcement, nil-guard) are
**Rejection** — both the dotted (`relational.schema.*`) and legacy
underscore (`relational.schema_*`) spellings. Operational wraps split:
row-scan/reconstruction is **Corruption**, DDL/queries/writes are
**Transient** (the SQL tier treats per-statement failures as retryable —
the caller decides via `errorfamily.IsRetryable`).

| Error                      | Family     | Code                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| -------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Schema validation (dotted) | Rejection  | `relational.schema.column_name_required`, `relational.schema.column_type_required`, `relational.schema.columns_required`, `relational.schema.duplicate_column`, `relational.schema.duplicate_table`, `relational.schema.index_no_name`, `relational.schema.no_tables`, `relational.schema.table_name_required`, `relational.schema.unique_no_name`, `relational.schema.unknown_index_column`, `relational.schema.unknown_pk_column`, `relational.schema.unknown_unique_column` |
| Schema validation (legacy) | Rejection  | `relational.schema_column_no_name`, `relational.schema_column_no_type`, `relational.schema_duplicate_column`, `relational.schema_duplicate_table`, `relational.schema_index_no_name`, `relational.schema_table`, `relational.schema_unknown_index_col`, `relational.schema_unknown_pk`, `relational.schema_unique_no_name`, `relational.schema_unknown_unique_col`                                                                                                             |
| Sink validation            | Rejection  | `relational.sink.counter_in_key`, `relational.sink.empty_row`, `relational.sink.key_missing_pk`, `relational.sink.no_rows`, `relational.sink.unknown_column`, `relational.sink.unknown_table`, `relational.sink_counter_in_key`, `relational.sink_key_missing_pk`, `relational.sink_no_rows`, `relational.sink_unknown_column`, `relational.sink_unknown_table`                                                                                                                |
| Nil guards / operator      | Rejection  | `relational.nil_db`, `relational.nil_dialect`, `relational.nil_handler`, `relational.no_name`, `relational.conditions`, `relational.query_no_columns`, `relational.unknown_column`, `relational.unknown_table`, `relational.unsupported_operator`                                                                                                                                                                                                                              |
| Row scan / reconstruct     | Corruption | `relational.scan_row`, `relational.sink_query`                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| DDL / query / write wraps  | Transient  | `relational.count`, `relational.migrate`, `relational.query`, `relational.rows_err`, `relational.projection_begin_tx`, `relational.projection_commit`, `relational.projection_reset`, `relational.sink_delete`, `relational.sink_ensure`, `relational.sink_increment`, `relational.sink_update`, `relational.sink_upsert`, `relational.sink_upsert_cols`, `relational.sink_upsert_expr`                                                                                        |

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

| Error                   | Family         | Code                                                                                                                                                                                                                                                                                                                                                                       |
| ----------------------- | -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Constructor violations  | Rejection      | `projectionhost.already_started`, `projectionhost.checkpoint_store_required`, `projectionhost.duplicate_name`, `projectionhost.empty_projection_name`, `projectionhost.journal_required`, `projectionhost.nil_db`, `projectionhost.no_dead_letter_store`, `projectionhost.register_after_start`, `projectionhost.reset_while_running`, `projectionhost.unknown_projection` |
| Shutdown timeout        | Infrastructure | `projectionhost.shutdown_timeout`                                                                                                                                                                                                                                                                                                                                          |
| Checkpoint / worker ops | Infrastructure | `projectionhost.reset_checkpoint`, `projectionhost.reset_projection`, `projectionhost.save_checkpoint_live`, `projectionhost.worker_failed`                                                                                                                                                                                                                                |
| DLQ operations          | Infrastructure | `projectionhost.dlq_count`, `projectionhost.dlq_delete`, `projectionhost.dlq_list`, `projectionhost.dlq_list_paged`, `projectionhost.dlq_purge`, `projectionhost.dlq_purge_before`, `projectionhost.dlq_scan`, `projectionhost.dlq_schema`, `projectionhost.dlq_store`, `projectionhost.list_dead_letters`, `projectionhost.reset_dlq_purge`                               |
| DLQ reconstruct         | Corruption     | `projectionhost.dlq_reconstruct`                                                                                                                                                                                                                                                                                                                                           |
| Staleness check         | Transient      | `projectionhost.stale`                                                                                                                                                                                                                                                                                                                                                     |

### transport/grpc

Transport wraps are **Infrastructure**; decoding a corrupt wire payload is
**Corruption**; request validation is **Rejection**.

| Error                    | Family         | Code                                                                                                              |
| ------------------------ | -------------- | ----------------------------------------------------------------------------------------------------------------- |
| Dispatch failure         | Infrastructure | `grpc.dispatch_failed`, `grpc.dispatch`                                                                           |
| Query failure            | Infrastructure | `grpc.query_failed`, `grpc.ask`                                                                                   |
| Event client streaming   | Infrastructure | `grpc.event_client.open_stream`, `grpc.event_client.receive`                                                      |
| Event server streaming   | Infrastructure | `grpc.event_server.send`, `grpc.event_server.subscribe`                                                           |
| Unmarshal/marshal result | Corruption     | `grpc.unmarshal_result`, `grpc.query.marshal_result`, `grpc.event_client.decode`, `grpc.event_client.reconstruct` |
| Missing command ID       | Rejection      | `grpc.missing_command_id`, `grpc.dispatch_missing_id`                                                             |
| Command/stream ID parse  | Rejection      | `grpc.command.create`, `grpc.command.parse_stream_id`, `grpc.event_client.parse_stream_id`                        |

### claiming

Lease-stamp failure is **Infrastructure**; losing a claim race is
**Orchestration** — a distributed-coordination race, not a caller bug.

| Error               | Family         | Code                      |
| ------------------- | -------------- | ------------------------- |
| Lease stamp failure | Infrastructure | `claiming.stamp_lease`    |
| `ErrLeaseNotHeld`   | Orchestration  | `claiming.lease_not_held` |

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

| Error | Family         | Code                               |
| ----- | -------------- | ---------------------------------- |
| —     | Infrastructure | `pebble.adapter.get`               |
| —     | Infrastructure | `pebble.adapter.has`               |
| —     | Infrastructure | `pebble.adapter.new_iterator`      |
| —     | Infrastructure | `pebble.adapter.set_if_absent_get` |
| —     | Infrastructure | `pebble.adapter.set_if_absent_set` |
| —     | Infrastructure | `pebble.add_to_batch`              |
| —     | Infrastructure | `pebble.batch_dup_check`           |
| —     | Conflict       | `pebble.batch_existing_dup`        |
| —     | Conflict       | `pebble.batch_internal_dup`        |
| —     | Infrastructure | `pebble.command_batch_commit`      |
| —     | Infrastructure | `pebble.command_commit`            |
| —     | Corruption     | `pebble.command_corrupt`           |
| —     | Infrastructure | `pebble.command_iter`              |
| —     | Infrastructure | `pebble.command_iter_error`        |
| —     | Infrastructure | `pebble.command_journal_key`       |
| —     | Infrastructure | `pebble.command_stream_key`        |
| —     | Infrastructure | `pebble.commit_batch`              |
| —     | Infrastructure | `pebble.concurrency_check`         |
| —     | Corruption     | `pebble.corrupt_event`             |
| —     | Infrastructure | `pebble.create_iterator`           |
| —     | Infrastructure | `pebble.delete_snapshot`           |
| —     | Corruption     | `pebble.deserialize_checkpoint`    |
| —     | Corruption     | `pebble.deserialize_snapshot`      |
| —     | Conflict       | `pebble.duplicate_command`         |
| —     | Infrastructure | `pebble.duplicate_command_check`   |
| —     | Conflict       | `pebble.duplicate_query`           |
| —     | Infrastructure | `pebble.duplicate_query_check`     |
| —     | Rejection      | `pebble.empty_projection_name`     |
| —     | Infrastructure | `pebble.event_load_filtered`       |
| —     | Corruption     | `pebble.invalid_key_format`        |
| —     | Infrastructure | `pebble.iterator_error`            |
| —     | Rejection      | `pebble.nil_database`              |
| —     | Infrastructure | `pebble.open_backend`              |
| —     | Infrastructure | `pebble.parse_version`             |
| —     | Corruption     | `pebble.query_corrupt`             |
| —     | Infrastructure | `pebble.query_iter`                |
| —     | Infrastructure | `pebble.query_iter_error`          |
| —     | Infrastructure | `pebble.query_write`               |
| —     | Infrastructure | `pebble.read_checkpoint`           |
| —     | Infrastructure | `pebble.read_snapshot`             |
| —     | Corruption     | `pebble.reconstruct_command`       |
| —     | Corruption     | `pebble.reconstruct_event`         |
| —     | Corruption     | `pebble.reconstruct_query`         |
| —     | Infrastructure | `pebble.scan_journal`              |
| —     | Corruption     | `pebble.serialize_checkpoint`      |
| —     | Corruption     | `pebble.serialize_command`         |
| —     | Corruption     | `pebble.serialize_command_batch`   |
| —     | Corruption     | `pebble.serialize_event`           |
| —     | Corruption     | `pebble.serialize_query`           |
| —     | Corruption     | `pebble.serialize_snapshot`        |
| —     | Infrastructure | `pebble.stream_create_iterator`    |
| —     | Conflict       | `pebble.stream_id_mismatch`        |
| —     | Conflict       | `pebble.stream_type_mismatch`      |
| —     | Corruption     | `pebble.validate_event`            |
| —     | Conflict       | `pebble.version_conflict`          |
| —     | Conflict       | `pebble.version_mismatch`          |
| —     | Infrastructure | `pebble.write_checkpoint`          |
| —     | Infrastructure | `pebble.write_snapshot`            |

### watermill

| Error | Family         | Code                                    |
| ----- | -------------- | --------------------------------------- |
| —     | Infrastructure | `watermill.catchup.load_checkpoint`     |
| —     | Orchestration  | `watermill.catchup.replay_nacked`       |
| —     | Infrastructure | `watermill.catchup.replay_read`         |
| —     | Infrastructure | `watermill.catchup_subscriber_closed`   |
| —     | Infrastructure | `watermill.command_bus_publish`         |
| —     | Corruption     | `watermill.convert_message_failed`      |
| —     | Corruption     | `watermill.corrupt_metadata`            |
| —     | Rejection      | `watermill.create_catchup_subscriber`   |
| —     | Corruption     | `watermill.create_command_failed`       |
| —     | Corruption     | `watermill.create_event_failed`         |
| —     | Infrastructure | `watermill.event_bus_publish`           |
| —     | Rejection      | `watermill.missing_metadata`            |
| —     | Rejection      | `watermill.parse_event_id_failed`       |
| —     | Rejection      | `watermill.parse_failed`                |
| —     | Rejection      | `watermill.parse_id_field_failed`       |
| —     | Rejection      | `watermill.parse_occurred_at_failed`    |
| —     | Rejection      | `watermill.parse_schema_version_failed` |
| —     | Rejection      | `watermill.parse_stream_id_failed`      |
| —     | Rejection      | `watermill.parse_tombstone_status`      |
| —     | Rejection      | `watermill.parse_version_failed`        |
| —     | Infrastructure | `watermill.publish_command_failed`      |
| —     | Infrastructure | `watermill.publish_event_failed`        |
| —     | Infrastructure | `watermill.subscribe_failed`            |
| —     | Infrastructure | `watermill.topic_cancelled`             |
| —     | Infrastructure | `watermill.topic_closed`                |

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
