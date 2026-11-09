package adttest_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// The cap contract helpers are run from every engine suite, so a defect in the
// HELPER (a bad key extractor, the wrong survivor order) would surface as an
// engine failure and send the reader hunting in the wrong module. These two pin
// both helpers against the reference engine: the contract itself is exercised
// here, where a helper bug is unambiguous.
func TestAssertPlainCapContract_Memory(t *testing.T) {
	t.Parallel()

	adttest.AssertPlainCapContract(t, metaengine.NewMemoryEngine())
}

func TestAssertGroupedCapContract_Memory(t *testing.T) {
	t.Parallel()

	adttest.AssertGroupedCapContract(t, metaengine.NewMemoryEngine())
}
