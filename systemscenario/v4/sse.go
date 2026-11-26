package systemscenario

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-sse/sseparse"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// SSEStream receives Server-Sent Events streamed from one projection
// collection over a real HTTP round trip (an in-process httptest server
// serving [metaengine.ServeSSE] plus a streaming client), decoded as V.
// Tests assert on the received values with [SSEStream.Await].
type SSEStream[V any] struct {
	t       testing.TB
	watcher *metaengine.Watcher[V]

	mu      sync.Mutex
	vals    []V
	ids     []string
	lastErr error
}

// SubscribeSSE serves collection from the scenario's projection store via
// [metaengine.ServeSSE] on an in-process HTTP server and connects as its
// first client, recording every streamed value as V. The stream lives for
// the rest of the test: the response body, server, and watcher are closed
// via t.Cleanup.
//
// Subscribe BEFORE the Given/When acts whose projection changes you want to
// observe — notifications fire on writes, so a subscription that starts
// after the fold has settled sees nothing.
//
//	systemscenario.SubscribeSSE[TaskView](t, ctx, sc, "task_views")
//
// The wire path is the production one: watcher hub → ServeSSE JSON encoding
// → HTTP stream → client-side SSE parsing, so an assertion passing here
// means a browser EventSource on the same endpoint would see the value too.
func SubscribeSSE[V any](
	tb testing.TB, ctx context.Context, sc *Scenario, collection string,
) *SSEStream[V] {
	tb.Helper()

	store := sc.System().MetaEngine()
	if store == nil {
		tb.Fatalf("SubscribeSSE: scenario has no projection store (System().MetaEngine() is nil)")
	}

	watcher := metaengine.NewWatcher[V](store, collection)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = metaengine.ServeSSE(w, r, watcher)
	}))

	stream := &SSEStream[V]{t: tb, watcher: watcher}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		tb.Fatalf("SubscribeSSE: build request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		server.Close()
		tb.Fatalf("SubscribeSSE: connect to %s: %v", server.URL, err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		server.Close()
		tb.Fatalf("SubscribeSSE: status %s, want 200 OK", resp.Status)
	}

	go stream.readLoop(resp.Body)

	tb.Cleanup(func() {
		_ = resp.Body.Close()
		server.Close()
		watcher.Close()
	})

	return stream
}

// readLoop parses the SSE wire format via sseparse (the fleet's shared
// conformance-tested parser — the same one go-sse's own suite runs), so the
// harness exercises spec-true framing: comment/heartbeat frames never
// dispatch, multi-line data joins with \n, and id:/retry: fields are
// surfaced for diagnostics instead of being silently dropped.
// Line bounds: events are small JSON documents, but a payload-heavy
// projection row must not truncate the stream silently.
const sseMaxLineBytes = 1024 * 1024

func (s *SSEStream[V]) readLoop(body io.Reader) {
	sr := sseparse.NewStreamReader(body, sseparse.WithMaxLineBytes(sseMaxLineBytes))

	for {
		evt, err := sr.Next()
		if err != nil {
			s.mu.Lock()
			s.lastErr = err
			s.mu.Unlock()

			return
		}

		s.record(evt)
	}
}

// record decodes one SSE event's data payload as V and appends it to the
// received values (remembering the wire id: for diagnostics); decode
// failures are recorded as stream errors instead of failing the test, so
// Await's timeout message can show the real wire payload.
func (s *SSEStream[V]) record(evt sseparse.Event) {
	var val V

	payload := strings.Join(evt.DataLines, "\n")

	if err := json.Unmarshal([]byte(payload), &val); err != nil {
		s.mu.Lock()
		s.lastErr = fmt.Errorf("decode SSE payload %q as %T: %w", payload, val, err)
		s.mu.Unlock()

		return
	}

	s.mu.Lock()
	s.vals = append(s.vals, val)
	s.ids = append(s.ids, evt.ID)
	s.mu.Unlock()
}

// Received returns the values streamed so far, in arrival order.
func (s *SSEStream[V]) Received() []V {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.vals)
}

// ReceivedIDs returns the wire id: fields of the streamed events, in
// arrival order (empty strings when the serving watcher has no replay
// journal — ServeSSE only stamps ids on the reconnection path).
func (s *SSEStream[V]) ReceivedIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.ids)
}

// Await polls the received values until one satisfies match, returning it.
// On timeout it fails the test with what, the await window, and every value
// received so far (plus any stream error), so a missing value and a broken
// stream are distinguishable in the failure output.
func (s *SSEStream[V]) Await(
	tb testing.TB, within time.Duration, what string, match func(V) bool,
) V {
	tb.Helper()

	deadline := time.Now().Add(within)

	for {
		received := s.Received()

		for _, val := range received {
			if match(val) {
				return val
			}
		}

		if time.Now().After(deadline) {
			s.mu.Lock()
			lastErr := s.lastErr
			s.mu.Unlock()

			detail := fmt.Sprintf("received %d values, none matched: %+v", len(received), received)
			if lastErr != nil {
				detail += fmt.Sprintf("; stream error: %v", lastErr)
			}

			if ids := s.ReceivedIDs(); len(ids) > 0 {
				detail += fmt.Sprintf("; wire ids: %v", ids)
			}

			tb.Fatalf("SSEStream.Await: %s not received within %s: %s", what, within, detail)
		}

		time.Sleep(pollInterval)
	}
}
