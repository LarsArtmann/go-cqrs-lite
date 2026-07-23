// Package id provides type-safe branded identifiers backed by ULID.
//
// Each domain concept has its own branded type (StreamID, EventID, UserID, etc.)
// preventing accidental mixing of IDs at compile time. Custom branded types are
// created with a one-line type alias:
//
//	type OrderID = id.Of[orderMarker]
//	orderID := id.New[OrderID]()
//
// # Built-in Types
//
//	StreamID, EventID, CorrelationID, CausationID, RequestID, UserID, ClientID, CommandID
//
// # Serialization
//
// All branded IDs support JSON (including null), binary, text, and SQL Scan/Value.
// ULID provides binary-sortable, time-ordered 16-byte identifiers.
package id
