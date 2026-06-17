package middleware_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/middleware/v2"
)

func TestProfilingHandler_AllEndpoints(t *testing.T) {
	t.Parallel()

	handler := middleware.ProfilingHandler("/debug/pprof/")
	server := httptest.NewServer(handler)
	defer server.Close()

	client := server.Client()

	// Sequential — parallel subtests race with defer server.Close().
	for _, endpoint := range []string{
		"/debug/pprof/",
		"/debug/pprof/heap",
		"/debug/pprof/goroutine",
		"/debug/pprof/allocs",
		"/debug/pprof/block",
		"/debug/pprof/mutex",
		"/debug/pprof/threadcreate",
		"/debug/pprof/cmdline",
		"/debug/pprof/symbol",
	} {
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			server.URL+endpoint,
			nil,
		)
		if err != nil {
			t.Fatalf("NewRequest %s: %v", endpoint, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("GET %s: %v", endpoint, err)
		}

		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: got status %d, want %d", endpoint, resp.StatusCode, http.StatusOK)
		}

		if len(body) == 0 {
			t.Fatalf("%s: empty response body", endpoint)
		}
	}
}

func TestRegisterProfiling(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	middleware.RegisterProfiling(mux)

	server := httptest.NewServer(mux)
	defer server.Close()

	client := server.Client()

	for _, endpoint := range []string{
		"/debug/pprof/",
		"/debug/pprof/heap",
		"/debug/pprof/goroutine",
	} {
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			server.URL+endpoint,
			nil,
		)
		if err != nil {
			t.Fatalf("NewRequest %s: %v", endpoint, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("GET %s: %v", endpoint, err)
		}

		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: got status %d, want %d", endpoint, resp.StatusCode, http.StatusOK)
		}
	}
}
