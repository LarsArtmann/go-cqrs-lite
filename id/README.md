# id — Type-Safe Branded IDs

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/id/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/id/v4)

Type-safe branded identifiers backed by ULID. Prevents mixing different ID types at compile time: a `UserID` cannot be assigned to an `EventID`, even though both are strings under the hood.

```bash
go get github.com/larsartmann/go-cqrs-lite/id/v4
```

## Quick Start

```go
import "github.com/larsartmann/go-cqrs-lite/id/v4"

// Built-in types
aggID := id.NewAggregateID()
evtID := id.NewEventID()
corrID := id.NewCorrelationID()

// Custom branded type
type OrderMarker struct{}
type OrderID = id.Of[OrderMarker]
orderID := id.New[OrderID]()
parsed, err := id.Parse[OrderID](orderID.String())
```

## Built-in Types

| Type            | Marker              | Purpose                        |
| --------------- | ------------------- | ------------------------------ |
| `AggregateID`   | `AggregateMarker`   | Identifies an aggregate stream |
| `EventID`       | `EventMarker`       | Uniquely identifies an event   |
| `CorrelationID` | `CorrelationMarker` | Links events across a request  |
| `CausationID`   | `CausationMarker`   | Links an event to its cause    |
| `CommandID`     | `CommandMarker`     | Uniquely identifies a command  |
| `RequestID`     | `RequestMarker`     | HTTP request correlation       |
| `UserID`        | `UserMarker`        | Authenticated user             |
| `ClientID`      | `ClientMarker`      | API client / consumer          |

All 8 markers are exported for `BrandNamer` integration. Custom types use `id.Of[struct{}]`.

## API

| Function                    | Description                                              |
| --------------------------- | -------------------------------------------------------- |
| `New[T]()`                  | Generate a new random branded ID.                        |
| `Parse[T](s)`               | Parse a string into a branded ID.                        |
| `DeriveCommandID(...)`      | Deterministically derive a command ID (for idempotency). |
| `NewStreamID()`             | Shortcut for `New[StreamID]()`.                          |
| `NewAggregateRef(type, id)` | Create an aggregate reference for store operations.      |

## Serialization

All branded IDs support:

- **JSON** (including `null` marshaling)
- **Binary** (`encoding.BinaryMarshaler`/`Unmarshaler`)
- **Text** (`encoding.TextMarshaler`/`TextUnmarshaler`)
- **SQL** (`database/sql.Scanner` and `driver.Valuer`)

## Design

- **Powered by `go-branded-id`**: `id.Of[T]` = `cbid.ID[T, ulid.ULID]`. The type parameter `T` is a phantom marker type that exists only at compile time.
- **ULID backing**: 128-bit, lexicographically sortable, URL-safe. Time-ordered with millisecond precision.
- **Zero-value safety**: `IsZero()` method on every branded ID.
- **No runtime overhead**: The marker type is erased at compile time. IDs are just ULIDs at runtime.

## Related Modules

- [**event**](../event/README.md) — Uses `StreamID`, `EventID`, `CorrelationID`, `CausationID`
- [**command**](../command/README.md) — Uses `AggregateID`, `CommandID`
- [**query**](../query/README.md) — Uses `RequestID`
- [**decider**](../decider/README.md) — Aggregates keyed by branded `StreamID`
- [**id/idtest**](README.md) — Test helpers (`ParseAggregateID`, `ParseEventID`) that call `tb.Fatalf`
