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
// # Keys and persistence: Get(), never String()
//
// String() is a DISPLAY form: for brands whose marker implements Name()
// (StreamMarker does), go-branded-id renders String() as "<Marker>:<value>"
// (e.g. "StreamMarker:01M68XT7S85W53TZ6NTC000001"). The bare value — the one
// to use as map keys, storage rows, cache keys, and anything persisted or
// compared — comes from Get():
//
//	key := streamID.Get()      // bare value: persist THIS
//	label := streamID.String() // "StreamMarker:<value>": display only
//
// Persisting String() output silently poisons keys: lookups by the bare value
// miss, and nothing errors (a downstream consumer served empty reports for 26
// days on exactly this mistake). The text wire form (MarshalText) is bare, and
// the Parse* functions strip a leading brand prefix, so display forms still
// round-trip — but round-tripping a display form into storage is the trap.
//
// # Serialization
//
// All branded IDs support JSON (including null), binary, text, and SQL Scan/Value.
// ULID provides binary-sortable, time-ordered 16-byte identifiers.
package id
