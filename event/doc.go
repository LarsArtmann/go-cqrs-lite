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
//	    []any{UserCreated{Name: "Alice"}, EmailVerified{At: time.Now()}},
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
// # Reactive Streams
//
// Built on samber/ro (PublishSubject, ReplaySubject, BehaviorSubject):
//
//	bus := event.NewEventBus()
//	filtered := ro.Pipe1(bus, event.FilterEventType("user.created"))
//
// # Error Taxonomy
//
// Five families: Rejection, Conflict, Transient, Infrastructure, Corruption.
// Use NewRejection, NewConflict, etc. for classified errors.
//
// # Sub-packages
//
//   - eventtest: FakeStore, FakeBus, test assertions for testing
package event
