# id — Branded IDs

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/id/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/id/v3)

Type-safe branded identifiers backed by ULID. Prevents mixing different ID types at compile time.

```bash
go get github.com/larsartmann/go-cqrs-lite/id/v3
```

## Quick Start

```go
// Built-in types
aggID := id.NewAggregateID()
evtID := id.NewEventID()

// Custom branded type
type OrderID = id.Of[orderMarker]
orderID := id.New[OrderID]()
parsed, err := id.Parse[OrderID](orderID.String())
```

## Built-in Types

AggregateID, EventID, CorrelationID, CausationID, RequestID, UserID, ClientID, CommandID

## Serialization

All IDs support JSON (including null), binary, text, and SQL Scan/Value.

## Related Modules

- [**event/v2**](../event/README.md) — Uses `AggregateID`, `EventID`, `CorrelationID`, `CausationID`
- [**command/v2**](../command/README.md) — Uses `AggregateID`, `CommandID`
- [**query/v2**](../query/README.md) — Uses `RequestID`
- [**decider/v2**](../decider/README.md) — Aggregates keyed by branded `AggregateID`
