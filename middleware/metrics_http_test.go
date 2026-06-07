package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMetricsCollector(t *testing.T) {
	col := NewMetricsCollector()

	col.RecordRequest(http.StatusOK, 100*time.Millisecond)
	col.RecordRequest(http.StatusOK, 200*time.Millisecond)
	col.RecordRequest(http.StatusInternalServerError, 50*time.Millisecond)

	snap := col.Snapshot()

	if snap.RequestsTotal != 3 {
		t.Fatalf("expected 3 requests, got %d", snap.RequestsTotal)
	}

	if snap.RequestsError != 1 {
		t.Fatalf("expected 1 error, got %d", snap.RequestsError)
	}

	if snap.AvgResponseMs < 80 || snap.AvgResponseMs > 120 {
		t.Fatalf("expected avg ~100ms, got %f", snap.AvgResponseMs)
	}

	if snap.Goroutines < 1 {
		t.Fatal("expected at least 1 goroutine")
	}
}

func TestMetricsHandler(t *testing.T) {
	col := NewMetricsCollector()
	col.RecordRequest(http.StatusOK, 10*time.Millisecond)

	handler := MetricsHandler(col)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var snap MetricsSnapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if snap.RequestsTotal != 1 {
		t.Fatalf("expected 1 request, got %d", snap.RequestsTotal)
	}
}

func TestMetricsMiddleware(t *testing.T) {
	col := NewMetricsCollector()
	mw := MetricsMiddleware(col)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	snap := col.Snapshot()
	if snap.RequestsTotal != 1 {
		t.Fatalf("expected 1 request, got %d", snap.RequestsTotal)
	}
}
