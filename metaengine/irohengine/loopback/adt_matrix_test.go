package loopback_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/loopback/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestLoopbackADTMatrix runs the full adttest.RunMatrix against a loopback-backed
// replicated engine. This closes the middle tier of the three-tier transport
// testing pyramid (InProcess → loopback → QUIC), proving the TCP framing layer
// does not alter ADT semantics.
func TestLoopbackADTMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunMatrix(t, []adttest.Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			Name: "loopback",
			Create: func(t *testing.T) metaengine.Engine {
				tr, err := loopback.New()
				if err != nil {
					t.Fatalf("loopback.New: %v", err)
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

// TestCapabilityConformance verifies the loopback-backed wrapper's Profile()
// declarations against its implemented backend interfaces. The transport tier
// must not change the capability surface the in-process tier declares (the
// wave-4 capability loop noted loopback/quic had matrix tests but no
// conformance wiring — closed 2026-09-06 alongside graph WriteOp convergence).
func TestCapabilityConformance(t *testing.T) {
	t.Parallel()

	tr, err := loopback.New()
	if err != nil {
		t.Fatalf("loopback.New: %v", err)
	}
	t.Cleanup(func() { _ = tr.Close() })

	eng := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithTransport(tr),
	)
	t.Cleanup(func() { _ = eng.Close() })

	adttest.RunCapabilityConformance(t, "iroh-loopback", eng, nil)
}
