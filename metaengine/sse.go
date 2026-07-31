package metaengine

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
)

// ServeSSE streams Watcher notifications as Server-Sent Events over HTTP.
// Each value change is JSON-encoded and sent as an SSE "data" event.
// The stream stays open until the client disconnects (request context cancelled).
//
// Usage:
//
//	watcher := NewWatcher[UserView](store, "users")
//	defer watcher.Close()
//	http.HandleFunc("/events/users", func(w http.ResponseWriter, r *http.Request) {
//	    metaengine.ServeSSE(w, r, watcher)
//	})
//
// Clients connect with EventSource in the browser:
//
//	const es = new EventSource("/events/users");
//	es.onmessage = (e) => console.log(JSON.parse(e.data));
func ServeSSE[V any](w http.ResponseWriter, r *http.Request, watcher *Watcher[V]) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("metaengine.ServeSSE: response writer does not support flushing")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ctx := r.Context()
	ch := watcher.Watch(ctx, nil)

	for {
		select {
		case <-ctx.Done():
			return nil

		case val, ok := <-ch:
			if !ok {
				return nil
			}

			data, err := json.Marshal(val)
			if err != nil {
				continue
			}

			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return err
			}

			flusher.Flush()
		}
	}
}

// Inspect returns a human-readable summary of all collections, their ADTs,
// read patterns, assigned engines, and complexity scores. Useful for CLI
// tools, debug endpoints, and startup diagnostics.
func (s *Store) Inspect() string {
	collections := s.Collections()

	if len(collections) == 0 {
		return "metaengine: no collections registered"
	}

	result := fmt.Sprintf("metaengine: %d collection(s)\n", len(collections))

	for _, c := range collections {
		result += fmt.Sprintf(
			"  %-20s  ADT=%-10s  pattern=%-20s  engine=%-15s  complexity=%s\n",
			c.Name, c.ADT, c.ReadPattern, c.EngineName, c.Complexity,
		)
	}

	return result
}
