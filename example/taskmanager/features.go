package main

import (
	"crypto/rand"
	"fmt"

	otel "go.opentelemetry.io/otel"

	"github.com/larsartmann/go-cqrs-lite/middleware/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	"github.com/larsartmann/go-cqrs-lite/signing/v3"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

// ──────────────────────────────────────────────────────────────────────────
// Features — middleware, observability, and signing.
//
// This file showcases go-cqrs-lite's production-readiness features:
//   - Command middleware: Recovery, Logging, Retry, OTel tracing + metrics
//   - Event bus signing: HMAC-SHA256 tamper detection
//   - OTel setup: one-call provider configuration
//
// Each feature is opt-in. The domain code (decider.go) knows nothing about
// these concerns — they're applied at the composition boundary.
// ──────────────────────────────────────────────────────────────────────────

// setupFeatures wires production-grade middleware and event security.
func setupFeatures(s *Server) error {
	// ── OTel: one-call setup (no-op exporter when no OTLP endpoint) ────
	provider, err := cqrsotel.Setup(
		cqrsotel.WithService("taskmanager", "1.0.0", "dev"),
	)
	if err != nil {
		return err
	}

	s.otelProvider = provider

	tracer := otel.GetTracerProvider().Tracer("taskmanager")
	meter := otel.GetMeterProvider().Meter("taskmanager")

	// ── Command dispatcher middleware ─────────────────────────────────
	//
	// Order matters: Recovery first (outermost) to catch panics from any
	// downstream handler. Then logging, retry, and observability.
	otelBundle, err := middleware.NewOTelBundle(tracer, meter)
	if err != nil {
		return err
	}

	s.CmdDisp.Use(
		middleware.CommandRecovery(),
		middleware.CommandLogging(s.Logger),
		middleware.CommandRetry(middleware.DefaultRetryConfig()),
	)
	s.CmdDisp.Use(otelBundle.Command()...)

	// ── Event bus middleware: HMAC signing ────────────────────────────
	//
	// Sign on publish, verify on consume. Tampered events are rejected
	// before they reach projections — tamper-evident event streams.
	// The bundle stores the bus as event.Publisher; we type-assert to
	// *EventBus (which has UsePublish/Use) to install middleware.
	signer := newDemoSigner()
	s.signer = signer

	if bus, ok := s.Bundle.Publisher.(*cqrswatermill.EventBus); ok {
		if err := bus.UsePublish(signing.SignMiddleware(signer)); err != nil {
			return fmt.Errorf("setup: sign middleware: %w", err)
		}

		if err := bus.Use(signing.VerifyMiddleware(signer)); err != nil {
			return fmt.Errorf("setup: verify middleware: %w", err)
		}
	}

	return nil
}

// newDemoSigner creates an HMAC-SHA256 signer-verifier with a random key.
// In production, load the key from a secret manager (vault, KMS, etc.).
func newDemoSigner() signing.SignerVerifier {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic("failed to generate signing key: " + err.Error())
	}

	signer, err := signing.NewHMAC(key)
	if err != nil {
		panic("failed to create HMAC signer: " + err.Error())
	}

	return signer
}
