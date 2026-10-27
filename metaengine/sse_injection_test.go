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

	// Let the subscription attach before the value is applied, otherwise the
	// event may be missed and the assertion would pass for the wrong reason.
	time.Sleep(200 * time.Millisecond)

	forged := "innocuous\r\n\r\nid: 999\r\nevent: forged\r\ndata: injected"
	if err := store.Apply(ctx, "task_created", testTask{ID: "inject-1", Title: forged}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	raw := readUntil(t, conn, "injected")

	if strings.Contains(raw, "id: 999") {
		t.Fatalf("payload forged an SSE id field — the value escaped its data field:\n%q", raw)
	}

	if strings.Contains(raw, "event: forged") {
		t.Fatalf("payload forged an SSE event type — the value escaped its data field:\n%q", raw)
	}

	// Exactly one event block (one `id:` line) must have been written for one
	// applied value. A forged boundary would produce a second block.
	if got := strings.Count(raw, "\nid: "); got != 1 {
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
