package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHardening_RecoveryTurnsPanicInto500 pins the outermost middleware: a
// handler panic reaches the client as a 500 (logged, connection drained),
// never a dropped connection — via the exact wrapHandler chain StartHTTP
// installs.
func TestHardening_RecoveryTurnsPanicInto500(t *testing.T) {
	t.Parallel()

	srv := &Server{Logger: slog.New(slog.NewTextHandler(&strings.Builder{}, nil))}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /boom", func(http.ResponseWriter, *http.Request) {
		panic("handler exploded")
	})

	ts := httptest.NewServer(srv.wrapHandler(mux))
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/boom")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("panic status: got %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

// TestHardening_OversizedBodyReturns413 pins the request-body ceiling: a
// body larger than maxBodySize is rejected with 413 on a real decode path.
func TestHardening_OversizedBodyReturns413(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)

	ts := httptest.NewServer(srv.wrapHandler(srv.routes()))
	t.Cleanup(ts.Close)

	big := strings.NewReader(`{"title":"` + strings.Repeat("x", 2*int(maxBodyBytes)) + `"}`)

	resp, err := http.Post(ts.URL+"/api/tasks", "application/json", big)
	if err != nil {
		t.Fatalf("post oversized body: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body status: got %d, want %d", resp.StatusCode, http.StatusRequestEntityTooLarge)
	}
}

// TestHardening_ServerTimingGatedByConfig pins that the Server-Timing header
// is only emitted when the deployment opted in (Config.ServerTiming) — the
// header is a debugging affordance, not a default tax.
func TestHardening_ServerTimingGatedByConfig(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)

	off := httptest.NewServer(srv.wrapHandler(srv.routes()))
	t.Cleanup(off.Close)

	resp, err := http.Get(off.URL + "/health")
	if err != nil {
		t.Fatalf("get (timing off): %v", err)
	}
	_ = resp.Body.Close()

	if got := resp.Header.Get("Server-Timing"); got != "" {
		t.Fatalf("timing off: Server-Timing header leaked: %q", got)
	}

	srv.timingEnabled = true

	on := httptest.NewServer(srv.wrapHandler(srv.routes()))
	t.Cleanup(on.Close)

	resp2, err := http.Get(on.URL + "/health")
	if err != nil {
		t.Fatalf("get (timing on): %v", err)
	}
	_ = resp2.Body.Close()

	if got := resp2.Header.Get("Server-Timing"); got == "" {
		t.Fatal("timing on: Server-Timing header missing")
	}
}

// TestHardening_ReadinessProbeReportsStoreHealth pins GET /health/ready:
// with the in-memory SQLite stack healthy, the probe answers 200; the
// underlying s.ready() consults system.HealthCheck (engines + projections).
func TestHardening_ReadinessProbeReportsStoreHealth(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)

	if !srv.ready() {
		t.Fatal("ready(): got false on a healthy in-memory stack, want true")
	}

	ts := httptest.NewServer(srv.wrapHandler(srv.routes()))
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/health/ready")
	if err != nil {
		t.Fatalf("get ready: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ready status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
