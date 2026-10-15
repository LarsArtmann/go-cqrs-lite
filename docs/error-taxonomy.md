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

> Every module table below is GENERATED: `scripts/generate-error-taxonomy.sh`
> regenerates the region from erraudit's errorflow export (`erraudit errorflow
> ./...` per module — its codes array sees inline New/Newf/Wrap/Wrapf mints a
> plain regex cannot), and `scripts/check-error-taxonomy.sh` (CI + `nix run
> .#verify`) diffs the document against the modules' `errorfamily.*` call sites
> in both directions — a code missing here, a stale entry, or a wrong family
> label fails the build. Extend both by adding modules to `GATED_MODULES` in
> the gate script (the generator shares the table).

<!-- generate-error-taxonomy:begin (scripts/generate-error-taxonomy.sh writes this region — edit outside the markers) -->

### middleware

| Error                 | Family         | Code                                               |
| --------------------- | -------------- | -------------------------------------------------- |
| -                     | Infrastructure | `deadletter.clear`                                 |
| -                     | Transient      | `deadletter.count`                                 |
| -                     | Infrastructure | `deadletter.create_table`                          |
| -                     | Infrastructure | `deadletter.migrate`                               |
| -                     | Transient      | `deadletter.query`                                 |
| -                     | Transient      | `deadletter.rows_err`                              |
| -                     | Corruption     | `deadletter.scan`                                  |
| -                     | Rejection      | `middleware.cb_invalid_failure_threshold`          |
| -                     | Rejection      | `middleware.cb_invalid_success_threshold`          |
| -                     | Rejection      | `middleware.cb_invalid_timeout`                    |
| ErrCircuitBreakerOpen | Infrastructure | `middleware.circuit_breaker_open`                  |
| -                     | Transient      | `middleware.circuit_open`                          |
| ErrUnexpectedTimeType | Corruption     | `middleware.deadletter.unexpected_time_type`       |
| -                     | Corruption     | `middleware.deadletter_sql.parse_time`             |
| -                     | Corruption     | `middleware.deadletter_sql.unexpected_string_type` |
| -                     | Corruption     | `middleware.deadletter_sql.unexpected_time_type`   |
| -                     | Rejection      | `middleware.invalid_initial_delay`                 |
| -                     | Rejection      | `middleware.invalid_max_attempts`                  |
| -                     | Rejection      | `middleware.invalid_multiplier`                    |
| ErrMeterRequired      | Rejection      | `middleware.meter_required`                        |
| -                     | Infrastructure | `middleware.otel_recorder_init`                    |
| -                     | Corruption     | `middleware.panic_detail`                          |
| ErrPanicRecovered     | Corruption     | `middleware.panic_recovered`                       |
| -                     | Corruption     | `middleware.recovery`                              |
| ErrRetryCanceled      | Infrastructure | `middleware.retry_canceled`                        |
| ErrRetryExhausted     | Infrastructure | `middleware.retry_exhausted`                       |
| ErrValidationFailed   | Rejection      | `middleware.validation_failed`                     |

- Unreferenced named errors (dead-sentinel candidates): `ErrMeterRequired`
- 20 of 27 codes are inline mints (no named sentinel)

### graph

| Error                        | Family         | Code                                  |
| ---------------------------- | -------------- | ------------------------------------- |
| -                            | Rejection      | `graph.edge_from`                     |
| -                            | Rejection      | `graph.edge_to`                       |
| errEmptyEdgeType             | Rejection      | `graph.edgeref.type_required`         |
| errEmptyKeyProp              | Rejection      | `graph.noderef.key_prop_required`     |
| errEmptyLabel                | Rejection      | `graph.noderef.label_required`        |
| -                            | Infrastructure | `graph.projection.close`              |
| errNilDriver                 | Rejection      | `graph.projection.driver_required`    |
| errNilHandler                | Rejection      | `graph.projection.handler_required`   |
| errNoName                    | Rejection      | `graph.projection.name_required`      |
| ErrPathNotFound              | Rejection      | `graph.read.path_not_found`           |
| errSchemaDuplicateEdgeType   | Rejection      | `graph.schema.duplicate_edge_type`    |
| errSchemaDuplicateNodeLabel  | Rejection      | `graph.schema.duplicate_node_label`   |
| errSchemaDuplicateProp       | Rejection      | `graph.schema.duplicate_prop`         |
| errSchemaEmptyEdgeType       | Rejection      | `graph.schema.edge_type_required`     |
| errSchemaEmptyFromLabel      | Rejection      | `graph.schema.from_label_required`    |
| errSchemaEmptyIndexName      | Rejection      | `graph.schema.index_name_required`    |
| errSchemaKeyPropInProperties | Rejection      | `graph.schema.key_prop_in_properties` |
| errSchemaEmptyKeyProp        | Rejection      | `graph.schema.key_prop_required`      |
| errSchemaEmptyLabel          | Rejection      | `graph.schema.label_required`         |
| errSchemaNoNodeTypes         | Rejection      | `graph.schema.no_node_types`          |
| errSchemaEmptyPropName       | Rejection      | `graph.schema.prop_name_required`     |
| errSchemaEmptyToLabel        | Rejection      | `graph.schema.to_label_required`      |
| errSchemaUnknownFromLabel    | Rejection      | `graph.schema.unknown_from_label`     |
| errSchemaUnknownIndexLabel   | Rejection      | `graph.schema.unknown_index_label`    |
| errSchemaUnknownIndexProp    | Rejection      | `graph.schema.unknown_index_prop`     |
| errSchemaUnknownToLabel      | Rejection      | `graph.schema.unknown_to_label`       |
| -                            | Rejection      | `graph.shortest_path_from`            |
| -                            | Rejection      | `graph.shortest_path_to`              |
| errSinkEdgeEndpointMismatch  | Rejection      | `graph.sink.edge_endpoint_mismatch`   |
| errSinkUnknownEdgeType       | Rejection      | `graph.sink.unknown_edge_type`        |
| errSinkUnknownNodeLabel      | Rejection      | `graph.sink.unknown_node_label`       |
| errSinkUnknownProp           | Rejection      | `graph.sink.unknown_prop`             |
| errSinkWrongKeyProp          | Rejection      | `graph.sink.wrong_key_prop`           |

