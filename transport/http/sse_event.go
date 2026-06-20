package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

// SSEEventID is the event identifier sent in the SSE id: field.
// It must not contain newlines or carriage returns — the SSE spec
// treats a blank line as the event terminator.
type SSEEventID string

func (e SSEEventID) String() string { return string(e) }

func (e SSEEventID) IsZero() bool { return e == "" }

// ParseSSEEventID validates that s contains no \n or \r.
// These characters would corrupt the SSE wire format.
func ParseSSEEventID(s string) (SSEEventID, error) {
	if strings.ContainsAny(s, "\n\r") {
		return "", errors.New("SSE event ID must not contain newlines")
	}

	return SSEEventID(s), nil
}

// SSEEvent represents a single Server-Sent Events message.
//
// Per the SSE spec (https://html.spec.whatwg.org/multipage/server-sent-events.html):
//   - Type maps to the event: field. If empty, the default message event fires.
//   - ID maps to the id: field. Browsers send it back via Last-Event-ID on reconnect.
//   - Data maps to the data: field. Multi-line data is split so each line gets
//     its own "data:" prefix (required by the spec).
//   - Retry maps to the retry: field, suggesting a reconnection interval in milliseconds.
type SSEEvent struct {
	Type  string     // event: field (optional)
	ID    SSEEventID // id: field (optional)
	Data  string     // data: field (can be multi-line)
	Retry int        // retry: field in ms (optional, 0 = omit)
}

// WriteSSEEvent writes a spec-correct SSE event to w.
// Multi-line Data is handled per spec: each line gets its own "data:" prefix.
// Blank line terminator is included.
func WriteSSEEvent(w io.Writer, evt SSEEvent) error {
	var buf bytes.Buffer

	if evt.Type != "" {
		buf.WriteString("event: ")
		buf.WriteString(evt.Type)
		buf.WriteByte('\n')
	}

	for line := range strings.SplitSeq(evt.Data, "\n") {
		buf.WriteString("data: ")
		buf.WriteString(line)
		buf.WriteByte('\n')
	}

	if evt.ID != "" {
		buf.WriteString("id: ")
		buf.WriteString(evt.ID.String())
		buf.WriteByte('\n')
	}

	if evt.Retry > 0 {
		buf.WriteString("retry: ")
		fmt.Fprintf(&buf, "%d", evt.Retry)
		buf.WriteByte('\n')
	}

	buf.WriteByte('\n')

	_, err := w.Write(buf.Bytes())

	return err
}

// WriteSSEHeartbeat writes a comment frame (SSE comment line).
// Browsers ignore it, but it keeps the connection alive through
// ALB/Nginx/Cloudflare idle timeouts.
func WriteSSEHeartbeat(w io.Writer) error {
	_, err := w.Write([]byte(": heartbeat\n\n"))

	return err
}
