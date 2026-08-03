package http

import (
	"io"

	sse "github.com/larsartmann/go-sse"
)

// SSEEventID is a branded identifier for SSE event identifiers (the id: field
// and the Last-Event-ID request header). It prevents accidental cross-assignment
// with other string-typed IDs.
//
// SSE event IDs are arbitrary server-defined strings — they are NOT ULIDs.
// Use ParseSSEEventID to construct from a string (rejects control characters
// and newlines, which would corrupt the SSE wire format).
//
// This is a type alias for [sse.EventID] from github.com/larsartmann/go-sse:
// the wire-format identity, validation, and serialization are owned by go-sse
// (ADR-0097). The alias preserves this package's public API while delegating
// the implementation.
type SSEEventID = sse.EventID

// NewSSEEventID constructs an SSEEventID from a string. Performs no validation —
// use ParseSSEEventID for untrusted input (e.g., from request headers).
//
// Delegates to [sse.NewEventID].
func NewSSEEventID(s string) SSEEventID { return sse.NewEventID(s) }

// ParseSSEEventID converts a string to an SSEEventID, rejecting values that
// would corrupt the SSE wire format (newlines, carriage returns). Empty strings
// are allowed (representing "no ID" / initial connection).
//
// Delegates to [sse.ParseEventID].
func ParseSSEEventID(s string) (SSEEventID, error) {
	return sse.ParseEventID(s)
}

// MustParseSSEEventID is the panicking variant of ParseSSEEventID for tests
// and constants. Panics if the input contains newlines.
//
// Delegates to [sse.MustParseEventID].
func MustParseSSEEventID(s string) SSEEventID {
	return sse.MustParseEventID(s)
}

// SSEEvent represents a single Server-Sent Event.
//
// Per the SSE spec (https://html.spec.whatwg.org/multipage/server-sent-events.html):
//   - Event maps to the event: field. If empty, the default message event fires.
//   - ID maps to the id: field. Browsers send it back via Last-Event-ID on reconnect.
//   - Data maps to the data: field. Multi-line data is split so each line gets
//     its own "data:" prefix (required by the spec).
//   - Retry maps to the retry: field, suggesting a reconnection interval in milliseconds.
//
// This struct is intentionally distinct from [sse.Event] to preserve the
// historical Retry field type (int) in this package's public API. Wire-format
// serialization is delegated to [sse.WriteEvent] via [WriteSSEEvent].
type SSEEvent struct {
	// Event is the SSE event name. Must match the client's event listener.
	// For unnamed events, leave empty (the browser default "message" fires).
	Event string

	// Data is the event payload. Multi-line data is supported;
	// each line is prefixed with "data: " per the SSE specification.
	Data string

	// ID is an optional event identifier. The browser sends this as
	// Last-Event-ID on reconnection, enabling replay of missed events.
	ID SSEEventID

	// Retry is an optional reconnection time in milliseconds.
	// Instructs the browser to wait this long before reconnecting
	// after a connection drop.
	Retry int
}

// WriteSSEEvent writes a single SSE event to the writer in the standard
// Server-Sent Events wire format.
//
// Delegates to [sse.WriteEvent] (ADR-0097), converting the [SSEEvent] into an
// [sse.Event]. This eliminates the duplicated byte-append serializer and
// multi-line splitter that previously lived in this package — both are now
// owned and tested once in go-sse.
func WriteSSEEvent(w io.Writer, evt SSEEvent) error {
	return sse.WriteEvent(w, sse.Event{
		Event: evt.Event,
		Data:  evt.Data,
		ID:    evt.ID,
		Retry: uint(evt.Retry),
	})
}

// WriteSSEHeartbeat writes a comment frame (SSE comment line).
// Browsers ignore it, but it keeps the connection alive through
// ALB/Nginx/Cloudflare idle timeouts.
//
// Delegates to [sse.WriteHeartbeat].
func WriteSSEHeartbeat(w io.Writer) error {
	return sse.WriteHeartbeat(w)
}

// WriteSSERetry writes the SSE retry field, telling the browser how many
// milliseconds to wait before reconnecting after a connection drop.
// Per the SSE spec, this is sent once and persists until overwritten.
//
// Delegates to [sse.WriteRetry].
func WriteSSERetry(w io.Writer, ms int) error {
	return sse.WriteRetry(w, uint(ms)) //nolint:gosec // ms is a caller-controlled millisecond count, never negative in practice
}
