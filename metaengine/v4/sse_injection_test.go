package metaengine

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// SSE framing-injection pin (testing suite T25).
//
// A read-model value is caller data. If a value containing CRLF can end the
// current event early, the SENDER controls the wire framing: it can forge an
// `id:` (breaking Last-Event-ID reconnect for every later event on the
// connection) or an `event:` type. That is a protocol injection, not a
// formatting nit — and it is invisible in the happy-path SSE tests, which use
// values without newlines.
//
// The writer must emit the payload as SSE `data:` fields only, so a newline in
// the VALUE becomes a data-field separator, never a record separator.
func TestSSE_PayloadCannotForgeEventBoundary(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()

	store, err := Plan([]Engine{eng}, testTaskQuery())
	if err != nil {
		t.Fatal(err)
	}

	watcher := NewWatcher[testTask](store, "tasks")

	// Replay attached on purpose: ServeSSE writes the response headers BEFORE
	// it calls watcher.Watch (metaengine/sse.go), so a test that applies the
	// value "after the client connected" can still race the subscription. A
	// recorded value is replayed to a late subscriber, which makes this test
	// independent of that ordering — and needs no sleep to paper over it.
	watcher.WithReplay(100)

	defer watcher.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		_ = ServeSSE(w, r, watcher, WithSSETimeout(5*time.Second))
	})

	srv := &http.Server{Handler: mux}
	defer srv.Close()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv.Addr = ln.Addr().String()

	go func() { _ = srv.Serve(ln) }()

	conn, err := net.Dial("tcp", srv.Addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("GET /events HTTP/1.0\r\nHost: localhost\r\n\r\n")); err != nil {
		t.Fatalf("write request: %v", err)
	}

	forged := "innocuous\r\n\r\nid: 999\r\nevent: forged\r\ndata: injected"
	if err := store.Apply(
		ctx,
		"task_created",
		testTask{ID: "inject-1", Title: forged},
	); err != nil {
		t.Fatalf("apply: %v", err)
	}

	raw := readUntil(t, conn, "injected")

	// Assert on LINE STARTS, not substrings: the value is JSON-encoded, so the
	// payload's CRLF arrives as the two-character escape `\r\n` INSIDE one
	// `data:` line. A substring check would fire on that escaped text and
	// report a fake vulnerability (it did, first run). What matters is that no
	// REAL line begins with a field name the payload chose.
	body := raw[strings.Index(raw, "\r\n\r\n"):]
	for _, forgedLine := range []string{"\nid: 999", "\nevent: forged"} {
		if strings.Contains(body, forgedLine) {
			t.Fatalf("payload forged the SSE line %q — a newline escaped its data field:\n%q",
				strings.TrimPrefix(forgedLine, "\n"), raw)
		}
	}

	// Exactly one event block must have been written for one applied value: a
	// blank line terminates an event, so the body must contain exactly one.
	// (This watcher has no replay attached, so no `id:` field is written —
	// counting blank lines is the boundary metric that works in both shapes.)
	if got := strings.Count(body, "\n\n"); got != 1 {
		t.Fatalf("wrote %d event blocks for one applied value, want 1:\n%q", got, raw)
	}
}

// readUntil accumulates conn output until needle appears or the deadline
// passes, then returns everything read.
func readUntil(t *testing.T, conn net.Conn, needle string) string {
	t.Helper()

	var out string

	buf := make([]byte, 8192)
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		if err := conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
			return out
		}

		n, err := conn.Read(buf)
		if n > 0 {
			out += string(buf[:n])

			if strings.Contains(out, needle) {
				return out
			}
		}

		if err != nil {
			continue
		}
	}

	t.Fatalf("timed out waiting for %q in the SSE stream; got:\n%q", needle, out)

	return out
}