- Unreferenced named errors (dead-sentinel candidates): `errEmptyEdgeType`, `errEmptyKeyProp`, `errEmptyLabel`, `errNilDriver`, `errNilHandler`, `errNoName`, `errSchemaNoNodeTypes`
- 5 of 33 codes are inline mints (no named sentinel)

### storage/relational

| Error                        | Family     | Code                                      |
| ---------------------------- | ---------- | ----------------------------------------- |
| -                            | Rejection  | `relational.conditions`                   |
| -                            | Transient  | `relational.count`                        |
| -                            | Transient  | `relational.migrate`                      |
| errRelationalNilDB           | Rejection  | `relational.nil_db`                       |
| errRelationalNilDialect      | Rejection  | `relational.nil_dialect`                  |
| errRelationalNilHandler      | Rejection  | `relational.nil_handler`                  |
| errRelationalNoName          | Rejection  | `relational.no_name`                      |
| -                            | Transient  | `relational.projection_begin_tx`          |
| -                            | Transient  | `relational.projection_commit`            |
| -                            | Transient  | `relational.projection_reset`             |
| -                            | Transient  | `relational.query`                        |
| -                            | Rejection  | `relational.query_no_columns`             |
| -                            | Transient  | `relational.rows_err`                     |
| -                            | Corruption | `relational.scan_row`                     |
| errSchemaColumnNoName        | Rejection  | `relational.schema.column_name_required`  |
| errSchemaColumnNoType        | Rejection  | `relational.schema.column_type_required`  |
| errSchemaTableNoColumns      | Rejection  | `relational.schema.columns_required`      |
| errSchemaDuplicateColumn     | Rejection  | `relational.schema.duplicate_column`      |
| errSchemaDuplicateTable      | Rejection  | `relational.schema.duplicate_table`       |
| errSchemaIndexNoName         | Rejection  | `relational.schema.index_no_name`         |
| errSchemaNoTables            | Rejection  | `relational.schema.no_tables`             |
| errSchemaTableNoName         | Rejection  | `relational.schema.table_name_required`   |
| errSchemaUniqueNoName        | Rejection  | `relational.schema.unique_no_name`        |
| errSchemaUnknownIndexColumn  | Rejection  | `relational.schema.unknown_index_column`  |
| errSchemaUnknownPKColumn     | Rejection  | `relational.schema.unknown_pk_column`     |
| errSchemaUnknownUniqueColumn | Rejection  | `relational.schema.unknown_unique_column` |
| -                            | Rejection  | `relational.schema_column_no_name`        |
| -                            | Rejection  | `relational.schema_column_no_type`        |
| -                            | Rejection  | `relational.schema_duplicate_column`      |
| -                            | Rejection  | `relational.schema_duplicate_table`       |
| -                            | Rejection  | `relational.schema_index_no_name`         |
| -                            | Rejection  | `relational.schema_table`                 |
| -                            | Rejection  | `relational.schema_unique_no_name`        |
| -                            | Rejection  | `relational.schema_unknown_index_col`     |
| -                            | Rejection  | `relational.schema_unknown_pk`            |
| -                            | Rejection  | `relational.schema_unknown_unique_col`    |
| errSinkCounterInKey          | Rejection  | `relational.sink.counter_in_key`          |
| errSinkEmptyRow              | Rejection  | `relational.sink.empty_row`               |
| errSinkKeyMissingPK          | Rejection  | `relational.sink.key_missing_pk`          |
| errSinkNoRows                | Rejection  | `relational.sink.no_rows`                 |
| errSinkUnknownColumn         | Rejection  | `relational.sink.unknown_column`          |
| errSinkUnknownTable          | Rejection  | `relational.sink.unknown_table`           |
| -                            | Rejection  | `relational.sink_counter_in_key`          |
| -                            | Transient  | `relational.sink_delete`                  |
| -                            | Transient  | `relational.sink_ensure`                  |
| -                            | Transient  | `relational.sink_increment`               |
| -                            | Rejection  | `relational.sink_key_missing_pk`          |
| -                            | Rejection  | `relational.sink_no_rows`                 |
| -                            | Corruption | `relational.sink_query`                   |
| -                            | Rejection  | `relational.sink_unknown_column`          |
| -                            | Rejection  | `relational.sink_unknown_table`           |
| -                            | Transient  | `relational.sink_update`                  |
| -                            | Transient  | `relational.sink_upsert`                  |
| -                            | Transient  | `relational.sink_upsert_cols`             |
| -                            | Transient  | `relational.sink_upsert_expr`             |
| -                            | Rejection  | `relational.unknown_column`               |
| -                            | Rejection  | `relational.unknown_table`                |
| -                            | Rejection  | `relational.unsupported_operator`         |

- Unreferenced named errors (dead-sentinel candidates): `errRelationalNilDB`, `errRelationalNilDialect`, `errRelationalNilHandler`, `errRelationalNoName`, `errSchemaNoTables`, `errSchemaTableNoColumns`, `errSchemaTableNoName`, `errSinkEmptyRow`
- 36 of 58 codes are inline mints (no named sentinel)

### projectionhost

| Error              | Family         | Code                                        |
| ------------------ | -------------- | ------------------------------------------- |
| -                  | Rejection      | `projectionhost.already_started`            |
| -                  | Infrastructure | `projectionhost.check_projection_staleness` |
| -                  | Infrastructure | `projectionhost.check_staleness`            |
| -                  | Rejection      | `projectionhost.checkpoint_store_required`  |
| -                  | Infrastructure | `projectionhost.dlq_count`                  |
| -                  | Infrastructure | `projectionhost.dlq_delete`                 |
| -                  | Infrastructure | `projectionhost.dlq_list`                   |
| -                  | Infrastructure | `projectionhost.dlq_list_paged`             |
| -                  | Infrastructure | `projectionhost.dlq_purge`                  |
| -                  | Infrastructure | `projectionhost.dlq_purge_before`           |
| -                  | Corruption     | `projectionhost.dlq_reconstruct`            |
| -                  | Infrastructure | `projectionhost.dlq_scan`                   |
| -                  | Infrastructure | `projectionhost.dlq_schema`                 |
| -                  | Infrastructure | `projectionhost.dlq_store`                  |
| -                  | Rejection      | `projectionhost.duplicate_name`             |
| -                  | Rejection      | `projectionhost.empty_projection_name`      |
| -                  | Rejection      | `projectionhost.journal_required`           |
| -                  | Infrastructure | `projectionhost.list_dead_letters`          |
| -                  | Rejection      | `projectionhost.nil_db`                     |
| -                  | Rejection      | `projectionhost.no_dead_letter_store`       |
| -                  | Rejection      | `projectionhost.register_after_start`       |
| -                  | Infrastructure | `projectionhost.reset_checkpoint`           |
| -                  | Infrastructure | `projectionhost.reset_dlq_purge`            |
| -                  | Infrastructure | `projectionhost.reset_projection`           |
| -                  | Rejection      | `projectionhost.reset_while_running`        |
| -                  | Infrastructure | `projectionhost.save_checkpoint_live`       |
| -                  | Infrastructure | `projectionhost.shutdown_timeout`           |
| ErrProjectionStale | Transient      | `projectionhost.stale`                      |
| -                  | Rejection      | `projectionhost.unknown_projection`         |
| ErrWorkerFailed    | Infrastructure | `projectionhost.worker_failed`              |

