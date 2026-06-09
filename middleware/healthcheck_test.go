package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testVersion = "v2.2.0"

func TestHealthCheckHandler_Live(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{"LiveEndpoint", "/health/live"},
		{"DefaultEndpoint", "/health"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := HealthCheckHandler(testVersion)

			req := httptest.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				tt.path,
				nil,
			)
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
		})
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

	handler := HealthCheckHandler(testVersion, checker)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/health/ready",
		nil,
	)
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

	handler := HealthCheckHandler(testVersion, checker)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/health/ready",
		nil,
	)
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
