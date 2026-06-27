// Package aggregate is the event-sourcing core of the todo example.
//
// It defines the todo aggregate type, command and event type constants, the
// TodoPayload event schema, the TodoState fold state, and a decider.Decider
// built from a pure Apply fold plus Decide* functions (one per command) that
// validate state and produce events.
package aggregate