- 28 of 30 codes are inline mints (no named sentinel)

### transport/grpc

| Error               | Family         | Code                                |
| ------------------- | -------------- | ----------------------------------- |
| -                   | Infrastructure | `grpc.ask`                          |
| -                   | Rejection      | `grpc.command.create`               |
| -                   | Rejection      | `grpc.command.parse_stream_id`      |
| -                   | Infrastructure | `grpc.dispatch`                     |
| errDispatchFailed   | Infrastructure | `grpc.dispatch_failed`              |
| -                   | Rejection      | `grpc.dispatch_missing_id`          |
| -                   | Corruption     | `grpc.event_client.decode`          |
| -                   | Infrastructure | `grpc.event_client.open_stream`     |
| -                   | Rejection      | `grpc.event_client.parse_stream_id` |
| -                   | Infrastructure | `grpc.event_client.receive`         |
| -                   | Corruption     | `grpc.event_client.reconstruct`     |
| -                   | Infrastructure | `grpc.event_server.send`            |
| -                   | Infrastructure | `grpc.event_server.subscribe`       |
| errMissingCommandID | Rejection      | `grpc.missing_command_id`           |
| -                   | Corruption     | `grpc.query.marshal_result`         |
| errQueryFailed      | Infrastructure | `grpc.query_failed`                 |
| -                   | Corruption     | `grpc.unmarshal_result`             |

- Unreferenced named errors (dead-sentinel candidates): `errDispatchFailed`, `errQueryFailed`
- 14 of 17 codes are inline mints (no named sentinel)

### claiming

| Error           | Family         | Code                      |
| --------------- | -------------- | ------------------------- |
| ErrLeaseNotHeld | Orchestration  | `claiming.lease_not_held` |
| -               | Infrastructure | `claiming.stamp_lease`    |

- Unreferenced named errors (dead-sentinel candidates): `ErrLeaseNotHeld`, `ErrUnsupported`
- 1 of 2 codes are inline mints (no named sentinel)

### queue

| Error           | Family        | Code                   |
| --------------- | ------------- | ---------------------- |
| ErrDanglingDep  | Rejection     | `queue.dangling_dep`   |
| ErrLeaseNotHeld | Orchestration | `queue.lease_not_held` |

- Unreferenced named errors (dead-sentinel candidates): `ErrDanglingDep`, `ErrDuplicateID`, `ErrEmptyType`, `ErrInvalidTransition`, `ErrLeaseNotHeld`, `ErrNotFound`, `errObservationAbandoned`, `errDSNRequired`, `errNilPool`

### watermill

| Error               | Family         | Code                                    |
| ------------------- | -------------- | --------------------------------------- |
| -                   | Infrastructure | `watermill.catchup.load_checkpoint`     |
| -                   | Orchestration  | `watermill.catchup.replay_nacked`       |
| -                   | Infrastructure | `watermill.catchup.replay_read`         |
| -                   | Infrastructure | `watermill.catchup_subscriber_closed`   |
| -                   | Infrastructure | `watermill.command_bus_publish`         |
| -                   | Corruption     | `watermill.convert_message_failed`      |
| -                   | Corruption     | `watermill.corrupt_metadata`            |
| -                   | Rejection      | `watermill.create_catchup_subscriber`   |
| -                   | Corruption     | `watermill.create_command_failed`       |
| -                   | Corruption     | `watermill.create_event_failed`         |
| -                   | Infrastructure | `watermill.event_bus_publish`           |
| ErrMissingMetadata  | Rejection      | `watermill.missing_metadata`            |
| -                   | Rejection      | `watermill.parse_event_id_failed`       |
| -                   | Rejection      | `watermill.parse_failed`                |
| -                   | Rejection      | `watermill.parse_id_field_failed`       |
| -                   | Rejection      | `watermill.parse_occurred_at_failed`    |
| -                   | Rejection      | `watermill.parse_optional`              |
| -                   | Rejection      | `watermill.parse_schema_version_failed` |
| -                   | Rejection      | `watermill.parse_stream_id_failed`      |
| -                   | Rejection      | `watermill.parse_tombstone_status`      |
| -                   | Rejection      | `watermill.parse_version_failed`        |
| -                   | Infrastructure | `watermill.publish_command_failed`      |
| -                   | Infrastructure | `watermill.publish_event_failed`        |
| ErrReentrantPublish | Orchestration  | `watermill.reentrant_publish`           |
| -                   | Infrastructure | `watermill.subscribe_failed`            |
| -                   | Infrastructure | `watermill.topic_cancelled`             |
| -                   | Infrastructure | `watermill.topic_closed`                |

- Unreferenced named errors (dead-sentinel candidates): `ErrMissingMetadata`
- 25 of 27 codes are inline mints (no named sentinel)

### storage/pebble

