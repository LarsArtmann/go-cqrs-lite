// Package prometheus provides an OTel→Prometheus bridge for exposing CQRS
// metrics via the standard /metrics endpoint.
//
// This module wraps the OpenTelemetry Prometheus exporter into a convenient
// API. It creates a MeterProvider backed by a Prometheus registry, so all
// OTel instruments (including middleware.CommandOTelMetricsWithCounter and
// middleware.NewOTelMetricsRecorder) are automatically exposed as Prometheus
// metrics.
//
// # Quick Start
//
//	provider, handler, err := prometheus.Setup()
//	if err != nil { panic(err) }
//	defer provider.Shutdown(context.Background())
//
//	otel.SetMeterProvider(provider)
//
//	// Expose /metrics for Prometheus scraping
//	mux.Handle("/metrics", handler)
//
// # Custom Registry
//
//	reg := prometheus_client.NewRegistry()
//	provider, handler, err := prometheus.Setup(prometheus.WithRegistry(reg))
package prometheus
