// Package event provides the core event sourcing primitives for CQRS applications.
//
// It defines immutable events, store/bus interfaces, the 5-family error taxonomy,
// reactive streams, and the aggregate lifecycle. Zero infrastructure dependencies
// (no HTTP, no database, no message broker).
//
// # Creating Events
//
// Use New for typed payloads (auto-marshaled via codec) or NewEvent for raw []byte:
//
//	evt, err := event.New("user.created", aggID, "User", event.Version(1),
//	    UserCreated{Name: "Alice"},
//	    event.WithCorrelationID(corrID),
//	)
//
//	events, err := event.NewEvents(aggID, "User", 0,
//	    []event.Type{"user.created", "user.email_verified"},
//	    []any{UserCreated{Name: "Alice"}, EmailVerified{At: time.Now().UTC()}},
//	)
//
// # Store Interface (ISP Split)
//
// EventSink (write) and EventSource (read) are separated for interface segregation:
//
//	type Store interface { EventSink; EventSource }
//	type Journal interface { ReadAll(ctx) ([]Event, error) }
//	type SeekableJournal interface { Journal; ReadFrom(ctx, afterEventID, limit) ([]Event, error) }
//
// # Error Taxonomy
//
// Five families: Rejection, Conflict, Transient, Infrastructure, Corruption.
// Use NewRejection, NewConflict, etc. for classified errors.
//
// # Time Handling
//
// time.Time values in event payloads MUST be UTC-normalized before encoding.
// The CBOR codec encodes time as epoch (stripping timezone), so non-UTC times
// are silently corrupted. Use time.Now().UTC() or the Instant type.
//
// For wall-clock times ("9am, for whom?"), NEVER use time.Time — use WallTime
// or a string with explicit IANA timezone. See docs/TIMEZONE_HANDLING.md.
//
// # Sub-packages
//
//   - eventtest: FakeStore, FakeBus, test assertions for testing
package event
