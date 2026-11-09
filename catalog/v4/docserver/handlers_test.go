package docserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestD2Handler_StatusAndContentType(t *testing.T) {
	t.Parallel()

	cat := testProvider()
	handler := D2Handler(cat)

	req := newTestRequest("/diagram.d2")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("expected text/plain, got %q", ct)
	}
}

func TestD2Handler_BodyContainsService(t *testing.T) {
	t.Parallel()

	cat := testProvider()
	handler := D2Handler(cat)

	req := newTestRequest("/diagram.d2")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "test-svc") && !strings.Contains(body, "Test API") {
		t.Error("expected D2 diagram to contain service identifier")
	}
}

func TestD2Handler_WithDescription(t *testing.T) {
	t.Parallel()

	cat := testProvider()
	handler := D2Handler(cat, WithD2Description("My architecture diagram"))

	req := newTestRequest("/diagram.d2")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHealthCheckHandler_Healthy(t *testing.T) {
	t.Parallel()

	cat := testProvider()
	handler := HealthCheckHandler(cat)

	req := newTestRequest("/health")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := decodeJSON(t, w)
	if body["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %v", body["status"])
	}

	services, ok := body["services"].(float64)
	if !ok {
		t.Fatalf("expected 'services' count in body, got %T", body["services"])
	}

	if services < 1 {
		t.Errorf("expected at least 1 service, got %v", services)
	}
}

func TestHealthCheckHandler_NilCatalog(t *testing.T) {
	t.Parallel()

	handler := HealthCheckHandler(nil)

	req := newTestRequest("/health")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}

	body := decodeJSON(t, w)
	if body["status"] != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got %v", body["status"])
	}

	if body["message"] != "catalog has no services" {
		t.Errorf("expected message 'catalog has no services', got %v", body["message"])
	}
}

func TestGenerateEventCatalog(t *testing.T) {
	t.Parallel()

	cat := testProvider()

	dir := t.TempDir()
	if err := GenerateEventCatalog(cat, dir); err != nil {
		t.Fatalf("GenerateEventCatalog failed: %v", err)
	}

	// Verify at least the services directory was created.
	if _, err := os.Stat(dir + "/services"); err != nil {
		t.Errorf("expected services directory to exist: %v", err)
	}
}

func TestGenerateEventCatalog_BadOutputDir(t *testing.T) {
	t.Parallel()

	cat := testProvider()

	// A path under /proc should be unwritable on Linux, triggering the error path.
	err := GenerateEventCatalog(cat, "/proc/cannot-write-here")
	if err == nil {
		t.Fatal("expected error writing to unwritable directory")
	}
}
