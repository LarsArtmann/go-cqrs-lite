package middleware

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// MetricsSnapshot holds a point-in-time view of runtime metrics.
type MetricsSnapshot struct {
	Timestamp     string  `json:"timestamp"`
	UptimeSeconds float64 `json:"uptime_seconds"`
	RequestsTotal uint64  `json:"requests_total"`
	RequestsError uint64  `json:"requests_error"`
	AvgResponseMs float64 `json:"avg_response_ms"`
	Goroutines    int     `json:"goroutines"`
	MemoryAllocMB float64 `json:"memory_alloc_mb"`
	MemorySysMB   float64 `json:"memory_sys_mb"`
	GCCount       uint32  `json:"gc_count"`
}

// MetricsCollector tracks HTTP request metrics.
type MetricsCollector struct {
	startTime     time.Time
	requestsTotal atomic.Uint64
	requestsError atomic.Uint64
	responseSum   atomic.Uint64 // sum of response times in microseconds
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime: time.Now(),
	}
}

// RecordRequest records a completed HTTP request.
func (m *MetricsCollector) RecordRequest(statusCode int, duration time.Duration) {
	m.requestsTotal.Add(1)
	m.responseSum.Add(uint64(duration.Microseconds()))
	if statusCode >= 400 {
		m.requestsError.Add(1)
	}
}

// Snapshot returns the current metrics snapshot.
func (m *MetricsCollector) Snapshot() MetricsSnapshot {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	total := m.requestsTotal.Load()
	var avgMs float64
	if total > 0 {
		avgMs = float64(m.responseSum.Load()) / float64(total) / 1000.0
	}

	return MetricsSnapshot{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		UptimeSeconds: time.Since(m.startTime).Seconds(),
		RequestsTotal: total,
		RequestsError: m.requestsError.Load(),
		AvgResponseMs: avgMs,
		Goroutines:    runtime.NumGoroutine(),
		MemoryAllocMB: float64(mem.Alloc) / 1024.0 / 1024.0,
		MemorySysMB:   float64(mem.Sys) / 1024.0 / 1024.0,
		GCCount:       mem.NumGC,
	}
}

// MetricsHandler returns an HTTP handler that exposes metrics.
func MetricsHandler(collector *MetricsCollector) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(collector.Snapshot())
	})
}

// MetricsMiddleware wraps an HTTP handler to collect request metrics.
func MetricsMiddleware(collector *MetricsCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rec, r)
			collector.RecordRequest(rec.statusCode, time.Since(start))
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}
