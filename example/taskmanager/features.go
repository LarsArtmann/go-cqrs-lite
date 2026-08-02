package main

import (
	"crypto/rand"
	"fmt"
	"time"

	otel "go.opentelemetry.io/otel"

	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/signing/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
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

const (
	idempotencyTTL = 10 * time.Minute
	hmacKeyBytes   = 32
)

// setupFeatures wires production-grade middleware and event security.
func setupFeatures(s *Server) error {
	// ── Idempotency: deduplicate commands by ID ─────────────────────
	// Prevents double-execution when a client retries a command due to
	// network failure. Uses the command's minted ID as the dedup key.
	s.idemStore = idempotency.NewMemoryStore(0)

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
		middleware.CommandIdempotency(s.idemStore, idempotencyTTL, nil),
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
//
//nolint:ireturn // factory returning interface for signing abstraction
func newDemoSigner() signing.SignerVerifier {
	key := make([]byte, hmacKeyBytes)
	if _, err := rand.Read(key); err != nil {
		//cqrs-lint:ignore(C009) library code or intentional pattern
		panic("failed to generate signing key: " + err.Error())
	}

	signer, err := signing.NewHMAC(key)
	if err != nil {
		//cqrs-lint:ignore(C009) library code or intentional pattern
		panic("failed to create HMAC signer: " + err.Error())
	}

	return signer
}
