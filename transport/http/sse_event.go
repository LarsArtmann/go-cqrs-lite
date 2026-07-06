package http

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	brandid "github.com/larsartmann/go-branded-id"
	errorfamily "github.com/larsartmann/go-error-family"
)

// sseEventBrand is the phantom brand type for SSEEventID.
type sseEventBrand struct{}

func (sseEventBrand) Name() string { return "SSEEvent" }

// SSEEventID is a branded identifier for SSE event identifiers (the id: field
// and the Last-Event-ID request header). It prevents accidental cross-assignment
// with other string-typed IDs.
//
// SSE event IDs are arbitrary server-defined strings — they are NOT ULIDs.
// Use ParseSSEEventID to construct from a string (rejects control characters
// and newlines, which would corrupt the SSE wire format).
type SSEEventID = brandid.ID[sseEventBrand, string]

// NewSSEEventID constructs an SSEEventID from a string. Performs no validation —
// use ParseSSEEventID for untrusted input (e.g., from request headers).
func NewSSEEventID(s string) SSEEventID { return brandid.NewID[sseEventBrand](s) }

// base10 is the numeric base for decimal integer formatting (strconv base).
const base10 = 10

// errSSEEventIDInvalid is returned by ParseSSEEventID for malformed values.
var errSSEEventIDInvalid = errorfamily.NewRejection(
	"http.sse.event_id_invalid",
	"sse event id: contains forbidden character (newline or carriage return)",
)

// ParseSSEEventID converts a string to an SSEEventID, rejecting values that
// would corrupt the SSE wire format (newlines, carriage returns). Empty strings
// are allowed (representing "no ID" / initial connection).
func ParseSSEEventID(s string) (SSEEventID, error) {
	if strings.ContainsAny(s, "\n\r") {
		return SSEEventID{}, errorfamily.Wrapf(errSSEEventIDInvalid, errorfamily.Rejection,
			"http.sse.event_id_invalid", "%q", s)
	}

	return NewSSEEventID(s), nil
}

// MustParseSSEEventID is the panicking variant of ParseSSEEventID for tests
// and constants. Panics if the input contains newlines.
func MustParseSSEEventID(s string) SSEEventID {
	id, err := ParseSSEEventID(s)
	if err != nil {
		panic(fmt.Sprintf("MustParseSSEEventID: %v", err))
	}

	return id
}

// SSEEvent represents a single Server-Sent Event.
//
// Per the SSE spec (https://html.spec.whatwg.org/multipage/server-sent-events.html):
//   - Event maps to the event: field. If empty, the default message event fires.
//   - ID maps to the id: field. Browsers send it back via Last-Event-ID on reconnect.
//   - Data maps to the data: field. Multi-line data is split so each line gets
//     its own "data:" prefix (required by the spec).
//   - Retry maps to the retry: field, suggesting a reconnection interval in milliseconds.
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
// Server-Sent Events wire format. Uses io.WriteString and direct byte writes
// instead of fmt.Fprintf to minimize allocations on the SSE hot path.
func WriteSSEEvent(w io.Writer, evt SSEEvent) error {
	var buf []byte

	if evt.Event != "" {
		buf = append(buf, 'e', 'v', 'e', 'n', 't', ':', ' ')
		buf = append(buf, evt.Event...)
		buf = append(buf, '\n')
	}

	for _, line := range splitSSELines(evt.Data) {
		buf = append(buf, 'd', 'a', 't', 'a', ':', ' ')
		buf = append(buf, line...)
		buf = append(buf, '\n')
	}

	if !evt.ID.IsZero() {
		buf = append(buf, 'i', 'd', ':', ' ')
		buf = append(buf, evt.ID.Get()...)
		buf = append(buf, '\n')
	}

	if evt.Retry > 0 {
		buf = append(buf, 'r', 'e', 't', 'r', 'y', ':', ' ')
		buf = strconv.AppendInt(buf, int64(evt.Retry), base10)
		buf = append(buf, '\n')
	}

	buf = append(buf, '\n')

	if _, err := w.Write(buf); err != nil {
		return errorfamily.Wrapf(
			err,
			errorfamily.Transient,
			"http.sse.write_failed",
			"write sse event",
		)
	}

	return nil
}

// WriteSSEHeartbeat writes a comment frame (SSE comment line).
// Browsers ignore it, but it keeps the connection alive through
// ALB/Nginx/Cloudflare idle timeouts.
func WriteSSEHeartbeat(w io.Writer) error {
	_, err := w.Write([]byte(": heartbeat\n\n"))

	return err
}

// WriteSSERetry writes the SSE retry field, telling the browser how many
// milliseconds to wait before reconnecting after a connection drop.
// Per the SSE spec, this is sent once and persists until overwritten.
func WriteSSERetry(w io.Writer, ms int) error {
	_, err := fmt.Fprintf(w, "retry: %d\n\n", ms)

	return err
}

// splitSSELines splits a string into lines for SSE data field formatting.
// Each line in the SSE spec must be prefixed with "data: ".
// Fast path: if the data contains no newline, returns a single-element
// slice without allocating a backing array.
func splitSSELines(s string) []string {
	if s == "" {
		return []string{""}
	}

	if !strings.Contains(s, "\n") {
		return []string{s}
	}

	var lines []string

	start := 0

	for i := range len(s) {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}

			lines = append(lines, line)
			start = i + 1
		}
	}

	if start < len(s) {
		lines = append(lines, s[start:])
	}

	if len(lines) == 0 {
		return []string{""}
	}

	return lines
}
