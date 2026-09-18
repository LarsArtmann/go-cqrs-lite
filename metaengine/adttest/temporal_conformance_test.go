package adttest_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestTemporalConformance_Memory pins the ADR-0141 versioned-cell contract on
// the versioned Memory engine. Every engine module claiming temporal
// capabilities runs the same assertion (see AssertTemporalConformance).
func TestTemporalConformance_Memory(t *testing.T) {
	t.Parallel()

	eng := metaengine.NewMemoryEngineWithVersioning()

	if closer, ok := eng.(interface{ Close() error }); ok {
		defer closer.Close() //nolint:errcheck // test cleanup
	}

	adttest.AssertTemporalConformance(t, eng)
}
