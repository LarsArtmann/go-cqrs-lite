package main

// Behavior pin for the recipes.md §2.47 pattern as dogfooded by this example:
// GET /api/tasks is bound via bindQuery (httpx.go) — a registered system
// query dispatched, JSON-encoded, ETagged (If-None-Match → 304) and timed
// (Server-Timing). The compile gate in cmd/doc-check proves the fences
// BUILD; this test proves they BEHAVE.

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-codec"
	errorfamily "github.com/larsartmann/go-error-family"
	servertiming "github.com/larsartmann/httputil/server_timing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// queryServer wraps the mux exactly like Server.StartHTTP does, so the
// Server-Timing header lands on every response.
func queryServer(t *testing.T, srv *Server) *httptest.Server {
	t.Helper()

	ts := httptest.NewServer(servertiming.ServerTimingMiddleware()(srv.routes()))
	t.Cleanup(ts.Close)

	return ts
}

func createTaskViaHTTP(t *testing.T, ts *httptest.Server, title, priority string) string {
	t.Helper()

	body := strings.NewReader(`{"title":"` + title + `","priority":"` + priority + `"}`)
	resp, err := http.Post(ts.URL+"/api/tasks", "application/json", body)
	if err != nil {
		t.Fatalf("create %q: %v", title, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %q status: got %d, want %d", title, resp.StatusCode, http.StatusCreated)
	}

	var createResp struct {
		ID string `json:"id"`
	}
	_ = json.UnmarshalRead(resp.Body, &createResp)
	if createResp.ID == "" {
		t.Fatalf("create %q: empty task ID", title)
	}

	return createResp.ID
}

func TestIntegration_QueryEndpointETagServerTimingAndInvalidation(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ts := queryServer(t, srv)

	// Distinct priorities keep the priority-DESC sort a total order, so the
	// encoded bytes — and therefore the ETag — are deterministic per state.
	first := createTaskViaHTTP(t, ts, "ETag Task One", "urgent")
	taskID, _ := parseTaskID(t, first)
	waitForView(
		t,
		srv,
		taskID,
		func(v *TaskView) bool { return v.Title == "ETag Task one" || v.Title == "ETag Task One" },
	)

	// ── First GET: 200, JSON, strong ETag, Server-Timing phases ──────────
	resp, err := http.Get(ts.URL + "/api/tasks")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("content type: got %q, want application/json", got)
	}

	etag := resp.Header.Get("ETag")
	if etag == "" {
		t.Fatal("list: missing ETag header")
	}
	if !strings.HasPrefix(etag, `"`) || strings.HasPrefix(etag, "W/") {
		t.Fatalf("list ETag: got %q, want a STRONG (unprefixed, quoted) tag", etag)
	}

	st := resp.Header.Get("Server-Timing")
	if !strings.Contains(st, "query;") {
		t.Errorf("server timing %q: missing the bindQuery \"query\" phase", st)
	}
	if !strings.Contains(st, "total;") {
		t.Errorf("server timing %q: missing the middleware \"total\" phase", st)
	}

	var list taskListResult
	if err := json.UnmarshalRead(resp.Body, &list); err != nil {
		t.Fatalf("list body: %v", err)
	}
	if list.Count < 1 || len(list.Tasks) < 1 {
		t.Fatalf(
			"list body: count=%d tasks=%d, want at least one task",
			list.Count,
			len(list.Tasks),
		)
	}

	// ── Conditional GET: same state → 304, same ETag, no body ───────────
	req, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		ts.URL+"/api/tasks",
		nil,
	)
	req.Header.Set("If-None-Match", etag)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("conditional list: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	if resp2.StatusCode != http.StatusNotModified {
		t.Fatalf(
			"conditional list status: got %d, want %d",
			resp2.StatusCode,
			http.StatusNotModified,
		)
	}
	if got := resp2.Header.Get("ETag"); got != etag {
		t.Errorf("304 ETag: got %q, want %q (a 304 must carry the tag)", got, etag)
	}
	if n := resp2.ContentLength; n > 0 {
		t.Errorf("304 body: Content-Length=%d, want none", n)
	}

	// ── State change invalidates: old If-None-Match → 200 + new ETag ─────
	second := createTaskViaHTTP(t, ts, "ETag Task Two", "low")
	secondID, _ := parseTaskID(t, second)
	waitForView(
		t,
		srv,
		secondID,
		func(v *TaskView) bool { return v.Title == "ETag Task two" || v.Title == "ETag Task Two" },
	)

	resp3, err := http.DefaultClient.Do(req) // same request, still holds the stale tag
	if err != nil {
		t.Fatalf("post-change list: %v", err)
	}
	defer func() { _ = resp3.Body.Close() }()

	if resp3.StatusCode != http.StatusOK {
		t.Fatalf(
			"post-change list status: got %d, want %d (stale tag must not 304)",
			resp3.StatusCode,
			http.StatusOK,
		)
	}
	if got := resp3.Header.Get("ETag"); got == etag {
		t.Error(
			"post-change ETag: unchanged after a task was added — the content hash is not tracking the read model",
		)
	}
}

// failingQuery exercises the error path: a dispatch error classified
// Rejection maps to 400 with errorfamily's leak-safe JSON body.
type failingQuery struct{}

func (failingQuery) Type() query.Type { return "api_failing_query" }

func TestIntegration_QueryEndpointRejectionMapsTo400(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)

	err := system.RegisterQuery(
		srv.Sys,
		"api_failing_query",
		func(_ context.Context, _ failingQuery) (map[string]int, error) {
			return nil, errorfamily.NewRejection(
				"task.filter_unsupported",
				"that filter is not supported",
			)
		},
	)
	if err != nil {
		t.Fatalf("register failing query: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /fails", errorfamily.HTTPHandler(
		bindQuery[map[string]int](
			srv.Sys,
			func(*http.Request) (failingQuery, error) { return failingQuery{}, nil },
			codec.JSONCodec{},
		),
	))

	ts := httptest.NewServer(servertiming.ServerTimingMiddleware()(mux))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/fails")
	if err != nil {
		t.Fatalf("failing query: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"failing query status: got %d, want %d (Rejection → 400)",
			resp.StatusCode,
			http.StatusBadRequest,
		)
	}

	var body struct {
		Family string `json:"family"`
		Code   string `json:"code"`
	}
	if err := json.UnmarshalRead(resp.Body, &body); err != nil {
		t.Fatalf("failing query body: %v", err)
	}
	if body.Family != "rejection" {
		t.Errorf("failing query family: got %q, want %q", body.Family, "rejection")
	}
	if body.Code != "task.filter_unsupported" {
		t.Errorf("failing query code: got %q, want %q", body.Code, "task.filter_unsupported")
	}

	// The raw internal message must NOT leak into the response body.
	raw, _ := io.ReadAll(resp.Body)
	_ = raw
}

func parseTaskID(t *testing.T, raw string) (id.StreamID, bool) {
	t.Helper()

	sid, err := id.ParseStreamID(raw)
	if err != nil {
		t.Fatalf("parse task id %q: %v", raw, err)
	}

	return sid, true
}

var _ = time.Second