| Error                 | Family         | Code                               |
| --------------------- | -------------- | ---------------------------------- |
| -                     | Infrastructure | `pebble.adapter.get`               |
| -                     | Infrastructure | `pebble.adapter.has`               |
| -                     | Infrastructure | `pebble.adapter.new_iterator`      |
| -                     | Infrastructure | `pebble.adapter.set_if_absent_get` |
| -                     | Infrastructure | `pebble.adapter.set_if_absent_set` |
| -                     | Infrastructure | `pebble.add_to_batch`              |
| -                     | Infrastructure | `pebble.batch_dup_check`           |
| -                     | Conflict       | `pebble.batch_existing_dup`        |
| -                     | Conflict       | `pebble.batch_internal_dup`        |
| -                     | Corruption     | `pebble.checkpoint_cbor`           |
| -                     | Corruption     | `pebble.checkpoint_json`           |
| -                     | Infrastructure | `pebble.command_batch_commit`      |
| -                     | Infrastructure | `pebble.command_commit`            |
| -                     | Corruption     | `pebble.command_corrupt`           |
| -                     | Infrastructure | `pebble.command_iter`              |
| -                     | Infrastructure | `pebble.command_iter_error`        |
| -                     | Infrastructure | `pebble.command_journal_key`       |
| -                     | Infrastructure | `pebble.command_stream_key`        |
| -                     | Infrastructure | `pebble.commit_batch`              |
| -                     | Infrastructure | `pebble.concurrency_check`         |
| -                     | Corruption     | `pebble.corrupt_event`             |
| -                     | Infrastructure | `pebble.create_iterator`           |
| -                     | Infrastructure | `pebble.delete_snapshot`           |
| -                     | Corruption     | `pebble.deserialize_checkpoint`    |
| -                     | Corruption     | `pebble.deserialize_snapshot`      |
| -                     | Conflict       | `pebble.duplicate_command`         |
| -                     | Infrastructure | `pebble.duplicate_command_check`   |
| -                     | Conflict       | `pebble.duplicate_query`           |
| -                     | Infrastructure | `pebble.duplicate_query_check`     |
| -                     | Rejection      | `pebble.empty_projection_name`     |
| -                     | Infrastructure | `pebble.event_load_filtered`       |
| -                     | Corruption     | `pebble.invalid_key_format`        |
| -                     | Infrastructure | `pebble.iterator_error`            |
| -                     | Corruption     | `pebble.journal_corrupt_event`     |
| -                     | Infrastructure | `pebble.journal_iterator`          |
| ErrNilDatabase        | Rejection      | `pebble.nil_database`              |
| -                     | Infrastructure | `pebble.open_backend`              |
| -                     | Infrastructure | `pebble.parse_version`             |
| -                     | Corruption     | `pebble.query_corrupt`             |
| -                     | Infrastructure | `pebble.query_iter`                |
| -                     | Infrastructure | `pebble.query_iter_error`          |
| -                     | Infrastructure | `pebble.query_write`               |
| -                     | Infrastructure | `pebble.read_checkpoint`           |
| -                     | Infrastructure | `pebble.read_snapshot`             |
| -                     | Corruption     | `pebble.reconstruct_command`       |
| -                     | Corruption     | `pebble.reconstruct_event`         |
| -                     | Corruption     | `pebble.reconstruct_query`         |
| -                     | Infrastructure | `pebble.scan_journal`              |
| -                     | Corruption     | `pebble.serialize_checkpoint`      |
| -                     | Corruption     | `pebble.serialize_command`         |
| -                     | Corruption     | `pebble.serialize_command_batch`   |
| -                     | Corruption     | `pebble.serialize_event`           |
| -                     | Corruption     | `pebble.serialize_query`           |
| -                     | Corruption     | `pebble.serialize_snapshot`        |
| -                     | Corruption     | `pebble.snapshot_cbor`             |
| -                     | Corruption     | `pebble.snapshot_json`             |
| -                     | Corruption     | `pebble.stream_corrupt_event`      |
| -                     | Infrastructure | `pebble.stream_create_iterator`    |
| ErrStreamIDMismatch   | Conflict       | `pebble.stream_id_mismatch`        |
| -                     | Infrastructure | `pebble.stream_iterator`           |
| ErrStreamTypeMismatch | Conflict       | `pebble.stream_type_mismatch`      |
| -                     | Corruption     | `pebble.validate_event`            |
| -                     | Conflict       | `pebble.version_conflict`          |
| ErrVersionMismatch    | Conflict       | `pebble.version_mismatch`          |
| -                     | Infrastructure | `pebble.write_checkpoint`          |
| -                     | Infrastructure | `pebble.write_snapshot`            |

- Unreferenced named errors (dead-sentinel candidates): `ErrNilDatabase`
- 62 of 66 codes are inline mints (no named sentinel)

### core/event

| Error                     | Family         | Code                                 |
| ------------------------- | -------------- | ------------------------------------ |
| -                         | Corruption     | `event.build_failed`                 |
| ErrBusClosed              | Infrastructure | `event.bus_closed`                   |
| -                         | Rejection      | `event.date_cbor_decode`             |
| -                         | Rejection      | `event.date_parse`                   |
| -                         | Corruption     | `event.decode_custom_bytes`          |
| -                         | Corruption     | `event.decode_payload_auto_no_codec` |
| -                         | Corruption     | `event.decode_payload_failed`        |
| ErrEmptyEventType         | Rejection      | `event.empty_event_type`             |
| -                         | Rejection      | `event.empty_source`                 |
| ErrEmptyStreamType        | Rejection      | `event.empty_stream_type`            |
| -                         | Rejection      | `event.encoding_mismatch`            |
| ErrEventNotFound          | Rejection      | `event.event_not_found`              |
| ErrInnerStoreNotBackwards | Rejection      | `event.inner_store_not_backwards`    |
| ErrInnerStoreNotJournal   | Rejection      | `event.inner_store_not_journal`      |
| ErrInnerStoreNotMultiSink | Rejection      | `event.inner_store_not_multi_sink`   |
| ErrInnerStoreNotSeekable  | Rejection      | `event.inner_store_not_seekable`     |
| ErrInnerStoreNotStreaming | Rejection      | `event.inner_store_not_streaming`    |
| -                         | Rejection      | `event.instant_cbor_decode`          |
| -                         | Rejection      | `event.instant_parse`                |
| ErrInvalidDate            | Rejection      | `event.invalid_date`                 |
| ErrInvalidHour            | Rejection      | `event.invalid_hour`                 |
| -                         | Rejection      | `event.invalid_ip_address`           |
| ErrInvalidMinute          | Rejection      | `event.invalid_minute`               |
| -                         | Rejection      | `event.invalid_schema_version`       |
| -                         | Corruption     | `event.marshal_payload_failed`       |
| ErrMismatchedEventCount   | Rejection      | `event.mismatched_event_count`       |
| ErrNilBus                 | Infrastructure | `event.nil_bus`                      |
| -                         | Rejection      | `event.nil_event`                    |
| ErrNilPayload             | Rejection      | `event.nil_payload`                  |
| ErrNilStreamID            | Rejection      | `event.nil_stream_id`                |
| ErrSchemaVersionUnderflow | Rejection      | `event.schema_version_underflow`     |
| ErrStoreClosed            | Infrastructure | `event.store_closed`                 |
| -                         | Rejection      | `event.store_not_backwards`          |
| -                         | Rejection      | `event.store_not_journal`            |
| -                         | Rejection      | `event.store_not_multi_sink`         |
| -                         | Rejection      | `event.store_not_streaming`          |
| ErrStreamNotFound         | Rejection      | `event.stream_not_found`             |
| ErrVersionConflict        | Conflict       | `event.version_conflict`             |
| ErrVersionNotPositive     | Rejection      | `event.version_not_positive`         |
| ErrVersionUnderflow       | Rejection      | `event.version_underflow`            |
| -                         | Rejection      | `event.walltime_cbor_decode`         |
| -                         | Rejection      | `event.walltime_invalid_tz`          |

