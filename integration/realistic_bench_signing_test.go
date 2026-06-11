//go:build scale

package integration_test

import (
	"slices"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/signing/v2"
)

// ---------------------------------------------------------------------------
// 4. HMAC Signing — sign + verify 10K events
// ---------------------------------------------------------------------------

func BenchmarkRealistic_Signing(b *testing.B) {
	b.ReportAllocs()

	secret := slices.Repeat([]byte("x"), 32)
	signer, err := signing.NewHMAC(secret)
	if err != nil {
		b.Fatalf("NewHMAC: %v", err)
	}

	eventCount := 10_000
	events := make([]event.Event, eventCount)
	for i := range events {
		aggID := id.NewAggregateID()
		events[i] = newRealisticEvent(
			b,
			"OrderCreated",
			aggID,
			1,
			OrderCreated{
				OrderID:   aggID.String(),
				Customer:  "alice",
				Total:     199.99,
				Items:     10,
				Timestamp: time.Now().Format(time.RFC3339),
			},
		)
	}

	sigs := make([]signing.Signature, eventCount)
	for i, evt := range events {
		sigs[i], _ = signer.Sign(evt)
	}

	b.Run("Sign", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			for _, evt := range events {
				if _, err := signer.Sign(evt); err != nil {
					b.Fatalf("Sign: %v", err)
				}
			}
		}

		b.ReportMetric(float64(eventCount), "events")
		b.ReportMetric(float64(b.N*eventCount)/b.Elapsed().Seconds(), "signs/sec")
	})

	b.Run("Verify", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			for i, evt := range events {
				if err := signer.Verify(evt, sigs[i]); err != nil {
					b.Fatalf("Verify: %v", err)
				}
			}
		}

		b.ReportMetric(float64(eventCount), "events")
		b.ReportMetric(float64(b.N*eventCount)/b.Elapsed().Seconds(), "verifies/sec")
	})
}
