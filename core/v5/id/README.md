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
streamID := id.NewStreamID()
evtID := id.NewEventID()
corrID := id.NewCorrelationID()

// StreamRef bundles stream type + ID for store operations
ref := id.NewStreamRef("User", streamID)

// Custom branded type
type OrderMarker struct{}
type OrderID = id.Of[OrderMarker]
orderID := id.New[OrderID]()
parsed, err := id.Parse[OrderID](orderID.String())
```

## Built-in Types

| Type            | Marker              | Purpose                       |
| --------------- | ------------------- | ----------------------------- |
| `StreamID`      | `StreamMarker`      | Identifies an event stream    |
| `EventID`       | `EventMarker`       | Uniquely identifies an event  |
| `CorrelationID` | `CorrelationMarker` | Links events across a request |
| `CausationID`   | `CausationMarker`   | Links an event to its cause   |
| `CommandID`     | `CommandMarker`     | Uniquely identifies a command |
| `RequestID`     | `RequestMarker`     | HTTP request correlation      |
| `UserID`        | `UserMarker`        | Authenticated user            |
| `ClientID`      | `ClientMarker`      | API client / consumer         |

`StreamID` is string-backed (`id.Of[StreamMarker, string]`) so caller-chosen
semantic keys survive intact; ULID backing (`id.NewStreamID()`) is for
system-minted IDs. `AggregateID` is a **deprecated alias** of `StreamID`.
`ActorID` (ADR-0111) is a distinct struct type for actor attribution
(`NewUserActor`, `NewBotActor`, `NewSystemActor`, `NewServiceActor`).

## API

| Function                 | Description                                              |
| ------------------------ | -------------------------------------------------------- |
| `New[T]()`               | Generate a new random branded ID.                        |
| `Parse[T](s)`            | Parse a string into a branded ID.                        |
| `DeriveCommandID(...)`   | Deterministically derive a command ID (for idempotency). |
| `NewStreamID()`          | Shortcut for `New[StreamID]()`.                          |
| `NewStreamRef(type, id)` | Create a stream reference for store operations.          |

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
- [**command**](../command/README.md) — Uses `AggregateID` (deprecated alias of `StreamID`), `CommandID`
- [**query**](../query/README.md) — Uses `RequestID`
- [**decider**](../decider/README.md) — Aggregates keyed by branded `StreamID`
- [**id/idtest**](idtest/doc.go) — Test helpers (`ParseStreamID`, `ParseEventID`) that call `tb.Fatalf`
