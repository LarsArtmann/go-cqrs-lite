// Package otel provides shared OpenTelemetry instrumentation utilities for go-cqrs-lite.
//
// This module centralizes tracer/meter creation, semantic attribute constants,
// and span helpers so that every go-cqrs-lite module produces consistent,
// convention-compliant telemetry without duplicating boilerplate.
//
// All instrumentation is opt-in: if no TracerProvider or MeterProvider is
// configured, the global defaults return no-op implementations with zero overhead.
//
// # SDK Setup Recipe
//
// For production deployments, configure trace and meter providers with
// exporters (OTLP, stdout, etc.), resource attributes, and CQRS-optimized views:
//
//	// 1. Create a resource identifying your service
//	res, _ := resource.New(ctx,
//	    resource.WithAttributes(cqrsotel.ServiceResourceAttributes(
//	        "my-service", "1.0.0", "instance-1")...),
//	)
//
//	// 2. Set up propagators (W3C trace context + baggage)
//	otel.SetTextMapPropagator(cqrsotel.NewTextMapPropagator())
//
//	// 3. Create tracer provider with exporter + sampler
//	tp, _ := sdktrace.NewProvider(
//	    sdktrace.WithResource(res),
//	    sdktrace.WithBatchSpanProcessor(otlpExporter),
//	)
//	otel.SetTracerProvider(tp)
//
//	// 4. Create meter provider with CQRS-optimized histogram views
//	mp, _ := sdkmetric.NewMeterProvider(
//	    sdkmetric.WithResource(res),
//	    sdkmetric.WithReader(reader),
//	    sdkmetric.WithView(cqrsotel.NewCQRSViews()...),
//	)
//	otel.SetMeterProvider(mp)
//
// # Distributed Correlation IDs
//
// Use WithCorrelationID and CorrelationIDFromContext to propagate correlation
// IDs across service boundaries via OTel baggage:
//
//	ctx = cqrsotel.WithCorrelationID(ctx, "abc-123")
//	// ... HTTP/gRPC call propagates baggage automatically ...
//	corrID := cqrsotel.CorrelationIDFromContext(ctx)
package otel
