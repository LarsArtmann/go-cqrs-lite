package prometheus_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	promClient "github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"

	cqrsprom "github.com/larsartmann/go-cqrs-lite/prometheus/v2"
)

func TestSetup_CreatesProviderAndHandler(t *testing.T) {
	t.Parallel()

	provider, err := cqrsprom.Setup()
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer func() { _ = provider.Shutdown(context.Background()) }()

	if provider.AsMeterProvider() == nil {
		t.Fatal("expected non-nil MeterProvider")
	}

	if provider.Handler() == nil {
		t.Fatal("expected non-nil HTTP handler")
	}
}

func TestSetup_WithCustomRegistry(t *testing.T) {
	t.Parallel()

	reg := promClient.NewRegistry()
	provider, err := cqrsprom.Setup(cqrsprom.WithRegistry(reg))
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer func() { _ = provider.Shutdown(context.Background()) }()

	counter, err := provider.AsMeterProvider().
		Meter("test").
		Int64Counter("test_counter")
	if err != nil {
		t.Fatalf("Int64Counter: %v", err)
	}

	counter.Add(context.Background(), 42)

	metricFamilies, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}

	found := false

	for _, family := range metricFamilies {
		if family.GetName() == "test_counter_total" {
			found = true

			if val := family.GetMetric()[0].GetCounter().GetValue(); val != 42 {
				t.Errorf("expected counter value 42, got %f", val)
			}
		}
	}

	if !found {
		t.Fatal("test_counter_total metric not found in registry")
	}
}

func TestHandler_ServesMetrics(t *testing.T) {
	t.Parallel()

	provider, err := cqrsprom.Setup()
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer func() { _ = provider.Shutdown(context.Background()) }()

	otel.SetMeterProvider(provider.AsMeterProvider())

	counter, err := otel.GetMeterProvider().Meter("test").
		Int64Counter("cqrs_test_total")
	if err != nil {
		t.Fatalf("Int64Counter: %v", err)
	}

	counter.Add(context.Background(), 7)

	ts := httptest.NewServer(provider.Handler())
	defer ts.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if !strings.Contains(string(body), "cqrs_test_total") {
		t.Errorf("expected metrics to contain cqrs_test_total, got:\n%s", body)
	}
}

func TestMustSetup_Succeeds(t *testing.T) {
	t.Parallel()

	p := cqrsprom.MustSetup()
	defer func() { _ = p.Shutdown(context.Background()) }()

	if p.AsMeterProvider() == nil {
		t.Fatal("expected non-nil MeterProvider from MustSetup")
	}
}

func TestHandler_EmptyInitially(t *testing.T) {
	t.Parallel()

	provider, err := cqrsprom.Setup()
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer func() { _ = provider.Shutdown(context.Background()) }()

	ts := httptest.NewServer(provider.Handler())
	defer ts.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestSetup_HistogramMetrics(t *testing.T) {
	t.Parallel()

	reg := promClient.NewRegistry()
	provider, err := cqrsprom.Setup(cqrsprom.WithRegistry(reg))
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer func() { _ = provider.Shutdown(context.Background()) }()

	meter := provider.AsMeterProvider().Meter("test-app")
	hist, err := meter.Float64Histogram("cqrs_latency_ms")
	if err != nil {
		t.Fatalf("Float64Histogram: %v", err)
	}

	hist.Record(context.Background(), 15.5)
	hist.Record(context.Background(), 25.0)
	hist.Record(context.Background(), 5.0)

	metricFamilies, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}

	for _, family := range metricFamilies {
		if family.GetName() == "cqrs_latency_ms" {
			if len(family.GetMetric()) != 1 {
				t.Fatalf("expected 1 metric, got %d", len(family.GetMetric()))
			}

			histData := family.GetMetric()[0].GetHistogram()
			if histData.GetSampleCount() != 3 {
				t.Errorf("expected 3 samples, got %d", histData.GetSampleCount())
			}

			return
		}
	}

	t.Fatal("cqrs_latency_ms metric not found")
}
