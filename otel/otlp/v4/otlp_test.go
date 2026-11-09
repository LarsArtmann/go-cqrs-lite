package otlp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	cqrsotlp "github.com/larsartmann/go-cqrs-lite/otel/otlp/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// collectorStub is an OTLP/HTTP collector stand-in: it records the paths of
// POST requests (and headers on the first one) without parsing protobuf.
type collectorStub struct {
	mu      sync.Mutex
	paths   map[string]int
	headers map[string]http.Header
}

func (c *collectorStub) handler(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.paths == nil {
		c.paths = map[string]int{}
		c.headers = map[string]http.Header{}
	}

	if r.Method == http.MethodPost {
		c.paths[r.URL.Path]++
		c.headers[r.URL.Path] = r.Header.Clone()
	}

	w.WriteHeader(http.StatusOK)
}

func (c *collectorStub) count(path string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.paths[path]
}

func (c *collectorStub) header(path string) http.Header {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.headers[path]
}

// SetupOTLP ships both signals over OTLP/HTTP to the configured endpoint:
// after a span and a counter are recorded, Shutdown (which flushes) leaves
// one export request on each collector path — with the configured auth
// headers attached.
func TestSetupOTLP_ExportsTracesAndMetrics(t *testing.T) {
	t.Parallel()

	stub := &collectorStub{}
	server := httptest.NewServer(http.HandlerFunc(stub.handler))
	t.Cleanup(server.Close)

	ctx := context.Background()

	provider, err := cqrsotlp.SetupOTLP(ctx, cqrsotlp.OTLPConfig{
		Endpoint:    server.Listener.Addr().String(),
		Insecure:    true,
		Headers:     map[string]string{"x-scope-token": "secret"},
		ServiceName: "otlp-test",
	}, cqrsotel.WithoutGlobalRegistration())
	if err != nil {
		t.Fatal(err)
	}

	tracer := provider.AsTracerProvider().Tracer("cqrs/test")
	_, span := tracer.Start(ctx, "cqrs.test.span")
	span.End()

	counter, err := provider.AsMeterProvider().Meter("cqrs/test").Int64Counter("cqrs.test.count")
	if err != nil {
		t.Fatal(err)
	}

	counter.Add(ctx, 1)

	if err := provider.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	if got := stub.count("/v1/traces"); got < 1 {
		t.Fatalf("collector saw %d trace exports, want >= 1", got)
	}

	if got := stub.count("/v1/metrics"); got < 1 {
		t.Fatalf("collector saw %d metric exports, want >= 1", got)
	}

	if got := stub.header("/v1/traces").Get("X-Scope-Token"); got != "secret" {
		t.Fatalf("trace export auth header = %q, want %q", got, "secret")
	}
}

// Options appended after the OTLP wiring win: a custom metric reader
// replaces the OTLP one, so metrics land in the manual reader and no metric
// export ever reaches the collector.
func TestSetupOTLP_TrailingOptionsOverride(t *testing.T) {
	t.Parallel()

	stub := &collectorStub{}
	server := httptest.NewServer(http.HandlerFunc(stub.handler))
	t.Cleanup(server.Close)

	ctx := context.Background()

	manual := sdkmetric.NewManualReader()

	provider, err := cqrsotlp.SetupOTLP(ctx, cqrsotlp.OTLPConfig{
		Endpoint: server.Listener.Addr().String(),
		Insecure: true,
	}, cqrsotel.WithMetricReader(manual), cqrsotel.WithoutGlobalRegistration())
	if err != nil {
		t.Fatal(err)
	}

	counter, err := provider.AsMeterProvider().Meter("cqrs/test").Int64Counter("cqrs.test.count")
	if err != nil {
		t.Fatal(err)
	}

	counter.Add(ctx, 1)

	var rm metricdata.ResourceMetrics
	if err := manual.Collect(ctx, &rm); err != nil {
		t.Fatal(err)
	}

	found := slices.ContainsFunc(rm.ScopeMetrics, func(sm metricdata.ScopeMetrics) bool {
		return slices.ContainsFunc(sm.Metrics, func(m metricdata.Metrics) bool {
			return m.Name == "cqrs.test.count"
		})
	})

	if !found {
		t.Fatal("manual reader did not receive the counter — override lost")
	}

	// Shutdown AFTER collecting: it closes the manual reader too.
	if err := provider.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	if got := stub.count("/v1/metrics"); got != 0 {
		t.Fatalf("collector saw %d metric exports despite reader override", got)
	}
}
