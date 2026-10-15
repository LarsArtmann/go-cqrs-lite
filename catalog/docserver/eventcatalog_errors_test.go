package docserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// errorflowFixture writes a minimal erraudit errorflow export
// (schemaVersion 1) and returns its path.
func errorflowFixture(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "errorflow.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write errorflow fixture: %v", err)
	}

	return path
}

const errorflowFixtureBody = `{
	"schemaVersion": 1,
	"package": "example.com/demo",
	"filesScanned": 6,
	"errors": [
		{"name": "ErrEmptyOrder", "kind": "sentinel", "code": "order.empty", "family": "rejection", "status": "unreferenced", "message": "order is empty"},
		{"name": "ErrUnknownDomain", "kind": "sentinel", "code": "domain.unknown", "family": "rejection", "status": "unreferenced", "message": "unknown domain"},
		{"name": "ErrPaid", "kind": "sentinel", "code": "billing.paid", "family": "rejection", "status": "referenced", "message": "already paid", "wraps": "ErrInvoice"}
	]
}`

func TestEventCatalogErrorsPage_Renders(t *testing.T) {
	t.Parallel()

	srv := NewDocsServer(testProvider, Config{
		ServiceName:             "Test",
		Version:                 "1.0.0",
		EventCatalogErrorsFile:  errorflowFixture(t, errorflowFixtureBody),
	})
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	req := newTestRequest("/docs/eventcatalog/errors")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, want := range []string{
		"Error Surface",
		"ErrEmptyOrder", "ErrUnknownDomain", "ErrPaid",
		"order.empty", "billing.paid",
		"unreferenced", "referenced",
		"Dead sentinels",
		"2 error(s) are unreferenced",
		"files scanned",
		"example.com/demo",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
}

func TestEventCatalogErrorsPage_NotAttachedIsExplicit(t *testing.T) {
	t.Parallel()

	srv := NewDocsServer(testProvider, Config{ServiceName: "Test", Version: "1.0.0"})
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	req := newTestRequest("/docs/eventcatalog/errors")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "No error surface attached") {
		t.Error("expected explicit not-attached state, got something else")
	}
	if strings.Contains(body, "Dead sentinels") {
		t.Error("not-attached state must not render surface content")
	}
}

func TestEventCatalogErrorsPage_BrokenFileIsVisibleError(t *testing.T) {
	t.Parallel()

	srv := NewDocsServer(testProvider, Config{
		ServiceName: "Test",
		Version:     "1.0.0",
		// schemaVersion 2 is a future version — the reader must reject it,
		// and the page must say so instead of rendering an empty table.
		EventCatalogErrorsFile: errorflowFixture(t, `{"schemaVersion": 2, "errors": []}`),
	})
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	req := newTestRequest("/docs/eventcatalog/errors")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "Error surface unavailable") {
		t.Error("expected visible error state for schema mismatch")
	}
	if !strings.Contains(body, "schemaVersion") {
		t.Error("expected the underlying error to name the schema mismatch")
	}
	if strings.Contains(body, "Dead sentinels") {
		t.Error("broken-file state must not render surface content")
	}
}

// TestEventCatalogErrorsPage_DataProductCatalog pins the error-surface page
// rendering alongside a data-product-bearing catalog (the dataproduct build
// path must not interfere with the errors route).
func TestEventCatalogErrorsPage_DataProductCatalog(t *testing.T) {
	t.Parallel()

	srv := NewDocsServer(dataProductTestProvider, Config{
		ServiceName:            "Mesh API",
		Version:                "1.0.0",
		EventCatalogErrorsFile: errorflowFixture(t, errorflowFixtureBody),
	})
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	for _, path := range []string{
		"/docs/eventcatalog/errors",
		"/docs/eventcatalog/data-products/order-lifecycle",
	} {
		req := newTestRequest(path)
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s: expected 200, got %d", path, recorder.Code)
		}
	}

	req := newTestRequest("/docs/eventcatalog/errors")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	if body := recorder.Body.String(); !strings.Contains(body, "ErrEmptyOrder") {
		t.Error("errors page must render surface rows on a dataproduct catalog")
	}
}
