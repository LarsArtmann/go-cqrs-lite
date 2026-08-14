package main

import (
	"bufio"
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gomust "github.com/larsartmann/go-must"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

const (
	sseTestTitle   = "SSE Stream Test"
	sseTestTimeout = 6 * time.Second
)

// TestIntegration_SSEStreamsTaskViewUpdates verifies the /events endpoint:
// a client that subscribes before a command is dispatched receives the
// projected TaskView as an SSE data event. The endpoint is served by
// metaengine.ServeSSE on the task_views watcher (go-sse under the hood,
// ADR-0127) — proving the watcher fires for metaengine auto-projections.
func TestIntegration_SSEStreamsTaskViewUpdates(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)

	// Subscribe FIRST — the watcher only notifies on changes after connect.
	ctx, cancel := context.WithTimeout(context.Background(), sseTestTimeout)
	t.Cleanup(cancel)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/events", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect SSE stream: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	taskID := id.NewStreamID()

	if err := srv.CmdDisp.Dispatch(context.Background(), CreateTaskCmd{
		BasicCommand: gomust.Must(command.New(cmdCreateTask, taskID)),
		Title:        sseTestTitle,
		Priority:     PriorityHigh,
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}

	var last TaskView
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue // skip heartbeat comments, id:, and retry: lines
		}

		payload := strings.TrimPrefix(line, "data: ")

		if err := json.Unmarshal([]byte(payload), &last); err != nil {
			t.Fatalf("decode SSE data %q: %v", payload, err)
		}

		if last.ID == taskID.String() && last.Title == sseTestTitle {
			return // pass: projected view arrived over SSE
		}
	}

	t.Fatalf("SSE stream ended before TaskView arrived (err=%v, last=%+v)", scanner.Err(), last)
}
