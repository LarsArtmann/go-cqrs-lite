package tursoengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

func TestVectorDimensionGuard(t *testing.T) {
	t.Parallel()

	eng, err := tursoengine.New("")
	if err != nil {
		t.Skipf("turso not available: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })

	adttest.AssertVectorDimensionGuard(t, eng)
}