- Unreferenced named errors (dead-sentinel candidates): `ErrBusClosed`, `ErrEventNotFound`, `ErrNilBus`, `ErrStoreClosed`
- 20 of 42 codes are inline mints (no named sentinel)

### core/command

| Error                 | Family         | Code                               |
| --------------------- | -------------- | ---------------------------------- |
| ErrDispatcherClosed   | Infrastructure | `command.dispatcher_closed`        |
| ErrDuplicateCommand   | Conflict       | `command.duplicate`                |
| ErrEmptyCommandType   | Rejection      | `command.empty_command_type`       |
| ErrEmptyStreamType    | Rejection      | `command.empty_stream_type`        |
| ErrHandlerNotFound    | Rejection      | `command.handler_not_found`        |
| -                     | Rejection      | `command.memory_bus.subscribe`     |
| errNilBusHandler      | Rejection      | `command.nil_handler`              |
| ErrNilStreamID        | Rejection      | `command.nil_stream_id`            |
| errNilBusSubscribeAll | Rejection      | `command.nil_subscribe_all`        |
| ErrCommandNotFound    | Rejection      | `command.not_found`                |
| -                     | Rejection      | `command.parse_stream_type`        |
| ErrStoreClosed        | Infrastructure | `command.store_closed`             |
| ErrTypeAssertion      | Rejection      | `command.type_assertion`           |
| -                     | Corruption     | `command.typed_store.decode`       |
| -                     | Corruption     | `command.typed_store.encode`       |
| -                     | Corruption     | `command.typed_store.encode_batch` |
| -                     | Infrastructure | `command.typed_store.load`         |

- Unreferenced named errors (dead-sentinel candidates): `ErrCommandNotFound`, `ErrDispatcherClosed`, `ErrStoreClosed`, `ErrTypeAssertion`, `errNilBusSubscribeAll`
- 6 of 17 codes are inline mints (no named sentinel)

### core/query

| Error               | Family         | Code                       |
| ------------------- | -------------- | -------------------------- |
| ErrDispatcherClosed | Infrastructure | `query.dispatcher_closed`  |
| ErrDuplicateQuery   | Conflict       | `query.duplicate`          |
| ErrEmptyQueryType   | Rejection      | `query.empty_query_type`   |
| ErrHandlerNotFound  | Rejection      | `query.handler_not_found`  |
| -                   | Rejection      | `query.invalid_page`       |
| -                   | Rejection      | `query.invalid_page_size`  |
| ErrQueryNotFound    | Rejection      | `query.not_found`          |
| ErrStoreClosed      | Infrastructure | `query.store_closed`       |
| ErrTypeAssertion    | Rejection      | `query.type_assertion`     |
| -                   | Corruption     | `query.type_mismatch`      |
| -                   | Corruption     | `query.typed_store.decode` |
| -                   | Corruption     | `query.typed_store.encode` |
| -                   | Infrastructure | `query.typed_store.load`   |

- Unreferenced named errors (dead-sentinel candidates): `ErrDispatcherClosed`, `ErrQueryNotFound`, `ErrStoreClosed`, `ErrTypeAssertion`
- 6 of 13 codes are inline mints (no named sentinel)

### storage/view

| Error                    | Family         | Code                                       |
| ------------------------ | -------------- | ------------------------------------------ |
| -                        | Corruption     | `storage.view.auto.scan_row`               |
| -                        | Transient      | `storage.view.batch_chunk`                 |
| -                        | Rejection      | `storage.view.batch_nil`                   |
| -                        | Rejection      | `storage.view.conditions`                  |
| -                        | Transient      | `storage.view.count`                       |
| -                        | Infrastructure | `storage.view.create_index`                |
| -                        | Infrastructure | `storage.view.create_indexes`              |
| -                        | Infrastructure | `storage.view.create_table`                |
| -                        | Transient      | `storage.view.delete`                      |
| -                        | Transient      | `storage.view.delete_all`                  |
| -                        | Corruption     | `storage.view.get`                         |
| errMapperColumnNameEmpty | Rejection      | `storage.view.mapper.column_name_required` |
| errMapperColumnsRequired | Rejection      | `storage.view.mapper.columns_required`     |
| errMapperDuplicateColumn | Rejection      | `storage.view.mapper.duplicate_column`     |
| errMapperExtractRequired | Rejection      | `storage.view.mapper.extract_required`     |
| errMapperKeyReserved     | Rejection      | `storage.view.mapper.key_reserved`         |
| errMapperScanRowRequired | Rejection      | `storage.view.mapper.scan_row_required`    |
| errMapperTableRequired   | Rejection      | `storage.view.mapper.table_required`       |
| -                        | Infrastructure | `storage.view.migrate`                     |
| -                        | Infrastructure | `storage.view.new_handle`                  |
| errNilViewValue          | Rejection      | `storage.view.nil_value`                   |
| -                        | Transient      | `storage.view.query`                       |
| -                        | Transient      | `storage.view.scan`                        |
| -                        | Corruption     | `storage.view.scan_row`                    |
| -                        | Transient      | `storage.view.scan_rows_err`               |
| -                        | Transient      | `storage.view.set`                         |
| -                        | Rejection      | `storage.view.set_nil`                     |
| -                        | Rejection      | `storage.view.unknown_column`              |
| -                        | Rejection      | `storage.view.unsupported_operator`        |
| -                        | Rejection      | `storage.view.validate_mapper`             |

