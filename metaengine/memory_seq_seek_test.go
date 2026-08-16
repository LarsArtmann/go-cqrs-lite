package metaengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestMemory_SeqSeekableStreamLog runs the token-resumption conformance
// contract against the memory engine.
func TestMemory_SeqSeekableStreamLog(t *testing.T) {
	t.Parallel()

	enginetest.RunSeqSeekableStreamLogTest(t, metaengine.NewMemoryEngine())
}
