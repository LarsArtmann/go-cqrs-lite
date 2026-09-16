package metaengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

func TestVectorDimensionGuard_Memory(t *testing.T) {
	t.Parallel()

	eng := metaengine.NewMemoryEngine()
	t.Cleanup(func() { _ = eng.Close() })

	adttest.AssertVectorDimensionGuard(t, eng)
}