- Unreferenced named errors (dead-sentinel candidates): `errMapperColumnsRequired`, `errMapperScanRowRequired`, `errMapperTableRequired`
- 22 of 30 codes are inline mints (no named sentinel)

### stack

| Error                | Family         | Code                                    |
| -------------------- | -------------- | --------------------------------------- |
| -                    | Infrastructure | `bbolt_preset.open_backend`             |
| -                    | Infrastructure | `bbolt_preset.wire_bundle`              |
| -                    | Infrastructure | `duckdb.create_secondary_backend`       |
| -                    | Infrastructure | `duckdb.init_schema`                    |
| -                    | Infrastructure | `duckdb.open`                           |
| -                    | Infrastructure | `duckdb_preset.finalize_bundle`         |
| -                    | Infrastructure | `duckdb_preset.init_schema`             |
| -                    | Infrastructure | `duckdb_preset.init_stack`              |
| -                    | Infrastructure | `memory.wire_bundle`                    |
| ErrNoDatabase        | Rejection      | `mysql.no_database`                     |
| -                    | Infrastructure | `mysql_preset.create_secondary_backend` |
| -                    | Infrastructure | `mysql_preset.finalize_bundle`          |
| -                    | Infrastructure | `mysql_preset.init_schema`              |
| -                    | Infrastructure | `mysql_preset.init_stack`               |
| -                    | Infrastructure | `mysql_preset.open_secondary`           |
| -                    | Infrastructure | `mysql_preset.secondary_init_schema`    |
| -                    | Infrastructure | `pebble_preset.open_backend`            |
| -                    | Infrastructure | `pebble_preset.wire_bundle`             |
| -                    | Infrastructure | `postgres.create_secondary_backend`     |
| -                    | Infrastructure | `postgres.init_schema`                  |
| ErrNoDatabase        | Rejection      | `postgres.no_database`                  |
| -                    | Infrastructure | `postgres.open_secondary`               |
| -                    | Infrastructure | `postgres_preset.finalize_bundle`       |
| -                    | Infrastructure | `postgres_preset.init_schema`           |
| -                    | Infrastructure | `postgres_preset.init_stack`            |
| -                    | Infrastructure | `sqlite.create_secondary_backend`       |
| -                    | Infrastructure | `sqlite.enable_fk`                      |
| -                    | Infrastructure | `sqlite.enable_wal`                     |
| -                    | Infrastructure | `sqlite.init_schema`                    |
| ErrNoDatabase        | Rejection      | `sqlite.no_database`                    |
| -                    | Infrastructure | `sqlite.open`                           |
| -                    | Infrastructure | `sqlite.optimize`                       |
| -                    | Infrastructure | `sqlite_preset.apply_durability`        |
| -                    | Infrastructure | `sqlite_preset.apply_optimizations`     |
| -                    | Infrastructure | `sqlite_preset.cache_size`              |
| -                    | Infrastructure | `sqlite_preset.enable_foreign_keys`     |
| -                    | Infrastructure | `sqlite_preset.enable_wal`              |
| -                    | Infrastructure | `sqlite_preset.finalize_bundle`         |
| -                    | Infrastructure | `sqlite_preset.init_schema`             |
| -                    | Infrastructure | `sqlite_preset.init_stack`              |
| -                    | Infrastructure | `stack.bundle.graceful_drain`           |
| -                    | Infrastructure | `stack.bundle.health_check`             |
| ErrEmpty             | Rejection      | `stack.bundle_empty`                    |
| -                    | Rejection      | `stack.invalid_subscriber_type`         |
| -                    | Corruption     | `stack.materialize.decode`              |
| -                    | Rejection      | `stack.materialize.extract_key`         |
| ErrMissingCheckpoint | Rejection      | `stack.missing_checkpoint`              |
| ErrMissingEventStore | Rejection      | `stack.missing_event_store`             |
| ErrMissingJournal    | Rejection      | `stack.missing_journal`                 |
| ErrMissingPublisher  | Rejection      | `stack.missing_publisher`               |
| ErrMissingQueryStore | Rejection      | `stack.missing_query_store`             |
| ErrMissingReadModels | Rejection      | `stack.missing_read_models`             |
| ErrMissingSubscriber | Rejection      | `stack.missing_subscriber`              |
| -                    | Infrastructure | `stack.run_projections.catchup`         |
| -                    | Corruption     | `stack.run_projections.decode`          |
| -                    | Infrastructure | `stack.run_projections.subscribe`       |
| -                    | Infrastructure | `turso.apply_durability`                |
| -                    | Infrastructure | `turso.create_backend`                  |
| -                    | Infrastructure | `turso.create_secondary_backend`        |
| -                    | Infrastructure | `turso.create_view_backend`             |
| -                    | Infrastructure | `turso.enable_fk`                       |
| -                    | Infrastructure | `turso.enable_wal`                      |
| -                    | Infrastructure | `turso.init_schema`                     |
| -                    | Infrastructure | `turso.kv_store`                        |
| ErrNoDatabase        | Rejection      | `turso.no_database`                     |
| -                    | Infrastructure | `turso.open`                            |
| -                    | Infrastructure | `turso.open_secondary`                  |
| -                    | Infrastructure | `turso.open_view_db`                    |
| -                    | Infrastructure | `turso.view_kv_store`                   |
| -                    | Infrastructure | `turso_preset.create_backend`           |
| -                    | Infrastructure | `turso_preset.kv_store`                 |
| -                    | Rejection      | `turso_preset.multi_db_incompatible`    |
| -                    | Infrastructure | `turso_preset.open_event_db`            |
| -                    | Infrastructure | `turso_preset.open_local_backend`       |
| -                    | Infrastructure | `turso_preset.open_query_db`            |
| -                    | Infrastructure | `turso_preset.open_sync_db`             |
| -                    | Infrastructure | `turso_preset.schema_pragmas`           |
| -                    | Infrastructure | `turso_preset.view_options`             |
| -                    | Infrastructure | `turso_preset.wire_local_bundle`        |
| -                    | Infrastructure | `turso_preset.wire_sync_bundle`         |

