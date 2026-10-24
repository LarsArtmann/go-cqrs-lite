package bboltengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestValueEncodeRejection pins the stored-value contract: a
// non-JSON-representable value must fail the write with a Rejection carrying
// keycodec.encode_value, never silently store its %v rendering.
func TestValueEncodeRejection(t *testing.T) {
	t.Parallel()

	eng := mustNewBboltEngine(t)

	adttest.AssertValueEncodeRejection(t, eng)
}
