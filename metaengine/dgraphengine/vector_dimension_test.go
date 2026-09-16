package dgraphengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

func TestVectorDimensionGuard(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)

	adttest.AssertVectorDimensionGuard(t, eng)
}