- Unreferenced named errors (dead-sentinel candidates): `ErrEmpty`, `ErrMissingCheckpoint`, `ErrMissingEventStore`, `ErrMissingJournal`, `ErrMissingProjection`, `ErrMissingPublisher`, `ErrMissingQueryStore`, `ErrMissingReadModels`, `ErrMissingSubscriber`, `ErrNoDatabase`
- 68 of 80 codes are inline mints (no named sentinel)

### deriver

| Error            | Family        | Code                     |
| ---------------- | ------------- | ------------------------ |
| ErrDepthExceeded | Orchestration | `deriver.depth_exceeded` |
| ErrNilDispatcher | Rejection     | `deriver.nil_dispatcher` |

- Unreferenced named errors (dead-sentinel candidates): `ErrNilDispatcher`

### storage (SQL facade)

| Error                    | Family         | Code                                       |
| ------------------------ | -------------- | ------------------------------------------ |
| -                        | Infrastructure | `backend.checkpoint_store`                 |
| -                        | Infrastructure | `backend.command_store`                    |
| -                        | Infrastructure | `backend.event_store`                      |
| -                        | Infrastructure | `backend.kv_store`                         |
| -                        | Infrastructure | `backend.query_store`                      |
| -                        | Infrastructure | `backend.snapshot_store`                   |
| -                        | Infrastructure | `kv_sql.begin_batch`                       |
| -                        | Infrastructure | `kv_sql.create_handle`                     |
| -                        | Transient      | `kv_sql.get`                               |
| -                        | Transient      | `kv_sql.has`                               |
| -                        | Transient      | `kv_sql.iterator`                          |
| -                        | Corruption     | `kv_sql.scan`                              |
| -                        | Infrastructure | `listing.create_table`                     |
| -                        | Rejection      | `listing.invalid_prefix`                   |
| -                        | Rejection      | `listing.invalid_table_prefix`             |
| -                        | Infrastructure | `listing.sql_list`                         |
| -                        | Infrastructure | `listing.sql_parse_id`                     |
| -                        | Infrastructure | `listing.sql_rows`                         |
| -                        | Infrastructure | `listing.sql_scan`                         |
| -                        | Rejection      | `listing.type_required`                    |
| -                        | Infrastructure | `sql.invalid_identifier`                   |
| -                        | Infrastructure | `storage.batch_insert_events`              |
| -                        | Infrastructure | `storage.begin_tx`                         |
| -                        | Infrastructure | `storage.check_version`                    |
| ErrClosed                | Infrastructure | `storage.closed`                           |
| -                        | Infrastructure | `storage.commit_tx`                        |
| -                        | Infrastructure | `storage.delete_by_stream`                 |
| -                        | Infrastructure | `storage.due_timers`                       |
| -                        | Conflict       | `storage.duplicate_command`                |
| -                        | Conflict       | `storage.duplicate_query`                  |
| -                        | Infrastructure | `storage.enable_foreign_keys`              |
| -                        | Rejection      | `storage.event_not_found`                  |
| -                        | Infrastructure | `storage.exec_ddl`                         |
| -                        | Infrastructure | `storage.insert_event`                     |
| -                        | Infrastructure | `storage.insert_events`                    |
| -                        | Infrastructure | `storage.iterate_rows`                     |
| -                        | Infrastructure | `storage.iterate_timers`                   |
| -                        | Rejection      | `storage.listing_table_prefix`             |
| -                        | Infrastructure | `storage.load_checkpoint`                  |
| -                        | Infrastructure | `storage.load_snapshot`                    |
| -                        | Infrastructure | `storage.load_snapshot_version`            |
| -                        | Corruption     | `storage.marshal_metadata`                 |
| ErrNilDB                 | Infrastructure | `storage.nil_db`                           |
| -                        | Infrastructure | `storage.open_duckdb`                      |
| -                        | Infrastructure | `storage.open_sqlite`                      |
| -                        | Infrastructure | `storage.open_sqlite_in_memory`            |
| -                        | Corruption     | `storage.parse_command_id`                 |
| -                        | Corruption     | `storage.parse_command_metadata`           |
| -                        | Corruption     | `storage.parse_event_id`                   |
| -                        | Corruption     | `storage.parse_fire_at`                    |
| -                        | Corruption     | `storage.parse_occurred_at`                |
| -                        | Corruption     | `storage.parse_processed_at`               |
| -                        | Corruption     | `storage.parse_query_metadata`             |
| -                        | Corruption     | `storage.parse_received_at`                |
| -                        | Corruption     | `storage.parse_request_id`                 |
| -                        | Corruption     | `storage.parse_snapshot_created_at`        |
| -                        | Corruption     | `storage.parse_stream_id`                  |
| -                        | Corruption     | `storage.parse_stream_type`                |
| -                        | Infrastructure | `storage.postgres_ddl_lock`                |
| -                        | Infrastructure | `storage.query_by_event_id`                |
| -                        | Infrastructure | `storage.query_from_position`              |
| -                        | Infrastructure | `storage.query_queries`                    |
| -                        | Infrastructure | `storage.read_stream`                      |
| -                        | Corruption     | `storage.reconstruct_command`              |
| -                        | Corruption     | `storage.reconstruct_query`                |
| -                        | Infrastructure | `storage.resolve_cursor_timestamp`         |
| -                        | Infrastructure | `storage.save_checkpoint`                  |
| -                        | Infrastructure | `storage.save_snapshot`                    |
| -                        | Infrastructure | `storage.scan_by_event_id`                 |
| -                        | Infrastructure | `storage.scan_command`                     |
| -                        | Infrastructure | `storage.scan_event`                       |
| -                        | Infrastructure | `storage.scan_query`                       |
| -                        | Infrastructure | `storage.scan_snapshot`                    |
| -                        | Corruption     | `storage.scan_timer`                       |
| -                        | Infrastructure | `storage.schedule_timer`                   |
| -                        | Corruption     | `storage.schedule_timer_marshal`           |
| -                        | Infrastructure | `storage.set_synchronous`                  |
| -                        | Infrastructure | `storage.set_synchronous_commit`           |
| -                        | Corruption     | `storage.snapshot_column_mixed`            |
| -                        | Infrastructure | `storage.snapshot_column_probe`            |
| -                        | Infrastructure | `storage.snapshot_column_rename`           |
| -                        | Rejection      | `storage.snapshot_not_found`               |
| -                        | Rejection      | `storage.sql.invalid_column`               |
| -                        | Rejection      | `storage.sql.invalid_operator`             |
| -                        | Infrastructure | `storage.sql_load`                         |
| -                        | Infrastructure | `storage.stream_from_start`                |
| ErrStreamIDMismatch      | Conflict       | `storage.stream_id_mismatch`               |
| -                        | Infrastructure | `storage.stream_iterate`                   |
| -                        | Infrastructure | `storage.stream_query`                     |
| -                        | Infrastructure | `storage.stream_query_from`                |
| -                        | Infrastructure | `storage.stream_read_all`                  |
| -                        | Infrastructure | `storage.stream_read_from`                 |
| -                        | Infrastructure | `storage.stream_resolve_cursor`            |
| ErrStreamTypeMismatch    | Conflict       | `storage.stream_type_mismatch`             |
| ErrUnexpectedTimeType    | Corruption     | `storage.unexpected_time_type`             |
| -                        | Corruption     | `storage.unmarshal_timer_payload`          |
| ErrUnsupportedTimestamp  | Corruption     | `storage.unsupported_timestamp`            |
| ErrVersionMismatch       | Conflict       | `storage.version_mismatch`                 |
| -                        | Corruption     | `storage.view.auto.scan_row`               |
| -                        | Transient      | `storage.view.batch_chunk`                 |
| -                        | Rejection      | `storage.view.batch_nil`                   |
| -                        | Rejection      | `storage.view.conditions`                  |
| -                        | Transient      | `storage.view.count`                       |
| -                        | Infrastructure | `storage.view.create_index`                |
| -                        | Infrastructure | `storage.view.create_indexes`              |
| -                        | Infrastructure | `storage.view.create_table`                |
| -                        | Transient      | `storage.view.delete`                      |
| -                        | Transient      | `storage.view.delete_all`                  |
| -                        | Corruption     | `storage.view.get`                         |
| errMapperColumnNameEmpty | Rejection      | `storage.view.mapper.column_name_required` |
| errMapperColumnsRequired | Rejection      | `storage.view.mapper.columns_required`     |
| errMapperDuplicateColumn | Rejection      | `storage.view.mapper.duplicate_column`     |
| errMapperExtractRequired | Rejection      | `storage.view.mapper.extract_required`     |
| errMapperKeyReserved     | Rejection      | `storage.view.mapper.key_reserved`         |
| errMapperScanRowRequired | Rejection      | `storage.view.mapper.scan_row_required`    |
| errMapperTableRequired   | Rejection      | `storage.view.mapper.table_required`       |
| -                        | Infrastructure | `storage.view.migrate`                     |
| -                        | Infrastructure | `storage.view.new_handle`                  |
| errNilViewValue          | Rejection      | `storage.view.nil_value`                   |
| -                        | Transient      | `storage.view.query`                       |
| -                        | Transient      | `storage.view.scan`                        |
| -                        | Corruption     | `storage.view.scan_row`                    |
| -                        | Transient      | `storage.view.scan_rows_err`               |
| -                        | Transient      | `storage.view.set`                         |
| -                        | Rejection      | `storage.view.set_nil`                     |
| -                        | Rejection      | `storage.view.unknown_column`              |
| -                        | Rejection      | `storage.view.unsupported_operator`        |
| -                        | Rejection      | `storage.view.validate_mapper`             |

