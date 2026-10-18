package docserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// TestDocsServer_ArchitectureReference pins the opt-in route: 200 with an
// HTML reference containing a known section anchor, 404 when not enabled.
func TestDocsServer_ArchitectureReference(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		srv := NewDocsServer(testProvider, Config{
			ServiceName:           "Test",
			Version:               "1.0.0",
			ArchitectureReference: true,
		})

		mux := http.NewServeMux()
		srv.RegisterRoutes(mux)

		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, newTestRequest("/docs/architecture"))

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}

		ct := recorder.Header().Get("Content-Type")
		if !strings.Contains(ct, "text/html") {
			t.Errorf("content type: %q, want text/html", ct)
		}

		body := recorder.Body.String()

		for _, want := range []string{
			"<!doctype html>",
			`id="service-`,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("reference is missing %q", want)
			}
		}
	})

	t.Run("disabled by default", func(t *testing.T) {
		srv := NewDocsServer(testProvider, Config{
			ServiceName: "Test",
			Version:     "1.0.0",
		})

		mux := http.NewServeMux()
		srv.RegisterRoutes(mux)

		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, newTestRequest("/docs/architecture"))

		if recorder.Code != http.StatusNotFound {
			t.Errorf("expected 404 when not opted in, got %d", recorder.Code)
		}
	})
}

// TestDocsServer_ArchitectureCacheInvalidation pins the invalidation half:
// a changed catalog instance must change the served bytes.
func TestDocsServer_ArchitectureCacheInvalidation(t *testing.T) {
	cat := testProvider()

	srv := NewDocsServer(func() *catalog.Catalog { return cat }, Config{
		ServiceName:           "Test",
		Version:               "1.0.0",
		ArchitectureReference: true,
	})

	handler := srv.ArchitectureReferenceHandler()

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, newTestRequest("/docs/architecture"))

	// A new catalog instance with an extra service must invalidate.
	builder := catalog.NewBuilder("Test", "1.0.0")
	builder.AddService("extra-service", "Extra", "1.0.0", "Added after the first render.")

	cat = builder.Build()

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, newTestRequest("/docs/architecture"))

	if second.Body.String() == first.Body.String() {
		t.Error("a new catalog instance must re-render, not serve stale bytes")
	}

	if !strings.Contains(second.Body.String(), `id="service-extra-service"`) {
		t.Error("the re-render lost the added service")
	}
}
