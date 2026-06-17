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
//
// # Relationship to event.WithCorrelationID
//
// CQRS has TWO complementary correlation mechanisms — use BOTH:
//
//   - event.WithCorrelationID(id.CorrelationID) — stores a branded ULID in
//     event metadata for domain-level command→event traceability within a
//     single service.
//   - cqrsotel.WithCorrelationID(ctx, string) — stores a string in OTel
//     baggage for infrastructure-level distributed tracing across services.
//
// The domain correlation ID answers: "Which command produced this event?"
// The OTel correlation ID answers: "Which distributed trace does this
// request belong to?"
//
// They use different ID types (branded ULID vs arbitrary string) and serve
// different purposes. Propagate both:
//
//	// In the command handler:
//	ctx = cqrsotel.WithCorrelationID(ctx, traceID.String())
//	evt, _ := event.NewEvent("user.created", aggID, "User", 1, payload,
//	    event.WithCorrelationID(domainCorrID))
package otel
