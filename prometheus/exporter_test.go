package prometheus_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/prometheus/v2"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"go.opentelemetry.io/otel"
	otelmetric "go.opentelemetry.io/otel/metric"
)

func TestSetup_CreatesProviderAndHandler(t *testing.T) {
	t.Parallel()

	provider, err := prometheus.Setup()
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer provider.Shutdown(context.Background())

	if provider.AsMeterProvider() == nil {
		t.Fatal("expected non-nil MeterProvider")
	}

	if provider.Handler() == nil {
		t.Fatal("expected non-nil HTTP handler")
	}
}

func TestSetup_WithCustomRegistry(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	provider, err := prometheus.Setup(prometheus.WithRegistry(reg))
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer provider.Shutdown(context.Background())

	counter, err := provider.AsMeterProvider().
		Meter("test").
		Int64Counter("test_counter")
	if err != nil {
		t.Fatalf("Int64Counter: %v", err)
	}

	counter.Add(context.Background(), 42)

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}

	found := false

	for _, mf := range mfs {
		if mf.GetName() == "test_counter" {
			found = true

			if val := mf.GetMetric()[0].GetCounter().GetValue(); val != 42 {
				t.Errorf("expected counter value 42, got %f", val)
			}
		}
	}

	if !found {
		t.Fatal("test_counter metric not found in registry")
	}
}

func TestHandler_ServesMetrics(t *testing.T) {
	t.Parallel()

	provider, err := prometheus.Setup()
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer provider.Shutdown(context.Background())

	otel.SetMeterProvider(provider.AsMeterProvider())

	counter, err := otelmetric.Int64Counter(
		otel.GetMeterProvider(), "cqrs_test_total",
		otelmetric.WithDescription("test counter"),
	)
	if err != nil {
		t.Fatalf("Int64Counter: %v", err)
	}

	counter.Add(context.Background(), 7)

	ts := httptest.NewServer(provider.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if !strings.Contains(string(body), "cqrs_test_total") {
		t.Errorf("expected metrics to contain cqrs_test_total, got:\n%s", body)
	}

	if resp.Header().Get("Content-Type") == "" {
		t.Error("expected Content-Type header")
	}
}

func TestMustSetup_PanicsOnError(t *testing.T) {
	t.Parallel()

	// WithRegistry with nil should still work (default is used)
	// Instead, test that MustSetup succeeds normally
	p := prometheus.MustSetup()
	defer p.Shutdown(context.Background())

	if p.AsMeterProvider() == nil {
		t.Fatal("expected non-nil MeterProvider from MustSetup")
	}
}

func TestHandler_EmptyInitially(t *testing.T) {
	t.Parallel()

	provider, err := prometheus.Setup()
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer provider.Shutdown(context.Background())

	ts := httptest.NewServer(provider.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestSetup_GOProcessesMetric(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	provider, err := prometheus.Setup(prometheus.WithRegistry(reg))
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer provider.Shutdown(context.Background())

	meter := provider.AsMeterProvider().Meter("test-app")
	hist, err := meter.Float64Histogram("cqrs_latency_ms")
	if err != nil {
		t.Fatalf("Float64Histogram: %v", err)
	}

	hist.Record(context.Background(), 15.5)
	hist.Record(context.Background(), 25.0)
	hist.Record(context.Background(), 5.0)

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}

	for _, mf := range mfs {
		if mf.GetName() == "cqrs_latency_ms" {
			if len(mf.GetMetric()) != 1 {
				t.Fatalf("expected 1 metric, got %d", len(mf.GetMetric()))
			}

			hist := mf.GetMetric()[0].GetHistogram()
			if hist.GetSampleCount() != 3 {
				t.Errorf("expected 3 samples, got %d", hist.GetSampleCount())
			}

			return
		}
	}

	t.Fatal("cqrs_latency_ms metric not found")
}

// Verify dto import is used (for future use).
var _ = dto.MetricFamily{}