- Unreferenced named errors (dead-sentinel candidates): `ErrClosed`, `ErrNilDB`, `ErrVersionMismatch`, `errMapperColumnsRequired`, `errMapperScanRowRequired`, `errMapperTableRequired`, `errRelationalNilDB`, `errRelationalNilDialect`, `errRelationalNilHandler`, `errRelationalNoName`, `errSchemaNoTables`, `errSchemaTableNoColumns`, `errSchemaTableNoName`, `errSinkEmptyRow`
- 113 of 128 codes are inline mints (no named sentinel)

### metaengine

| Error                        | Family        | Code                                      |
| ---------------------------- | ------------- | ----------------------------------------- |
| ErrBackfillUnsupported       | Rejection     | `metaengine.backfill_unsupported`         |
| ErrClaimLeaseNotHeld         | Orchestration | `metaengine.claim_lease_not_held`         |
| ErrPlannedColumnTypeMismatch | Rejection     | `metaengine.planned_column_type_mismatch` |
| ErrVectorDimensionMismatch   | Rejection     | `metaengine.vector_dimension_mismatch`    |

- Unreferenced named errors (dead-sentinel candidates): `ErrDuplicateEvent`, `ErrPlannedColumnTypeMismatch`, `errCannotInferADT`, `errCollectionCountMismatch`, `errEdgeRemoval`, `errNoEngine`, `errNoEventLog`, `errNoQuery`, `errNoQueryDecls`, `errNotBinaryVector`, `errSSENoFlusher`, `errUndirectedGraph`, `errUnsupportedCounterOps`, `errUnsupportedCounterReads`, `errUnsupportedGraphOps`, `errUnsupportedGraphReads`, `errUnsupportedLogOps`, `errUnsupportedLogReads`, `errUnsupportedMapOps`, `errUnsupportedMultiReads`, `errUnsupportedMultimapOps`, `errUnsupportedSearchOps`, `errUnsupportedSearchReads`, `errUnsupportedSetOps`, `errUnsupportedSpatialOps`, `errUnsupportedSpatialReads`, `errUnsupportedVectorFilters`, `errUnsupportedVectorOps`, `errUnsupportedVectorReads`, `errTransportClosed`, `errQuicTransportClosed`, `ErrFrameTooLarge`, `ErrGraphBackendNotImplemented`, `ErrMapUpdaterNotImplemented`, `ErrScanBackendNotImplemented`, `ErrSearchBackendNotImplemented`, `ErrSpatialBackendNotImplemented`, `ErrTransportClosed`, `ErrVectorBackendNotImplemented`, `ErrNoProber`

<!-- generate-error-taxonomy:end -->

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
