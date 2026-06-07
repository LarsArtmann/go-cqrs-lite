package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheckHandler_Live(t *testing.T) {
	handler := HealthCheckHandler("v2.2.0")

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health/live", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp HealthCheckResponse

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != HealthStatusPass {
		t.Fatalf("expected status pass, got %s", resp.Status)
	}

	if _, ok := resp.Checks["liveness"]; !ok {
		t.Fatal("expected liveness check")
	}
}

func TestHealthCheckHandler_Ready(t *testing.T) {
	checker := func(_ context.Context) Check {
		return Check{
			ComponentID:   "db",
			ComponentType: "datastore",
			Status:        HealthStatusPass,
		}
	}

	handler := HealthCheckHandler("v2.2.0", checker)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp HealthCheckResponse

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != HealthStatusPass {
		t.Fatalf("expected status pass, got %s", resp.Status)
	}

	if _, ok := resp.Checks["db"]; !ok {
		t.Fatal("expected db check")
	}
}

func TestHealthCheckHandler_ReadyFail(t *testing.T) {
	checker := func(_ context.Context) Check {
		return Check{
			ComponentID:   "db",
			ComponentType: "datastore",
			Status:        HealthStatusFail,
		}
	}

	handler := HealthCheckHandler("v2.2.0", checker)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}

	var resp HealthCheckResponse

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != HealthStatusFail {
		t.Fatalf("expected status fail, got %s", resp.Status)
	}
}

func TestHealthCheckHandler_Default(t *testing.T) {
	handler := HealthCheckHandler("v2.2.0")

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp HealthCheckResponse

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != HealthStatusPass {
		t.Fatalf("expected status pass, got %s", resp.Status)
	}

	if _, ok := resp.Checks["liveness"]; !ok {
		t.Fatal("expected liveness check")
	}
}
