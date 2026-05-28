# Error Taxonomy

> The go-cqrs-lite library uses [go-error-family](https://github.com/LarsArtmann/go-error-family) v0.2.0 for structured, classified error handling.

## The 5 Error Families

Every error produced by the library belongs to exactly one family. This enables consumers to make **programmable decisions** — retry, reject, escalate — without string matching or type assertions.

| Family             | Semantic                                    | Retryable?            | Example                                      |
| ------------------ | ------------------------------------------- | --------------------- | -------------------------------------------- |
| **Rejection**      | Business rule violation or invalid input    | No                    | `email is required`, `handler not found`     |
| **Conflict**       | Optimistic concurrency or state collision   | No (consumer decides) | `version conflict`, `user already exists`    |
| **Transient**      | Temporary failure likely to resolve         | Yes                   | `connection reset`, `timeout`                |
| **Infrastructure** | System-level failure requiring intervention | Maybe                 | `dispatcher closed`, `database unavailable`  |
| **Corruption**     | Data integrity violation                    | No                    | `type assertion failed`, `checksum mismatch` |

## Usage from Consumer Code

### Creating Classified Errors

```go
import "github.com/larsartmann/go-cqrs-lite/core/event"

// Business rule violation — consumer's input is invalid
err := event.NewRejection("order.create.negative_total", "total must be positive")

// Optimistic concurrency — another write beat us
err := event.NewConflict("order.version_conflict", "order was modified by another request")

// Transient — network blip, safe to retry
err := event.NewTransient("order.publish.timeout", "failed to publish event within deadline")
```

### Classifying Errors

```go
family := event.Classify(err)
switch family {
case event.Rejection:
    // Return 400 to the client
case event.Conflict:
    // Return 409, maybe ask user to refresh
case event.Transient:
    // Retry with backoff
case event.Infrastructure:
    // Alert on-call, return 500
case event.Corruption:
    // Page on-call, investigate data integrity
}

// Or use the boolean helper
if event.IsRetryable(err) {
    // Exponential backoff and retry
}
```

### Wrapping Errors

```go
// Preserve classification through fmt.Errorf wrapping
err := fmt.Errorf("save order %s: %w", orderID, event.NewConflict("order.version_conflict", "version mismatch"))

// Classification still works
event.Classify(err) // => Conflict
```

## Error Families by Module

### core/event

| Error                     | Family    | Code                           |
| ------------------------- | --------- | ------------------------------ |
| `ErrEmptyEventType`       | Rejection | `event.empty_event_type`       |
| `ErrNilAggregateID`       | Rejection | `event.nil_aggregate_id`       |
| `ErrEmptyAggregateType`   | Rejection | `event.empty_aggregate_type`   |
| `ErrVersionNotPositive`   | Rejection | `event.version_not_positive`   |
| `ErrNilPayload`           | Rejection | `event.nil_payload`            |
| `ErrMismatchedEventCount` | Rejection | `event.mismatched_event_count` |
| `ErrVersionConflict`      | Conflict  | `event.version_conflict`       |
| `ErrAggregateNotFound`    | Rejection | `event.aggregate_not_found`    |

### core/command

| Error                 | Family         | Code                         |
| --------------------- | -------------- | ---------------------------- |
| `ErrHandlerNotFound`  | Rejection      | `command.handler_not_found`  |
| `ErrDispatcherClosed` | Infrastructure | `command.dispatcher_closed`  |
| `ErrEmptyCommandType` | Rejection      | `command.empty_command_type` |
| `ErrNilAggregateID`   | Rejection      | `command.nil_aggregate_id`   |
| `ErrTypeAssertion`    | Corruption     | `command.type_assertion`     |

### core/query

| Error                  | Family         | Code                      |
| ---------------------- | -------------- | ------------------------- |
| `ErrQueryNotSupported` | Rejection      | `query.not_supported`     |
| `ErrDispatcherClosed`  | Infrastructure | `query.dispatcher_closed` |
| `ErrEmptyQueryType`    | Rejection      | `query.empty_query_type`  |

### middleware

| Context              | Family     | Behavior             |
| -------------------- | ---------- | -------------------- |
| Validation failure   | Rejection  | Propagated as-is     |
| Retry exhausted      | Transient  | Wraps original error |
| Panic recovery       | Corruption | Captures panic value |
| Circuit breaker open | Transient  | Returns immediately  |

## Design Principles

1. **Sentinel errors** in `errors.go` files — every module's errors are centralized
2. **Contextual wrapping** — `fmt.Errorf("operation %s: %w", name, err)` preserves classification
3. **No panics** — all errors are returned as values
4. **Consumer decides** — the library classifies, the consumer chooses the response strategy
5. **Codes are namespaced** — `module.subdomain.specific_error` format for uniqueness
