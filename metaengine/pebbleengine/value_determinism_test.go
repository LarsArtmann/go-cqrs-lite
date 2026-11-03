package pebbleengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestValueDeterminism pins the stored-bytes contract end-to-end: the same
// map-typed value must always persist as the same JSON byte sequence
// (json/v2 dropped v1's implicit map-key sorting; keycodec pins it back).
func TestValueDeterminism(t *testing.T) {
	t.Parallel()

	eng := mustNewPebbleEngine(t)

	adttest.AssertDeterministicMapEncoding(t, eng)
}
