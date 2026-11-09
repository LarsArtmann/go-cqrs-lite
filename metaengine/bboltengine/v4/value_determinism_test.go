package bboltengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestValueDeterminism pins the stored-bytes contract end-to-end. bbolt has
// no RawValueReader, so the harness skips here — its encoder path is pinned
// directly by the keycodec determinism tests. Wired so the byte-level pin
// activates automatically if raw reads ever land.
func TestValueDeterminism(t *testing.T) {
	t.Parallel()

	eng := mustNewBboltEngine(t)

	adttest.AssertDeterministicMapEncoding(t, eng)
}
