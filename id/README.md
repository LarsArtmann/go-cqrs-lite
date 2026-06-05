# id — Branded IDs

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/id/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/id/v2)

Type-safe branded identifiers backed by ULID. Prevents mixing different ID types at compile time.

```bash
go get github.com/larsartmann/go-cqrs-lite/id/v2
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
