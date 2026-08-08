//go:build cgo

package quic_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/quic/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestQuicADTMatrix runs the full adttest.RunMatrix against a QUIC-backed
// replicated engine. Each factory.Create spins up a real QUIC endpoint,
// wraps a Memory engine in irohengine.Replicated with that transport, and
// registers cleanup via t.Cleanup. This verifies that every CRDT-safe ADT
// produces identical canonical output to the bare Memory engine, ensuring the
// QUIC transport wrapper does not alter semantics.
//
// StreamLogBackend is auto-skipped (replicatedEngine does not implement it —
// stream logs are not CRDT-safe). The remaining 10 ADTs must all pass.
func TestQuicADTMatrix(t *testing.T) {
	adttest.RunMatrix(t, []adttest.Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			Name: "quic",
			Create: func(t *testing.T) metaengine.Engine {
				tr, err := quic.New(quic.WithLocalOnly())
				if err != nil {
					t.Fatalf("quic.New: %v", err)
				}
				t.Cleanup(func() { _ = tr.Close() })
				eng := irohengine.Replicated(
					metaengine.NewMemoryEngine(),
					irohengine.WithTransport(tr),
				)
				t.Cleanup(func() { _ = eng.Close() })
				return eng
			},
		},
	})
}
