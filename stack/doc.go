// Package stack is the composition root for go-cqrs-lite.
//
// A [Bundle] is a bag of peer capability fields — the event sink, event source,
// journals, publishers, snapshot store, checkpoint store, and read-model
// backend that a deployment wires together. The deployer assembles a Bundle
// (directly via [New] and the With* options, or via a preset such as
// stack/sqlite); the application consumes it through typed accessors
// ([Bundle.Repository], [Bundle.ReadModel], [Bundle.ProjectionRunner]) that
// never reference an infrastructure type.
//
// # Why "Bundle", not "Stack" or "Container"
//
// A stack implies LIFO ordering and a call-passing chain; a container implies
// lifecycle ownership of injected objects. A Bundle is neither: its fields are
// peers (no ordering between them), and ownership is explicit — [Bundle.Close]
// closes every resource the Bundle's options registered, deduplicated by
// pointer so a *sql.DB shared across capabilities is closed exactly once.
//
// # Interface Segregation at the field level
//
// The Bundle does not store a fat event.Store. It stores the segregated
// interfaces (EventSink, EventSource, Journal, SeekableJournal) so a consumer
// that only writes events can depend on [Bundle.EventSink] and stay oblivious
// to read paths. The convenience options ([WithEventStore], [WithBus]) set
// several fields from one composite value for ergonomics.
//
// # No Provider interface
//
// Go does not support partial interface implementation, so a Provider
// interface would force every preset to implement every method. Instead,
// presets are ordinary functions returning (*Bundle, error), and capabilities
// are wired with ordinary option functions. This is the same pattern the
// rest of the codebase uses (event.Option, snapshot.Option).
//
// # Resource lifecycle
//
// [Bundle.Close] scans every capability field that implements [io.Closer],
// deduplicates by pointer, and closes each once. Options that open resources
// (used by presets) register the resource's closer; if a preset fails partway
// through construction, it closes everything it already opened before
// returning the error. [New] itself only validates — it does not open
// resources, so it has nothing to roll back.
package stack
